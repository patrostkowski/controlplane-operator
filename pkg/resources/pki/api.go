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

import (
	"path/filepath"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	corev1 "k8s.io/api/core/v1"
)

// Bundle is the only exported way to access PKI.
type Bundle struct {
	rootDir  string
	certFile string
	keyFile  string
	caFile   string

	// CAs
	clusterCA      common.SecretMount
	etcdCA         common.SecretMount
	frontProxyCA   common.SecretMount
	konnectivityCA common.SecretMount

	// ETCD serving/peer
	etcdServer common.SecretMount
	etcdPeer   common.SecretMount

	// Serving
	apiServerServing          common.SecretMount
	konnectivityServerServing common.SecretMount
	konnectivityAgent         common.SecretMount

	// Clients
	apiServerEtcdClient     common.SecretMount
	apiServerKubeletClient  common.SecretMount
	controllerManagerClient common.SecretMount
	schedulerClient         common.SecretMount
	frontProxyClient        common.SecretMount
	adminClient             common.SecretMount

	// Special
	serviceAccountSigner common.SecretMount
}

// New builds the PKI bundle (one place for mount root + filenames).
func New(_ *mcpv1alpha1.ManagedControlPlane) Bundle {
	root := common.PKIMountRoot
	m := func(secret string) common.SecretMount {
		return common.SecretMount{
			SecretName: secret,
			MountDir:   filepath.Join(root, secret),
			CertFile:   common.TLSCrtKey,
			KeyFile:    common.TLSKeyKey,
			CAFile:     common.CACrtKey,
		}
	}

	return Bundle{
		rootDir:  root,
		certFile: common.TLSCrtKey,
		keyFile:  common.TLSKeyKey,
		caFile:   common.CACrtKey,

		clusterCA:      m(secretManagedCA),
		etcdCA:         m(secretEtcdCA),
		frontProxyCA:   m(secretFrontProxyCA),
		konnectivityCA: m(secretKonnectivityCA),

		etcdServer: m(secretEtcdServerTLS),
		etcdPeer:   m(secretEtcdPeerTLS),

		apiServerServing:          m(secretAPIServerTLS),
		konnectivityServerServing: m(secretKonnectivityTLS),
		konnectivityAgent:         m(secretKonnectivityAgentTLS),

		apiServerEtcdClient:    m(secretAPIServerEtcd),
		apiServerKubeletClient: m(secretAPIServerKubelet),

		controllerManagerClient: m(secretCMClient),
		schedulerClient:         m(secretSchedulerClient),

		frontProxyClient: m(secretFrontProxyClient),
		adminClient:      m(secretAdminClient),

		serviceAccountSigner: m(secretSASigner),
	}
}

// APIServerView is what apiserver needs. It’s explicit, self-documenting and hard to misuse.
type APIServerView struct {
	ClientCA       common.SecretMount
	KonnectivityCA common.SecretMount

	Serving             common.SecretMount
	KonnectivityServing common.SecretMount

	EtcdCA     common.SecretMount
	EtcdClient common.SecretMount

	KubeletClient common.SecretMount

	FrontProxyCA     common.SecretMount
	FrontProxyClient common.SecretMount

	ServiceAccountSigner common.SecretMount
}

func (b Bundle) APIServer() APIServerView {
	return APIServerView{
		ClientCA:       b.clusterCA,
		KonnectivityCA: b.konnectivityCA,

		Serving:             b.apiServerServing,
		KonnectivityServing: b.konnectivityServerServing,

		EtcdCA:     b.etcdCA,
		EtcdClient: b.apiServerEtcdClient,

		KubeletClient: b.apiServerKubeletClient,

		FrontProxyCA:     b.frontProxyCA,
		FrontProxyClient: b.frontProxyClient,

		ServiceAccountSigner: b.serviceAccountSigner,
	}
}

// SchedulerView is what kube-scheduler needs to build kubeconfig + mount PKI.
type SchedulerView struct {
	ClientCA common.SecretMount // cluster CA
	Client   common.SecretMount // scheduler client cert
}

// Scheduler returns the scheduler PKI view.
func (b Bundle) Scheduler() SchedulerView {
	return SchedulerView{
		ClientCA: b.clusterCA,
		Client:   b.schedulerClient,
	}
}

func (p SchedulerView) Volumes() []corev1.Volume {
	return []corev1.Volume{
		p.Client.Volume(),
		p.ClientCA.Volume(),
	}
}

