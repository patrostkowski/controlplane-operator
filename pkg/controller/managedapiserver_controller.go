// Copyright 2025 controlplane.patrostkowski.dev
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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/apiserver"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedAPIServerReconciler struct {
	controlplane.BaseReconciler
}

func (r *ManagedAPIServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedapiserver", req.NamespacedName)

	apiObj := &mcpv1alpha1.ManagedAPIServer{}
	if err := r.Get(ctx, req.NamespacedName, apiObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// cond := apimeta.FindStatusCondition(apiObj.Status.Conditions, string(controlplane.ConditionReady))
	// if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != string(controlplane.ReasonReconciling) {
	// 	if err := r.UpdateCondition(ctx, apiObj, controlplane.Conditions{
	// 		Type:    controlplane.ConditionReady,
	// 		Status:  metav1.ConditionFalse,
	// 		Reason:  controlplane.ReasonReconciling,
	// 		Message: controlplane.MessageReconciling,
	// 	}); err != nil {
	// 		return ctrl.Result{}, err
	// 	}
	// }

	log.Info("Reconciling ManagedAPIServer")

	resources := apiserver.Resources(apiObj)

	if err := r.ensureResources(ctx, apiObj, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// if err := r.updateCondition(ctx, apiObj, allReady); err != nil {
	// 	return ctrl.Result{}, err
	// }

	if !allReady {
		log.Info("requeueing reconcile for ManagedAPIServer until Deployment is Ready")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("Finished reconciling ManagedAPIServer")
	return ctrl.Result{}, nil
}

func (r *ManagedAPIServerReconciler) ensureResources(
	ctx context.Context,
	apiObj *mcpv1alpha1.ManagedAPIServer,
	resources []client.Object,
	log logr.Logger,
) error {
	for _, desired := range resources {
		log.Info("Ensuring APIServer resource", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", desired.GetName())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, apiObj, desired, log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *corev1.Service:
				d := desired.(*corev1.Service)
				// Preserve ClusterIP on updates (immutable)
				if o.CreationTimestamp.IsZero() {
					o.Spec = d.Spec
				} else {
					o.Spec.Ports = d.Spec.Ports
					o.Spec.Selector = d.Spec.Selector
					o.Spec.Type = d.Spec.Type
				}
			case *appsv1.Deployment:
				d := desired.(*appsv1.Deployment)
				o.Spec = d.Spec
			}
			return nil
		})
		if err != nil {
			log.Error(err, "failed to ensure APIServer resource", "name", desired.GetName())
			return err
		}
	}
	return nil
}

func (r *ManagedAPIServerReconciler) checkResourcesReady(
	ctx context.Context,
	resources []client.Object,
	log logr.Logger,
) (bool, error) {
	allReady := true

	for _, desired := range resources {
		key := client.ObjectKey{Namespace: desired.GetNamespace(), Name: desired.GetName()}

		switch desired.(type) {
		case *corev1.Service:
			// nothing to wait on; just ensure it exists
			svc := &corev1.Service{}
			if err := r.Get(ctx, key, svc); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to get Service", "name", key.Name)
				}
				allReady = false
			}

		case *appsv1.Deployment:
			deploy := &appsv1.Deployment{}
			if err := r.Get(ctx, key, deploy); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to get Deployment", "name", key.Name)
				}
				allReady = false
				continue
			}

			if !r.IsDeploymentReady(deploy) {
				log.Info("Deployment not ready", "name", deploy.Name, "readyReplicas", deploy.Status.ReadyReplicas)
				allReady = false
			}
		}
	}

	return allReady, nil
}

// func (r *ManagedAPIServerReconciler) updateCondition(
// 	ctx context.Context,
// 	apiObj *mcpv1alpha1.ManagedAPIServer,
// 	allReady bool,
// ) error {

// 	if allReady {
// 		return r.UpdateCondition(ctx, apiObj, controlplane.Conditions{
// 			Type:    controlplane.ConditionReady,
// 			Status:  metav1.ConditionTrue,
// 			Reason:  controlplane.ReasonDeploymentReady,
// 			Message: controlplane.MessageDeploymentReady,
// 		})
// 	}

// 	return r.UpdateCondition(ctx, apiObj, controlplane.Conditions{
// 		Type:    controlplane.ConditionReady,
// 		Status:  metav1.ConditionFalse,
// 		Reason:  controlplane.ReasonWaitingForDeployment,
// 		Message: controlplane.MessageWaitingForDeployment,
// 	})
// }

func SetupManagedAPIServerReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedAPIServer{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedAPIServerReconciler{
			BaseReconciler: controlplane.BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedAPIServer"),
				Recorder: mgr.GetEventRecorderFor("managedapiserver"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
