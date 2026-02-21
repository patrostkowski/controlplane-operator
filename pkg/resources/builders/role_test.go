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

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestNewRole_SetsNameNamespaceAndMergesMeta(t *testing.T) {
	r := NewRole().
		WithName("r1").
		WithNamespace("ns1")
	if r == nil || r.Role == nil {
		t.Fatalf("expected non-nil RoleTemplate")
	}
	if r.Name != "r1" || r.Namespace != "ns1" {
		t.Fatalf("unexpected metadata: %s/%s", r.Namespace, r.Name)
	}

	// merge behavior
	r.Labels = nil
	r.Annotations = nil
	r.WithLabels(map[string]string{"a": "1"}).
		WithLabels(map[string]string{"a": "2", "b": "3"}).
		WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	if !reflect.DeepEqual(r.Labels, map[string]string{"a": "2", "b": "3"}) {
		t.Fatalf("unexpected labels: %#v", r.Labels)
	}
	if !reflect.DeepEqual(r.Annotations, map[string]string{"x": "yy", "z": "w"}) {
		t.Fatalf("unexpected annotations: %#v", r.Annotations)
	}
}

func TestRole_WithRules_Appends(t *testing.T) {
	r1 := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get", "list"},
	}
	r2 := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"secrets"},
		Verbs:     []string{"get"},
	}

	role := NewRole().WithName("r").WithNamespace("ns").WithRules(r1).WithRules(r2)
	if len(role.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(role.Rules))
	}
	if !reflect.DeepEqual(role.Rules[0], r1) || !reflect.DeepEqual(role.Rules[1], r2) {
		t.Fatalf("unexpected rules: %#v", role.Rules)
	}
}

func TestRole_Build_ReturnsDeepCopy(t *testing.T) {
	role := NewRole().
		WithName("r").
		WithNamespace("ns").
		WithLabels(map[string]string{"app": "x"}).
		WithRules(rbacv1.PolicyRule{Resources: []string{"pods"}, Verbs: []string{"get"}})

	b1 := role.Build()
	b2 := role.Build()

	if b1 == role.Role {
		t.Fatalf("Build() must not return the same pointer as template role")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.Rules[0].Verbs[0] = "mut"

	if role.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", role.Labels)
	}
	if role.Rules[0].Verbs[0] != "get" {
		t.Fatalf("template rules mutated unexpectedly: %#v", role.Rules)
	}
	if b2.Labels["app"] != "x" || b2.Rules[0].Verbs[0] != "get" {
		t.Fatalf("b2 mutated unexpectedly: labels=%#v rules=%#v", b2.Labels, b2.Rules)
	}
}

func TestNewRoleBinding_SetsNameNamespaceAndRefs(t *testing.T) {
	roleRef := rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: "r"}
	s1 := rbacv1.Subject{Kind: "ServiceAccount", Name: "sa1", Namespace: "ns1"}
	s2 := rbacv1.Subject{Kind: "User", Name: "user1"}

	rb := NewRoleBinding().
		WithName("rb1").
		WithNamespace("ns1").
		WithRefs(roleRef, s1).
		WithRefs(roleRef, s2)

	if rb == nil || rb.RoleBinding == nil {
		t.Fatalf("expected non-nil RoleBindingTemplate")
	}
	if rb.Name != "rb1" || rb.Namespace != "ns1" {
		t.Fatalf("unexpected metadata: %s/%s", rb.Namespace, rb.Name)
	}
	if !reflect.DeepEqual(rb.RoleRef, roleRef) {
		t.Fatalf("unexpected roleRef: %#v", rb.RoleRef)
	}
	if len(rb.Subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(rb.Subjects))
	}
	if !reflect.DeepEqual(rb.Subjects[0], s1) || !reflect.DeepEqual(rb.Subjects[1], s2) {
		t.Fatalf("unexpected subjects: %#v", rb.Subjects)
	}
}

func TestRoleBinding_Build_ReturnsDeepCopy(t *testing.T) {
	roleRef := rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: "r"}
	s1 := rbacv1.Subject{Kind: "ServiceAccount", Name: "sa1", Namespace: "ns1"}

	rb := NewRoleBinding().
		WithName("rb").
		WithNamespace("ns").
		WithLabels(map[string]string{"app": "x"}).
		WithRefs(roleRef, s1)

	b1 := rb.Build()
	b2 := rb.Build()

	if b1 == rb.RoleBinding {
		t.Fatalf("Build() must not return the same pointer as template rolebinding")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.Subjects[0].Name = "mut"

	if rb.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", rb.Labels)
	}
	if rb.Subjects[0].Name != "sa1" {
		t.Fatalf("template subjects mutated unexpectedly: %#v", rb.Subjects)
	}
	if b2.Labels["app"] != "x" || b2.Subjects[0].Name != "sa1" {
		t.Fatalf("b2 mutated unexpectedly: labels=%#v subjects=%#v", b2.Labels, b2.Subjects)
	}
}
