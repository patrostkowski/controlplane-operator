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

package pki

import (
	"testing"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- small helpers ---

func findIssuer(objs []any, name, ns string) *certmanagerv1.Issuer {
	for _, o := range objs {
		if iss, ok := o.(*certmanagerv1.Issuer); ok {
			if iss.Name == name && iss.Namespace == ns {
				return iss
			}
		}
	}
	return nil
}

func findCert(objs []any, name, ns string) *certmanagerv1.Certificate {
	for _, o := range objs {
		if c, ok := o.(*certmanagerv1.Certificate); ok {
			if c.Name == name && c.Namespace == ns {
				return c
			}
		}
	}
	return nil
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func TestDefaultDurations(t *testing.T) {
	d := defaultDurations()

	if d.tenYears.Duration != 87600*time.Hour {
		t.Fatalf("tenYears: expected %v, got %v", 87600*time.Hour, d.tenYears.Duration)
	}
	if d.thirtyDays.Duration != 720*time.Hour {
		t.Fatalf("thirtyDays: expected %v, got %v", 720*time.Hour, d.thirtyDays.Duration)
	}
}

func TestIssuerSpecs(t *testing.T) {
	ns := "mcp"
	specs := issuerSpecs(ns)

	if len(specs) != 6 {
		t.Fatalf("expected 6 issuer specs, got %d: %#v", len(specs), specs)
	}

	// exact order matters for readability; behavior-wise, count+contents is what matters
	want := []PKIIssuerSpec{
		{Name: issuerSelfSigned, Namespace: ns, SelfSigned: true},
		{Name: issuerCA, Namespace: ns, CASecret: secretManagedCA},

		{Name: issuerEtcdSelfSigned, Namespace: ns, SelfSigned: true},
		{Name: issuerEtcdCA, Namespace: ns, CASecret: secretEtcdCA},

		{Name: issuerFrontProxySelf, Namespace: ns, SelfSigned: true},
		{Name: issuerFrontProxyCA, Namespace: ns, CASecret: secretFrontProxyCA},
	}

	for i := range want {
		if specs[i] != want[i] {
			t.Fatalf("issuer spec[%d]: expected %#v, got %#v", i, want[i], specs[i])
		}
	}
}

func TestResources_CountsAndKeyFields(t *testing.T) {
	ns := "mcp"
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: ns,
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Networking: &mcpv1alpha1.NetworkingSpec{
				ServiceCIDR: "10.96.0.0/12",
			},
		},
	}
	// LB address should be included as an IP SAN
	mcp.Status.Address = "1.2.3.4"

	// Resources returns []client.Object; cast to []any for generic helper funcs
	raw := Resources(mcp)
	objs := make([]any, 0, len(raw))
	for _, o := range raw {
		objs = append(objs, o)
	}

	// Current composition:
	// - issuers: 6
	// - certs:
	//   root CAs 3
	//   sa signer 1
	//   apiserver serving 1
	//   apiserver kubelet client 1
	//   etcd leafs 4
	//   front-proxy client 1
	//   component clients 2
	//   admin client 1
	//   total certs: 14
	// total objects = 20
	if len(objs) != 20 {
		t.Fatalf("expected 20 objects, got %d", len(objs))
	}

	// ensure key issuers exist
	if got := findIssuer(objs, issuerSelfSigned, ns); got == nil {
		t.Fatalf("expected issuer %q in ns %q", issuerSelfSigned, ns)
	}
	if got := findIssuer(objs, issuerCA, ns); got == nil {
		t.Fatalf("expected issuer %q in ns %q", issuerCA, ns)
	}
	if got := findIssuer(objs, issuerEtcdCA, ns); got == nil {
		t.Fatalf("expected issuer %q in ns %q", issuerEtcdCA, ns)
	}

	// ensure key cert exists + validate SAN behavior
	apiserverTLS := findCert(objs, secretAPIServerTLS, ns)
	if apiserverTLS == nil {
		t.Fatalf("expected certificate %q in ns %q", secretAPIServerTLS, ns)
	}
	if apiserverTLS.Spec.IssuerRef.Name != issuerCA {
		t.Fatalf("apiserver tls: expected issuer %q, got %q", issuerCA, apiserverTLS.Spec.IssuerRef.Name)
	}
	if !contains(apiserverTLS.Spec.DNSNames, "kubernetes.default.svc") {
		t.Fatalf("apiserver tls: expected DNS SAN kubernetes.default.svc; got %#v", apiserverTLS.Spec.DNSNames)
	}
	// first IP in service CIDR + lb addr + 127.0.0.1 should appear
	if !contains(apiserverTLS.Spec.IPAddresses, "10.96.0.1") {
		t.Fatalf("apiserver tls: expected service IP SAN 10.96.0.1; got %#v", apiserverTLS.Spec.IPAddresses)
	}
	if !contains(apiserverTLS.Spec.IPAddresses, "1.2.3.4") {
		t.Fatalf("apiserver tls: expected LB IP SAN 1.2.3.4; got %#v", apiserverTLS.Spec.IPAddresses)
	}
	if !contains(apiserverTLS.Spec.IPAddresses, "127.0.0.1") {
		t.Fatalf("apiserver tls: expected loopback IP SAN 127.0.0.1; got %#v", apiserverTLS.Spec.IPAddresses)
	}

	// admin client should be in system:masters
	admin := findCert(objs, secretAdminClient, ns)
	if admin == nil {
		t.Fatalf("expected certificate %q in ns %q", secretAdminClient, ns)
	}
	if admin.Spec.Subject == nil || len(admin.Spec.Subject.Organizations) != 1 || admin.Spec.Subject.Organizations[0] != orgSystemMasters {
		t.Fatalf("admin client: expected org %q, got %#v", orgSystemMasters, admin.Spec.Subject)
	}
}

func TestAPIServerSANs_HostnameAddressGoesToDNS(t *testing.T) {
	ns := "mcp"
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: ns,
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Networking: &mcpv1alpha1.NetworkingSpec{
				ServiceCIDR: "10.96.0.0/12",
			},
		},
	}
	mcp.Status.Address = "api.example.com"

	dns, ips := apiserverSANs(mcp, ns)

	if !contains(dns, "api.example.com") {
		t.Fatalf("expected hostname address to be added to DNS SANs; dns=%#v", dns)
	}
	if contains(ips, "api.example.com") {
		t.Fatalf("expected hostname address NOT to be added to IP SANs; ips=%#v", ips)
	}
}
