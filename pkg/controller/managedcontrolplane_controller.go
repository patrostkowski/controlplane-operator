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
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const RequeueAfterFailure = 10 * time.Second

const ManagedControlPlaneFinalizer = "controlplane.patrostkowski.dev/finalizer"

type ManagedControlPlaneReconciler struct {
	BaseReconciler
	cp *controlplane.ControlPlaneClient
}

func (r *ManagedControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedcontrolplane", req.NamespacedName)
	var err error

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			// MCP already gone
			return ctrl.Result{}, nil
		}
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Version)

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

	// API Server Service
	if res, err := r.reconcileAPIServiceSvc(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageAPIServerSvcFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageAPIServerSvcWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// PKI
	if res, err := r.reconcilePKI(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessagePKIFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessagePKIWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// ETCD
	if res, err := r.reconcileETCD(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageETCDFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageETCDWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// APIServer
	if res, err := r.reconcileAPIServer(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageAPIServerFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageAPIServerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// ControllerManager
	if res, err := r.reconcileControllerManager(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageControllerManagerFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageControllerManagerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// Scheduler
	if res, err := r.reconcileScheduler(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageSchedulerFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageSchedulerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileAdminConfig(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessagePKIFailed)
		log.Error(err, "failed to ensure admin kubeconfig")
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageSchedulerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	r.cp, err = r.controlPlaneClient(ctx, mcpObj)
	if err != nil {
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileKubeletJoinResources(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageKubeResourcesFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageKubeResourcesWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileAddons(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageAddonsFailed)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageAddonsWaiting)
		return res, nil
	}

	_ = r.statusReady(ctx, mcpObj)
	log.Info("Finished reconciling ManagedControlPlane")
	return ctrl.Result{}, nil
}

func SetupManagedControlPlaneController(mgr ctrl.Manager) error {
	certPred := predicate.GenerationChangedPredicate{}
	issuerPred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
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
			BaseReconciler: BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName(mcpv1alpha1.KindManagedControlPlane),
				Recorder: mgr.GetEventRecorderFor("managedcontrolplane"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
