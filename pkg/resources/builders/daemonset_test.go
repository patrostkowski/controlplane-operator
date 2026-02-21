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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestNewDaemonSet_SetsBasics(t *testing.T) {
	labels := map[string]string{"app": "demo"}
	selector := map[string]string{"app": "demo", "tier": "node"}

	ds := NewDaemonSet().
		WithName("ds1").
		WithNamespace("ns1").
		WithLabels(labels).
		WithSelector(selector).
		WithPodLabels(map[string]string{"pod": "x"})

	if ds == nil || ds.DaemonSet == nil {
		t.Fatalf("expected non-nil DaemonsetTemplate")
	}
	if ds.Namespace != "ns1" || ds.Name != "ds1" {
		t.Fatalf("unexpected metadata: %s/%s", ds.Namespace, ds.Name)
	}
	if !reflect.DeepEqual(ds.Labels, labels) {
		t.Fatalf("unexpected labels: %#v", ds.Labels)
	}
	if ds.Spec.Selector == nil || !reflect.DeepEqual(ds.Spec.Selector.MatchLabels, selector) {
		t.Fatalf("unexpected selector: %#v", ds.Spec.Selector)
	}
	// selector does NOT automatically apply to pod labels for DaemonSet; WithPodLabels does.
	if ds.Spec.Template.Labels["pod"] != "x" {
		t.Fatalf("unexpected pod labels: %#v", ds.Spec.Template.Labels)
	}
}

func TestDaemonSet_WithUpdateStrategy_SetsType(t *testing.T) {
	ds := NewDaemonSet().WithName("ds").WithUpdateStrategy(appsv1.RollingUpdateDaemonSetStrategyType)
	if ds.Spec.UpdateStrategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
		t.Fatalf("unexpected update strategy: %#v", ds.Spec.UpdateStrategy)
	}
}

func TestDaemonSet_PodMutations(t *testing.T) {
	aff := corev1.Affinity{}
	tol := corev1.Toleration{Key: "k", Operator: corev1.TolerationOpExists}
	c := corev1.Container{Name: "c1", Image: "img"}

	ds := NewDaemonSet().
		WithName("ds").
		WithNamespace("ns").
		WithServiceAccount("sa").
		WithAffinity(aff).
		WithTolerations(tol).
		WithNodeSelector(map[string]string{"k": "v"}).
		WithHostNetwork().
		WithDNSPolicy(corev1.DNSClusterFirstWithHostNet).
		WithPriorityClass("pc").
		WithContainer(c)

	pt := ds.Spec.Template
	if pt.Spec.ServiceAccountName != "sa" {
		t.Fatalf("unexpected serviceAccount: %q", pt.Spec.ServiceAccountName)
	}
	if pt.Spec.Affinity == nil {
		t.Fatalf("expected affinity to be set")
	}
	if len(pt.Spec.Tolerations) != 1 || pt.Spec.Tolerations[0].Key != "k" {
		t.Fatalf("unexpected tolerations: %#v", pt.Spec.Tolerations)
	}
	if pt.Spec.NodeSelector["k"] != "v" {
		t.Fatalf("unexpected nodeSelector: %#v", pt.Spec.NodeSelector)
	}
	if !pt.Spec.HostNetwork {
		t.Fatalf("expected hostNetwork=true")
	}
	if pt.Spec.DNSPolicy != corev1.DNSClusterFirstWithHostNet {
		t.Fatalf("unexpected dnsPolicy: %q", pt.Spec.DNSPolicy)
	}
	if pt.Spec.PriorityClassName != "pc" {
		t.Fatalf("unexpected priorityClass: %q", pt.Spec.PriorityClassName)
	}
	if len(pt.Spec.Containers) != 1 || pt.Spec.Containers[0].Name != "c1" {
		t.Fatalf("unexpected containers: %#v", pt.Spec.Containers)
	}
}

func TestDaemonSet_Build_ReturnsDeepCopy(t *testing.T) {
	ds := NewDaemonSet().
		WithName("ds").
		WithNamespace("ns").
		WithLabels(map[string]string{"app": "x"}).
		WithSelector(map[string]string{"app": "x"}).
		WithPodLabels(map[string]string{"app": "x"}).
		WithContainer(corev1.Container{Name: "c1", Image: "img"})

	b1 := ds.Build()
	b2 := ds.Build()

	if b1 == ds.DaemonSet {
		t.Fatalf("Build() must not return same pointer as template")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.Spec.Template.Labels["app"] = "mut"
	b1.Spec.Template.Spec.Containers[0].Name = "mut"

	if ds.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", ds.Labels)
	}
	if ds.Spec.Template.Labels["app"] != "x" {
		t.Fatalf("template pod labels mutated unexpectedly: %#v", ds.Spec.Template.Labels)
	}
	if ds.Spec.Template.Spec.Containers[0].Name != "c1" {
		t.Fatalf("template containers mutated unexpectedly: %#v", ds.Spec.Template.Spec.Containers)
	}
	if b2.Labels["app"] != "x" || b2.Spec.Template.Labels["app"] != "x" || b2.Spec.Template.Spec.Containers[0].Name != "c1" {
		t.Fatalf("b2 mutated unexpectedly")
	}
}
