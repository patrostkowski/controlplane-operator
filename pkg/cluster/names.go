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

package cluster

func (n Names) prefix(s string) string {
	// for now lets keep it backward compatible
	// return n.Name + "-" + s
	return s
}

// Admin kubeconfig
func (n Names) AdminKubeconfigSecretName() string {
	return n.prefix("admin-kubeconfig")
}

// API Server
func (n Names) APIServerServiceName() string {
	return n.prefix("kube-apiserver")
}

func (n Names) APIServerDeploymentName() string {
	return n.prefix("kube-apiserver")
}

func (n Names) KonnectivityConfigMapName() string {
	return n.prefix("konnectivity-egress-selector")
}

// ETCD
func (n Names) EtcdServiceName() string {
	return n.prefix("etcd")
}

func (n Names) EtcdStatefulSetName() string {
	return n.prefix("etcd")
}

// Controller Manager
func (n Names) ControllerManagerDeploymentName() string {
	return n.prefix("kube-controller-manager")
}

func (n Names) ControllerManagerKubeconfigConfigMapName() string {
	return n.prefix("controller-kubeconfig")
}

// Scheduler
func (n Names) SchedulerDeploymentName() string {
	return n.prefix("kube-scheduler")
}

func (n Names) SchedulerKubeconfigConfigMapName() string {
	return n.prefix("scheduler-kubeconfig")
}

// Bootstrap token management (management cluster secret)
func (n Names) BootstrapTokenMgmtSecretName() string {
	return n.prefix("bootstrap-token")
}

// Issuers
func (n Names) IssuerSelfSignedName() string {
	return n.prefix("issuer-selfsigned")
}

func (n Names) IssuerCAName() string {
	return n.prefix("issuer-ca")
}

func (n Names) IssuerEtcdSelfSignedName() string {
	return n.prefix("issuer-etcd-selfsigned")
}

func (n Names) IssuerEtcdCAName() string {
	return n.prefix("issuer-etcd-ca")
}

func (n Names) IssuerFrontProxySelfSignedName() string {
	return n.prefix("issuer-front-proxy-selfsigned")
}

func (n Names) IssuerFrontProxyCAName() string {
	return n.prefix("issuer-front-proxy-ca")
}

func (n Names) IssuerKonnectivitySelfSignedName() string {
	return n.prefix("issuer-konnectivity-selfsigned")
}

func (n Names) IssuerKonnectivityCAName() string {
	return n.prefix("issuer-konnectivity-ca")
}

// CA secrets
func (n Names) SecretManagedCAName() string {
	return n.prefix("managed-ca")
}

func (n Names) SecretEtcdCAName() string {
	return n.prefix("etcd-ca")
}

func (n Names) SecretFrontProxyCAName() string {
	return n.prefix("front-proxy-ca")
}

func (n Names) SecretServiceAccountSignerName() string {
	return n.prefix("sa-signer")
}

// kube-apiserver certs
func (n Names) SecretAPIServerServingTLSName() string {
	return n.prefix("apiserver-tls")
}

func (n Names) SecretAPIServerKubeletClientName() string {
	return n.prefix("apiserver-kubelet-client")
}

func (n Names) SecretAPIServerEtcdClientName() string {
	return n.prefix("apiserver-etcd-client")
}

// etcd certs
func (n Names) SecretEtcdServerTLSName() string {
	return n.prefix("etcd-server-tls")
}

func (n Names) SecretEtcdPeerTLSName() string {
	return n.prefix("etcd-peer-tls")
}

func (n Names) SecretEtcdHealthcheckClientName() string {
	return n.prefix("etcd-healthcheck-client")
}

// front-proxy
func (n Names) SecretFrontProxyClientName() string {
	return n.prefix("front-proxy-client")
}

// component clients
func (n Names) SecretControllerManagerClientName() string {
	return n.prefix("cm-client") // todo change
}

func (n Names) SecretSchedulerClientName() string {
	return n.prefix("scheduler-client")
}

func (n Names) SecretAdminClientName() string {
	return n.prefix("admin-client")
}

// konnectivity
func (n Names) SecretKonnectivityCAName() string {
	return n.prefix("konnectivity-ca")
}

func (n Names) SecretKonnectivityServerTLSName() string {
	return n.prefix("konnectivity-server-tls")
}

func (n Names) SecretKonnectivityAgentTLSName() string {
	return n.prefix("konnectivity-agent-tls")
}

