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

	corev1 "k8s.io/api/core/v1"
)

func TestNewSecret_SetsNameNamespaceAndInitializesMaps(t *testing.T) {
	s := NewSecret().
		WithName("sec1").
		WithNamespace("ns1")

	if s == nil || s.Secret == nil {
		t.Fatalf("expected non-nil SecretTemplate")
	}
	if s.Name != "sec1" || s.Namespace != "ns1" {
		t.Fatalf("unexpected metadata: %s/%s", s.Namespace, s.Name)
	}
	if s.Data == nil {
		t.Fatalf("expected Data to be initialized")
	}
	if s.StringData == nil {
		t.Fatalf("expected StringData to be initialized")
	}
}

func TestSecret_WithLabelsAndAnnotations_MergesAndInitializes(t *testing.T) {
	s := NewSecret().WithName("sec").WithNamespace("ns")
	s.Labels = nil
	s.Annotations = nil

	s.WithLabels(map[string]string{"a": "1"}).
		WithLabels(map[string]string{"a": "2", "b": "3"}).
		WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	if !reflect.DeepEqual(s.Labels, map[string]string{"a": "2", "b": "3"}) {
		t.Fatalf("unexpected labels: %#v", s.Labels)
	}
	if !reflect.DeepEqual(s.Annotations, map[string]string{"x": "yy", "z": "w"}) {
		t.Fatalf("unexpected annotations: %#v", s.Annotations)
	}
}

func TestSecret_WithType_SetsType(t *testing.T) {
	s := NewSecret().WithName("sec").WithNamespace("ns").WithType(corev1.SecretTypeTLS)
	if s.Type != corev1.SecretTypeTLS {
		t.Fatalf("expected type %q got %q", corev1.SecretTypeTLS, s.Type)
	}
}

func TestSecret_Put_SetsStringDataKey(t *testing.T) {
	s := NewSecret().WithName("sec").WithNamespace("ns")
	s.StringData = nil // ensure Put handles nil map

	s.Put("k1", "v1").Put("k2", "v2").Put("k1", "v11")

	want := map[string]string{"k1": "v11", "k2": "v2"}
	if !reflect.DeepEqual(s.StringData, want) {
		t.Fatalf("unexpected stringData: want=%#v got=%#v", want, s.StringData)
	}
}

func TestSecret_PutBytes_SetsDataKey(t *testing.T) {
	s := NewSecret().WithName("sec").WithNamespace("ns")
	s.Data = nil // ensure PutBytes handles nil map

	s.PutBytes("b1", []byte("x")).PutBytes("b2", []byte("y")).PutBytes("b1", []byte("xx"))

	if got := string(s.Data["b1"]); got != "xx" {
		t.Fatalf("unexpected b1: %q", got)
	}
	if got := string(s.Data["b2"]); got != "y" {
		t.Fatalf("unexpected b2: %q", got)
	}
}

func TestSecret_Build_ReturnsDeepCopy(t *testing.T) {
	tpl := NewSecret().
		WithName("sec").
		WithNamespace("ns").
		WithLabels(map[string]string{"app": "x"}).
		WithType(corev1.SecretTypeOpaque).
		Put("k", "v").
		PutBytes("b", []byte("x"))

	b1 := tpl.Build()
	b2 := tpl.Build()

	if b1 == tpl.Secret {
		t.Fatalf("Build() must not return same pointer as template")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.StringData["k"] = "mut"
	b1.Data["b"][0] = 'm'

	if tpl.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", tpl.Labels)
	}
	if tpl.StringData["k"] != "v" {
		t.Fatalf("template StringData mutated unexpectedly: %#v", tpl.StringData)
	}
	if string(tpl.Data["b"]) != "x" {
		t.Fatalf("template Data mutated unexpectedly: %#v", tpl.Data)
	}
	if b2.StringData["k"] != "v" || string(b2.Data["b"]) != "x" {
		t.Fatalf("b2 mutated unexpectedly: stringData=%#v data=%#v", b2.StringData, b2.Data)
	}
}
