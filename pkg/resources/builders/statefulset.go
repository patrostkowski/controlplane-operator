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
)

type StatefulSetTemplate struct {
	*appsv1.StatefulSet
	pod PodTemplateMutator
}

func NewStatefulSet(ns, name string, labels map[string]string, replicas int32, serviceName string) *StatefulSetTemplate {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: serviceName,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       corev1.PodSpec{},
			},
		},
	}

	w := &StatefulSetTemplate{StatefulSet: sts}
	w.pod = PodTemplateMutator{obj: w}
	return w
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