func (n Names) CertificateEtcdPeerName() string {
	return n.prefix("etcd-peer")
}

// Konnectivity agent
func (n Names) KonnectivityAgentNamespace() string {
	return "kube-system"
}

func (n Names) KonnectivityAgentDaemonSetName() string {
	return "konnectivity-agent"
}

// CoreDNS
func (n Names) CoreDNSNamespaceName() string {
	return "kube-system"
}

func (n Names) CoreDNSServiceAccountName() string {
	return "coredns"
}

func (n Names) CoreDNSConfigMapName() string {
	return "coredns"
}

func (n Names) CoreDNSDeploymentName() string {
	return "coredns"
}

func (n Names) CoreDNSServiceName() string {
	return "kube-dns"
}

func (n Names) CoreDNSClusterRoleName() string {
	return "system:coredns"
}

func (n Names) CoreDNSClusterRoleBindingName() string {
	return "system:coredns"
}

func (n Names) CoreDNSRoleName() string {
	return "coredns"
}

func (n Names) CoreDNSRoleBindingName() string {
	return "coredns"
}

// Flannel
func (n Names) FlannelNamespaceName() string {
	return "kube-flannel"
}

func (n Names) FlannelServiceAccountName() string {
	return "flannel"
}

func (n Names) FlannelConfigMapName() string {
	return "kube-flannel-cfg"
}

func (n Names) FlannelDaemonSetName() string {
	return "kube-flannel-ds"
}

func (n Names) FlannelClusterRoleName() string {
	return "flannel"
}

func (n Names) FlannelClusterRoleBindingName() string {
	return "flannel"
}

// CSI local-path-provisioner
func (n Names) CSINamespaceName() string {
	return "local-path-storage"
}

func (n Names) CSIServiceAccountName() string {
	return "local-path-provisioner-service-account"
}

func (n Names) CSIRoleName() string {
	return "local-path-provisioner-role"
}

func (n Names) CSIClusterRoleName() string {
	return "local-path-provisioner-role"
}

func (n Names) CSIRoleBindingName() string {
	return "local-path-provisioner-bind"
}

func (n Names) CSIClusterRoleBindingName() string {
	return "local-path-provisioner-bind"
}

func (n Names) CSIDeploymentName() string {
	return "local-path-provisioner"
}

func (n Names) CSIConfigMapName() string {
	return "local-path-config"
}

func (n Names) CSIStorageClassName() string {
	return "local-path"
}

func DefaultKeys() Keys {
	return Keys{
		TLSCrt:                   "tls.crt",
		TLSKey:                   "tls.key",
		CACrt:                    "ca.crt",
		LivezPath:                "/livez",
		ReadyzPath:               "/readyz",
		HealthPath:               "/healthz",
		AdminKubeconfigKey:       "config",
		AdminConfigKubeconfigKey: "config",
		LabelKeyApp:              "app",
	}
}

func DefaultLayout() Layout {
	return Layout{PKIRoot: "/var/run/k8s"}
}

func DefaultVolumes() Volumes {
	return Volumes{Kubeconfig: "kubeconfig"}
}

func DefaultWellKnown() WellKnown {
	return WellKnown{
		KonnectivityAgentNamespace: "kube-system",
		CoreAPIGroup:               "",
		DiscoveryAPIGroup:          "discovery.k8s.io",
	}
}

func DefaultContract() Contract {
	var c Contract
	c.APIServer.ServiceNameLegacy = "kube-apiserver"
	c.APIServer.SecurePort = 6443
	c.APIServer.AppLabelKey = "app"
	c.APIServer.AppLabelVal = "kube-apiserver"

	c.Konnectivity.ServerName = "konnectivity-server"
	c.Konnectivity.AgentName = "konnectivity-agent"
	c.Konnectivity.ServerPort = 8132
	c.Konnectivity.EgressSelectorKind = "EgressSelectorConfiguration"
	c.Konnectivity.EgressSelectorAPIVersion = "apiserver.k8s.io/v1beta1"
	c.Konnectivity.ConfigMapName = "konnectivity-egress-selector"
	c.Konnectivity.ConfigVolumeName = "egress-selector"
	c.Konnectivity.ConfigMapKey = "konnectivity-egress-selector.yaml"
	c.Konnectivity.ServerUDSVolume = "konnectivity-uds"
	c.Konnectivity.UDSFileName = "konnectivity-uds.sock"
	return c
}
