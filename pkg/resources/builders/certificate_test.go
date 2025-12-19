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

package builders

import (
	"reflect"
	"testing"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewCertificate_SetsNameNamespaceAndPrivateKeyDefaults(t *testing.T) {
	c := NewCertificate("ns1", "cert1")
	if c == nil || c.Certificate == nil {
		t.Fatalf("expected non-nil CertificateTemplate")
	}
	if c.Namespace != "ns1" || c.Name != "cert1" {
		t.Fatalf("unexpected metadata: %s/%s", c.Namespace, c.Name)
	}
	if c.Spec.PrivateKey == nil {
		t.Fatalf("expected PrivateKey to be initialized")
	}
	if c.Spec.PrivateKey.Algorithm != certmanagerv1.RSAKeyAlgorithm || c.Spec.PrivateKey.Size != 2048 {
		t.Fatalf("unexpected private key defaults: %#v", c.Spec.PrivateKey)
	}
}

func TestCertificate_Basics(t *testing.T) {
	dur := &metav1.Duration{Duration: 10 * time.Hour}
	renew := &metav1.Duration{Duration: 1 * time.Hour}

	c := NewCertificate("ns", "cert").
		WithSecretName("s").
		WithCommonName("cn").
		IsCA(true).
		Issuer("my-issuer").
		WithDuration(dur).
		WithRenewBefore(renew).
		WithUsages(certmanagerv1.UsageServerAuth, certmanagerv1.UsageKeyEncipherment).
		WithDNSNames("a", "b").
		WithIPAddresses("1.2.3.4").
		WithOrganizations("org1")

	b := c.Build()

	if b.Spec.SecretName != "s" || b.Spec.CommonName != "cn" || !b.Spec.IsCA {
		t.Fatalf("unexpected basics: %#v", b.Spec)
	}
	if b.Spec.IssuerRef.Name != "my-issuer" || b.Spec.IssuerRef.Kind != IssuerKind {
		t.Fatalf("unexpected issuerRef: %#v", b.Spec.IssuerRef)
	}
	if b.Spec.Duration == nil || b.Spec.Duration.Duration != dur.Duration {
		t.Fatalf("unexpected duration: %#v", b.Spec.Duration)
	}
	if b.Spec.RenewBefore == nil || b.Spec.RenewBefore.Duration != renew.Duration {
		t.Fatalf("unexpected renewBefore: %#v", b.Spec.RenewBefore)
	}
	if !reflect.DeepEqual(b.Spec.DNSNames, []string{"a", "b"}) {
		t.Fatalf("unexpected DNSNames: %#v", b.Spec.DNSNames)
	}
	if !reflect.DeepEqual(b.Spec.IPAddresses, []string{"1.2.3.4"}) {
		t.Fatalf("unexpected IPAddresses: %#v", b.Spec.IPAddresses)
	}
	if b.Spec.Subject == nil || !reflect.DeepEqual(b.Spec.Subject.Organizations, []string{"org1"}) {
		t.Fatalf("unexpected subject orgs: %#v", b.Spec.Subject)
	}
}

func TestCertificate_WithLabelsAndAnnotations_MergesAndInitializes(t *testing.T) {
	c := NewCertificate("ns", "cert")
	c.Labels = nil
	c.Annotations = nil

	c.WithLabels(map[string]string{"a": "1"}).
		WithLabels(map[string]string{"a": "2", "b": "3"}).
		WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	wantLabels := map[string]string{"a": "2", "b": "3"}
	if !reflect.DeepEqual(c.Labels, wantLabels) {
		t.Fatalf("unexpected labels: want=%#v got=%#v", wantLabels, c.Labels)
	}

	wantAnn := map[string]string{"x": "yy", "z": "w"}
	if !reflect.DeepEqual(c.Annotations, wantAnn) {
		t.Fatalf("unexpected annotations: want=%#v got=%#v", wantAnn, c.Annotations)
	}
}

func TestCertificate_Build_ReturnsDeepCopy(t *testing.T) {
	tpl := NewCertificate("ns", "cert").
		WithSecretName("s").
		WithDNSNames("a")

	b1 := tpl.Build()
	b2 := tpl.Build()

	if b1 == tpl.Certificate {
		t.Fatalf("Build() must not return same pointer as template")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	// mutate b1 and ensure template/b2 unchanged
	b1.Spec.DNSNames[0] = "mut"
	if tpl.Spec.DNSNames[0] == "mut" {
		t.Fatalf("template mutated unexpectedly")
	}
	if b2.Spec.DNSNames[0] == "mut" {
		t.Fatalf("b2 mutated unexpectedly")
	}
}
