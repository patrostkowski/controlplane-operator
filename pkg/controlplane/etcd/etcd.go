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

package etcd

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns the Service + StatefulSet required for etcd.
func Resources(etcdObj *mcpv1alpha1.ManagedETCD) []client.Object {
	ns := etcdObj.Namespace
	// For now we hardcode "etcd" to match your cert SANs: etcd-0.etcd.<ns>.svc
	name := "etcd"

	return []client.Object{
		buildService(ns, name),
		buildStatefulSet(ns, name),
	}
}

func buildService(namespace, name string) *corev1.Service {
	labels := map[string]string{
		"app": "etcd",
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None", // headless for StatefulSet
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{
					Name: "client",
					Port: 2379,
				},
				{
					Name: "peer",
					Port: 2380,
				},
			},
		},
	}
}

func buildStatefulSet(namespace, name string) *appsv1.StatefulSet {
	labels := map[string]string{
		"app": "etcd",
	}

	replicas := int32(1)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &replicas,
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
							Name:  "etcd",
							Image: "registry.k8s.io/etcd:3.5.15-0",
							Command: []string{
								"etcd",
							},
							Args: []string{
								"--name=etcd-0",
								"--data-dir=/var/lib/etcd",
								"--listen-client-urls=https://0.0.0.0:2379",
								"--advertise-client-urls=https://etcd-0." + name + "." + namespace + ".svc:2379",
								"--listen-peer-urls=https://0.0.0.0:2380",
								"--initial-advertise-peer-urls=https://etcd-0." + name + "." + namespace + ".svc:2380",
								"--initial-cluster=etcd-0=https://etcd-0." + name + "." + namespace + ".svc:2380",
								"--initial-cluster-state=new",
								"--client-cert-auth=true",
								"--peer-client-cert-auth=true",
								"--trusted-ca-file=/etc/etcd/pki/ca/ca.crt",
								"--cert-file=/etc/etcd/pki/server/tls.crt",
								"--key-file=/etc/etcd/pki/server/tls.key",
								"--peer-trusted-ca-file=/etc/etcd/pki/ca/ca.crt",
								"--peer-cert-file=/etc/etcd/pki/peer/tls.crt",
								"--peer-key-file=/etc/etcd/pki/peer/tls.key",
							},
							Ports: []corev1.ContainerPort{
								{Name: "client", ContainerPort: 2379},
								{Name: "peer", ContainerPort: 2380},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(2379),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(2379),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "etcd-data",
									MountPath: "/var/lib/etcd",
								},
								{
									Name:      "etcd-server-tls",
									MountPath: "/etc/etcd/pki/server",
									ReadOnly:  true,
								},
								{
									Name:      "etcd-peer-tls",
									MountPath: "/etc/etcd/pki/peer",
									ReadOnly:  true,
								},
								{
									Name:      "etcd-ca",
									MountPath: "/etc/etcd/pki/ca",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "etcd-server-tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "etcd-server-tls",
								},
							},
						},
						{
							Name: "etcd-peer-tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "etcd-peer-tls",
								},
							},
						},
						{
							Name: "etcd-ca",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "etcd-ca",
								},
							},
						},
					},
				},
			},
			// Use PVC for data; default StorageClass will be used when StorageClassName is nil
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "etcd-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
						// StorageClassName: nil -> use default StorageClass
					},
				},
			},
		},
	}
}
