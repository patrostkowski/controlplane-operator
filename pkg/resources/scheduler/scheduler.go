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
	"strconv"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	kcfg := buildSchedulerKubeconfig(ns)

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

func buildSchedulerKubeconfig(namespace string) *clientcmdapi.Config {
	serverURL := "https://" + apiserver.KubeAPIServerSvcName + "." + namespace + ".svc:" + strconv.Itoa(int(apiserver.KubeAPIServerSecurePort))

	cfg := clientcmdapi.NewConfig()

	secretCA := pki.SecretManagedCA
	secretSchedulerClient := pki.SecretSchedulerClient

	caDir := filepath.Join(common.PKIMountRoot, secretCA)
	schedulerClientDir := filepath.Join(common.PKIMountRoot, secretSchedulerClient)

	// --- Cluster ---
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:               serverURL,
		CertificateAuthority: filepath.Join(caDir, common.TLSCrtKey),
	}

	// --- User ---
	cfg.AuthInfos["scheduler"] = &clientcmdapi.AuthInfo{
		ClientCertificate: filepath.Join(schedulerClientDir, common.TLSCrtKey),
		ClientKey:         filepath.Join(schedulerClientDir, common.TLSKeyKey),
	}

	// --- Context ---
	cfg.Contexts["local"] = &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "scheduler",
	}
	cfg.CurrentContext = "local"

	return cfg
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
							LivenessProbe:  httpsProbe(securePort, livezPath, 10, 10),
							ReadinessProbe: httpsProbe(securePort, readyzPath, 5, 5),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volKubeconfig,
									MountPath: kubeconfigMountDir,
									ReadOnly:  true,
								},
								utils.SecretMount(secretSchedulerClient, schedulerClientDir),
								utils.SecretMount(secretCA, caDir),
							},
						},
					},
					Volumes: []corev1.Volume{
						configMapVolume(cmKubeconfigName),
						secretVolume(secretSchedulerClient),
						secretVolume(secretCA),
					},
				},
			},
		},
	}
}

func configMapVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: volKubeconfig,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
				Items: []corev1.KeyToPath{
					{Key: cmKubeconfigKey, Path: cmKubeconfigFileName},
				},
			},
		},
	}
}

func secretVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	}
}

func httpsProbe(port int32, path string, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Host:   "127.0.0.1",
				Port:   intstr.FromInt(int(port)),
				Path:   path,
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}
