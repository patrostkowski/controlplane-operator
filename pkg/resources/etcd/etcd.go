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

type builder struct{ cc cluster.ETCDSpec }

func NewBuilder(cc cluster.ETCDSpec) cluster.ObjectProducer {
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
	e := cc.Etcd()

	labels := map[string]string{appLabelKey: appLabelVal}
	ns := cc.Namespace()

	return builders.NewService().
		WithName(e.ServiceName()).
		WithNamespace(ns).
		Headless().
		WithLabels(labels).
		WithSelector(labels).
		AddPort("client", e.ClientPort(), e.ClientPort(), corev1.ProtocolTCP).
		AddPort("peer", e.PeerPort(), e.PeerPort(), corev1.ProtocolTCP).
		Build()
}

func (b builder) buildStatefulSet() *appsv1.StatefulSet {
	cc := b.cc
	e := cc.Etcd()

	ca := e.CASecret()
	srv := e.ServerTLSSecret()
	peer := e.PeerTLSSecret()

	labels := map[string]string{appLabelKey: appLabelVal}
	ns := cc.Namespace()
	replicas := int32(1)

	// service must match buildService() name
	svc := e.ServiceName()

	podFQDNClient := e.MemberFQDNClient()
	podFQDNPeer := e.MemberFQDNPeer()

	health := e.HealthClientTLSSecret()

	vols := []corev1.Volume{
		cc.SecretVolume(ca),
		cc.SecretVolume(srv),
		cc.SecretVolume(peer),
		cc.SecretVolume(health),
	}
	mounts := []corev1.VolumeMount{
		cc.SecretMount(ca, true),
		cc.SecretMount(srv, true),
		cc.SecretMount(peer, true),
		cc.SecretMount(health, true),
	}

	etcdContainer := corev1.Container{
		Name:  svc,
		Image: "registry.k8s.io/etcd:" + EtcdVersion,
		Command: []string{
			"etcd",
		},
		Env: []corev1.EnvVar{{Name: "ETCDCTL_API", Value: "3"}},
		Args: []string{
			"--name=" + e.MemberName(),
			"--data-dir=" + e.DataDir(),

			"--listen-client-urls=https://0.0.0.0:" + utils.PortString(e.ClientPort()),
			"--advertise-client-urls=https://" + podFQDNClient,

			"--listen-peer-urls=https://0.0.0.0:" + utils.PortString(e.PeerPort()),
			"--initial-advertise-peer-urls=https://" + podFQDNPeer,

			"--initial-cluster=" + e.MemberName() + "=https://" + podFQDNPeer,
			"--initial-cluster-state=new",

			"--client-cert-auth=true",
			"--peer-client-cert-auth=true",

			"--trusted-ca-file=" + e.CAPath(),
			"--cert-file=" + e.ServerCertPath(),
			"--key-file=" + e.ServerKeyPath(),

			"--peer-trusted-ca-file=" + e.CAPath(),
			"--peer-cert-file=" + e.PeerCertPath(),
			"--peer-key-file=" + e.PeerKeyPath(),
		},
		Ports: []corev1.ContainerPort{
			{Name: "client", ContainerPort: clientPort},
			{Name: "peer", ContainerPort: peerPort},
		},
		StartupProbe:   utils.ETCDHealthProbe(e.CAPath(), e.HealthClientCertPath(), e.HealthClientKeyPath()),
		LivenessProbe:  utils.ETCDHealthProbe(e.CAPath(), e.HealthClientCertPath(), e.HealthClientKeyPath()),
		ReadinessProbe: utils.ETCDHealthProbe(e.CAPath(), e.HealthClientCertPath(), e.HealthClientKeyPath()),
		VolumeMounts: append(
			[]corev1.VolumeMount{
				{Name: e.StatefulSetName(), MountPath: dataDir},
			},
			mounts...,
		),
	}

	claim := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: e.StatefulSetName()},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(e.DefaultStorage()),
				},
			},
		},
	}

	return builders.NewStatefulSet().
		WithName(e.StatefulSetName()).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithServiceName(svc).
		WithContainer(etcdContainer).
		AddVolumes(vols...).
		WithVolumeClaims(claim).
		Build()
}
