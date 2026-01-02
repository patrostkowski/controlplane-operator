package builders

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRoleTemplate struct {
	*rbacv1.ClusterRole
	meta MetaMutator
}

type ClusterRoleBindingTemplate struct {
	*rbacv1.ClusterRoleBinding
	meta MetaMutator
}

func NewClusterRole() *ClusterRoleTemplate {
	cr := &rbacv1.ClusterRole{}
	newCR := &ClusterRoleTemplate{ClusterRole: cr}
	newCR.meta = MetaMutator{obj: cr}
	return newCR
}

func NewClusterRoleBinding() *ClusterRoleBindingTemplate {
	crb := &rbacv1.ClusterRoleBinding{}
	newCRB := &ClusterRoleBindingTemplate{ClusterRoleBinding: crb}
	newCRB.meta = MetaMutator{obj: crb}
	return newCRB
}

// ClusterRole
func (c *ClusterRoleTemplate) GetMeta() *metav1.ObjectMeta {
	return &c.ClusterRole.ObjectMeta
}

func (c *ClusterRoleTemplate) WithLabels(labels map[string]string) *ClusterRoleTemplate {
	c.meta.WithLabels(labels)
	return c
}

func (c *ClusterRoleTemplate) WithAnnotations(ann map[string]string) *ClusterRoleTemplate {
	c.meta.WithAnnotations(ann)
	return c
}

func (c *ClusterRoleTemplate) WithName(name string) *ClusterRoleTemplate {
	c.meta.WithName(name)
	return c
}

func (c *ClusterRoleTemplate) WithRules(rules []rbacv1.PolicyRule) *ClusterRoleTemplate {
	c.ClusterRole.Rules = append(c.ClusterRole.Rules, rules...)
	return c
}

func (c *ClusterRoleTemplate) Build() *rbacv1.ClusterRole {
	return c.ClusterRole.DeepCopy()
}

// ClusterRoleBinding
func (b *ClusterRoleBindingTemplate) GetMeta() *metav1.ObjectMeta {
	return &b.ClusterRoleBinding.ObjectMeta
}

func (b *ClusterRoleBindingTemplate) WithLabels(labels map[string]string) *ClusterRoleBindingTemplate {
	b.meta.WithLabels(labels)
	return b
}

func (b *ClusterRoleBindingTemplate) WithAnnotations(ann map[string]string) *ClusterRoleBindingTemplate {
	b.meta.WithAnnotations(ann)
	return b
}

func (b *ClusterRoleBindingTemplate) WithName(name string) *ClusterRoleBindingTemplate {
	b.meta.WithName(name)
	return b
}

func (b *ClusterRoleBindingTemplate) WithRefs(subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) *ClusterRoleBindingTemplate {
	b.ClusterRoleBinding.Subjects = append(b.ClusterRoleBinding.Subjects, subjects...)
	b.ClusterRoleBinding.RoleRef = roleRef
	return b
}

func (b *ClusterRoleBindingTemplate) Build() *rbacv1.ClusterRoleBinding {
	return b.ClusterRoleBinding.DeepCopy()
}
