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

type SecretTemplate struct {
	*corev1.Secret
	meta MetaMutator
}

func NewSecret() *SecretTemplate {
	secret := &corev1.Secret{}
	newSecret := &SecretTemplate{Secret: secret}
	newSecret.meta = MetaMutator{obj: secret}
	newSecret.Data = map[string][]byte{}
	newSecret.StringData = map[string]string{}
	return newSecret
}

func (c *SecretTemplate) GetMeta() *metav1.ObjectMeta {
	return &c.Secret.ObjectMeta
}

func (c *SecretTemplate) WithLabels(labels map[string]string) *SecretTemplate {
	c.meta.WithLabels(labels)
	return c
}

func (c *SecretTemplate) WithAnnotations(ann map[string]string) *SecretTemplate {
	c.meta.WithAnnotations(ann)
	return c
}

func (c *SecretTemplate) WithName(name string) *SecretTemplate {
	c.meta.WithName(name)
	return c
}

func (c *SecretTemplate) WithNamespace(ns string) *SecretTemplate {
	c.meta.WithNamespace(ns)
	return c
}

func (c *SecretTemplate) WithType(t corev1.SecretType) *SecretTemplate {
	c.Secret.Type = t
	return c
}

func (c *SecretTemplate) Put(key, val string) *SecretTemplate {
	if c.StringData == nil {
		c.StringData = map[string]string{}
	}
	c.StringData[key] = val
	return c
}

func (c *SecretTemplate) PutBytes(key string, val []byte) *SecretTemplate {
	if c.Data == nil {
		c.Data = map[string][]byte{}
	}
	c.Data[key] = val
	return c
}

func (c *SecretTemplate) Build() *corev1.Secret {
	return c.Secret.DeepCopy()
}
