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

func TestPKI_IssuerNames(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	cc := NewClusterContext(mcp, logr.Logger{})

	i := cc.PKI().Issuer()

	if got, want := i.SelfSigned(), "demo-selfsigned"; got != want {
		t.Fatalf("SelfSigned()=%q want %q", got, want)
	}
	if got, want := i.CA(), "demo-ca-issuer"; got != want {
		t.Fatalf("CA()=%q want %q", got, want)
	}
	if got, want := i.EtcdSelfSigned(), "demo-etcd-selfsigned"; got != want {
		t.Fatalf("EtcdSelfSigned()=%q want %q", got, want)
	}
	if got, want := i.EtcdCA(), "demo-etcd-ca-issuer"; got != want {
		t.Fatalf("EtcdCA()=%q want %q", got, want)
	}
	if got, want := i.FrontProxySelfSigned(), "demo-front-proxy-selfsigned"; got != want {
		t.Fatalf("FrontProxySelfSigned()=%q want %q", got, want)
	}
	if got, want := i.FrontProxyCA(), "demo-front-proxy-ca-issuer"; got != want {
		t.Fatalf("FrontProxyCA()=%q want %q", got, want)
	}
	if got, want := i.KonnectivitySelfSigned(), "demo-konnectivity-selfsigned"; got != want {
		t.Fatalf("KonnectivitySelfSigned()=%q want %q", got, want)
	}
	if got, want := i.KonnectivityCA(), "demo-konnectivity-ca-issuer"; got != want {
		t.Fatalf("KonnectivityCA()=%q want %q", got, want)
	}
}

func TestPKI_CertificateSecretNames(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	cc := NewClusterContext(mcp, logr.Logger{})

	c := cc.PKI().Certificate()

	// root CAs
	if got, want := c.ManagedCA(), "demo-managed-ca"; got != want {
		t.Fatalf("ManagedCA()=%q want %q", got, want)
	}
	if got, want := c.EtcdCA(), "demo-etcd-ca"; got != want {
		t.Fatalf("EtcdCA()=%q want %q", got, want)
	}
	if got, want := c.FrontProxyCA(), "demo-front-proxy-ca"; got != want {
		t.Fatalf("FrontProxyCA()=%q want %q", got, want)
	}
	if got, want := c.KonnectivityCA(), "demo-konnectivity-ca"; got != want {
		t.Fatalf("KonnectivityCA()=%q want %q", got, want)
	}

	// service account signer
	if got, want := c.SASigner(), "demo-sa-signer"; got != want {
		t.Fatalf("SASigner()=%q want %q", got, want)
	}

	// apiserver
	if got, want := c.APIServerTLS(), "demo-apiserver-tls"; got != want {
		t.Fatalf("APIServerTLS()=%q want %q", got, want)
	}
	if got, want := c.APIServerKubeletClient(), "demo-apiserver-kubelet-client"; got != want {
		t.Fatalf("APIServerKubeletClient()=%q want %q", got, want)
	}
	if got, want := c.APIServerEtcdClient(), "demo-apiserver-etcd-client"; got != want {
		t.Fatalf("APIServerEtcdClient()=%q want %q", got, want)
	}

	// etcd
	if got, want := c.EtcdServerTLS(), "demo-etcd-server-tls"; got != want {
		t.Fatalf("EtcdServerTLS()=%q want %q", got, want)
	}
	if got, want := c.EtcdPeerTLS(), "demo-etcd-peer-tls"; got != want {
		t.Fatalf("EtcdPeerTLS()=%q want %q", got, want)
	}
	if got, want := c.EtcdHealthClient(), "demo-etcd-healthcheck-client"; got != want {
		t.Fatalf("EtcdHealthClient()=%q want %q", got, want)
	}

	// front proxy
	if got, want := c.FrontProxyClient(), "demo-front-proxy-client"; got != want {
		t.Fatalf("FrontProxyClient()=%q want %q", got, want)
	}

	// component clients
	if got, want := c.CMClient(), "demo-cm-client"; got != want {
		t.Fatalf("CMClient()=%q want %q", got, want)
	}
	if got, want := c.SchedulerClient(), "demo-scheduler-client"; got != want {
		t.Fatalf("SchedulerClient()=%q want %q", got, want)
	}
	if got, want := c.AdminClient(), "demo-admin-client"; got != want {
		t.Fatalf("AdminClient()=%q want %q", got, want)
	}

	// konnectivity
	if got, want := c.KonnectivityServerTLS(), "demo-konnectivity-server-tls"; got != want {
		t.Fatalf("KonnectivityServerTLS()=%q want %q", got, want)
	}
	if got, want := c.KonnectivityAgentTLS(), "demo-konnectivity-agent-tls"; got != want {
		t.Fatalf("KonnectivityAgentTLS()=%q want %q", got, want)
	}
}
