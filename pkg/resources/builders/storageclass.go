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
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StorageClassTemplate struct {
	*storagev1.StorageClass
	meta MetaMutator
}

func NewStorageClass() *StorageClassTemplate {
	sc := &storagev1.StorageClass{}
	newSC := &StorageClassTemplate{StorageClass: sc}
	newSC.meta = MetaMutator{obj: sc}
	return newSC
}

func (s *StorageClassTemplate) GetMeta() *metav1.ObjectMeta {
	return &s.StorageClass.ObjectMeta
}

func (s *StorageClassTemplate) WithLabels(labels map[string]string) *StorageClassTemplate {
	s.meta.WithLabels(labels)
	return s
}

func (s *StorageClassTemplate) WithAnnotations(ann map[string]string) *StorageClassTemplate {
	s.meta.WithAnnotations(ann)
	return s
}

func (s *StorageClassTemplate) WithName(name string) *StorageClassTemplate {
	s.meta.WithName(name)
	return s
}

func (s *StorageClassTemplate) WithProvisioner(name string) *StorageClassTemplate {
	s.StorageClass.Provisioner = name
	return s
}

func (s *StorageClassTemplate) WithPolicy(policy corev1.PersistentVolumeReclaimPolicy) *StorageClassTemplate {
	s.StorageClass.ReclaimPolicy = &policy
	return s
}

func (s *StorageClassTemplate) WithBindingMode(mode storagev1.VolumeBindingMode) *StorageClassTemplate {
	s.StorageClass.VolumeBindingMode = &mode
	return s
}

func (s *StorageClassTemplate) Build() *storagev1.StorageClass {
	return s.StorageClass.DeepCopy()
}