func (p SchedulerView) Mounts(readOnly bool) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		p.Client.Mount(readOnly),
		p.ClientCA.Mount(readOnly),
	}
}

// ControllerManagerView is what kube-controller-manager needs for PKI mounts/paths.
type ControllerManagerView struct {
	ClientCA             common.SecretMount
	ServiceAccountSigner common.SecretMount
	Client               common.SecretMount
}

func (b Bundle) ControllerManager() ControllerManagerView {
	return ControllerManagerView{
		ClientCA:             b.clusterCA,
		ServiceAccountSigner: b.serviceAccountSigner,
		Client:               b.controllerManagerClient,
	}
}

func (p ControllerManagerView) Volumes() []corev1.Volume {
	return []corev1.Volume{
		p.Client.Volume(),
		p.ServiceAccountSigner.Volume(),
		p.ClientCA.Volume(),
	}
}

func (p ControllerManagerView) Mounts(readOnly bool) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		p.Client.Mount(readOnly),
		p.ServiceAccountSigner.Mount(readOnly),
		p.ClientCA.Mount(readOnly),
	}
}

type EtcdView struct {
	CA     common.SecretMount
	Server common.SecretMount
	Peer   common.SecretMount
}

const (
	etcdMountRoot = "/etc/etcd/pki"
	etcdDirCA     = "ca"
	etcdDirServer = "server"
	etcdDirPeer   = "peer"

	etcdCAFile   = "ca.crt"
	etcdCertFile = "tls.crt"
	etcdKeyFile  = "tls.key"
)

func (b Bundle) ETCD() EtcdView {
	m := func(secret, dir string) common.SecretMount {
		return common.SecretMount{
			SecretName: secret,
			MountDir:   filepath.Join(etcdMountRoot, dir),
			CertFile:   etcdCertFile,
			KeyFile:    etcdKeyFile,
			CAFile:     etcdCAFile,
		}
	}

	return EtcdView{
		CA:     m(b.etcdCA.SecretName, etcdDirCA),
		Server: m(b.etcdServer.SecretName, etcdDirServer),
		Peer:   m(b.etcdPeer.SecretName, etcdDirPeer),
	}
}

func (p EtcdView) Volumes() []corev1.Volume {
	return []corev1.Volume{
		p.CA.Volume(),
		p.Server.Volume(),
		p.Peer.Volume(),
	}
}

func (p EtcdView) Mounts(readOnly bool) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		p.CA.Mount(readOnly),
		p.Server.Mount(readOnly),
		p.Peer.Mount(readOnly),
	}
}

// AdminView is what reconcileAdminConfig needs.
type AdminView struct {
	Client common.SecretMount // admin client cert secret (contains tls.crt/tls.key/ca.crt)
}

func (b Bundle) Admin() AdminView {
	return AdminView{
		Client: b.adminClient,
	}
}

const (
	konnectivityAgentMountRoot = "/certs"
	konnectivityAgentCAFile    = "ca.crt"
	konnectivityAgentCertFile  = "tls.crt"
	konnectivityAgentKeyFile   = "tls.key"
)

type KonnectivityAgentView struct {
	KonnectivityCA    common.SecretMount
	KonnectivityAgent common.SecretMount
}

func (b Bundle) KonnectivityAgentView() KonnectivityAgentView {
	m := func(secret, dir string) common.SecretMount {
		return common.SecretMount{
			SecretName: secret,
			MountDir:   dir,
			CertFile:   konnectivityAgentCertFile,
			KeyFile:    konnectivityAgentKeyFile,
			CAFile:     konnectivityAgentCAFile,
		}
	}

	return KonnectivityAgentView{
		KonnectivityCA:    m(b.konnectivityCA.SecretName, konnectivityAgentMountRoot),
		KonnectivityAgent: m(b.konnectivityAgent.SecretName, konnectivityAgentMountRoot),
	}
}

func (k KonnectivityAgentView) Volumes() []corev1.Volume {
	return []corev1.Volume{
		k.KonnectivityCA.Volume(),
		k.KonnectivityAgent.Volume(),
	}
}

func (k KonnectivityAgentView) Mounts(readOnly bool) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		k.KonnectivityCA.Mount(readOnly),
		k.KonnectivityAgent.Mount(readOnly),
	}
}
