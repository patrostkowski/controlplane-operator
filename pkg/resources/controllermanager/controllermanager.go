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
	"path/filepath"

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

// Resources returns ConfigMap + Deployment for kube-controller-manager.
func Resources(cm *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildConfigMap(cm),
		buildDeployment(cm),
	}
}

func buildConfigMap(cm *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	ns := cm.Namespace
	kcfg := buildControllerManagerKubeconfig(ns)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err) // should never happen
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

func buildControllerManagerKubeconfig(namespace string) *clientcmdapi.Config {
	serverURL := "https://" + apiserver.KubeAPIServerSvcName + "." + namespace + ".svc:" + utils.PortString(apiserver.KubeAPIServerSecurePort)

	cfg := clientcmdapi.NewConfig()

	// Reuse PKI secret names as mount directory names.
	caDir := filepath.Join(common.PKIMountRoot, pki.SecretManagedCA)
	cmClientDir := filepath.Join(common.PKIMountRoot, pki.SecretCMClient)

	// --- Cluster ---
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:               serverURL,
		CertificateAuthority: filepath.Join(caDir, common.TLSCrtKey),
	}

	// --- User ---
	cfg.AuthInfos["cm"] = &clientcmdapi.AuthInfo{
		ClientCertificate: filepath.Join(cmClientDir, common.TLSCrtKey),
		ClientKey:         filepath.Join(cmClientDir, common.TLSKeyKey),
	}

	// --- Context ---
	cfg.Contexts["local"] = &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "cm",
	}
	cfg.CurrentContext = "local"

	return cfg
}

func buildDeployment(cm *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	ns := cm.Namespace
	version := cm.Spec.Version

	labels := map[string]string{common.LabelKeyApp: labelValApp}

	// PKI secret names (canonical)
	secretCA := pki.SecretManagedCA
	secretSA := pki.SecretSASigner
	secretCMClient := pki.SecretCMClient

	// Mount dirs based on secret names (same pattern as apiserver/etcd)
	caDir := filepath.Join(common.PKIMountRoot, secretCA)
	saDir := filepath.Join(common.PKIMountRoot, secretSA)
	cmClientDir := filepath.Join(common.PKIMountRoot, secretCMClient)

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
							Image:           "registry.k8s.io/kube-controller-manager:" + version,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"kube-controller-manager"},
							Args: []string{
								"--bind-address=0.0.0.0",

								"--cluster-name=" + cm.GetObjectMeta().GetName(),

								// kubeconfig wiring
								"--kubeconfig=" + kubeconfigPath,
								"--authentication-kubeconfig=" + kubeconfigPath,
								"--authorization-kubeconfig=" + kubeconfigPath,

								// leader election + sa creds
								"--leader-elect=true",
								"--use-service-account-credentials=true",
								"--controllers=*,bootstrapsigner,tokencleaner",

								// service account signing key
								"--service-account-private-key-file=" + filepath.Join(saDir, common.TLSKeyKey),

								// cluster signing
								"--cluster-signing-cert-file=" + filepath.Join(caDir, common.TLSCrtKey),
								"--cluster-signing-key-file=" + filepath.Join(caDir, common.TLSKeyKey),

								// CA wiring
								"--client-ca-file=" + filepath.Join(caDir, common.TLSCrtKey),
								"--root-ca-file=" + filepath.Join(caDir, common.TLSCrtKey),

								// networking
								"--cluster-cidr=" + cm.Spec.Networking.PodCIDR,
								"--allocate-node-cidrs=true",
							},
							Ports: []corev1.ContainerPort{
								{Name: "https", ContainerPort: securePort},
							},
							StartupProbe:   httpsHealthProbe(securePort, healthPath, 10, 10, 15, 24),
							LivenessProbe:  httpsHealthProbe(securePort, healthPath, 10, 10, 15, 8),
							ReadinessProbe: httpsHealthProbe(securePort, healthPath, 5, 5, 15, 3),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      volKubeconfig,
									MountPath: kubeconfigMountDir,
									ReadOnly:  true,
								},
								secretMount(secretCMClient, cmClientDir),
								secretMount(secretSA, saDir),
								secretMount(secretCA, caDir),
							},
						},
					},
					Volumes: []corev1.Volume{
						configMapVolume(cmKubeconfigName),
						secretVolume(secretCMClient),
						secretVolume(secretSA),
						secretVolume(secretCA),
					},
				},
			},
		},
	}
}

const volKubeconfig = "kcfg"

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

func secretMount(volumeName, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		ReadOnly:  true,
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

func httpsHealthProbe(port int32, path string, initialDelay, period, timeout, failureThreshold int32) *corev1.Probe {
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
		TimeoutSeconds:      timeout,
		FailureThreshold:    failureThreshold,
		SuccessThreshold:    1,
	}
}
