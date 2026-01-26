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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiserverv1beta "k8s.io/apiserver/pkg/apis/apiserver/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var KonnectivityServerVersion = "v0.1.3"

const (
	egressConnectionNameCluster      = "cluster"
	egressConnectionNameControlPlane = "controlplane"
	egressConnectionGRPCType         = apiserverv1beta.ProtocolGRPC
	egressConnectionDirectType       = apiserverv1beta.ProtocolDirect
	konnectivityUDSFile              = "konnectivity-uds.sock"
)

// EndpointResources returns only the Service (LB) for kube-apiserver
// and Service for Konnectivity server(sidecar)
func EndpointResources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildService(mcp),
	}
}

// WorkloadResources returns only the Deployment for kube-apiserver
func WorkloadResources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildDeployment(mcp),
		buildKonnectivityConfigMap(mcp),
	}
}

func buildService(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.Service {
	ns := mcp.Namespace
	labels := map[string]string{appLabelKey: appLabelVal}

	return builders.NewService().
		WithName(apiServerName).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		AddPort("https", 6443, 6443, corev1.ProtocolTCP).
		AddPort("grpc", 8132, 8132, corev1.ProtocolTCP).
		WithType(corev1.ServiceTypeLoadBalancer).
		Build()
}

func buildKonnectivityConfigMap(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	ns := mcp.Namespace
	socketPath := filepath.Join(common.PKIMountRoot, konnectivityServerUDS, konnectivityUDSFile)

	e := apiserverv1beta.EgressSelectorConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       common.EgressSelectorKind,
			APIVersion: common.EgressSelectorAPIVersion,
		},
		EgressSelections: []apiserverv1beta.EgressSelection{
			{
				Name: egressConnectionNameCluster,
				Connection: apiserverv1beta.Connection{
					ProxyProtocol: egressConnectionGRPCType,
					Transport: &apiserverv1beta.Transport{
						UDS: &apiserverv1beta.UDSTransport{
							UDSName: socketPath,
						},
					},
				},
			},
			{
				Name: egressConnectionNameControlPlane,
				Connection: apiserverv1beta.Connection{
					ProxyProtocol: egressConnectionDirectType,
				},
			},
		},
	}

	y := utils.GetObjYaml(&e)

	return builders.NewConfigMap().
		WithName(konnectivityConfigMapName).
		WithNamespace(ns).
		Put(konnectivityConfigMapKey, y).
		Build()
}

func buildDeployment(mcp *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	p := pki.New(mcp).APIServer()

	ns := mcp.Namespace
	labels := map[string]string{appLabelKey: appLabelVal}
	replicas := int32(1)
	version := mcp.Spec.Kubernetes.Version

	konnectivityConfigVolume := utils.ConfigMapVolume(
		konnectivityConfigVolumeName,
		konnectivityConfigMapName,
		konnectivityConfigMapKey,
		konnectivityConfFileName,
	)
	konnectivityConfigVolumeMount := corev1.VolumeMount{
		Name:      konnectivityConfigVolumeName,
		ReadOnly:  true,
		MountPath: konnectivityServerMountDir,
	}

	konnectivityUDSDir := filepath.Join(common.PKIMountRoot, konnectivityServerUDS)
	udsFile := filepath.Join(konnectivityUDSDir, konnectivityUDSFile)
	konnectivityUDSVolume := utils.EmptyDirVolume(konnectivityServerUDS)

	konnectivityUDSVolumeMount := corev1.VolumeMount{
		Name:      konnectivityServerUDS,
		ReadOnly:  false,
		MountPath: konnectivityUDSDir,
	}

	c := corev1.Container{
		Name:            "apiserver",
		Image:           "registry.k8s.io/kube-apiserver:" + version,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"kube-apiserver"},
		Args: []string{
			"--advertise-address=" + mcp.Status.Address,
			"--bind-address=0.0.0.0",
			"--secure-port=6443",
			"--service-cluster-ip-range=" + mcp.Spec.Kubernetes.Networking.ServiceCIDR,

			// etcd
			"--etcd-servers=https://etcd-0.etcd." + ns + ".svc:2379",
			"--etcd-cafile=" + p.EtcdCA.CertPath(),
			"--etcd-certfile=" + p.EtcdClient.CertPath(),
			"--etcd-keyfile=" + p.EtcdClient.KeyPath(),

			// serving + client-ca
			"--client-ca-file=" + p.ClientCA.CertPath(),
			"--tls-cert-file=" + p.Serving.CertPath(),
			"--tls-private-key-file=" + p.Serving.KeyPath(),

			// kubelet client
			"--kubelet-client-certificate=" + p.KubeletClient.CertPath(),
			"--kubelet-client-key=" + p.KubeletClient.KeyPath(),
			"--kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalIP,ExternalDNS",

			"--authorization-mode=Node,RBAC",
			"--enable-bootstrap-token-auth=true",
			"--enable-admission-plugins=NodeRestriction",

			// service account signing
			"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
			"--service-account-key-file=" + p.ServiceAccountSigner.CertPath(),
			"--service-account-signing-key-file=" + p.ServiceAccountSigner.KeyPath(),

			"--allow-privileged=true",

			"--egress-selector-config-file=" + konnectivityConfFilePath,

			"--logging-format=json",
		},
		Ports: []corev1.ContainerPort{
			{Name: "https", ContainerPort: securePort},
		},
		LivenessProbe:  utils.HttpsHealthProbe(securePort, common.LivezPath, 10, 10, 10, 10),
		ReadinessProbe: utils.HttpsHealthProbe(securePort, common.ReadyzPath, 5, 5, 5, 5),
	}

	c2 := corev1.Container{
		Name:  konnectivityServerName,
		Image: "registry.k8s.io/kas-network-proxy/proxy-server:" + KonnectivityServerVersion,
		Args: []string{
			"--mode=grpc",
			"--uds-name=" + udsFile,
			"--delete-existing-uds-file=true",
			"--cluster-cert=" + p.KonnectivityServing.CertPath(),
			"--cluster-key=" + p.KonnectivityServing.KeyPath(),
			"--cluster-ca-cert=" + p.KonnectivityCA.CertPath(),
			"--agent-port=" + utils.PortString(konnectivityServerPort),
			"--server-port=0",
			"--v=2",
		},
	}

	return builders.NewDeployment().
		WithName(apiServerName).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithContainer(c).
		AddVolumes(
			konnectivityUDSVolume,
			konnectivityConfigVolume,
			p.ClientCA.Volume(),
			p.Serving.Volume(),
			p.EtcdClient.Volume(),
			p.KubeletClient.Volume(),
			p.ServiceAccountSigner.Volume(),
			p.EtcdCA.Volume(),
			p.FrontProxyCA.Volume(),
			p.FrontProxyClient.Volume(),
			p.KonnectivityServing.Volume(),
			p.KonnectivityCA.Volume(),
		).
		AddVolumeMounts(c.Name,
			konnectivityConfigVolumeMount,
			konnectivityUDSVolumeMount,
			p.ClientCA.Mount(true),
			p.Serving.Mount(true),
			p.EtcdClient.Mount(true),
			p.KubeletClient.Mount(true),
			p.ServiceAccountSigner.Mount(true),
			p.EtcdCA.Mount(true),
			p.FrontProxyCA.Mount(true),
			p.FrontProxyClient.Mount(true),
		).
		WithContainer(c2).
		AddVolumeMounts(c2.Name,
			konnectivityUDSVolumeMount,
			p.KonnectivityServing.Mount(true),
			p.KonnectivityCA.Mount(true),
		).
		Build()
}
