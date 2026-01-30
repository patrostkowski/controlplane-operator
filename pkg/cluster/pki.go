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
	return issuers{}
}

// Certificate returns the certificate secret name configuration.
func (p pki) Certificate() CertificateConfig {
	return certificates{}
}

// certificates is a names-only implementation of CertificateConfig.
type certificates struct{}

// issuers is a names-only implementation of IssuerConfig.
type issuers struct{}

// SelfSigned returns the name of the root self-signed issuer.
func (issuers) SelfSigned() string {
	return issuerSelfSigned
}

// CA returns the name of the root CA issuer.
func (issuers) CA() string {
	return issuerCA
}

// EtcdSelfSigned returns the name of the etcd self-signed issuer.
func (issuers) EtcdSelfSigned() string {
	return issuerEtcdSelfSigned
}

// EtcdCA returns the name of the etcd CA issuer.
func (issuers) EtcdCA() string {
	return issuerEtcdCA
}

// FrontProxySelfSigned returns the name of the front-proxy self-signed issuer.
func (issuers) FrontProxySelfSigned() string {
	return issuerFrontProxySelf
}

// FrontProxyCA returns the name of the front-proxy CA issuer.
func (issuers) FrontProxyCA() string {
	return issuerFrontProxyCA
}

// KonnectivitySelfSigned returns the name of the konnectivity self-signed issuer.
func (issuers) KonnectivitySelfSigned() string {
	return issuerKonnectivitySelf
}

// KonnectivityCA returns the name of the konnectivity CA issuer.
func (issuers) KonnectivityCA() string {
	return issuerKonnectivityCA
}

// ManagedCA returns the Secret name holding the cluster root CA.
func (certificates) ManagedCA() string {
	return secretManagedCA
}

// EtcdCA returns the Secret name holding the etcd CA.
func (certificates) EtcdCA() string {
	return secretEtcdCA
}

// FrontProxyCA returns the Secret name holding the front-proxy CA.
func (certificates) FrontProxyCA() string {
	return secretFrontProxyCA
}

// SASigner returns the Secret name holding the service-account signing key.
func (certificates) SASigner() string {
	return secretSASigner
}

// APIServerTLS returns the Secret name holding the API server serving certificate.
func (certificates) APIServerTLS() string {
	return secretAPIServerTLS
}

// APIServerKubeletClient returns the Secret name holding the API server kubelet client certificate.
func (certificates) APIServerKubeletClient() string {
	return secretAPIServerKubelet
}

// EtcdServerTLS returns the Secret name holding the etcd server certificate.
func (certificates) EtcdServerTLS() string {
	return secretEtcdServerTLS
}

// EtcdPeerTLS returns the Secret name holding the etcd peer certificate.
func (certificates) EtcdPeerTLS() string {
	return secretEtcdPeerTLS
}

// EtcdHealthClient returns the Secret name holding the etcd health check client certificate.
func (certificates) EtcdHealthClient() string {
	return secretEtcdHealthClient
}

// APIServerEtcdClient returns the Secret name holding the API server etcd client certificate.
func (certificates) APIServerEtcdClient() string {
	return secretAPIServerEtcd
}

// FrontProxyClient returns the Secret name holding the front-proxy client certificate.
func (certificates) FrontProxyClient() string {
	return secretFrontProxyClient
}

// CMClient returns the Secret name holding the controller-manager client certificate.
func (certificates) CMClient() string {
	return secretCMClient
}

// SchedulerClient returns the Secret name holding the scheduler client certificate.
func (certificates) SchedulerClient() string {
	return secretSchedulerClient
}

// AdminClient returns the Secret name holding the admin client certificate.
func (certificates) AdminClient() string {
	return secretAdminClient
}

// KonnectivityCA returns the Secret name holding the konnectivity CA.
func (certificates) KonnectivityCA() string {
	return secretKonnectivityCA
}

// KonnectivityServerTLS returns the Secret name holding the konnectivity server certificate.
func (certificates) KonnectivityServerTLS() string {
	return secretKonnectivityTLS
}

// KonnectivityAgentTLS returns the Secret name holding the konnectivity agent certificate.
func (certificates) KonnectivityAgentTLS() string {
	return secretKonnectivityAgentTLS
}
