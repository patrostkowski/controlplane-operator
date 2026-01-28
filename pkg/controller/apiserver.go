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

type APIServerServiceComponent struct {
	r *ManagedControlPlaneReconciler
}

func (c *APIServerServiceComponent) Name() string {
	return "apiserver-service"
}

func (c *APIServerServiceComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileAPIServiceSvc(ctx, cc)
}

func (c *APIServerServiceComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerSvcWaiting
}

func (c *APIServerServiceComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerSvcFailed
}

type APIServerComponent struct {
	r *ManagedControlPlaneReconciler
}

func (c *APIServerComponent) Name() string {
	return "apiserver"
}

func (c *APIServerComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileAPIServer(ctx, cc)
}

func (c *APIServerComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerWaiting
}

func (c *APIServerComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageAPIServerFailed
}

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
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	mcp := cc.MCP
	log := r.Log.WithValues("api-endpoint", mcp.Namespace)

	endpoint := apiserver.NewEndpointBuilder(cc)
	if err := r.apply(ctx, r.Client, r.applyOpts(mcp), endpoint.Objects()...); err != nil {
		return ctrl.Result{}, err
	}

	addr, err := r.tryEndpointAddress(ctx, cc.Names.APIServerServiceName(), mcp.Namespace)
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
		if err := r.updateMCPAddress(ctx, mcp, addr); err != nil {
			log.Error(err, "failed to update address")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileAPIServer(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	mcp := cc.MCP

	workload := apiserver.NewWorkloadBuilder(cc)
	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(mcp),
		workload.Objects()...,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
