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
