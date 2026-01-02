package builders

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleTemplate struct {
	*rbacv1.Role
	meta MetaMutator
}

type RoleBindingTemplate struct {
	*rbacv1.RoleBinding
	meta MetaMutator
}

func NewRole() *RoleTemplate {
	r := &rbacv1.Role{}
	newRole := &RoleTemplate{Role: r}
	newRole.meta = MetaMutator{obj: r}
	return newRole
}

func NewRoleBinding() *RoleBindingTemplate {
	rb := &rbacv1.RoleBinding{}
	newRB := &RoleBindingTemplate{RoleBinding: rb}
	newRB.meta = MetaMutator{obj: rb}
	return newRB
}

// Role
func (r *RoleTemplate) GetMeta() *metav1.ObjectMeta {
	return &r.Role.ObjectMeta
}

func (r *RoleTemplate) WithLabels(labels map[string]string) *RoleTemplate {
	r.meta.WithLabels(labels)
	return r
}

func (r *RoleTemplate) WithAnnotations(ann map[string]string) *RoleTemplate {
	r.meta.WithAnnotations(ann)
	return r
}

func (r *RoleTemplate) WithName(name string) *RoleTemplate {
	r.meta.WithName(name)
	return r
}

func (r *RoleTemplate) WithNamespace(ns string) *RoleTemplate {
	r.meta.WithNamespace(ns)
	return r
}

func (r *RoleTemplate) WithRules(rules []rbacv1.PolicyRule) *RoleTemplate {
	r.Role.Rules = append(r.Role.Rules, rules...)
	return r
}

func (r *RoleTemplate) Build() *rbacv1.Role {
	return r.Role.DeepCopy()
}

// RoleBinding
func (b *RoleBindingTemplate) GetMeta() *metav1.ObjectMeta {
	return &b.RoleBinding.ObjectMeta
}

func (b *RoleBindingTemplate) WithLabels(labels map[string]string) *RoleBindingTemplate {
	b.meta.WithLabels(labels)
	return b
}

func (b *RoleBindingTemplate) WithAnnotations(ann map[string]string) *RoleBindingTemplate {
	b.meta.WithAnnotations(ann)
	return b
}

func (b *RoleBindingTemplate) WithName(name string) *RoleBindingTemplate {
	b.meta.WithName(name)
	return b
}

func (b *RoleBindingTemplate) WithNamespace(ns string) *RoleBindingTemplate {
	b.meta.WithNamespace(ns)
	return b
}

func (b *RoleBindingTemplate) WithRefs(subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) *RoleBindingTemplate {
	b.RoleBinding.Subjects = append(b.RoleBinding.Subjects, subjects...)
	b.RoleBinding.RoleRef = roleRef
	return b
}

func (b *RoleBindingTemplate) Build() *rbacv1.RoleBinding {
	return b.RoleBinding.DeepCopy()
}
