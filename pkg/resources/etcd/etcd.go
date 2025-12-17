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

package etcd

import (
	"path/filepath"
	"strconv"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	appLabelKey = "app"
	appLabelVal = "etcd"

	nameEtcd = "etcd"

	clientPort int32 = 2379
	peerPort   int32 = 2380

	dataDir = "/var/lib/etcd"

	mountRoot = "/etc/etcd/pki"
	dirCA     = "ca"
	dirServer = "server"
	dirPeer   = "peer"

	caCrt  = "ca.crt"
	tlsCrt = "tls.crt"
	tlsKey = "tls.key"

	defaultStorage = "10Gi"

	// Member config (single node for now)
	memberName  = "etcd-0"
	clusterName = "etcd-0"
)

var EtcdVersion = "3.6.5-0"

// Resources returns the Service + StatefulSet required for etcd.
func Resources(cp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildService(cp),
		buildStatefulSet(cp),
	}
}

func buildService(cp *mcpv1alpha1.ManagedControlPlane) *corev1.Service {
	labels := map[string]string{appLabelKey: appLabelVal}
	ns := cp.Namespace

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameEtcd,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{Name: "client", Port: clientPort},
				{Name: "peer", Port: peerPort},
			},
		},
	}
}

func buildStatefulSet(cp *mcpv1alpha1.ManagedControlPlane) *appsv1.StatefulSet {
	labels := map[string]string{appLabelKey: appLabelVal}
	ns := cp.Namespace
	replicas := int32(1)

	secretCA := pki.SecretEtcdCA
	secretServer := pki.SecretEtcdServerTLS
	secretPeer := pki.SecretEtcdPeerTLS

	volCA := secretCA
	volServer := secretServer
	volPeer := secretPeer

	caPath := filepath.Join(mountRoot, dirCA, caCrt)
	serverCrt := filepath.Join(mountRoot, dirServer, tlsCrt)
	serverKey := filepath.Join(mountRoot, dirServer, tlsKey)
	peerCrt := filepath.Join(mountRoot, dirPeer, tlsCrt)
	peerKey := filepath.Join(mountRoot, dirPeer, tlsKey)

	podFQDNClient := memberName + "." + nameEtcd + "." + ns + ".svc:" + portString(clientPort)
	podFQDNPeer := memberName + "." + nameEtcd + "." + ns + ".svc:" + portString(peerPort)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameEtcd,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: nameEtcd,
			Replicas:    &replicas,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  nameEtcd,
							Image: "registry.k8s.io/etcd:" + EtcdVersion,
							Command: []string{
								"etcd",
							},
							Args: []string{
								"--name=" + memberName,
								"--data-dir=" + dataDir,

								"--listen-client-urls=https://0.0.0.0:" + portString(clientPort),
								"--advertise-client-urls=https://" + podFQDNClient,

								"--listen-peer-urls=https://0.0.0.0:" + portString(peerPort),
								"--initial-advertise-peer-urls=https://" + podFQDNPeer,

								"--initial-cluster=" + clusterName + "=https://" + podFQDNPeer,
								"--initial-cluster-state=new",

								"--client-cert-auth=true",
								"--peer-client-cert-auth=true",

								"--trusted-ca-file=" + caPath,
								"--cert-file=" + serverCrt,
								"--key-file=" + serverKey,

								"--peer-trusted-ca-file=" + caPath,
								"--peer-cert-file=" + peerCrt,
								"--peer-key-file=" + peerKey,
							},
							Ports: []corev1.ContainerPort{
								{Name: "client", ContainerPort: clientPort},
								{Name: "peer", ContainerPort: peerPort},
							},
							LivenessProbe:  tcpProbe(clientPort, 10, 10),
							ReadinessProbe: tcpProbe(clientPort, 5, 5),
							VolumeMounts: []corev1.VolumeMount{
								{Name: "etcd-data", MountPath: dataDir},
								secretMount(volServer, filepath.Join(mountRoot, dirServer)),
								secretMount(volPeer, filepath.Join(mountRoot, dirPeer)),
								secretMount(volCA, filepath.Join(mountRoot, dirCA)),
							},
						},
					},
					Volumes: []corev1.Volume{
						secretVol(volServer, secretServer),
						secretVol(volPeer, secretPeer),
						secretVol(volCA, secretCA),
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "etcd-data"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(defaultStorage),
							},
						},
					},
				},
			},
		},
	}
}

func tcpProbe(port int32, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(port)),
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

func secretMount(volumeName, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		ReadOnly:  true,
	}
}

func secretVol(volumeName, secretName string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	}
}

func portString(p int32) string {
	return strconv.Itoa(int(p))
}
