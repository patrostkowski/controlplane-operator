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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcileAPIServer(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("apiserver", mcp.GetObjectMeta().GetNamespace())

	log.Info("Reconciling API Server")

	resources := apiserver.WorkloadResources(mcp)

	if err := r.ensureAPIServerResources(ctx, mcp, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkAPIServerResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for API Server")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Finished reconciling API Server")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) ensureAPIServerResources(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	resources []client.Object,
	log logr.Logger,
) error {
	for _, desired := range resources {
		log.Info("Ensuring APIServer resource", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", desired.GetName())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcp, desired, log, func(obj client.Object) error {
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

func (r *ManagedControlPlaneReconciler) checkAPIServerResourcesReady(
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

func (r *ManagedControlPlaneReconciler) reconcileAPIServiceSvc(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("api-endpoint", mcp.Namespace)
	log.Info("Reconciling API Service (endpoint)")

	resources := apiserver.EndpointResources(mcp) // Service only

	if err := r.ensureAPIServerResources(ctx, mcp, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	apiSvc := &corev1.Service{}
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: mcp.Namespace,
		Name:      apiserver.KubeAPIServerSvcName,
	}, apiSvc); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("API Service not created yet")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	var addr string
	if len(apiSvc.Status.LoadBalancer.Ingress) > 0 {
		ing := apiSvc.Status.LoadBalancer.Ingress[0]
		if ing.IP != "" {
			addr = ing.IP
		} else if ing.Hostname != "" {
			addr = ing.Hostname
		}
	}

	if addr == "" {
		r.Log.Info("Waiting for API Service LoadBalancer address", "service", apiSvc.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.UpdateMCPAddress(ctx, mcp, addr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
