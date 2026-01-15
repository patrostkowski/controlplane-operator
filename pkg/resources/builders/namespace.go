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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceTemplate struct {
	*corev1.Namespace
	meta MetaMutator
}

func NewNamespace() *NamespaceTemplate {
	obj := &corev1.Namespace{}
	b := &NamespaceTemplate{Namespace: obj}
	b.meta = MetaMutator{obj: obj}
	return b
}

func (n *NamespaceTemplate) GetMeta() *metav1.ObjectMeta {
	return &n.Namespace.ObjectMeta
}

func (n *NamespaceTemplate) WithLabels(labels map[string]string) *NamespaceTemplate {
	n.meta.WithLabels(labels)
	return n
}

func (n *NamespaceTemplate) WithAnnotations(ann map[string]string) *NamespaceTemplate {
	n.meta.WithAnnotations(ann)
	return n
}

func (n *NamespaceTemplate) WithName(name string) *NamespaceTemplate {
	n.meta.WithName(name)
	return n
}

func (n *NamespaceTemplate) WithNamespace(ns string) *NamespaceTemplate {
	n.meta.WithNamespace(ns)
	return n
}

func (n *NamespaceTemplate) Build() *corev1.Namespace {
	return n.Namespace.DeepCopy()
}
