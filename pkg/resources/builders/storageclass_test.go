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
	storagev1 "k8s.io/api/storage/v1"
)

func TestNewStorageClass_SetsNameAndMergesMeta(t *testing.T) {
	sc := NewStorageClass().
		WithName("sc1").
		WithLabels(map[string]string{"a": "1"}).
		WithLabels(map[string]string{"a": "2", "b": "3"}).
		WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	if sc == nil || sc.StorageClass == nil {
		t.Fatalf("expected non-nil StorageClassTemplate")
	}
	if sc.Name != "sc1" {
		t.Fatalf("expected name %q got %q", "sc1", sc.Name)
	}
	if !reflect.DeepEqual(sc.Labels, map[string]string{"a": "2", "b": "3"}) {
		t.Fatalf("unexpected labels: %#v", sc.Labels)
	}
	if !reflect.DeepEqual(sc.Annotations, map[string]string{"x": "yy", "z": "w"}) {
		t.Fatalf("unexpected annotations: %#v", sc.Annotations)
	}
}

func TestStorageClass_WithProvisionerPolicyAndBindingMode(t *testing.T) {
	policy := corev1.PersistentVolumeReclaimDelete
	mode := storagev1.VolumeBindingWaitForFirstConsumer

	sc := NewStorageClass().
		WithName("sc").
		WithProvisioner("kubernetes.io/no-provisioner").
		WithPolicy(policy).
		WithBindingMode(mode)

	b := sc.Build()
	if b.Provisioner != "kubernetes.io/no-provisioner" {
		t.Fatalf("unexpected provisioner: %q", b.Provisioner)
	}
	if b.ReclaimPolicy == nil || *b.ReclaimPolicy != policy {
		t.Fatalf("unexpected reclaim policy: %#v", b.ReclaimPolicy)
	}
	if b.VolumeBindingMode == nil || *b.VolumeBindingMode != mode {
		t.Fatalf("unexpected binding mode: %#v", b.VolumeBindingMode)
	}
}

func TestStorageClass_Build_ReturnsDeepCopy(t *testing.T) {
	policy := corev1.PersistentVolumeReclaimRetain
	sc := NewStorageClass().
		WithName("sc").
		WithLabels(map[string]string{"app": "x"}).
		WithProvisioner("p").
		WithPolicy(policy)

	b1 := sc.Build()
	b2 := sc.Build()

	if b1 == sc.StorageClass {
		t.Fatalf("Build() must not return same pointer as template")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	if sc.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", sc.Labels)
	}
	if b2.Labels["app"] != "x" {
		t.Fatalf("b2 labels mutated unexpectedly: %#v", b2.Labels)
	}
}
