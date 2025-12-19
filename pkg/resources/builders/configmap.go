package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapTemplate struct {
	*corev1.ConfigMap
}

func NewConfigMap(ns, name string) *ConfigMapTemplate {
	return &ConfigMapTemplate{
		ConfigMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Data:       map[string]string{},
			BinaryData: map[string][]byte{},
		},
	}
}

func (c *ConfigMapTemplate) WithLabels(labels map[string]string) *ConfigMapTemplate {
	if c.Labels == nil {
		c.Labels = map[string]string{}
	}
	for k, v := range labels {
		c.Labels[k] = v
	}
	return c
}

func (c *ConfigMapTemplate) WithAnnotations(ann map[string]string) *ConfigMapTemplate {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	for k, v := range ann {
		c.Annotations[k] = v
	}
	return c
}

func (c *ConfigMapTemplate) Put(key, val string) *ConfigMapTemplate {
	if c.Data == nil {
		c.Data = map[string]string{}
	}
	c.Data[key] = val
	return c
}

func (c *ConfigMapTemplate) PutBytes(key string, val []byte) *ConfigMapTemplate {
	if c.BinaryData == nil {
		c.BinaryData = map[string][]byte{}
	}
	c.BinaryData[key] = val
	return c
}

func (c *ConfigMapTemplate) Build() *corev1.ConfigMap {
	return c.ConfigMap.DeepCopy()
}
