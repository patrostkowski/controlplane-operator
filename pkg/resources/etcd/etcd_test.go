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

package etcd

import (
	"testing"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildService_Etcd(t *testing.T) {
	cp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-mcp",
			Namespace: "mcp",
		},
	}

	svc := buildService(cp)

	// metadata
	if svc.Name != nameEtcd {
		t.Fatalf("expected service name %q, got %q", nameEtcd, svc.Name)
	}
	if svc.Namespace != cp.Namespace {
		t.Fatalf("expected service namespace %q, got %q", cp.Namespace, svc.Namespace)
	}

	// labels
	wantLabels := map[string]string{appLabelKey: appLabelVal}
	if svc.Labels == nil || svc.Labels[appLabelKey] != appLabelVal {
		t.Fatalf("expected labels %#v, got %#v", wantLabels, svc.Labels)
	}

	// selector should match labels (critical for endpoints)
	if svc.Spec.Selector == nil || svc.Spec.Selector[appLabelKey] != appLabelVal {
		t.Fatalf("expected selector %#v, got %#v", wantLabels, svc.Spec.Selector)
	}

	// headless service
	if svc.Spec.ClusterIP != "None" {
		t.Fatalf("expected ClusterIP %q (headless), got %q", "None", svc.Spec.ClusterIP)
	}

	// ports
	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d: %#v", len(svc.Spec.Ports), svc.Spec.Ports)
	}

	expectPort(t, svc.Spec.Ports, "client", clientPort)
	expectPort(t, svc.Spec.Ports, "peer", peerPort)

	// protocol defaults to TCP if empty, don't be strict here
	for _, p := range svc.Spec.Ports {
		if p.Protocol != "" && p.Protocol != corev1.ProtocolTCP {
			t.Fatalf("expected protocol TCP/empty-default, got %q for port %#v", p.Protocol, p)
		}
	}
}

func expectPort(t *testing.T, ports []corev1.ServicePort, name string, port int32) {
	t.Helper()
	for _, p := range ports {
		if p.Name == name {
			if p.Port != port {
				t.Fatalf("port %q: expected %d, got %d", name, port, p.Port)
			}
			return
		}
	}
	t.Fatalf("expected port %q not found; ports=%#v", name, ports)
}
