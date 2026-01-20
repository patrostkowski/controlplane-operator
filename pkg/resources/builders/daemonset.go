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

type DaemonsetTemplate struct {
	*appsv1.DaemonSet
	pod  PodTemplateMutator
	meta MetaMutator
}

func NewDaemonSet() *DaemonsetTemplate {
	obj := &appsv1.DaemonSet{}
	b := &DaemonsetTemplate{DaemonSet: obj}
	b.meta = MetaMutator{obj: obj}
	b.pod = PodTemplateMutator{obj: b}
	return b
}

func (w *DaemonsetTemplate) GetMeta() *metav1.ObjectMeta {
	return &w.DaemonSet.ObjectMeta
}

func (w *DaemonsetTemplate) WithLabels(labels map[string]string) *DaemonsetTemplate {
	w.meta.WithLabels(labels)
	return w
}

func (w *DaemonsetTemplate) WithSelector(sel map[string]string) *DaemonsetTemplate {
	if w.Spec.Selector == nil {
		w.Spec.Selector = &metav1.LabelSelector{}
	}
	if w.Spec.Selector.MatchLabels == nil {
		w.Spec.Selector.MatchLabels = map[string]string{}
	}
	for k, v := range sel {
		w.Spec.Selector.MatchLabels[k] = v
	}
	return w
}

func (w *DaemonsetTemplate) WithPodLabels(labels map[string]string) *DaemonsetTemplate {
	w.pod.WithLabels(labels)
	return w
}

func (w *DaemonsetTemplate) WithAnnotations(ann map[string]string) *DaemonsetTemplate {
	w.meta.WithAnnotations(ann)
	return w
}

func (w *DaemonsetTemplate) WithName(name string) *DaemonsetTemplate {
	w.meta.WithName(name)
	return w
}

func (w *DaemonsetTemplate) WithNamespace(ns string) *DaemonsetTemplate {
	w.meta.WithNamespace(ns)
	return w
}

func (w *DaemonsetTemplate) GetPodTemplate() *corev1.PodTemplateSpec {
	return &w.Spec.Template
}

func (w *DaemonsetTemplate) WithServiceAccount(sa string) *DaemonsetTemplate {
	w.pod.WithServiceAccount(sa)
	return w
}

func (w *DaemonsetTemplate) WithAffinity(aff corev1.Affinity) *DaemonsetTemplate {
	w.pod.WithAffinity(aff)
	return w
}

func (w *DaemonsetTemplate) WithTolerations(toleration ...corev1.Toleration) *DaemonsetTemplate {
	w.pod.WithTolerations(toleration...)
	return w
}

func (w *DaemonsetTemplate) WithNodeSelector(sel map[string]string) *DaemonsetTemplate {
	w.pod.WithNodeSelector(sel)
	return w
}

func (w *DaemonsetTemplate) WithUpdateStrategy(strategy appsv1.DaemonSetUpdateStrategyType) *DaemonsetTemplate {
	w.Spec.UpdateStrategy.Type = strategy
	return w
}

func (w *DaemonsetTemplate) WithHostNetwork() *DaemonsetTemplate {
	w.pod.WithHostNetwork()
	return w
}

func (w *DaemonsetTemplate) WithDNSPolicy(policy corev1.DNSPolicy) *DaemonsetTemplate {
	w.pod.WithDNSPolicy(policy)
	return w
}

func (w *DaemonsetTemplate) WithPriorityClass(name string) *DaemonsetTemplate {
	w.pod.WithPriorityClass(name)
	return w
}

func (w *DaemonsetTemplate) WithContainer(c corev1.Container) *DaemonsetTemplate {
	w.pod.WithContainer(c)
	return w
}

func (w *DaemonsetTemplate) WithInitContainers(c ...corev1.Container) *DaemonsetTemplate {
	w.pod.WithInitContainers(c...)
	return w
}

func (w *DaemonsetTemplate) AddVolumes(vols ...corev1.Volume) *DaemonsetTemplate {
	w.pod.AddVolumes(vols...)
	return w
}

func (w *DaemonsetTemplate) AddVolumeMounts(containerName string, mounts ...corev1.VolumeMount) *DaemonsetTemplate {
	w.pod.AddVolumeMounts(containerName, mounts...)
	return w
}

func (w *DaemonsetTemplate) PatchContainer(name string, fn func(*corev1.Container)) *DaemonsetTemplate {
	w.pod.PatchContainer(name, fn)
	return w
}

func (w *DaemonsetTemplate) Build() *appsv1.DaemonSet {
	return w.DaemonSet.DeepCopy()
}
