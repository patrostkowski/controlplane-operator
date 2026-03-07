// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

const RequeueAfterFailure = 10 * time.Second // RequeueAfterFailure specifies the duration to wait before requeueing a failed reconciliation.

const ManagedControlPlaneFinalizer = "controlplane.patrostkowski.dev/finalizer" // ManagedControlPlaneFinalizer is the finalizer used for ManagedControlPlane objects.

// ManagedControlPlaneReconciler reconciles ManagedControlPlane objects.
type ManagedControlPlaneReconciler struct {
	BaseReconciler
	client.Client
	manager.Manager
}

// Reconcile performs the main reconciliation loop for ManagedControlPlane objects.
func (r *ManagedControlPlaneReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("control-plane", req.NamespacedName)

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			// MCP already gone
			return ctrl.Result{}, nil
		}
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Kubernetes.Version)

	if !mcpObj.ObjectMeta.DeletionTimestamp.IsZero() {
		// If finalizer not present, nothing to do
		if !controllerutil.ContainsFinalizer(mcpObj, ManagedControlPlaneFinalizer) {
			return ctrl.Result{}, nil
		}

		log.Info("ManagedControlPlane is being deleted, deleting child resources and removing finalizer")

		controllerutil.RemoveFinalizer(mcpObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, mcpObj); err != nil {
			log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
			return ctrl.Result{}, err
		}

		log.Info("Reconcile finished with deleted resources", "resource", mcpObj.GetName())
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(mcpObj, ManagedControlPlaneFinalizer) {
		log.Info("Adding finalizer to ManagedControlPlane")
		controllerutil.AddFinalizer(mcpObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, mcpObj); err != nil {
			log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	cc := cluster.NewClusterContext(mcpObj, r.Log)

	components := []Component{
		&APIServerServiceComponent{r: r},
		&PKIComponent{r: r},
		&ETCDComponent{r: r},
		&APIServerComponent{r: r},
		&ControllerManagerComponent{r: r},
		&SchedulerComponent{r: r},
		&AdminConfigComponent{r: r},
	}

	for _, c := range components {
		res, err := c.Reconcile(ctx, cc)
		if err != nil {
			_ = r.statusFailed(ctx, mcpObj, c.FailedMessage())
			log.Error(err, "component failed", "component", c.Name(), "after", RequeueAfterFailure)
			return ctrl.Result{}, err
		}
		if !res.IsZero() {
			_ = r.statusWaiting(ctx, mcpObj, c.WaitingMessage())
			return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
		}
	}

	_ = r.statusReady(ctx, mcpObj)
	log.Info("Finished reconciling managed controlplane")
	return ctrl.Result{}, nil
}

// SetupManagedControlPlaneController sets up the ManagedControlPlane controller with the Kubernetes manager.
func SetupManagedControlPlaneController(mgr mcmanager.Manager) error {
	certPred := predicate.GenerationChangedPredicate{}
	issuerPred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr.GetLocalManager()).
		Named("control-plane-controller").
		For(&mcpv1alpha1.ManagedControlPlane{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&certmanagerv1.Certificate{}, builder.WithPredicates(certPred)).
		Owns(&certmanagerv1.Issuer{}, builder.WithPredicates(issuerPred)).
		WithOptions(
			controller.Options{
				// MaxConcurrentReconciles: 1,
				RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
					5*time.Second,
					60*time.Second,
				),
			},
		).
		Complete(&ManagedControlPlaneReconciler{
			Manager: mgr.GetLocalManager(),
			BaseReconciler: BaseReconciler{
				Log:      ctrl.Log.WithName("controller").WithName(mcpv1alpha1.KindManagedControlPlane),
				Recorder: mgr.GetLocalManager().GetEventRecorder("managedcontrolplane"),
				Scheme:   mgr.GetLocalManager().GetScheme(),
			},
			Client: mgr.GetLocalManager().GetClient(),
		})
}
