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

type DeploymentTemplate struct {
	*appsv1.Deployment
	pod PodTemplateMutator
}

func NewDeployment(ns, name string, labels map[string]string, replicas int32) *DeploymentTemplate {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       corev1.PodSpec{},
			},
		},
	}
	w := &DeploymentTemplate{Deployment: d}
	w.pod = PodTemplateMutator{obj: w}
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
