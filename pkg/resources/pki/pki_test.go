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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func findIssuer(objs []client.Object, name, ns string) *certmanagerv1.Issuer {
	for _, o := range objs {
		iss, ok := o.(*certmanagerv1.Issuer)
		if ok && iss.Name == name && iss.Namespace == ns {
			return iss
		}
	}
	return nil
}

func findCert(objs []client.Object, name, ns string) *certmanagerv1.Certificate {
	for _, o := range objs {
		c, ok := o.(*certmanagerv1.Certificate)
		if ok && c.Name == name && c.Namespace == ns {
			return c
		}
	}
	return nil
}

func TestPKI_BuildsIssuersAndRootCAs(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Networking: &mcpv1alpha1.NetworkingSpec{ServiceCIDR: "10.96.0.0/12"},
			},
		},
		Status: mcpv1alpha1.ManagedControlPlaneStatus{Address: "192.0.2.10"},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})

	objs := NewBuilder(cc).Objects()

	// Issuers (names come from PKI facade)
	issSelf := cc.PKI().Issuer().SelfSigned()
	issCA := cc.PKI().Issuer().CA()

	if findIssuer(objs, issSelf, mcp.Namespace) == nil {
		t.Fatalf("expected issuer %q in namespace %q", issSelf, mcp.Namespace)
	}
	if findIssuer(objs, issCA, mcp.Namespace) == nil {
		t.Fatalf("expected issuer %q in namespace %q", issCA, mcp.Namespace)
	}

	// Root CA certificates (names come from PKI facade)
	managedCA := cc.PKI().Certificate().ManagedCA()
	etcdCA := cc.PKI().Certificate().EtcdCA()
	fpCA := cc.PKI().Certificate().FrontProxyCA()
	konCA := cc.PKI().Certificate().KonnectivityCA()

	for _, name := range []string{managedCA, etcdCA, fpCA, konCA} {
		c := findCert(objs, name, mcp.Namespace)
		if c == nil {
			t.Fatalf("expected certificate %q in namespace %q", name, mcp.Namespace)
		}
		if !c.Spec.IsCA {
			t.Fatalf("certificate %q: expected IsCA=true", name)
		}
		if c.Spec.SecretName != name {
			t.Fatalf("certificate %q: SecretName=%q want %q", name, c.Spec.SecretName, name)
		}
	}
}

func TestPKI_AllCertificatesHaveIssuerKindIssuer(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Networking: &mcpv1alpha1.NetworkingSpec{ServiceCIDR: "10.96.0.0/12"},
			},
		},
		Status: mcpv1alpha1.ManagedControlPlaneStatus{Address: "192.0.2.10"},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	objs := NewBuilder(cc).Objects()

	for _, o := range objs {
		c, ok := o.(*certmanagerv1.Certificate)
		if !ok {
			continue
		}
		if c.Spec.IssuerRef.Kind != "Issuer" {
			t.Fatalf("certificate %s/%s: issuerRef.kind=%q want %q", c.Namespace, c.Name, c.Spec.IssuerRef.Kind, "Issuer")
		}
		if c.Spec.IssuerRef.Name == "" {
			t.Fatalf("certificate %s/%s: issuerRef.name must not be empty", c.Namespace, c.Name)
		}
	}
}
