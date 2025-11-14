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
	mcpv1alpha1 "github.com/patrostkowski/operator-template/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/operator-template/pkg/controlplane/controllermanager"
	"github.com/patrostkowski/operator-template/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ManagedControllerManagerReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
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

	resources := controllermanager.Resources(cmObj)

	if err := r.ensureResources(ctx, cmObj, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateReadyCondition(ctx, cmObj, allReady); err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for ManagedControllerManager until Deployment is Ready")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("Finished reconciling ManagedControllerManager")
	return reconcile.Result{}, nil
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
			if !isDeploymentReady(deploy) {
				log.Info("Deployment not ready", "name", deploy.Name, "readyReplicas", deploy.Status.ReadyReplicas)
				allReady = false
			}
		}
	}

	return allReady, nil
}

func (r *ManagedControllerManagerReconciler) updateReadyCondition(
	ctx context.Context,
	cmObj *mcpv1alpha1.ManagedControllerManager,
	allReady bool,
) error {
	var cond metav1.Condition

	if allReady {
		cond = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReady",
			Message:            "kube-controller-manager Deployment is Ready",
			ObservedGeneration: cmObj.Generation,
		}
	} else {
		cond = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "WaitingForDeployment",
			Message:            "Waiting for kube-controller-manager Deployment to become Ready",
			ObservedGeneration: cmObj.Generation,
		}
	}

	if apimeta.SetStatusCondition(&cmObj.Status.Conditions, cond) {
		return r.Status().Update(ctx, cmObj)
	}
	return nil
}

func SetupManagedControllerManagerReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedControllerManager{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedControllerManagerReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controller").WithName("ManagedControllerManager"),
			Recorder: mgr.GetEventRecorderFor("managedcontrollermanager"),
			Scheme:   mgr.GetScheme(),
		})
}
