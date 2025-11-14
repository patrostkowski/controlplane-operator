// Copyright 2025 controlplane.patrostkowski.dev
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
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns ConfigMap + Deployment for kube-scheduler.
func Resources(ms *mcpv1alpha1.ManagedScheduler) []client.Object {
	ns := ms.Namespace

	return []client.Object{
		buildConfigMap(ns),
		buildDeployment(ns),
	}
}

func buildConfigMap(namespace string) *corev1.ConfigMap {
	kcfg := buildSchedulerKubeconfig(namespace)

	// marshal kubeconfig to YAML
	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		// should never happen with in-memory cfg
		panic(err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduler-kubeconfig",
			Namespace: namespace,
		},
		Data: map[string]string{
			"scheduler.conf": string(kubeconfigData),
		},
	}
}

func buildSchedulerKubeconfig(namespace string) *clientcmdapi.Config {
	serverURL := "https://kube-apiserver." + namespace + ".svc:443"

	cfg := clientcmdapi.NewConfig()

	// --- Cluster ---
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:               serverURL,
		CertificateAuthority: "/var/run/k8s/ca/ca.crt",
	}

	// --- User ---
	cfg.AuthInfos["scheduler"] = &clientcmdapi.AuthInfo{
		ClientCertificate: "/var/run/k8s/scheduler/tls.crt",
		ClientKey:         "/var/run/k8s/scheduler/tls.key",
	}

	// --- Context ---
	cfg.Contexts["local"] = &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "scheduler",
	}

	cfg.CurrentContext = "local"

	return cfg
}

func buildDeployment(namespace string) *appsv1.Deployment {
	labels := map[string]string{
		"app": "ks",
	}
	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-scheduler",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "ks",
							Image:           "registry.k8s.io/kube-scheduler:v1.31.3",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"kube-scheduler",
							},
							Args: []string{
								"--bind-address=0.0.0.0",
								"--kubeconfig=/etc/kubernetes/scheduler.conf",
								"--authentication-kubeconfig=/etc/kubernetes/scheduler.conf",
								"--authorization-kubeconfig=/etc/kubernetes/scheduler.conf",
								"--leader-elect=true",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "https",
									ContainerPort: 10259,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTPS,
										Host:   "127.0.0.1",
										Port:   intstr.FromInt(10259),
										Path:   "/livez",
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTPS,
										Host:   "127.0.0.1",
										Port:   intstr.FromInt(10259),
										Path:   "/readyz",
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kcfg",
									MountPath: "/etc/kubernetes",
									ReadOnly:  true,
								},
								{
									Name:      "schcert",
									MountPath: "/var/run/k8s/scheduler",
									ReadOnly:  true,
								},
								{
									Name:      "kubernetes-ca",
									MountPath: "/var/run/k8s/ca",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kcfg",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "scheduler-kubeconfig",
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "scheduler.conf",
											Path: "scheduler.conf",
										},
									},
								},
							},
						},
						{
							Name: "schcert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "scheduler-client",
								},
							},
						},
						{
							Name: "kubernetes-ca",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "managed-ca",
								},
							},
						},
					},
				},
			},
		},
	}
}
