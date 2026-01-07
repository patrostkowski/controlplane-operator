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
	meta MetaMutator
}

func NewConfigMap() *ConfigMapTemplate {
	obj := &corev1.ConfigMap{}
	b := &ConfigMapTemplate{ConfigMap: obj}
	b.meta = MetaMutator{obj: obj}
	b.Data = map[string]string{}
	b.BinaryData = map[string][]byte{}
	return b
}

func (c *ConfigMapTemplate) GetMeta() *metav1.ObjectMeta {
	return &c.ConfigMap.ObjectMeta
}

func (c *ConfigMapTemplate) WithLabels(labels map[string]string) *ConfigMapTemplate {
	c.meta.WithLabels(labels)
	return c
}

func (c *ConfigMapTemplate) WithAnnotations(ann map[string]string) *ConfigMapTemplate {
	c.meta.WithAnnotations(ann)
	return c
}

func (c *ConfigMapTemplate) WithName(name string) *ConfigMapTemplate {
	c.meta.WithName(name)
	return c
}

func (c *ConfigMapTemplate) WithNamespace(ns string) *ConfigMapTemplate {
	c.meta.WithNamespace(ns)
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
