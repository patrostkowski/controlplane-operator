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

package apiserver

import (
	"path/filepath"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns Service + Deployment for the kube-apiserver.
func Resources(api *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildService(api),
		buildDeployment(api),
	}
}

// EndpointResources returns only the Service (LB) for kube-apiserver.
func EndpointResources(api *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{buildService(api)}
}

// WorkloadResources returns only the Deployment for kube-apiserver.
func WorkloadResources(api *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{buildDeployment(api)}
}

func buildService(api *mcpv1alpha1.ManagedControlPlane) *corev1.Service {
	ns := api.Namespace
	labels := map[string]string{appLabelKey: appLabelVal}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeAPIServerSvcName,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       securePort,
					TargetPort: intstrFromInt(securePort),
				},
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}
}

func buildDeployment(api *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	ns := api.Namespace
	labels := map[string]string{appLabelKey: appLabelVal}
	replicas := int32(1)

	version := api.Spec.Version

	// Volume names == mounted directory names under mountRoot
	caVol := pki.SecretManagedCA
	apiTLSVol := pki.SecretAPIServerTLS
	etcdCAVol := pki.SecretEtcdCA
	etcdClientVol := pki.SecretAPIServerEtcd
	kubeletClientVol := pki.SecretAPIServerKubelet
	saVol := pki.SecretSASigner
	frontProxyCAVol := pki.SecretFrontProxyCA
	frontProxyClientVol := pki.SecretFrontProxyClient

	caDir := filepath.Join(common.PKIMountRoot, caVol)
	apiTLSDir := filepath.Join(common.PKIMountRoot, apiTLSVol)
	etcdCADir := filepath.Join(common.PKIMountRoot, etcdCAVol)
	etcdClientDir := filepath.Join(common.PKIMountRoot, etcdClientVol)
	kubeletClientDir := filepath.Join(common.PKIMountRoot, kubeletClientVol)
	saDir := filepath.Join(common.PKIMountRoot, saVol)
	frontProxyCADir := filepath.Join(common.PKIMountRoot, frontProxyCAVol)
	frontProxyClientDir := filepath.Join(common.PKIMountRoot, frontProxyClientVol)

	// Helper: compute file paths inside mounts
	certPath := func(vol string) string { return filepath.Join(common.PKIMountRoot, vol, common.TLSCrtKey) }
	keyPath := func(vol string) string { return filepath.Join(common.PKIMountRoot, vol, common.TLSKeyKey) }

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServerName,
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
							Name:            "apiserver",
							Image:           "registry.k8s.io/kube-apiserver:" + version,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"kube-apiserver"},
							Args: []string{
								"--advertise-address=" + api.Status.Address,
								"--bind-address=0.0.0.0",
								"--secure-port=6443",
								"--service-cluster-ip-range=" + api.Spec.Networking.ServiceCIDR,

								// etcd
								"--etcd-servers=https://etcd-0.etcd." + ns + ".svc:2379",
								"--etcd-cafile=" + certPath(etcdCAVol),
								"--etcd-certfile=" + certPath(etcdClientVol),
								"--etcd-keyfile=" + keyPath(etcdClientVol),

								// serving + client-ca
								"--client-ca-file=" + certPath(caVol),
								"--tls-cert-file=" + certPath(apiTLSVol),
								"--tls-private-key-file=" + keyPath(apiTLSVol),

								// kubelet client
								"--kubelet-client-certificate=" + certPath(kubeletClientVol),
								"--kubelet-client-key=" + keyPath(kubeletClientVol),
								"--kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalIP,ExternalDNS",

								"--authorization-mode=Node,RBAC",
								"--enable-bootstrap-token-auth=true",

								// service account signing
								"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
								"--service-account-key-file=" + certPath(saVol),
								"--service-account-signing-key-file=" + keyPath(saVol),

								"--allow-privileged=true",

								// front-proxy
								"--requestheader-client-ca-file=" + certPath(frontProxyCAVol),
								"--requestheader-allowed-names=" + pki.CNFrontProxyClient,
								"--requestheader-extra-headers-prefix=X-Remote-Extra-",
								"--requestheader-group-headers=X-Remote-Group",
								"--requestheader-username-headers=X-Remote-User",
								"--proxy-client-cert-file=" + certPath(frontProxyClientVol),
								"--proxy-client-key-file=" + keyPath(frontProxyClientVol),
							},
							Ports: []corev1.ContainerPort{
								{Name: "https", ContainerPort: securePort},
							},
							LivenessProbe:  probeHTTPS(securePort, "/livez", 10, 10),
							ReadinessProbe: probeHTTPS(securePort, "/readyz", 5, 5),
							VolumeMounts: []corev1.VolumeMount{
								utils.SecretMount(caVol, caDir),
								utils.SecretMount(apiTLSVol, apiTLSDir),
								utils.SecretMount(etcdCAVol, etcdCADir),
								utils.SecretMount(etcdClientVol, etcdClientDir),
								utils.SecretMount(kubeletClientVol, kubeletClientDir),
								utils.SecretMount(saVol, saDir),
								utils.SecretMount(frontProxyCAVol, frontProxyCADir),
								utils.SecretMount(frontProxyClientVol, frontProxyClientDir),
							},
						},
					},
					Volumes: []corev1.Volume{
						secretVol(caVol),
						secretVol(apiTLSVol),
						secretVol(etcdCAVol),
						secretVol(etcdClientVol),
						secretVol(kubeletClientVol),
						secretVol(saVol),
						secretVol(frontProxyCAVol),
						secretVol(frontProxyClientVol),
					},
				},
			},
		},
	}
}

func probeHTTPS(port int32, path string, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Host:   "127.0.0.1",
				Port:   intstrFromInt(port),
				Path:   path,
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

func intstrFromInt(port int32) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: port}
}

func secretVol(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	}
}
