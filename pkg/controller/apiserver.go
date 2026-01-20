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

	corev1 "k8s.io/api/core/v1"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) tryEndpointAddress(ctx context.Context, name, namespace string) (string, error) {
	svc := &corev1.Service{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc); err != nil {
		return "", err
	}
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return "", nil
	}
	ing := svc.Status.LoadBalancer.Ingress[0]
	if ing.IP != "" {
		return ing.IP, nil
	}
	if ing.Hostname != "" {
		return ing.Hostname, nil
	}
	return "", nil
}

func (r *ManagedControlPlaneReconciler) reconcileAPIServiceSvc(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("api-endpoint", mcp.Namespace)

	if err := apply(ctx, r.Client, r.Scheme, r.applyOpts(mcp), apiserver.EndpointResources(mcp)...); err != nil {
		return ctrl.Result{}, err
	}

	addr, err := r.tryEndpointAddress(ctx, common.KubeAPIServerName, mcp.Namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to read API Service address")
		return ctrl.Result{}, err
	}

	if addr == "" {
		log.Info("API server address is empty, requeue")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if mcp.Status.Address != addr {
		if err := r.UpdateMCPAddress(ctx, mcp, addr); err != nil {
			log.Error(err, "failed to update address")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileAPIServer(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	if err := apply(ctx, r.Client, r.Scheme, r.applyOpts(mcp), apiserver.WorkloadResources(mcp)...); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
