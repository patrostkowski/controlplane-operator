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

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	applier "github.com/patrostkowski/controlplane-operator/pkg/controller/apply"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APIServer struct {
	*applier.Applier
	mcp *mcpv1alpha1.ManagedControlPlane
	log logr.Logger
}

func NewAPIServer(mcp *mcpv1alpha1.ManagedControlPlane, k8s client.Client, scheme *runtime.Scheme, log logr.Logger) *APIServer {
	return &APIServer{
		Applier: applier.NewApplier(k8s, scheme, log, fieldOwner),
		mcp:     mcp,
		log:     log.WithName("apiserver"),
	}
}

func (a *APIServer) Ensure(ctx context.Context, resources []client.Object) error {
	return a.Apply(ctx, a.mcp, resources...)
}

func (a *APIServer) endpointManifests() []client.Object {
	return apiserver.EndpointResources(a.mcp)
}

func (a *APIServer) workloadManifests() []client.Object {
	return apiserver.WorkloadResources(a.mcp)
}

func (a *APIServer) tryEndpointAddress(ctx context.Context) (string, error) {
	svc := &corev1.Service{}
	if err := a.Get(ctx, client.ObjectKey{Namespace: a.mcp.Namespace, Name: apiserver.KubeAPIServerSvcName}, svc); err != nil {
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
