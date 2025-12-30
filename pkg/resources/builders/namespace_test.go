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
)

func TestNewNS_SetsName(t *testing.T) {
	n := NewNamespace().WithName("ns1")

	if n == nil || n.Namespace == nil {
		t.Fatalf("expected non-nil NamespaceTemplate")
	}
	if n.Name != "ns1" {
		t.Fatalf("expected name %q, got %q", "ns1", n.Name)
	}
}

func TestNSWithLabels_MergesAndInitializes(t *testing.T) {
	n := NewNamespace().WithName("ns")
	n.Labels = nil

	n.WithLabels(map[string]string{"a": "1", "b": "2"}).
		WithLabels(map[string]string{"b": "22", "c": "3"})

	want := map[string]string{"a": "1", "b": "22", "c": "3"}
	if !reflect.DeepEqual(n.Labels, want) {
		t.Fatalf("unexpected labels: want=%#v got=%#v", want, n.Labels)
	}
}

func TestNSWithAnnotations_MergesAndInitializes(t *testing.T) {
	n := NewNamespace().WithName("ns")
	n.Annotations = nil

	n.WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	want := map[string]string{"x": "yy", "z": "w"}
	if !reflect.DeepEqual(n.Annotations, want) {
		t.Fatalf("unexpected annotations: want=%#v got=%#v", want, n.Annotations)
	}
}
