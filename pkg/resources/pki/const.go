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

package pki

const (
	kindIssuer = "Issuer"

	// Issuers
	issuerSelfSigned     = "selfsigned"
	issuerCA             = "ca-issuer"
	issuerEtcdSelfSigned = "etcd-selfsigned"
	issuerEtcdCA         = "etcd-ca-issuer"
	issuerFrontProxySelf = "front-proxy-selfsigned"
	issuerFrontProxyCA   = "front-proxy-ca-issuer"

	// Secrets / cert names
	secretManagedCA        = "managed-ca"
	secretEtcdCA           = "etcd-ca"
	secretFrontProxyCA     = "front-proxy-ca"
	secretSASigner         = "sa-signer"
	secretAPIServerTLS     = "apiserver-tls"
	secretAPIServerKubelet = "apiserver-kubelet-client"
	secretEtcdServerTLS    = "etcd-server-tls"
	secretEtcdPeerTLS      = "etcd-peer-tls"
	secretEtcdHealthClient = "etcd-healthcheck-client"
	secretAPIServerEtcd    = "apiserver-etcd-client"
	secretFrontProxyClient = "front-proxy-client"
	secretCMClient         = "cm-client"
	secretSchedulerClient  = "scheduler-client"
	secretAdminClient      = "admin-client"

	// CommonNames
	cnManagedCA        = "managed-ca"
	cnEtcdCA           = "etcd-ca"
	cnFrontProxyCA     = "kubernetes-front-proxy-ca"
	cnSASigner         = "sa-signer"
	cnAPIServer        = "kube-apiserver"
	cnAPIServerKubelet = "kube-apiserver-kubelet-client"
	cnEtcdHealth       = "kube-etcd-healthcheck-client"
	cnAPIServerEtcd    = "kube-apiserver-etcd-client"
	cnFrontProxyClient = "front-proxy-client"
	cnCMClient         = "system:kube-controller-manager"
	cnSchedulerClient  = "system:kube-scheduler"
	cnAdminClient      = "kubernetes-admin"

	orgSystemMasters = "system:masters"
)
