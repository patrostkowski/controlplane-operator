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

func TestNewStatefulSet_SetsBasics(t *testing.T) {
	labels := map[string]string{"app": "demo"}
	sel := map[string]string{"app": "demo", "role": "db"}
	replicas := int32(3)

	ss := NewStatefulSet().
		WithName("ss1").
		WithNamespace("ns1").
		WithLabels(labels).
		WithServiceName("svc").
		WithSelector(sel).
		WithReplicas(replicas)

	if ss == nil || ss.StatefulSet == nil {
		t.Fatalf("expected non-nil StatefulSetTemplate")
	}
	if ss.Namespace != "ns1" || ss.Name != "ss1" {
		t.Fatalf("unexpected metadata: %s/%s", ss.Namespace, ss.Name)
	}
	if !reflect.DeepEqual(ss.Labels, labels) {
		t.Fatalf("unexpected labels: %#v", ss.Labels)
	}
	if ss.Spec.ServiceName != "svc" {
		t.Fatalf("unexpected serviceName: %q", ss.Spec.ServiceName)
	}
	if ss.Spec.Replicas == nil || *ss.Spec.Replicas != replicas {
		t.Fatalf("unexpected replicas: %#v", ss.Spec.Replicas)
	}
	if ss.Spec.Selector == nil || !reflect.DeepEqual(ss.Spec.Selector.MatchLabels, sel) {
		t.Fatalf("unexpected selector: %#v", ss.Spec.Selector)
	}
	// WithSelector also applies selector labels onto pod template labels for StatefulSet.
	for k, v := range sel {
		if ss.Spec.Template.Labels[k] != v {
			t.Fatalf("expected pod label %q=%q got %#v", k, v, ss.Spec.Template.Labels)
		}
	}
}

func TestStatefulSet_WithVolumeClaims_Appends(t *testing.T) {
	pvc1 := corev1.PersistentVolumeClaim{}
	pvc2 := corev1.PersistentVolumeClaim{}

	ss := NewStatefulSet().WithName("ss").WithNamespace("ns").WithVolumeClaims(pvc1).WithVolumeClaims(pvc2)
	if len(ss.Spec.VolumeClaimTemplates) != 2 {
		t.Fatalf("expected 2 volume claims, got %d", len(ss.Spec.VolumeClaimTemplates))
	}
}

func TestStatefulSet_PodMutationsAndVolumes(t *testing.T) {
	c := corev1.Container{Name: "c1", Image: "img"}
	vol := corev1.Volume{Name: "v1"}
	mount := corev1.VolumeMount{Name: "v1", MountPath: "/x"}

	ss := NewStatefulSet().
		WithName("ss").
		WithNamespace("ns").
		WithServiceAccount("sa").
		WithContainer(c).
		AddVolumes(vol).
		AddVolumeMounts("c1", mount)

	pt := ss.Spec.Template
	if pt.Spec.ServiceAccountName != "sa" {
		t.Fatalf("unexpected serviceAccount: %q", pt.Spec.ServiceAccountName)
	}
	if len(pt.Spec.Containers) != 1 || pt.Spec.Containers[0].Name != "c1" {
		t.Fatalf("unexpected containers: %#v", pt.Spec.Containers)
	}
	if len(pt.Spec.Volumes) != 1 || pt.Spec.Volumes[0].Name != "v1" {
		t.Fatalf("unexpected volumes: %#v", pt.Spec.Volumes)
	}
	if len(pt.Spec.Containers[0].VolumeMounts) != 1 || pt.Spec.Containers[0].VolumeMounts[0].MountPath != "/x" {
		t.Fatalf("unexpected mounts: %#v", pt.Spec.Containers[0].VolumeMounts)
	}
}

func TestStatefulSet_Build_ReturnsDeepCopy(t *testing.T) {
	ss := NewStatefulSet().
		WithName("ss").
		WithNamespace("ns").
		WithLabels(map[string]string{"app": "x"}).
		WithSelector(map[string]string{"app": "x"}).
		WithContainer(corev1.Container{Name: "c1", Image: "img"})

	b1 := ss.Build()
	b2 := ss.Build()

	if b1 == ss.StatefulSet {
		t.Fatalf("Build() must not return same pointer as template")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.Spec.Template.Labels["app"] = "mut"
	b1.Spec.Template.Spec.Containers[0].Name = "mut"

	if ss.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", ss.Labels)
	}
	if ss.Spec.Template.Labels["app"] != "x" {
		t.Fatalf("template pod labels mutated unexpectedly: %#v", ss.Spec.Template.Labels)
	}
	if ss.Spec.Template.Spec.Containers[0].Name != "c1" {
		t.Fatalf("template containers mutated unexpectedly: %#v", ss.Spec.Template.Spec.Containers)
	}
	if b2.Labels["app"] != "x" || b2.Spec.Template.Labels["app"] != "x" || b2.Spec.Template.Spec.Containers[0].Name != "c1" {
		t.Fatalf("b2 mutated unexpectedly")
	}
}
