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
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func TestIssuerResources_BuildsExpectedIssuers(t *testing.T) {
	ns := "mcp"

	raw := issuerResources(ns)
	objs := make([]any, 0, len(raw))
	for _, o := range raw {
		objs = append(objs, o)
	}

	if len(objs) != 6 {
		t.Fatalf("expected 6 issuer objects, got %d", len(objs))
	}

	// managed
	self := findIssuer(objs, issuerSelfSigned, ns)
	if self == nil {
		t.Fatalf("expected issuer %q/%q", ns, issuerSelfSigned)
	}
	if self.Spec.SelfSigned == nil {
		t.Fatalf("expected %q to be SelfSigned issuer", issuerSelfSigned)
	}

	ca := findIssuer(objs, issuerCA, ns)
	if ca == nil {
		t.Fatalf("expected issuer %q/%q", ns, issuerCA)
	}
	if ca.Spec.CA == nil || ca.Spec.CA.SecretName != secretManagedCA {
		t.Fatalf("expected %q to be CA issuer with secret %q; got %#v", issuerCA, secretManagedCA, ca.Spec.CA)
	}

	// etcd
	etcdSelf := findIssuer(objs, issuerEtcdSelfSigned, ns)
	if etcdSelf == nil || etcdSelf.Spec.SelfSigned == nil {
		t.Fatalf("expected etcd self-signed issuer %q", issuerEtcdSelfSigned)
	}

	etcdCA := findIssuer(objs, issuerEtcdCA, ns)
	if etcdCA == nil || etcdCA.Spec.CA == nil || etcdCA.Spec.CA.SecretName != secretEtcdCA {
		t.Fatalf("expected etcd CA issuer %q with secret %q; got %#v", issuerEtcdCA, secretEtcdCA, etcdCA)
	}

	// front-proxy
	fpSelf := findIssuer(objs, issuerFrontProxySelf, ns)
	if fpSelf == nil || fpSelf.Spec.SelfSigned == nil {
		t.Fatalf("expected front-proxy self-signed issuer %q", issuerFrontProxySelf)
	}

	fpCA := findIssuer(objs, issuerFrontProxyCA, ns)
	if fpCA == nil || fpCA.Spec.CA == nil || fpCA.Spec.CA.SecretName != secretFrontProxyCA {
		t.Fatalf("expected front-proxy CA issuer %q with secret %q; got %#v", issuerFrontProxyCA, secretFrontProxyCA, fpCA)
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

	raw := Resources(mcp)
	objs := make([]any, 0, len(raw))
	for _, o := range raw {
		objs = append(objs, o)
	}

	// issuers: 6
	// certs: 14
	// total: 20
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
	// ensure kind matches builder constant ("Issuer")
	if apiserverTLS.Spec.IssuerRef.Kind != "Issuer" {
		t.Fatalf("apiserver tls: expected issuerRef kind %q, got %q", "Issuer", apiserverTLS.Spec.IssuerRef.Kind)
	}
	// some canonical DNS SAN
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

	// sanity: issuerRef kind should be builder-compatible (Issuer)
	if admin.Spec.IssuerRef.Kind != "Issuer" {
		t.Fatalf("admin client: expected issuerRef kind %q got %q", "Issuer", admin.Spec.IssuerRef.Kind)
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

func TestCertificate_IssuerRefKindIsIssuerEverywhere(t *testing.T) {
	// This test protects you from accidentally switching the builder
	// to ClusterIssuer, or leaving Kind empty.
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
	raw := Resources(mcp)
	objs := make([]any, 0, len(raw))
	for _, o := range raw {
		objs = append(objs, o)
	}

	for _, o := range objs {
		c, ok := o.(*certmanagerv1.Certificate)
		if !ok {
			continue
		}
		// cert-manager uses empty Kind as "Issuer" sometimes, but our builder sets it explicitly.
		// Keep this as a regression guard.
		if c.Spec.IssuerRef.Kind != "Issuer" {
			t.Fatalf("certificate %s/%s: expected issuerRef.kind=Issuer, got=%q (ref=%#v)",
				c.Namespace, c.Name, c.Spec.IssuerRef.Kind, c.Spec.IssuerRef)
		}

		// Bonus: Kind must match cert-manager meta API group behavior (Name is required)
		if c.Spec.IssuerRef.Name == "" {
			t.Fatalf("certificate %s/%s: issuerRef.name must not be empty", c.Namespace, c.Name)
		}
	}
}

// Ensure imports used (keeps go test happy if you remove tests)
var _ = certmanagermeta.IssuerReference{}
