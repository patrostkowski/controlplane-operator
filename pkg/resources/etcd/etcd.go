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
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	return builders.NewService().
		WithName(nameEtcd).
		WithNamespace(ns).
		Headless().
		WithLabels(labels).
		WithSelector(map[string]string(labels)).
		AddPort("client", clientPort, clientPort, corev1.ProtocolTCP).
		AddPort("peer", peerPort, peerPort, corev1.ProtocolTCP).
		Build()
}

func buildStatefulSet(cp *mcpv1alpha1.ManagedControlPlane) *appsv1.StatefulSet {
	p := pki.New(cp).ETCD()

	labels := map[string]string{appLabelKey: appLabelVal}
	ns := cp.Namespace
	replicas := int32(1)

	podFQDNClient := memberName + "." + nameEtcd + "." + ns + ".svc:" + utils.PortString(clientPort)
	podFQDNPeer := memberName + "." + nameEtcd + "." + ns + ".svc:" + utils.PortString(peerPort)

	etcdContainer := corev1.Container{
		Name:  nameEtcd,
		Image: "registry.k8s.io/etcd:" + EtcdVersion,
		Command: []string{
			"etcd",
		},
		Args: []string{
			"--name=" + memberName,
			"--data-dir=" + dataDir,

			"--listen-client-urls=https://0.0.0.0:" + utils.PortString(clientPort),
			"--advertise-client-urls=https://" + podFQDNClient,

			"--listen-peer-urls=https://0.0.0.0:" + utils.PortString(peerPort),
			"--initial-advertise-peer-urls=https://" + podFQDNPeer,

			"--initial-cluster=" + clusterName + "=https://" + podFQDNPeer,
			"--initial-cluster-state=new",

			"--client-cert-auth=true",
			"--peer-client-cert-auth=true",

			"--trusted-ca-file=" + p.CA.CAPath(),
			"--cert-file=" + p.Server.CertPath(),
			"--key-file=" + p.Server.KeyPath(),

			"--peer-trusted-ca-file=" + p.CA.CAPath(),
			"--peer-cert-file=" + p.Peer.CertPath(),
			"--peer-key-file=" + p.Peer.KeyPath(),
		},
		Ports: []corev1.ContainerPort{
			{Name: "client", ContainerPort: clientPort},
			{Name: "peer", ContainerPort: peerPort},
		},
		LivenessProbe:  utils.TcpProbe(clientPort, 10, 10),
		ReadinessProbe: utils.TcpProbe(clientPort, 5, 5),
		VolumeMounts: append(
			[]corev1.VolumeMount{
				{Name: "etcd-data", MountPath: dataDir},
			},
			p.Mounts(true)...,
		),
	}

	claim := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "etcd-data"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(defaultStorage),
				},
			},
		},
	}

	return builders.NewStatefulSet().
		WithName(nameEtcd).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithServiceName(nameEtcd).
		WithContainer(etcdContainer).
		AddVolumes(p.Volumes()...).
		WithVolumeClaims(claim).
		Build()
}
