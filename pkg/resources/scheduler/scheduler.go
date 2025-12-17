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

package scheduler

import (
	"path/filepath"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns ConfigMap + Deployment for kube-scheduler.
func Resources(ms *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildConfigMap(ms),
		buildDeployment(ms),
	}
}

func buildConfigMap(ms *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	ns := ms.Namespace
	kcfg := utils.BuildComponentKubeconfig(
		ns,
		apiserver.KubeAPIServerSvcName,
		apiserver.KubeAPIServerSecurePort,
		"scheduler",
		pki.SecretSchedulerClient,
	)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmKubeconfigName,
			Namespace: ns,
		},
		Data: map[string]string{
			cmKubeconfigKey: string(kubeconfigData),
		},
	}
}

func buildDeployment(ms *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	ns := ms.Namespace
	version := ms.Spec.Version

	labels := map[string]string{common.LabelKeyApp: labelValApp}

	secretCA := pki.SecretManagedCA
	secretSchedulerClient := pki.SecretSchedulerClient

	caDir := filepath.Join(common.PKIMountRoot, secretCA)
	schedulerClientDir := filepath.Join(common.PKIMountRoot, secretSchedulerClient)

	// Runtime
	var replicas int32 = 1

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      componentName,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            containerName,
							Image:           "registry.k8s.io/kube-scheduler:" + version,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"kube-scheduler"},
							Args: []string{
								"--bind-address=0.0.0.0",
								"--kubeconfig=" + kubeconfigPath,
								"--authentication-kubeconfig=" + kubeconfigPath,
								"--authorization-kubeconfig=" + kubeconfigPath,
								"--leader-elect=true",
							},
							Ports: []corev1.ContainerPort{
								{Name: "https", ContainerPort: securePort},
							},
							LivenessProbe:  utils.HttpsHealthProbe(securePort, common.LivezPath, 10, 10, 10, 10),
							ReadinessProbe: utils.HttpsHealthProbe(securePort, common.ReadyzPath, 5, 5, 5, 5),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      common.KubeconfigVolumeName,
									MountPath: kubeconfigMountDir,
									ReadOnly:  true,
								},
								utils.SecretMount(secretSchedulerClient, schedulerClientDir),
								utils.SecretMount(secretCA, caDir),
							},
						},
					},
					Volumes: []corev1.Volume{
						utils.ConfigMapVolume(
							common.KubeconfigVolumeName,
							cmKubeconfigName,
							cmKubeconfigKey,
							cmKubeconfigFileName,
						),
						utils.SecretVolume(secretSchedulerClient, secretSchedulerClient),
						utils.SecretVolume(secretCA, secretSchedulerClient),
					},
				},
			},
		},
	}
}
