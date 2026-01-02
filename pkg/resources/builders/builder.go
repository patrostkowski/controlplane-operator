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

type MetaMutator struct {
	obj metav1.Object
}

type HasPodTemplate interface {
	GetPodTemplate() *corev1.PodTemplateSpec
}

type PodTemplateMutator struct {
	obj HasPodTemplate
}

func (m PodTemplateMutator) WithServiceAccount(sa string) {
	m.obj.GetPodTemplate().Spec.ServiceAccountName = sa
}

func (m PodTemplateMutator) WithContainer(c corev1.Container) {
	pt := m.obj.GetPodTemplate()
	pt.Spec.Containers = append(pt.Spec.Containers, c)
}

func (m PodTemplateMutator) AddVolumes(vols ...corev1.Volume) {
	pt := m.obj.GetPodTemplate()
	pt.Spec.Volumes = append(pt.Spec.Volumes, vols...)
}

func (m PodTemplateMutator) WithAffinity(aff corev1.Affinity) {
	pt := m.obj.GetPodTemplate()
	pt.Spec.Affinity = &aff
}

func (m PodTemplateMutator) WithNodeSelector(nodeSelector map[string]string) {
	pt := m.obj.GetPodTemplate()
	if nodeSelector == nil {
		return
	}
	n := pt.Spec.NodeSelector
	if n == nil {
		n = map[string]string{}
		pt.Spec.NodeSelector = n
	}
	maps.Copy(n, nodeSelector)
}

func (m PodTemplateMutator) WithTolerations(toleration ...corev1.Toleration) {
	pt := m.obj.GetPodTemplate()
	pt.Spec.Tolerations = append(pt.Spec.Tolerations, toleration...)
}

func (m PodTemplateMutator) WithInitContainers(initContainer ...corev1.Container) {
	pt := m.obj.GetPodTemplate()
	pt.Spec.InitContainers = append(pt.Spec.InitContainers, initContainer...)
}

func (m PodTemplateMutator) WithPriorityClass(name string) {
	pt := m.obj.GetPodTemplate()
	pt.Spec.PriorityClassName = name
}

func (m *PodTemplateMutator) WithLabels(labels map[string]string) {
	pt := m.obj.GetPodTemplate()
	if labels == nil {
		return
	}
	l := pt.GetLabels()
	if l == nil {
		l = map[string]string{}
		pt.SetLabels(l)
	}
	maps.Copy(l, labels)
}

func (m *PodTemplateMutator) WithAnnotations(ann map[string]string) {
	pt := m.obj.GetPodTemplate()
	if ann == nil {
		return
	}
	a := pt.GetAnnotations()
	if a == nil {
		a = map[string]string{}
		pt.SetLabels(a)
	}
	maps.Copy(a, ann)
}

func (m PodTemplateMutator) PatchContainer(name string, fn func(*corev1.Container)) bool {
	cs := m.obj.GetPodTemplate().Spec.Containers
	for i := range cs {
		if cs[i].Name == name {
			fn(&cs[i])
			return true
		}
	}
	return false
}

func (m PodTemplateMutator) WithHostNetwork() {
	pt := m.obj.GetPodTemplate()
	pt.Spec.HostNetwork = true
}

func (m PodTemplateMutator) AddVolumeMounts(containerName string, mounts ...corev1.VolumeMount) bool {
	return m.PatchContainer(containerName, func(c *corev1.Container) {
		c.VolumeMounts = append(c.VolumeMounts, mounts...)
	})
}

func (m *MetaMutator) WithLabels(labels map[string]string) {
	l := m.obj.GetLabels()
	if l == nil {
		l = map[string]string{}
		m.obj.SetLabels(l)
	}
	maps.Copy(l, labels)
}

func (m *MetaMutator) WithAnnotations(ann map[string]string) {
	a := m.obj.GetAnnotations()
	if a == nil {
		a = map[string]string{}
		m.obj.SetLabels(a)
	}
	maps.Copy(a, ann)
}

func (m *MetaMutator) WithName(name string) {
	if name != "" {
		m.obj.SetName(name)
	}
}

func (m *MetaMutator) WithNamespace(ns string) {
	if ns != "" {
		m.obj.SetNamespace(ns)
	}
}
