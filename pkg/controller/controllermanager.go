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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/controllermanager"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcileControllerManager(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("controllermanager", mcp.GetObjectMeta().GetNamespace())

	log.Info("Reconciling Controller Manager")

	resources := controllermanager.Resources(mcp)

	if err := r.ensureControllerManagerResources(ctx, mcp, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkControllerManagerResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for Controller Manager")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Finished reconciling Controller Manager")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) ensureControllerManagerResources(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	resources []client.Object,
	log logr.Logger,
) error {

	for _, desired := range resources {
		log.Info("Ensuring ControllerManager resource", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", desired.GetName())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcp, desired, log, func(mcp client.Object) error {
			switch o := mcp.(type) {
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

func (r *ManagedControlPlaneReconciler) checkControllerManagerResourcesReady(
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
