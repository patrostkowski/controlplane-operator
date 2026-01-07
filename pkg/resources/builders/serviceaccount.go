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

type ServiceAccountTemplate struct {
	*corev1.ServiceAccount
	meta MetaMutator
}

// TODO: add automount token & pull secrets
func NewServiceAccount() *ServiceAccountTemplate {
	obj := &corev1.ServiceAccount{}
	b := &ServiceAccountTemplate{ServiceAccount: obj}
	b.meta = MetaMutator{obj: obj}
	return b
}

func (s *ServiceAccountTemplate) GetMeta() *metav1.ObjectMeta {
	return &s.ServiceAccount.ObjectMeta
}

func (s *ServiceAccountTemplate) WithLabels(labels map[string]string) *ServiceAccountTemplate {
	s.meta.WithLabels(labels)
	return s
}

func (s *ServiceAccountTemplate) WithAnnotations(ann map[string]string) *ServiceAccountTemplate {
	s.meta.WithAnnotations(ann)
	return s
}

func (s *ServiceAccountTemplate) WithName(name string) *ServiceAccountTemplate {
	s.meta.WithName(name)
	return s
}

func (s *ServiceAccountTemplate) WithNamespace(ns string) *ServiceAccountTemplate {
	s.meta.WithNamespace(ns)
	return s
}

func (s *ServiceAccountTemplate) Build() *corev1.ServiceAccount {
	return s.ServiceAccount.DeepCopy()
}
