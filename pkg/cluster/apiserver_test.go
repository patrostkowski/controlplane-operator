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

func TestAPIServerConfig_NamesAndSecrets(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	cc := NewClusterContext(mcp, logr.Logger{})
	a := cc.APIServer()

	// resource names
	if got, want := a.ServiceName(), "demo-apiserver"; got != want {
		t.Fatalf("ServiceName()=%q want %q", got, want)
	}
	if got, want := a.DeploymentName(), "demo-apiserver"; got != want {
		t.Fatalf("DeploymentName()=%q want %q", got, want)
	}
	if got, want := a.KonnectivityConfigMapName(), "demo-konnectivity-egress-selector"; got != want {
		t.Fatalf("KonnectivityConfigMapName()=%q want %q", got, want)
	}

	// secret wiring (via PKI facade)
	if got, want := a.ClientCASecret(), "demo-managed-ca"; got != want {
		t.Fatalf("ClientCASecret()=%q want %q", got, want)
	}
	if got, want := a.ServingTLSSecret(), "demo-apiserver-tls"; got != want {
		t.Fatalf("ServingTLSSecret()=%q want %q", got, want)
	}
	if got, want := a.EtcdCASecret(), "demo-etcd-ca"; got != want {
		t.Fatalf("EtcdCASecret()=%q want %q", got, want)
	}
	if got, want := a.EtcdClientSecret(), "demo-apiserver-etcd-client"; got != want {
		t.Fatalf("EtcdClientSecret()=%q want %q", got, want)
	}
	if got, want := a.KubeletClientSecret(), "demo-apiserver-kubelet-client"; got != want {
		t.Fatalf("KubeletClientSecret()=%q want %q", got, want)
	}
	if got, want := a.SASignerSecret(), "demo-sa-signer"; got != want {
		t.Fatalf("SASignerSecret()=%q want %q", got, want)
	}
	if got, want := a.FrontProxyCASecret(), "demo-front-proxy-ca"; got != want {
		t.Fatalf("FrontProxyCASecret()=%q want %q", got, want)
	}
	if got, want := a.FrontProxyClientSecret(), "demo-front-proxy-client"; got != want {
		t.Fatalf("FrontProxyClientSecret()=%q want %q", got, want)
	}
	if got, want := a.KonnectivityCASecret(), "demo-konnectivity-ca"; got != want {
		t.Fatalf("KonnectivityCASecret()=%q want %q", got, want)
	}
	if got, want := a.KonnectivityServerSecret(), "demo-konnectivity-server-tls"; got != want {
		t.Fatalf("KonnectivityServerSecret()=%q want %q", got, want)
	}
}
