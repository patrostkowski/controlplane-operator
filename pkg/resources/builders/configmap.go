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
