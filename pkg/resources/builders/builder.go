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
	"maps"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	Objects() []client.Object
}

type HasMeta interface {
	GetMeta() *metav1.ObjectMeta
}

type MetaMutator struct {
	h HasMeta
}

type HasPodTemplate interface {
	GetPodTemplate() *corev1.PodTemplateSpec
}

type PodTemplateMutator struct {
	h HasPodTemplate
}

func (m PodTemplateMutator) WithServiceAccount(sa string) {
	m.h.GetPodTemplate().Spec.ServiceAccountName = sa
}

func (m PodTemplateMutator) WithContainer(c corev1.Container) {
	pt := m.h.GetPodTemplate()
	pt.Spec.Containers = append(pt.Spec.Containers, c)
}

func (m PodTemplateMutator) AddVolumes(vols ...corev1.Volume) {
	pt := m.h.GetPodTemplate()
	pt.Spec.Volumes = append(pt.Spec.Volumes, vols...)
}

func (m *PodTemplateMutator) WithLabels(labels map[string]string) {
	pt := m.h.GetPodTemplate()
	if labels == nil {
		return
	}
	l := pt.GetLabels()
	if l == nil {
		l = map[string]string{}
	}

	maps.Copy(l, labels)
	pt.SetLabels(l)
}

func (m *PodTemplateMutator) WithAnnotations(ann map[string]string) {
	pt := m.h.GetPodTemplate()
	if ann == nil {
		return
	}
	a := pt.GetAnnotations()
	if a == nil {
		a = map[string]string{}
	}

	maps.Copy(a, ann)
	pt.SetLabels(a)
}

func (m PodTemplateMutator) PatchContainer(name string, fn func(*corev1.Container)) bool {
	cs := m.h.GetPodTemplate().Spec.Containers
	for i := range cs {
		if cs[i].Name == name {
			fn(&cs[i])
			return true
		}
	}
	return false
}

func (m PodTemplateMutator) AddVolumeMounts(containerName string, mounts ...corev1.VolumeMount) bool {
	return m.PatchContainer(containerName, func(c *corev1.Container) {
		c.VolumeMounts = append(c.VolumeMounts, mounts...)
	})
}

func (m *MetaMutator) WithLabels(labels map[string]string) {
	om := m.h.GetMeta()
	if labels == nil {
		return
	}
	l := om.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	maps.Copy(l, labels)
	om.SetLabels(l)
}

func (m *MetaMutator) WithAnnotations(ann map[string]string) {
	om := m.h.GetMeta()
	if ann == nil {
		return
	}
	a := om.GetAnnotations()
	if a == nil {
		a = map[string]string{}
	}
	maps.Copy(a, ann)
	om.SetLabels(a)
}

func (m *MetaMutator) WithName(name string) {
	om := m.h.GetMeta()
	if name != "" {
		om.SetName(name)
	}
}

func (m *MetaMutator) WithNamespace(ns string) {
	om := m.h.GetMeta()
	if ns != "" {
		om.SetNamespace(ns)
	}
}
