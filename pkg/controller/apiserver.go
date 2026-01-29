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
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/controller/state"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// APIServerServiceComponent reconciles the Kubernetes API server service.
type APIServerServiceComponent struct {
	r *ManagedControlPlaneReconciler
}

// Name returns the name of the API server service component.
func (c *APIServerServiceComponent) Name() string {
	return "apiserver-service"
}

// Reconcile reconciles the API server service.
func (c *APIServerServiceComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileAPIServiceSvc(ctx, cc)
}

// WaitingMessage returns the waiting message for the API server service.
func (c *APIServerServiceComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerSvcWaiting
}

// FailedMessage returns the failed message for the API server service.
func (c *APIServerServiceComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerSvcFailed
}

// APIServerComponent reconciles the Kubernetes API server deployment.
type APIServerComponent struct {
	r *ManagedControlPlaneReconciler
}

// Name returns the name of the API server component.
func (c *APIServerComponent) Name() string {
	return "apiserver"
}

// Reconcile reconciles the API server deployment.
func (c *APIServerComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileAPIServer(ctx, cc)
}

// WaitingMessage returns the waiting message for the API server.
func (c *APIServerComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerWaiting
}

// FailedMessage returns the failed message for the API server.
func (c *APIServerComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerFailed
}

// tryEndpointAddress attempts to retrieve the endpoint address for a given service.
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

// reconcileAPIServiceSvc reconciles the API server service for the control plane.
func (r *ManagedControlPlaneReconciler) reconcileAPIServiceSvc(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	log := r.Log.WithValues("api-endpoint", cc.Namespace())

	endpoint := apiserver.NewEndpointBuilder(cc)
	if err := r.apply(ctx, r.Client, r.applyOpts(cc.Owner()), endpoint.Objects()...); err != nil {
		return ctrl.Result{}, err
	}

	addr, err := r.tryEndpointAddress(ctx, cc.APIServer().ServiceName(), cc.Namespace())
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

	if cc.GetManagedControlPlaneStatus().Address != addr {
		if err := r.updateMCPAddress(ctx, cc.MCP(), addr); err != nil {
			log.Error(err, "failed to update address")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// reconcileAPIServer reconciles the API server deployment for the control plane.
func (r *ManagedControlPlaneReconciler) reconcileAPIServer(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	workload := apiserver.NewWorkloadBuilder(cc)
	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(cc.Owner()),
		workload.Objects()...,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
