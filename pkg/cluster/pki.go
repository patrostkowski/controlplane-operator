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

// PKISpec defines the inputs required to generate and wire PKI resources for a control plane.
type PKISpec interface {
	Namespacer

	GetManagedControlPlaneSpec() mcpv1alpha1.ManagedControlPlaneSpec
	GetManagedControlPlaneStatus() mcpv1alpha1.ManagedControlPlaneStatus

	PKI() PKIConfig
	Etcd() EtcdConfig
	APIServer() APIServerConfig
}

// PKIConfig exposes accessors for PKI issuer and certificate naming.
type PKIConfig interface {
	Issuer() IssuerConfig
	Certificate() CertificateConfig
}

// IssuerConfig defines stable names for cert-manager Issuers used by the control plane PKI.
type IssuerConfig interface {
	SelfSigned() string
	CA() string

	EtcdSelfSigned() string
	EtcdCA() string

	FrontProxySelfSigned() string
	FrontProxyCA() string

	KonnectivitySelfSigned() string
	KonnectivityCA() string
}

// CertificateConfig defines stable Secret names for all PKI certificates and keys.
type CertificateConfig interface {
	ManagedCA() string
	EtcdCA() string
	FrontProxyCA() string
	KonnectivityCA() string

	SASigner() string

	APIServerTLS() string
	APIServerKubeletClient() string
	APIServerEtcdClient() string

	EtcdServerTLS() string
	EtcdPeerTLS() string
	EtcdHealthClient() string

	FrontProxyClient() string

	CMClient() string
	SchedulerClient() string
	AdminClient() string

	KonnectivityServerTLS() string
	KonnectivityAgentTLS() string
}

// Compile-time check ensuring pki implements PKIConfig.
var _ PKIConfig = pki{}

// Compile-time check ensuring issuers implements IssuerConfig.
var _ IssuerConfig = issuers{}

// Compile-time check ensuring certificates implements CertificateConfig.
var _ CertificateConfig = certificates{}

// PKI returns a names-only PKI facade for issuer and certificate resources.
func (cc ClusterContext) PKI() PKIConfig {
	return pki{cc: cc}
}

// pki is an internal PKIConfig implementation backed by ClusterContext.
type pki struct {
	cc ClusterContext
}

// Issuer returns the issuer name configuration.
func (p pki) Issuer() IssuerConfig {
	return issuers(p)
}

// Certificate returns the certificate secret name configuration.
func (p pki) Certificate() CertificateConfig {
	return certificates(p)
}

// certificates is a names-only implementation of CertificateConfig.
type certificates struct {
	cc ClusterContext
}

// issuers is a names-only implementation of IssuerConfig.
type issuers struct {
	cc ClusterContext
}

// SelfSigned returns the name of the root self-signed issuer.
func (i issuers) SelfSigned() string {
	return i.cc.prefix(issuerSelfSigned)
}

// CA returns the name of the root CA issuer.
func (i issuers) CA() string {
	return i.cc.prefix(issuerCA)
}

// EtcdSelfSigned returns the name of the etcd self-signed issuer.
func (i issuers) EtcdSelfSigned() string {
	return i.cc.prefix(issuerEtcdSelfSigned)
}

// EtcdCA returns the name of the etcd CA issuer.
func (i issuers) EtcdCA() string {
	return i.cc.prefix(issuerEtcdCA)
}

// FrontProxySelfSigned returns the name of the front-proxy self-signed issuer.
func (i issuers) FrontProxySelfSigned() string {
	return i.cc.prefix(issuerFrontProxySelf)
}

// FrontProxyCA returns the name of the front-proxy CA issuer.
func (i issuers) FrontProxyCA() string {
	return i.cc.prefix(issuerFrontProxyCA)
}

// KonnectivitySelfSigned returns the name of the konnectivity self-signed issuer.
func (i issuers) KonnectivitySelfSigned() string {
	return i.cc.prefix(issuerKonnectivitySelf)
}

// KonnectivityCA returns the name of the konnectivity CA issuer.
func (i issuers) KonnectivityCA() string {
	return i.cc.prefix(issuerKonnectivityCA)
}

// ManagedCA returns the Secret name holding the cluster root CA.
func (c certificates) ManagedCA() string {
	return c.cc.prefix(secretManagedCA)
}

// EtcdCA returns the Secret name holding the etcd CA.
func (c certificates) EtcdCA() string {
	return c.cc.prefix(secretEtcdCA)
}

// FrontProxyCA returns the Secret name holding the front-proxy CA.
func (c certificates) FrontProxyCA() string {
	return c.cc.prefix(secretFrontProxyCA)
}

// SASigner returns the Secret name holding the service-account signing key.
func (c certificates) SASigner() string {
	return c.cc.prefix(secretSASigner)
}

// APIServerTLS returns the Secret name holding the API server serving certificate.
func (c certificates) APIServerTLS() string {
	return c.cc.prefix(secretAPIServerTLS)
}

// APIServerKubeletClient returns the Secret name holding the API server kubelet client certificate.
func (c certificates) APIServerKubeletClient() string {
	return c.cc.prefix(secretAPIServerKubelet)
}

// EtcdServerTLS returns the Secret name holding the etcd server certificate.
func (c certificates) EtcdServerTLS() string {
	return c.cc.prefix(secretEtcdServerTLS)
}

// EtcdPeerTLS returns the Secret name holding the etcd peer certificate.
func (c certificates) EtcdPeerTLS() string {
	return c.cc.prefix(secretEtcdPeerTLS)
}

// EtcdHealthClient returns the Secret name holding the etcd health check client certificate.
func (c certificates) EtcdHealthClient() string {
	return c.cc.prefix(secretEtcdHealthClient)
}

// APIServerEtcdClient returns the Secret name holding the API server etcd client certificate.
func (c certificates) APIServerEtcdClient() string {
	return c.cc.prefix(secretAPIServerEtcd)
}

// FrontProxyClient returns the Secret name holding the front-proxy client certificate.
func (c certificates) FrontProxyClient() string {
	return c.cc.prefix(secretFrontProxyClient)
}

// CMClient returns the Secret name holding the controller-manager client certificate.
func (c certificates) CMClient() string {
	return c.cc.prefix(secretCMClient)
}

// SchedulerClient returns the Secret name holding the scheduler client certificate.
func (c certificates) SchedulerClient() string {
	return c.cc.prefix(secretSchedulerClient)
}

// AdminClient returns the Secret name holding the admin client certificate.
func (c certificates) AdminClient() string {
	return c.cc.prefix(secretAdminClient)
}

// KonnectivityCA returns the Secret name holding the konnectivity CA.
func (c certificates) KonnectivityCA() string {
	return c.cc.prefix(secretKonnectivityCA)
}

// KonnectivityServerTLS returns the Secret name holding the konnectivity server certificate.
func (c certificates) KonnectivityServerTLS() string {
	return c.cc.prefix(secretKonnectivityTLS)
}

// KonnectivityAgentTLS returns the Secret name holding the konnectivity agent certificate.
func (c certificates) KonnectivityAgentTLS() string {
	return c.cc.prefix(secretKonnectivityAgentTLS)
}
