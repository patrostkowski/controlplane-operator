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

package apiserver

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns Service + Deployment for the kube-apiserver.
func Resources(api *mcpv1alpha1.ManagedAPIServer) []client.Object {
	return []client.Object{
		buildService(api),
		buildDeployment(api),
	}
}

func buildService(api *mcpv1alpha1.ManagedAPIServer) *corev1.Service {
	ns := api.Namespace
	name := "kube-apiserver"
	labels := map[string]string{
		"app": "kube-apiserver",
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstrFromInt(6443),
				},
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}
}

func buildDeployment(api *mcpv1alpha1.ManagedAPIServer) *appsv1.Deployment {
	ns := api.Namespace
	name := "kube-apiserver"
	labels := map[string]string{"app": "kube-apiserver"}
	replicas := int32(1)

	version := api.Spec.Version

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
							Name:            "apiserver",
							Image:           "registry.k8s.io/kube-apiserver" + ":" + version,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"kube-apiserver",
							},
							Args: []string{
								"--advertise-address=0.0.0.0",
								"--bind-address=0.0.0.0",
								"--secure-port=6443",
								"--service-cluster-ip-range=10.200.0.0/16",

								"--etcd-servers=https://etcd-0.etcd." + ns + ".svc:2379",
								"--etcd-cafile=/var/run/k8s/etcd-ca/ca.crt",
								"--etcd-certfile=/var/run/k8s/etcd-client/tls.crt",
								"--etcd-keyfile=/var/run/k8s/etcd-client/tls.key",

								"--client-ca-file=/var/run/k8s/ca/tls.crt",
								"--tls-cert-file=/var/run/k8s/apiserver/tls.crt",
								"--tls-private-key-file=/var/run/k8s/apiserver/tls.key",

								"--kubelet-client-certificate=/var/run/k8s/kubelet-client/tls.crt",
								"--kubelet-client-key=/var/run/k8s/kubelet-client/tls.key",
								"--kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalIP,ExternalDNS",

								"--authorization-mode=Node,RBAC",
								"--enable-bootstrap-token-auth=true",

								"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
								"--service-account-key-file=/var/run/k8s/sa/tls.crt",
								"--service-account-signing-key-file=/var/run/k8s/sa/tls.key",

								"--allow-privileged=true",

								"--requestheader-client-ca-file=/var/run/k8s/front-proxy-ca/tls.crt",
								"--requestheader-allowed-names=front-proxy-client",
								"--requestheader-extra-headers-prefix=X-Remote-Extra-",
								"--requestheader-group-headers=X-Remote-Group",
								"--requestheader-username-headers=X-Remote-User",
								"--proxy-client-cert-file=/var/run/k8s/front-proxy-client/tls.crt",
								"--proxy-client-key-file=/var/run/k8s/front-proxy-client/tls.key",
							},
							Ports: []corev1.ContainerPort{
								{Name: "https", ContainerPort: 6443},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTPS,
										Host:   "127.0.0.1",
										Port:   intstrFromInt(6443),
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
										Port:   intstrFromInt(6443),
										Path:   "/readyz",
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "ca", MountPath: "/var/run/k8s/ca", ReadOnly: true},
								{Name: "apiserver-tls", MountPath: "/var/run/k8s/apiserver", ReadOnly: true},
								{Name: "etcd-ca", MountPath: "/var/run/k8s/etcd-ca", ReadOnly: true},
								{Name: "etcd-client", MountPath: "/var/run/k8s/etcd-client", ReadOnly: true},
								{Name: "kubelet-client", MountPath: "/var/run/k8s/kubelet-client", ReadOnly: true},
								{Name: "sa-signer", MountPath: "/var/run/k8s/sa", ReadOnly: true},
								{Name: "front-proxy-ca", MountPath: "/var/run/k8s/front-proxy-ca", ReadOnly: true},
								{Name: "front-proxy-client", MountPath: "/var/run/k8s/front-proxy-client", ReadOnly: true},
							},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "ca", VolumeSource: secretVolume("managed-ca")},
						{Name: "apiserver-tls", VolumeSource: secretVolume("apiserver-tls")},
						{Name: "etcd-ca", VolumeSource: secretVolume("etcd-ca")},
						{Name: "etcd-client", VolumeSource: secretVolume("apiserver-etcd-client")},
						{Name: "kubelet-client", VolumeSource: secretVolume("apiserver-kubelet-client")},
						{Name: "sa-signer", VolumeSource: secretVolume("sa-signer")},
						{Name: "front-proxy-ca", VolumeSource: secretVolume("front-proxy-ca")},
						{Name: "front-proxy-client", VolumeSource: secretVolume("front-proxy-client")},
					},
				},
			},
		},
	}
}

func intstrFromInt(port int32) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: port}
}

func secretVolume(name string) corev1.VolumeSource {
	return corev1.VolumeSource{
		Secret: &corev1.SecretVolumeSource{
			SecretName: name,
		},
	}
}
