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
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/controllermanager"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedControllerManagerReconciler struct {
	controlplane.BaseReconciler
}

func (r *ManagedControllerManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedcontrollermanager", req.NamespacedName)

	cmObj := &mcpv1alpha1.ManagedControllerManager{}
	if err := r.Get(ctx, req.NamespacedName, cmObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedControllerManager")

	if err := r.UpdateCondition(ctx, cmObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReconciling,
			Status:  metav1.ConditionFalse,
			Reason:  controlplane.ReasonReconciling,
			Message: controlplane.MessageReconciling,
		},
		controlplane.Status{
			Ready:   false,
			Message: "reconciling",
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	resources := controllermanager.Resources(cmObj)

	if err := r.ensureResources(ctx, cmObj, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		_ = r.UpdateCondition(ctx, cmObj,
			controlplane.Conditions{
				Type:    controlplane.ConditionWaitingForResource,
				Status:  metav1.ConditionFalse,
				Reason:  controlplane.ReasonWaitingForResources,
				Message: controlplane.MessageWaitingForResources,
			},
			controlplane.Status{
				Ready:   false,
				Message: "awaiting all ready",
			},
		)
		log.Info("requeueing reconcile for Controller Manager controller")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.UpdateCondition(ctx, cmObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  controlplane.ReasonAllResourcesReady,
			Message: controlplane.MessageAllResourcesReady,
		},
		controlplane.Status{
			Ready:   true,
			Message: "all ready",
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling ManagedControllerManager")
	return ctrl.Result{}, nil
}

func (r *ManagedControllerManagerReconciler) ensureResources(
	ctx context.Context,
	cmObj *mcpv1alpha1.ManagedControllerManager,
	resources []client.Object,
	log logr.Logger,
) error {

	for _, desired := range resources {
		log.Info("Ensuring ControllerManager resource", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", desired.GetName())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, cmObj, desired, log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *corev1.ConfigMap:
				d := desired.(*corev1.ConfigMap)
				o.Data = d.Data
			case *appsv1.Deployment:
				d := desired.(*appsv1.Deployment)
				o.Spec = d.Spec
			}
			return nil
		})
		if err != nil {
			log.Error(err, "failed to ensure ControllerManager resource", "name", desired.GetName())
			return err
		}
	}

	return nil
}

func (r *ManagedControllerManagerReconciler) checkResourcesReady(
	ctx context.Context,
	resources []client.Object,
	log logr.Logger,
) (bool, error) {

	allReady := true

	for _, desired := range resources {
		key := client.ObjectKey{Namespace: desired.GetNamespace(), Name: desired.GetName()}

		switch desired.(type) {
		case *corev1.ConfigMap:
			cm := &corev1.ConfigMap{}
			if err := r.Get(ctx, key, cm); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to get ConfigMap", "name", key.Name)
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

func SetupManagedControllerManagerReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedControllerManager{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedControllerManagerReconciler{
			BaseReconciler: controlplane.BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedControllerManager"),
				Recorder: mgr.GetEventRecorderFor("managedcontrollermanager"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
