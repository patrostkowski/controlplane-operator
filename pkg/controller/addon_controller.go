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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ManagedAddonsReconciler struct {
	BaseReconciler
	client.Client
	cp *ControlPlaneClient
}

func (r *ManagedAddonsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("addons", req.NamespacedName)
	var err error

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			// MCP already gone
			return ctrl.Result{}, nil
		}
		log.Error(err, "addons failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	}

	log.Info("Reconciling addons", "version", mcpObj.Spec.Kubernetes.Version)

	r.cp, err = r.getControlPlaneClient(ctx, mcpObj)
	if err != nil {
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileKubeletJoinResources(ctx, mcpObj); err != nil {
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileAddons(ctx, mcpObj); err != nil {
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		return res, nil
	}

	log.Info("Finished reconciling addon")
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

func SetupManagedAddonController(mgr ctrl.Manager) error {
	certPred := predicate.GenerationChangedPredicate{}
	issuerPred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		Named("addons-controller").
		For(&mcpv1alpha1.ManagedControlPlane{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Secret{}).
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
		Complete(&ManagedAddonsReconciler{
			BaseReconciler: BaseReconciler{
				Log:      ctrl.Log.WithName("addon").WithName(mcpv1alpha1.KindManagedControlPlane),
				Recorder: mgr.GetEventRecorderFor("managedaddon"),
				Scheme:   mgr.GetScheme(),
			},
			Client: mgr.GetClient(),
		})
}
