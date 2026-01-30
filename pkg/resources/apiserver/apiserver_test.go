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

package apiserver

import (
	"strings"
	"testing"

	"github.com/go-logr/logr"
	testutil "github.com/patrostkowski/controlplane-operator/internal/test"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func findConfigMapByName(objs []client.Object, name string) *corev1.ConfigMap {
	for _, o := range objs {
		cm, ok := o.(*corev1.ConfigMap)
		if ok && cm.Name == name {
			return cm
		}
	}
	return nil
}

func TestEndpointBuilder_Service(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
				Networking: &mcpv1alpha1.NetworkingSpec{
					ServiceCIDR: "10.96.0.0/12",
				},
			},
		},
		Status: mcpv1alpha1.ManagedControlPlaneStatus{Address: "192.0.2.10"},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	a := cc.APIServer()

	objs := NewEndpointBuilder(cc).Objects()
	if len(objs) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objs))
	}
	svc := testutil.MustFindService(t, objs)

	if svc.Name != a.ServiceName() {
		t.Fatalf("service name=%q want %q", svc.Name, a.ServiceName())
	}
	if svc.Namespace != mcp.Namespace {
		t.Fatalf("service namespace=%q want %q", svc.Namespace, mcp.Namespace)
	}
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Fatalf("service type=%q want %q", svc.Spec.Type, corev1.ServiceTypeLoadBalancer)
	}
	testutil.ExpectPort(t, svc.Spec.Ports, "https", securePort)
	testutil.ExpectPort(t, svc.Spec.Ports, "grpc", grpcPort)
}

func TestWorkloadBuilder_DeploymentAndKonnectivityConfig(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
				Networking: &mcpv1alpha1.NetworkingSpec{
					ServiceCIDR: "10.96.0.0/12",
					PodCIDR:     "10.244.0.0/16",
				},
			},
		},
		Status: mcpv1alpha1.ManagedControlPlaneStatus{Address: "192.0.2.10"},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	a := cc.APIServer()

	objs := NewWorkloadBuilder(cc).Objects()

	// There must be a Deployment.
	var dep *appsv1.Deployment
	for _, o := range objs {
		if d, ok := o.(*appsv1.Deployment); ok {
			dep = d
			break
		}
	}
	if dep == nil {
		t.Fatalf("expected *appsv1.Deployment in objects")
	}
	if dep.Name != a.DeploymentName() {
		t.Fatalf("deployment name=%q want %q", dep.Name, a.DeploymentName())
	}
	if dep.Namespace != mcp.Namespace {
		t.Fatalf("deployment namespace=%q want %q", dep.Namespace, mcp.Namespace)
	}
	// Should have 2 containers (kube-apiserver + konnectivity-server).
	if got := len(dep.Spec.Template.Spec.Containers); got != 2 {
		t.Fatalf("expected 2 containers, got %d", got)
	}

	// There must be a konnectivity ConfigMap with expected key.
	cm := findConfigMapByName(objs, a.KonnectivityConfigMapName())
	if cm == nil {
		t.Fatalf("expected ConfigMap %q", a.KonnectivityConfigMapName())
	}
	val := strings.TrimSpace(cm.Data[konnectivityConfigMapKey])
	if val == "" {
		t.Fatalf("expected configmap %q data[%q] to be non-empty", cm.Name, konnectivityConfigMapKey)
	}
	// Quick sanity: config looks like EgressSelectorConfiguration.
	if !strings.Contains(val, egressSelectorKind) {
		t.Fatalf("expected konnectivity config to mention %q; got:\n%s", egressSelectorKind, val)
	}
}
