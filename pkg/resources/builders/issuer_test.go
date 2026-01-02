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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
)

func TestNewIssuer_SetsNameNamespace(t *testing.T) {
	i := NewIssuer().
		WithName("iss1").
		WithNamespace("ns1")
	if i == nil || i.Issuer == nil {
		t.Fatalf("expected non-nil IssuerTemplate")
	}
	if i.Namespace != "ns1" {
		t.Fatalf("expected namespace %q, got %q", "ns1", i.Namespace)
	}
	if i.Name != "iss1" {
		t.Fatalf("expected name %q, got %q", "iss1", i.Name)
	}
}

func TestIssuer_WithLabelsAndAnnotations_MergesAndInitializes(t *testing.T) {
	i := NewIssuer().
		WithName("iss").
		WithNamespace("ns")
	i.Labels = nil
	i.Annotations = nil

	i.WithLabels(map[string]string{"a": "1", "b": "2"}).
		WithLabels(map[string]string{"b": "22", "c": "3"}).
		WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	wantLabels := map[string]string{"a": "1", "b": "22", "c": "3"}
	if !reflect.DeepEqual(i.Labels, wantLabels) {
		t.Fatalf("unexpected labels: want=%#v got=%#v", wantLabels, i.Labels)
	}

	wantAnn := map[string]string{"x": "yy", "z": "w"}
	if !reflect.DeepEqual(i.Annotations, wantAnn) {
		t.Fatalf("unexpected annotations: want=%#v got=%#v", wantAnn, i.Annotations)
	}
}

func TestIssuer_SelfSigned_SetsSpec(t *testing.T) {
	i := NewIssuer().
		WithName("iss").
		WithNamespace("ns").
		SelfSigned().Build()
	if i.Spec.SelfSigned == nil {
		t.Fatalf("expected SelfSigned issuer config to be set")
	}
	if i.Spec.CA != nil {
		t.Fatalf("expected CA config to be nil for SelfSigned issuer")
	}
}

func TestIssuer_CA_SetsSpec(t *testing.T) {
	i := NewIssuer().
		WithName("iss").
		WithNamespace("ns").
		CA("my-ca-secret").Build()
	if i.Spec.CA == nil || i.Spec.CA.SecretName != "my-ca-secret" {
		t.Fatalf("expected CA secret %q, got %#v", "my-ca-secret", i.Spec.CA)
	}
	if i.Spec.SelfSigned != nil {
		t.Fatalf("expected SelfSigned config to be nil for CA issuer")
	}
}

func TestIssuer_Build_ReturnsDeepCopy(t *testing.T) {
	tpl := NewIssuer().
		WithName("iss").
		WithNamespace("ns").
		SelfSigned()
	b1 := tpl.Build()
	b2 := tpl.Build()

	if b1 == tpl.Issuer {
		t.Fatalf("Build() must not return same pointer as template")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	// mutate b1 and ensure template/b2 unchanged
	b1.Labels = map[string]string{"mut": "x"}

	if tpl.Labels != nil && tpl.Labels["mut"] == "x" {
		t.Fatalf("template mutated unexpectedly")
	}
	if b2.Labels != nil && b2.Labels["mut"] == "x" {
		t.Fatalf("b2 mutated unexpectedly")
	}

	// sanity: correct type
	var _ *certmanagerv1.Issuer = b1
}
