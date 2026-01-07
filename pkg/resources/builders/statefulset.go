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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type StatefulSetTemplate struct {
	*appsv1.StatefulSet
	pod  PodTemplateMutator
	meta MetaMutator
}

func NewStatefulSet() *StatefulSetTemplate {
	obj := &appsv1.StatefulSet{}
	b := &StatefulSetTemplate{StatefulSet: obj}
	b.meta = MetaMutator{obj: obj}
	b.pod = PodTemplateMutator{obj: b}
	return b
}

func (s *StatefulSetTemplate) GetMeta() *metav1.ObjectMeta {
	return &s.StatefulSet.ObjectMeta
}

func (s *StatefulSetTemplate) WithLabels(labels map[string]string) *StatefulSetTemplate {
	s.meta.WithLabels(labels)
	return s
}

func (s *StatefulSetTemplate) WithServiceName(name string) *StatefulSetTemplate {
	s.StatefulSet.Spec.ServiceName = name
	return s
}

func (s *StatefulSetTemplate) WithReplicas(replicas int32) *StatefulSetTemplate {
	s.StatefulSet.Spec.Replicas = ptr.To(replicas)
	return s
}

func (s *StatefulSetTemplate) WithSelector(sel map[string]string) *StatefulSetTemplate {
	if s.Spec.Selector == nil {
		s.Spec.Selector = &metav1.LabelSelector{}
	}
	if s.Spec.Selector.MatchLabels == nil {
		s.Spec.Selector.MatchLabels = map[string]string{}
	}
	for k, v := range sel {
		s.Spec.Selector.MatchLabels[k] = v
	}
	s.pod.WithLabels(sel)
	return s
}

func (s *StatefulSetTemplate) WithAnnotations(ann map[string]string) *StatefulSetTemplate {
	s.meta.WithAnnotations(ann)
	return s
}

func (s *StatefulSetTemplate) WithName(name string) *StatefulSetTemplate {
	s.meta.WithName(name)
	return s
}

func (s *StatefulSetTemplate) WithNamespace(ns string) *StatefulSetTemplate {
	s.meta.WithNamespace(ns)
	return s
}

func (w *StatefulSetTemplate) GetPodTemplate() *corev1.PodTemplateSpec {
	return &w.Spec.Template
}

func (w *StatefulSetTemplate) WithServiceAccount(sa string) *StatefulSetTemplate {
	w.pod.WithServiceAccount(sa)
	return w
}

func (w *StatefulSetTemplate) WithContainer(c corev1.Container) *StatefulSetTemplate {
	w.pod.WithContainer(c)
	return w
}

func (w *StatefulSetTemplate) AddVolumes(vols ...corev1.Volume) *StatefulSetTemplate {
	w.pod.AddVolumes(vols...)
	return w
}

func (w *StatefulSetTemplate) AddVolumeMounts(container string, mounts ...corev1.VolumeMount) *StatefulSetTemplate {
	w.pod.AddVolumeMounts(container, mounts...)
	return w
}

func (w *StatefulSetTemplate) PatchContainer(name string, fn func(*corev1.Container)) *StatefulSetTemplate {
	w.pod.PatchContainer(name, fn)
	return w
}

func (w *StatefulSetTemplate) WithVolumeClaims(vcts ...corev1.PersistentVolumeClaim) *StatefulSetTemplate {
	w.Spec.VolumeClaimTemplates = append(w.Spec.VolumeClaimTemplates, vcts...)
	return w
}

func (w *StatefulSetTemplate) Build() *appsv1.StatefulSet {
	return w.StatefulSet.DeepCopy()
}
