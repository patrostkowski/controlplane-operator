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

package cluster

import (
	"testing"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdminConfig_Names(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"}}
	cc := NewClusterContext(mcp, logr.Logger{})

	a := cc.Admin()

	if got, want := a.KubeconfigSecret(), "demo-admin-kubeconfig"; got != want {
		t.Fatalf("KubeconfigSecret()=%q want %q", got, want)
	}
	if got, want := a.KubeconfigDataKey(), "kubeconfig"; got != want {
		t.Fatalf("KubeconfigDataKey()=%q want %q", got, want)
	}
	if got, want := a.ClientSecret(), "demo-admin-client"; got != want {
		t.Fatalf("ClientSecret()=%q want %q", got, want)
	}
}

func TestManagedAddonsConfig_Names(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"}}
	cc := NewClusterContext(mcp, logr.Logger{})

	m := cc.ManagedAddons()

	if got, want := m.KonnectivityAgentNamespace(), "kube-system"; got != want {
		t.Fatalf("KonnectivityAgentNamespace()=%q want %q", got, want)
	}
	if got, want := m.BootstrapTokenMgmtSecretName(), "demo-bootstrap-token"; got != want {
		t.Fatalf("BootstrapTokenMgmtSecretName()=%q want %q", got, want)
	}
}

func TestKonnectivityConfig_Names(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"}}
	cc := NewClusterContext(mcp, logr.Logger{})

	k := cc.Konnectivity()

	if got, want := k.AgentName(), "konnectivity-agent"; got != want {
		t.Fatalf("AgentName()=%q want %q", got, want)
	}
	if got, want := k.CASecret(), "demo-konnectivity-ca"; got != want {
		t.Fatalf("CASecret()=%q want %q", got, want)
	}
	if got, want := k.ServerTLSSecret(), "demo-konnectivity-server-tls"; got != want {
		t.Fatalf("ServerTLSSecret()=%q want %q", got, want)
	}
	if got, want := k.AgentTLSSecret(), "demo-konnectivity-agent-tls"; got != want {
		t.Fatalf("AgentTLSSecret()=%q want %q", got, want)
	}
}
