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
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns Service + Deployment for the kube-apiserver.
func Resources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildService(mcp),
		buildDeployment(mcp),
	}
}

// EndpointResources returns only the Service (LB) for kube-apiserver.
func EndpointResources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{buildService(mcp)}
}

// WorkloadResources returns only the Deployment for kube-apiserver.
func WorkloadResources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{buildDeployment(mcp)}
}

func buildService(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.Service {
	ns := mcp.Namespace
	labels := map[string]string{appLabelKey: appLabelVal}

	return builders.NewService(ns, KubeAPIServerSvcName).
		WithLabels(labels).
		WithSelector(labels).
		AddPort("https", 6443, 6443, corev1.ProtocolTCP).
		WithType(corev1.ServiceTypeLoadBalancer).
		Build()
}

func buildDeployment(mcp *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	p := pki.New(mcp).APIServer()

	ns := mcp.Namespace
	labels := map[string]string{appLabelKey: appLabelVal}
	replicas := int32(1)

	version := mcp.Spec.Version

	c := corev1.Container{
		Name:            "apiserver",
		Image:           "registry.k8s.io/kube-apiserver:" + version,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"kube-apiserver"},
		Args: []string{
			"--advertise-address=" + mcp.Status.Address,
			"--bind-address=0.0.0.0",
			"--secure-port=6443",
			"--service-cluster-ip-range=" + mcp.Spec.Networking.ServiceCIDR,

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
			"--kubelet-certificate-authority=" + p.ClientCA.CertPath(),
			"--kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalIP,ExternalDNS",

			"--authorization-mode=Node,RBAC",
			"--enable-bootstrap-token-auth=true",

			// service account signing
			"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
			"--service-account-key-file=" + p.ServiceAccountSigner.CertPath(),
			"--service-account-signing-key-file=" + p.ServiceAccountSigner.KeyPath(),

			"--allow-privileged=true",

			// front-proxy
			"--proxy-client-cert-file=" + p.FrontProxyClient.CertPath(),
			"--proxy-client-key-file=" + p.FrontProxyClient.KeyPath(),
			"--requestheader-allowed-names=" + "front-proxy-client",
			"--requestheader-client-ca-file=" + p.FrontProxyCA.CertPath(),
			"--requestheader-extra-headers-prefix=X-Remote-Extra-",
			"--requestheader-group-headers=X-Remote-Group",
			"--requestheader-username-headers=X-Remote-User",

			"--logging-format=json",
		},
		Ports: []corev1.ContainerPort{
			{Name: "https", ContainerPort: securePort},
		},
		LivenessProbe:  utils.HttpsHealthProbe(securePort, common.LivezPath, 10, 10, 10, 10),
		ReadinessProbe: utils.HttpsHealthProbe(securePort, common.ReadyzPath, 5, 5, 5, 5),
	}

	return builders.NewDeployment(ns, apiServerName, labels, replicas).
		WithContainer(c).
		AddVolumes(
			p.ClientCA.Volume(),
			p.Serving.Volume(),
			p.EtcdClient.Volume(),
			p.KubeletClient.Volume(),
			p.ServiceAccountSigner.Volume(),
			p.EtcdCA.Volume(),
			p.FrontProxyCA.Volume(),
			p.FrontProxyClient.Volume(),
		).
		AddVolumeMounts(c.Name,
			p.ClientCA.Mount(true),
			p.Serving.Mount(true),
			p.EtcdClient.Mount(true),
			p.KubeletClient.Mount(true),
			p.ServiceAccountSigner.Mount(true),
			p.EtcdCA.Mount(true),
			p.FrontProxyCA.Mount(true),
			p.FrontProxyClient.Mount(true),
		).
		Build()
}
