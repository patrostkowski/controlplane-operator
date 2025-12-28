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

func TestNewServiceAccount_SetsNameNamespace(t *testing.T) {
	s := NewServiceAccount("ns1", "sa1")

	if s == nil || s.ServiceAccount == nil {
		t.Fatalf("expected non-nil ServiceAccountTemplate")
	}
	if s.Namespace != "ns1" {
		t.Fatalf("expected namespace %q, got %q", "ns1", s.Namespace)
	}
	if s.Name != "sa1" {
		t.Fatalf("expected name %q, got %q", "sa1", s.Name)
	}
}

func TestSAWithLabels_MergesAndInitializes(t *testing.T) {
	s := NewServiceAccount("ns", "sa")
	s.Labels = nil

	s.WithLabels(map[string]string{"a": "1", "b": "2"}).
		WithLabels(map[string]string{"b": "22", "c": "3"})

	want := map[string]string{"a": "1", "b": "22", "c": "3"}
	if !reflect.DeepEqual(s.Labels, want) {
		t.Fatalf("unexpected labels: want=%#v got=%#v", want, s.Labels)
	}
}

func TestSAWithAnnotations_MergesAndInitializes(t *testing.T) {
	s := NewServiceAccount("ns", "sa")
	s.Annotations = nil

	s.WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	want := map[string]string{"x": "yy", "z": "w"}
	if !reflect.DeepEqual(s.Annotations, want) {
		t.Fatalf("unexpected annotations: want=%#v got=%#v", want, s.Annotations)
	}
}
