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

	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
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

type endpointBuilder struct {
	cc *cluster.ClusterContext
}

func NewEndpointBuilder(cc *cluster.ClusterContext) cluster.ObjectProducer {
	return endpointBuilder{cc: cc}
}

// Objects implements cluster.ObjectProducer
func (e endpointBuilder) Objects() []client.Object {
	return []client.Object{
		e.buildService(),
	}
}

type workloadBuilder struct {
	cc *cluster.ClusterContext
}

func NewWorkloadBuilder(cc *cluster.ClusterContext) cluster.ObjectProducer {
	return workloadBuilder{cc: cc}
}

// Objects implements cluster.ObjectProducer
func (w workloadBuilder) Objects() []client.Object {
	return []client.Object{
		w.buildDeployment(),
		w.buildKonnectivityConfigMap(),
	}
}

func (e endpointBuilder) buildService() *corev1.Service {
	cc := e.cc
	mcp := cc.MCP
	ns := mcp.Namespace
	labels := map[string]string{cc.Contract.APIServer.AppLabelKey: cc.Contract.APIServer.AppLabelVal}

	return builders.NewService().
		WithName(cc.Names.APIServerServiceName()).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		AddPort("https", 6443, 6443, corev1.ProtocolTCP).
		AddPort("grpc", 8132, 8132, corev1.ProtocolTCP).
		WithType(corev1.ServiceTypeLoadBalancer).
		Build()
}

func (w workloadBuilder) buildKonnectivityConfigMap() *corev1.ConfigMap {
	cc := w.cc
	mcp := cc.MCP
	ns := mcp.Namespace

	socketPath := filepath.Join(cc.Layout.PKIRoot, konnectivityServerUDS, konnectivityUDSFile)

	e := apiserverv1beta.EgressSelectorConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       egressSelectorKind,
			APIVersion: egressSelectorAPIVersion,
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
		WithName(cc.Names.KonnectivityConfigMapName()).
		WithNamespace(ns).
		Put(konnectivityConfigMapKey, y).
		Build()
}

func (w workloadBuilder) buildDeployment() *appsv1.Deployment {
	cc := w.cc
	mcp := cc.MCP
	ns := mcp.Namespace

	clientCA := cc.Names.SecretManagedCAName()
	serving := cc.Names.SecretAPIServerServingTLSName()
	etcdCA := cc.Names.SecretEtcdCAName()
	etcdClient := cc.Names.SecretAPIServerEtcdClientName()
	kubeletClient := cc.Names.SecretAPIServerKubeletClientName()
	saSigner := cc.Names.SecretServiceAccountSignerName()
	fpCA := cc.Names.SecretFrontProxyCAName()
	fpClient := cc.Names.SecretFrontProxyClientName()
	konCA := cc.Names.SecretKonnectivityCAName()
	konSrv := cc.Names.SecretKonnectivityServerTLSName()

	labels := map[string]string{cc.Contract.APIServer.AppLabelKey: cc.Contract.APIServer.AppLabelVal}
	replicas := int32(1)
	version := mcp.Spec.Kubernetes.Version

	konnectivityConfigVolume := utils.ConfigMapVolume(
		konnectivityConfigVolumeName,
		cc.Names.KonnectivityConfigMapName(),
		konnectivityConfigMapKey,
		konnectivityConfFileName,
	)

	konnectivityConfigVolumeMount := corev1.VolumeMount{
		Name:      konnectivityConfigVolumeName,
		ReadOnly:  true,
		MountPath: konnectivityServerMountDir,
	}

	konnectivityUDSDir := filepath.Join(cc.Layout.PKIRoot, konnectivityServerUDS)
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

			"--etcd-servers=https://etcd-0." + cc.Names.EtcdServiceName() + "." + ns + ".svc.cluster.local:2379",

			"--etcd-cafile=" + cc.CAPath(etcdCA),
			"--etcd-certfile=" + cc.CertPath(etcdClient),
			"--etcd-keyfile=" + cc.KeyPath(etcdClient),

			"--client-ca-file=" + cc.CertPath(clientCA),
			"--tls-cert-file=" + cc.CertPath(serving),
			"--tls-private-key-file=" + cc.KeyPath(serving),

			"--kubelet-client-certificate=" + cc.CertPath(kubeletClient),
			"--kubelet-client-key=" + cc.KeyPath(kubeletClient),
			"--kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalIP,ExternalDNS",

			"--authorization-mode=Node,RBAC",
			"--enable-bootstrap-token-auth=true",
			"--enable-admission-plugins=NodeRestriction",

			"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
			"--service-account-key-file=" + cc.CertPath(saSigner),
			"--service-account-signing-key-file=" + cc.KeyPath(saSigner),

			"--allow-privileged=true",

			"--egress-selector-config-file=" + konnectivityConfFilePath,

			"--logging-format=json",
		},
		Ports: []corev1.ContainerPort{
			{Name: "https", ContainerPort: cc.Contract.APIServer.SecurePort},
		},
		LivenessProbe:  utils.HttpsHealthProbe(cc.Contract.APIServer.SecurePort, cc.Keys.LivezPath, 10, 10, 10, 10),
		ReadinessProbe: utils.HttpsHealthProbe(cc.Contract.APIServer.SecurePort, cc.Keys.ReadyzPath, 5, 5, 5, 5),
	}

	vols := []corev1.Volume{
		cc.SecretVolume(clientCA),
		cc.SecretVolume(serving),
		cc.SecretVolume(etcdCA),
		cc.SecretVolume(etcdClient),
		cc.SecretVolume(kubeletClient),
		cc.SecretVolume(saSigner),
		cc.SecretVolume(fpCA),
		cc.SecretVolume(fpClient),
		cc.SecretVolume(konSrv),
		cc.SecretVolume(konCA),
	}

	mounts := []corev1.VolumeMount{
		cc.SecretMount(clientCA, true),
		cc.SecretMount(serving, true),
		cc.SecretMount(etcdCA, true),
		cc.SecretMount(etcdClient, true),
		cc.SecretMount(kubeletClient, true),
		cc.SecretMount(saSigner, true),
		cc.SecretMount(fpCA, true),
		cc.SecretMount(fpClient, true),
	}

	c2 := corev1.Container{
		Name:  konnectivityServerName,
		Image: "registry.k8s.io/kas-network-proxy/proxy-server:" + KonnectivityServerVersion,
		Args: []string{
			"--mode=grpc",
			"--uds-name=" + udsFile,
			"--delete-existing-uds-file=true",
			"--cluster-cert=" + cc.CertPath(konSrv),
			"--cluster-key=" + cc.KeyPath(konSrv),
			"--cluster-ca-cert=" + cc.CAPath(konCA),
			"--agent-port=" + utils.PortString(konnectivityServerPort),
			"--server-port=0",
			"--v=2",
		},
	}

	return builders.NewDeployment().
		WithName(cc.Names.APIServerDeploymentName()).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithContainer(c).
		AddVolumes(append([]corev1.Volume{
			konnectivityUDSVolume,
			konnectivityConfigVolume,
		}, vols...)...).
		AddVolumeMounts(c.Name,
			append([]corev1.VolumeMount{
				konnectivityConfigVolumeMount,
				konnectivityUDSVolumeMount,
			}, mounts...)...,
		).
		WithContainer(c2).
		AddVolumeMounts(c2.Name,
			konnectivityUDSVolumeMount,
			cc.SecretMount(konSrv, true),
			cc.SecretMount(konCA, true),
		).
		Build()
}
