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

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
)

// APIServerSpec defines the interface for API server-related configurations and secrets.
type APIServerSpec interface {
	Namespacer
	MountLayout

	GetManagedControlPlaneSpec() mcpv1alpha1.ManagedControlPlaneSpec
	GetManagedControlPlaneStatus() mcpv1alpha1.ManagedControlPlaneStatus

	APIServer() APIServerConfig
	Etcd() EtcdConfig
}

// APIServerConfig defines methods to retrieve API server-related names and secret names.
type APIServerConfig interface {
	ServiceName() string
	DeploymentName() string
	KonnectivityConfigMapName() string

	ClientCASecret() string
	ServingTLSSecret() string
	EtcdCASecret() string
	EtcdClientSecret() string
	KubeletClientSecret() string
	SASignerSecret() string
	FrontProxyCASecret() string
	FrontProxyClientSecret() string
	KonnectivityCASecret() string
	KonnectivityServerSecret() string
}

var _ APIServerConfig = apiserver{}

// APIServer returns the APIServerConfig interface for managing API server-related configurations.
func (cc ClusterContext) APIServer() APIServerConfig {
	return apiserver{cc: cc}
}

// apiserver is an internal struct that implements the APIServerConfig interface.
type apiserver struct {
	cc ClusterContext
}

// ServiceName returns the name of the API server service.
func (a apiserver) ServiceName() string {
	return a.cc.prefix("apiserver")
}

// DeploymentName returns the name of the API server deployment.
func (a apiserver) DeploymentName() string {
	return a.cc.prefix("apiserver")
}

// KonnectivityConfigMapName returns the name of the Konnectivity config map.
func (a apiserver) KonnectivityConfigMapName() string {
	return a.cc.prefix("konnectivity-egress-selector")
}

// ClientCASecret returns the name of the secret containing the client CA certificate.
func (a apiserver) ClientCASecret() string { return a.cc.PKI().Certificate().ManagedCA() }

// ServingTLSSecret returns the name of the secret containing the API server serving TLS certificates.
func (a apiserver) ServingTLSSecret() string { return a.cc.PKI().Certificate().APIServerTLS() }

// EtcdCASecret returns the name of the secret containing the etcd CA certificate.
func (a apiserver) EtcdCASecret() string { return a.cc.PKI().Certificate().EtcdCA() }

// EtcdClientSecret returns the name of the secret containing the etcd client certificate for the API server.
func (a apiserver) EtcdClientSecret() string { return a.cc.PKI().Certificate().APIServerEtcdClient() }

// KubeletClientSecret returns the name of the secret containing the kubelet client certificate for the API server.
func (a apiserver) KubeletClientSecret() string {
	return a.cc.PKI().Certificate().APIServerKubeletClient()
}

// SASignerSecret returns the name of the secret containing the service account signer certificate.
func (a apiserver) SASignerSecret() string { return a.cc.PKI().Certificate().SASigner() }

// FrontProxyCASecret returns the name of the secret containing the front proxy CA certificate.
func (a apiserver) FrontProxyCASecret() string { return a.cc.PKI().Certificate().FrontProxyCA() }

// FrontProxyClientSecret returns the name of the secret containing the front proxy client certificate.
func (a apiserver) FrontProxyClientSecret() string {
	return a.cc.PKI().Certificate().FrontProxyClient()
}

// KonnectivityCASecret returns the name of the secret containing the Konnectivity CA certificate.
func (a apiserver) KonnectivityCASecret() string { return a.cc.PKI().Certificate().KonnectivityCA() }

// KonnectivityServerSecret returns the name of the secret containing the Konnectivity server certificate.
func (a apiserver) KonnectivityServerSecret() string {
	return a.cc.PKI().Certificate().KonnectivityServerTLS()
}
