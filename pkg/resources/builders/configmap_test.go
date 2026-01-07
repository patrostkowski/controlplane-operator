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

func TestNewConfigMap_SetsNameNamespaceAndInitializesMaps(t *testing.T) {
	c := NewConfigMap().
		WithName("cm1").
		WithNamespace("ns1")

	if c == nil || c.ConfigMap == nil {
		t.Fatalf("expected non-nil ConfigMapTemplate")
	}
	if c.Namespace != "ns1" {
		t.Fatalf("expected namespace %q, got %q", "ns1", c.Namespace)
	}
	if c.Name != "cm1" {
		t.Fatalf("expected name %q, got %q", "cm1", c.Name)
	}

	// Important: NewConfigMap should initialize Data/BinaryData to avoid nil map writes
	if c.Data == nil {
		t.Fatalf("expected Data to be initialized")
	}
	if c.BinaryData == nil {
		t.Fatalf("expected BinaryData to be initialized")
	}
}

func TestWithLabels_MergesAndInitializes(t *testing.T) {
	c := NewConfigMap().
		WithName("cm").
		WithNamespace("ns")
	c.Labels = nil

	c.WithLabels(map[string]string{"a": "1", "b": "2"}).
		WithLabels(map[string]string{"b": "22", "c": "3"})

	want := map[string]string{"a": "1", "b": "22", "c": "3"}
	if !reflect.DeepEqual(c.Labels, want) {
		t.Fatalf("unexpected labels: want=%#v got=%#v", want, c.Labels)
	}
}

func TestWithAnnotations_MergesAndInitializes(t *testing.T) {
	c := NewConfigMap().
		WithName("cm").
		WithNamespace("ns")
	c.Annotations = nil

	c.WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	want := map[string]string{"x": "yy", "z": "w"}
	if !reflect.DeepEqual(c.Annotations, want) {
		t.Fatalf("unexpected annotations: want=%#v got=%#v", want, c.Annotations)
	}
}

func TestPut_SetsDataKey(t *testing.T) {
	c := NewConfigMap().
		WithName("cm").
		WithNamespace("ns")
	c.Data = nil // ensure Put handles nil map

	c.Put("k1", "v1").
		Put("k2", "v2").
		Put("k1", "v11") // overwrite

	want := map[string]string{"k1": "v11", "k2": "v2"}
	if !reflect.DeepEqual(c.Data, want) {
		t.Fatalf("unexpected data: want=%#v got=%#v", want, c.Data)
	}
}

func TestPutBytes_SetsBinaryDataKey(t *testing.T) {
	c := NewConfigMap().
		WithName("cm").
		WithNamespace("ns")
	c.BinaryData = nil // ensure PutBytes handles nil map

	c.PutBytes("b1", []byte{1, 2, 3})
	c.PutBytes("b2", []byte("hello"))
	c.PutBytes("b1", []byte{9}) // overwrite

	if got := c.BinaryData["b1"]; !reflect.DeepEqual(got, []byte{9}) {
		t.Fatalf("unexpected b1: want=%v got=%v", []byte{9}, got)
	}
	if got := c.BinaryData["b2"]; !reflect.DeepEqual(got, []byte("hello")) {
		t.Fatalf("unexpected b2: want=%v got=%v", []byte("hello"), got)
	}
}

func TestConfigMapBuild_ReturnsDeepCopy(t *testing.T) {
	c := NewConfigMap().
		WithName("cm").
		WithNamespace("ns").
		WithLabels(map[string]string{"app": "x"}).
		WithAnnotations(map[string]string{"a": "b"}).
		Put("k", "v").
		PutBytes("bin", []byte{1, 2, 3})

	b1 := c.Build()
	b2 := c.Build()

	if b1 == c.ConfigMap {
		t.Fatalf("Build() must not return the same pointer as the template ConfigMap")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	// Mutate b1 and ensure template and b2 stay unchanged
	b1.Labels["app"] = "mutated"
	b1.Annotations["a"] = "mutated"
	b1.Data["k"] = "mutated"
	b1.BinaryData["bin"][0] = 9

	if c.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", c.Labels)
	}
	if c.Annotations["a"] != "b" {
		t.Fatalf("template annotations mutated unexpectedly: %#v", c.Annotations)
	}
	if c.Data["k"] != "v" {
		t.Fatalf("template data mutated unexpectedly: %#v", c.Data)
	}
	if got := c.BinaryData["bin"]; !reflect.DeepEqual(got, []byte{1, 2, 3}) {
		t.Fatalf("template binary data mutated unexpectedly: %#v", got)
	}

	if b2.Labels["app"] != "x" {
		t.Fatalf("b2 labels mutated unexpectedly: %#v", b2.Labels)
	}
	if b2.Annotations["a"] != "b" {
		t.Fatalf("b2 annotations mutated unexpectedly: %#v", b2.Annotations)
	}
	if b2.Data["k"] != "v" {
		t.Fatalf("b2 data mutated unexpectedly: %#v", b2.Data)
	}
	if got := b2.BinaryData["bin"]; !reflect.DeepEqual(got, []byte{1, 2, 3}) {
		t.Fatalf("b2 binary data mutated unexpectedly: %#v", got)
	}
}
