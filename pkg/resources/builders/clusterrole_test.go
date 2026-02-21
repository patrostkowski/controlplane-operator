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

func TestNewClusterRole_SetsNameAndMergesMeta(t *testing.T) {
	cr := NewClusterRole().
		WithName("cr1").
		WithLabels(map[string]string{"a": "1"}).
		WithLabels(map[string]string{"a": "2", "b": "3"}).
		WithAnnotations(map[string]string{"x": "y"}).
		WithAnnotations(map[string]string{"x": "yy", "z": "w"})

	if cr == nil || cr.ClusterRole == nil {
		t.Fatalf("expected non-nil ClusterRoleTemplate")
	}
	if cr.Name != "cr1" {
		t.Fatalf("expected name %q, got %q", "cr1", cr.Name)
	}
	if !reflect.DeepEqual(cr.Labels, map[string]string{"a": "2", "b": "3"}) {
		t.Fatalf("unexpected labels: %#v", cr.Labels)
	}
	if !reflect.DeepEqual(cr.Annotations, map[string]string{"x": "yy", "z": "w"}) {
		t.Fatalf("unexpected annotations: %#v", cr.Annotations)
	}
}

func TestClusterRole_WithRules_Appends(t *testing.T) {
	r1 := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get", "list"},
	}
	r2 := rbacv1.PolicyRule{
		APIGroups: []string{"apps"},
		Resources: []string{"deployments"},
		Verbs:     []string{"watch"},
	}

	cr := NewClusterRole().WithName("cr").WithRules(r1).WithRules(r2)
	if len(cr.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(cr.Rules))
	}
	if !reflect.DeepEqual(cr.Rules[0], r1) || !reflect.DeepEqual(cr.Rules[1], r2) {
		t.Fatalf("unexpected rules: %#v", cr.Rules)
	}
}

func TestClusterRole_Build_ReturnsDeepCopy(t *testing.T) {
	cr := NewClusterRole().
		WithName("cr").
		WithLabels(map[string]string{"app": "x"}).
		WithRules(rbacv1.PolicyRule{Resources: []string{"pods"}, Verbs: []string{"get"}})

	b1 := cr.Build()
	b2 := cr.Build()

	if b1 == cr.ClusterRole {
		t.Fatalf("Build() must not return the same pointer as template clusterrole")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.Rules[0].Verbs[0] = "mut"

	if cr.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", cr.Labels)
	}
	if cr.Rules[0].Verbs[0] != "get" {
		t.Fatalf("template rules mutated unexpectedly: %#v", cr.Rules)
	}
	if b2.Labels["app"] != "x" {
		t.Fatalf("b2 labels mutated unexpectedly: %#v", b2.Labels)
	}
	if b2.Rules[0].Verbs[0] != "get" {
		t.Fatalf("b2 rules mutated unexpectedly: %#v", b2.Rules)
	}
}

func TestNewClusterRoleBinding_SetsNameAndRefs(t *testing.T) {
	roleRef := rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cr"}
	s1 := rbacv1.Subject{Kind: "ServiceAccount", Name: "sa1", Namespace: "ns1"}
	s2 := rbacv1.Subject{Kind: "User", Name: "user1"}

	crb := NewClusterRoleBinding().
		WithName("crb1").
		WithRefs(roleRef, s1).
		WithRefs(roleRef, s2)

	if crb == nil || crb.ClusterRoleBinding == nil {
		t.Fatalf("expected non-nil ClusterRoleBindingTemplate")
	}
	if crb.Name != "crb1" {
		t.Fatalf("expected name %q, got %q", "crb1", crb.Name)
	}
	if !reflect.DeepEqual(crb.RoleRef, roleRef) {
		t.Fatalf("unexpected roleRef: %#v", crb.RoleRef)
	}
	if len(crb.Subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(crb.Subjects))
	}
	if !reflect.DeepEqual(crb.Subjects[0], s1) || !reflect.DeepEqual(crb.Subjects[1], s2) {
		t.Fatalf("unexpected subjects: %#v", crb.Subjects)
	}
}

func TestClusterRoleBinding_Build_ReturnsDeepCopy(t *testing.T) {
	roleRef := rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cr"}
	s1 := rbacv1.Subject{Kind: "ServiceAccount", Name: "sa1", Namespace: "ns1"}

	crb := NewClusterRoleBinding().
		WithName("crb").
		WithLabels(map[string]string{"app": "x"}).
		WithRefs(roleRef, s1)

	b1 := crb.Build()
	b2 := crb.Build()

	if b1 == crb.ClusterRoleBinding {
		t.Fatalf("Build() must not return the same pointer as template clusterrolebinding")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	b1.Labels["app"] = "mut"
	b1.Subjects[0].Name = "mut"

	if crb.Labels["app"] != "x" {
		t.Fatalf("template labels mutated unexpectedly: %#v", crb.Labels)
	}
	if crb.Subjects[0].Name != "sa1" {
		t.Fatalf("template subjects mutated unexpectedly: %#v", crb.Subjects)
	}
	if b2.Labels["app"] != "x" {
		t.Fatalf("b2 labels mutated unexpectedly: %#v", b2.Labels)
	}
	if b2.Subjects[0].Name != "sa1" {
		t.Fatalf("b2 subjects mutated unexpectedly: %#v", b2.Subjects)
	}
}
