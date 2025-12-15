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

package controllermanager

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

// Resources returns ConfigMap + Deployment for kube-controller-manager.
func Resources(cm *mcpv1alpha1.ManagedControllerManager) []client.Object {
	return []client.Object{
		buildConfigMap(cm),
		buildDeployment(cm),
	}
}

func buildConfigMap(cm *mcpv1alpha1.ManagedControllerManager) *corev1.ConfigMap {
	ns := cm.Namespace
	kcfg := buildControllerManagerKubeconfig(ns)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err) // should never happen
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "controller-kubeconfig",
			Namespace: ns,
		},
		Data: map[string]string{
			"controller-manager.conf": string(kubeconfigData),
		},
	}
}

func buildControllerManagerKubeconfig(namespace string) *clientcmdapi.Config {
	serverURL := "https://kube-apiserver." + namespace + ".svc:6443"

	cfg := clientcmdapi.NewConfig()

	// --- Cluster ---
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:               serverURL,
		CertificateAuthority: "/var/run/k8s/ca/tls.crt",
	}

	// --- User ---
	cfg.AuthInfos["cm"] = &clientcmdapi.AuthInfo{
		ClientCertificate: "/var/run/k8s/cm/tls.crt",
		ClientKey:         "/var/run/k8s/cm/tls.key",
	}

	// --- Context ---
	cfg.Contexts["local"] = &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "cm",
	}

	cfg.CurrentContext = "local"

	return cfg
}

func buildDeployment(cm *mcpv1alpha1.ManagedControllerManager) *appsv1.Deployment {
	ns := cm.Namespace
	labels := map[string]string{"app": "kcm"}
	replicas := int32(1)
	version := cm.Spec.Version

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-controller-manager",
			Namespace: ns,
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
							Name:            "kcm",
							Image:           "registry.k8s.io/kube-controller-manager" + ":" + version,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"kube-controller-manager",
							},
							Args: []string{
								// match kind-style bind address
								"--bind-address=127.0.0.1",
								"--cluster-name=managed",
								"--kubeconfig=/etc/kubernetes/controller-manager.conf",
								"--authentication-kubeconfig=/etc/kubernetes/controller-manager.conf",
								"--authorization-kubeconfig=/etc/kubernetes/controller-manager.conf",
								"--leader-elect=true",
								"--use-service-account-credentials=true",
								"--controllers=*,bootstrapsigner,tokencleaner",
								"--allocate-node-cidrs=true",
								"--service-account-private-key-file=/var/run/k8s/sa/tls.key",
								"--cluster-signing-cert-file=/var/run/k8s/ca/tls.crt",
								"--cluster-signing-key-file=/var/run/k8s/ca/tls.key",
								"--client-ca-file=/var/run/k8s/ca/tls.crt",
								"--root-ca-file=/var/run/k8s/ca/tls.crt",

								// optional but often nice to mirror apiserver:
								// "--service-cluster-ip-range=10.200.0.0/16",
								"--cluster-cidr=10.244.0.0/16", // TODO: adjust via CR
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "https",
									ContainerPort: 10257,
								},
							},
							// kind uses /healthz over HTTPS on port 10257
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTPS,
										Host:   "127.0.0.1",
										Port:   intstr.FromInt(10257),
										Path:   "/healthz",
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								TimeoutSeconds:      15,
								FailureThreshold:    24,
								SuccessThreshold:    1,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTPS,
										Host:   "127.0.0.1",
										Port:   intstr.FromInt(10257),
										Path:   "/healthz",
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								TimeoutSeconds:      15,
								FailureThreshold:    8,
								SuccessThreshold:    1,
							},
							// readiness on the same /healthz endpoint is fine here
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTPS,
										Host:   "127.0.0.1",
										Port:   intstr.FromInt(10257),
										Path:   "/healthz",
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
								TimeoutSeconds:      15,
								FailureThreshold:    3,
								SuccessThreshold:    1,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kcfg",
									MountPath: "/etc/kubernetes",
									ReadOnly:  true,
								},
								{
									Name:      "cmcert",
									MountPath: "/var/run/k8s/cm",
									ReadOnly:  true,
								},
								{
									Name:      "sa-signer",
									MountPath: "/var/run/k8s/sa",
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
										Name: "controller-kubeconfig",
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "controller-manager.conf",
											Path: "controller-manager.conf",
										},
									},
								},
							},
						},
						{
							Name: "cmcert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "cm-client",
								},
							},
						},
						{
							Name: "sa-signer",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "sa-signer",
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
