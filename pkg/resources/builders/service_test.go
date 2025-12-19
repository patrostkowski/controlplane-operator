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
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNewService_SetsNameNamespaceAndDefaults(t *testing.T) {
	s := NewService("ns1", "svc1")

	if s == nil || s.Service == nil {
		t.Fatalf("expected non-nil ServiceTemplate")
	}
	if s.Namespace != "ns1" {
		t.Fatalf("expected namespace %q, got %q", "ns1", s.Namespace)
	}
	if s.Name != "svc1" {
		t.Fatalf("expected name %q, got %q", "svc1", s.Name)
	}

	if s.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Fatalf("expected default type ClusterIP, got %q", s.Spec.Type)
	}
	if s.Spec.Selector == nil {
		t.Fatalf("expected Selector to be initialized")
	}
	if s.Spec.Ports == nil {
		t.Fatalf("expected Ports slice to be initialized (non-nil)")
	}
}

func TestWithLabels_Merges(t *testing.T) {
	s := NewService("ns", "svc")
	s.Labels = nil

	s.WithLabels(map[string]string{"a": "1", "b": "2"}).
		WithLabels(map[string]string{"b": "22", "c": "3"})

	want := map[string]string{"a": "1", "b": "22", "c": "3"}
	if !reflect.DeepEqual(s.Labels, want) {
		t.Fatalf("unexpected labels: want=%#v got=%#v", want, s.Labels)
	}
}

func TestWithAnnotations_Merges(t *testing.T) {
	s := NewService("ns", "svc")
	s.Annotations = nil

	s.WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	want := map[string]string{"x": "yy", "z": "w"}
	if !reflect.DeepEqual(s.Annotations, want) {
		t.Fatalf("unexpected annotations: want=%#v got=%#v", want, s.Annotations)
	}
}

func TestWithSelector_Merges(t *testing.T) {
	s := NewService("ns", "svc")
	s.Spec.Selector = nil

	s.WithSelector(map[string]string{"app": "x"}).
		WithSelector(map[string]string{"tier": "cp", "app": "y"})

	want := map[string]string{"app": "y", "tier": "cp"}
	if !reflect.DeepEqual(s.Spec.Selector, want) {
		t.Fatalf("unexpected selector: want=%#v got=%#v", want, s.Spec.Selector)
	}
}

func TestWithType_SetsType(t *testing.T) {
	s := NewService("ns", "svc").WithType(corev1.ServiceTypeNodePort)
	if s.Spec.Type != corev1.ServiceTypeNodePort {
		t.Fatalf("expected type NodePort, got %q", s.Spec.Type)
	}
}

func TestHeadless_SetsClusterIPNone(t *testing.T) {
	s := NewService("ns", "svc").Headless()
	if s.Spec.ClusterIP != corev1.ClusterIPNone {
		t.Fatalf("expected ClusterIP None, got %q", s.Spec.ClusterIP)
	}
}

func TestAddPorts_Appends(t *testing.T) {
	s := NewService("ns", "svc")

	p1 := corev1.ServicePort{Name: "https", Port: 443, Protocol: corev1.ProtocolTCP}
	p2 := corev1.ServicePort{Name: "metrics", Port: 10250, Protocol: corev1.ProtocolTCP}

	s.AddPorts(p1, p2)

	if len(s.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(s.Spec.Ports))
	}
	if s.Spec.Ports[0].Name != "https" || s.Spec.Ports[1].Name != "metrics" {
		t.Fatalf("unexpected ports: %#v", s.Spec.Ports)
	}
}

func TestAddPort_Convenience(t *testing.T) {
	s := NewService("ns", "svc").
		AddPort("https", 6443, 6443, corev1.ProtocolTCP)

	if len(s.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(s.Spec.Ports))
	}
	p := s.Spec.Ports[0]
	if p.Name != "https" || p.Port != 6443 || p.Protocol != corev1.ProtocolTCP {
		t.Fatalf("unexpected port: %#v", p)
	}
	if p.TargetPort != intstr.FromInt(6443) {
		t.Fatalf("unexpected targetPort: %#v", p.TargetPort)
	}
}

func TestBuild_ReturnsDeepCopy(t *testing.T) {
	s := NewService("ns", "svc").
		WithLabels(map[string]string{"app": "x"}).
		WithSelector(map[string]string{"app": "x"}).
		AddPort("https", 6443, 6443, corev1.ProtocolTCP)

	b1 := s.Build()
	b2 := s.Build()

	if b1 == s.Service {
		t.Fatalf("Build() must not return the same pointer as the template Service")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	// Mutate b1 and ensure template + b2 stay unchanged
	b1.Labels["app"] = "mutated"
	b1.Spec.Selector["app"] = "mutated"
	b1.Spec.Ports[0].Port = 9999

	if s.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", s.Labels)
	}
	if s.Spec.Selector["app"] != "x" {
		t.Fatalf("template selector mutated unexpectedly: %#v", s.Spec.Selector)
	}
	if s.Spec.Ports[0].Port != 6443 {
		t.Fatalf("template ports mutated unexpectedly: %#v", s.Spec.Ports)
	}

	if b2.Labels["app"] != "x" || b2.Spec.Selector["app"] != "x" || b2.Spec.Ports[0].Port != 6443 {
		t.Fatalf("b2 mutated unexpectedly: labels=%#v selector=%#v ports=%#v", b2.Labels, b2.Spec.Selector, b2.Spec.Ports)
	}
}
