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
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var EtcdVersion = "3.6.5-0"

type builder struct{ cc *cluster.ClusterContext }

func NewBuilder(cc *cluster.ClusterContext) cluster.ObjectProducer {
	return builder{cc: cc}
}

func (b builder) Objects() []client.Object {
	return []client.Object{
		b.buildService(),
		b.buildStatefulSet(),
	}
}

func (b builder) buildService() *corev1.Service {
	cc := b.cc
	mcp := cc.MCP

	labels := map[string]string{appLabelKey: appLabelVal}
	ns := mcp.Namespace

	return builders.NewService().
		WithName(cc.Names.EtcdServiceName()).
		WithNamespace(ns).
		Headless().
		WithLabels(labels).
		WithSelector(labels).
		AddPort("client", clientPort, clientPort, corev1.ProtocolTCP).
		AddPort("peer", peerPort, peerPort, corev1.ProtocolTCP).
		Build()
}

func (b builder) buildStatefulSet() *appsv1.StatefulSet {
	cc := b.cc
	mcp := cc.MCP

	ca := cc.Names.SecretEtcdCAName()
	srv := cc.Names.SecretEtcdServerTLSName()
	peer := cc.Names.SecretEtcdPeerTLSName()

	labels := map[string]string{appLabelKey: appLabelVal}
	ns := mcp.Namespace
	replicas := int32(1)

	// service must match buildService() name
	svc := cc.Names.EtcdServiceName()

	podFQDNClient := memberName + "." + svc + "." + ns + ".svc:" + utils.PortString(clientPort)
	podFQDNPeer := memberName + "." + svc + "." + ns + ".svc:" + utils.PortString(peerPort)

	vols := []corev1.Volume{
		cc.SecretVolume(ca),
		cc.SecretVolume(srv),
		cc.SecretVolume(peer),
	}
	mounts := []corev1.VolumeMount{
		cc.SecretMount(ca, true),
		cc.SecretMount(srv, true),
		cc.SecretMount(peer, true),
	}

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

			"--trusted-ca-file=" + cc.CAPath(ca),
			"--cert-file=" + cc.CertPath(srv),
			"--key-file=" + cc.KeyPath(srv),

			"--peer-trusted-ca-file=" + cc.CAPath(ca),
			"--peer-cert-file=" + cc.CertPath(peer),
			"--peer-key-file=" + cc.KeyPath(peer),
		},
		Ports: []corev1.ContainerPort{
			{Name: "client", ContainerPort: clientPort},
			{Name: "peer", ContainerPort: peerPort},
		},
		LivenessProbe:  utils.TcpProbe(clientPort, 10, 10),
		ReadinessProbe: utils.TcpProbe(clientPort, 5, 5),
		VolumeMounts: append(
			[]corev1.VolumeMount{
				{Name: cc.Names.EtcdStatefulSetName(), MountPath: dataDir},
			},
			mounts...,
		),
	}

	claim := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: cc.Names.EtcdStatefulSetName(),
		},
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
		AddVolumes(vols...).
		WithVolumeClaims(claim).
		Build()
}
