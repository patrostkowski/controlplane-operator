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

type DeploymentTemplate struct {
	*appsv1.Deployment
	pod  PodTemplateMutator
	meta MetaMutator
}

func NewDeployment() *DeploymentTemplate {
	obj := &appsv1.Deployment{}
	b := &DeploymentTemplate{Deployment: obj}
	b.meta = MetaMutator{obj: obj}
	b.pod = PodTemplateMutator{obj: b}
	return b
}

func (w *DeploymentTemplate) GetMeta() *metav1.ObjectMeta {
	return &w.Deployment.ObjectMeta
}

func (w *DeploymentTemplate) WithLabels(labels map[string]string) *DeploymentTemplate {
	w.meta.WithLabels(labels)
	return w
}

func (w *DeploymentTemplate) WithSelector(sel map[string]string) *DeploymentTemplate {
	if w.Spec.Selector.MatchLabels == nil {
		w.Spec.Selector.MatchLabels = map[string]string{}
	}
	for k, v := range sel {
		w.Spec.Selector.MatchLabels[k] = v
	}
	w.pod.WithLabels(sel)
	return w
}

func (w *DeploymentTemplate) WithAnnotations(ann map[string]string) *DeploymentTemplate {
	w.meta.WithAnnotations(ann)
	return w
}

func (w *DeploymentTemplate) WithName(name string) *DeploymentTemplate {
	w.meta.WithName(name)
	return w
}

func (w *DeploymentTemplate) WithNamespace(ns string) *DeploymentTemplate {
	w.meta.WithNamespace(ns)
	return w
}

func (w *DeploymentTemplate) WithReplicas(replicas int32) *DeploymentTemplate {
	w.Deployment.Spec.Replicas = ptr.To(replicas)
	return w
}

func (w *DeploymentTemplate) GetPodTemplate() *corev1.PodTemplateSpec {
	return &w.Spec.Template
}

func (w *DeploymentTemplate) WithServiceAccount(sa string) *DeploymentTemplate {
	w.pod.WithServiceAccount(sa)
	return w
}

func (w *DeploymentTemplate) WithContainer(c corev1.Container) *DeploymentTemplate {
	w.pod.WithContainer(c)
	return w
}

func (w *DeploymentTemplate) AddVolumes(vols ...corev1.Volume) *DeploymentTemplate {
	w.pod.AddVolumes(vols...)
	return w
}

func (w *DeploymentTemplate) AddVolumeMounts(containerName string, mounts ...corev1.VolumeMount) *DeploymentTemplate {
	w.pod.AddVolumeMounts(containerName, mounts...)
	return w
}

func (w *DeploymentTemplate) PatchContainer(name string, fn func(*corev1.Container)) *DeploymentTemplate {
	w.pod.PatchContainer(name, fn)
	return w
}

func (w *DeploymentTemplate) Build() *appsv1.Deployment {
	return w.Deployment.DeepCopy()
}
