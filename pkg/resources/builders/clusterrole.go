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
	obj := &rbacv1.ClusterRole{}
	b := &ClusterRoleTemplate{ClusterRole: obj}
	b.meta = MetaMutator{obj: obj}
	return b
}

func NewClusterRoleBinding() *ClusterRoleBindingTemplate {
	crb := &rbacv1.ClusterRoleBinding{}
	newCRB := &ClusterRoleBindingTemplate{ClusterRoleBinding: crb}
	newCRB.meta = MetaMutator{obj: crb}
	return newCRB
}

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

func (c *ClusterRoleTemplate) WithRules(rules ...rbacv1.PolicyRule) *ClusterRoleTemplate {
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

func (b *ClusterRoleBindingTemplate) WithRefs(roleRef rbacv1.RoleRef, subjects ...rbacv1.Subject) *ClusterRoleBindingTemplate {
	b.ClusterRoleBinding.Subjects = append(b.ClusterRoleBinding.Subjects, subjects...)
	b.ClusterRoleBinding.RoleRef = roleRef
	return b
}

func (b *ClusterRoleBindingTemplate) Build() *rbacv1.ClusterRoleBinding {
	return b.ClusterRoleBinding.DeepCopy()
}
