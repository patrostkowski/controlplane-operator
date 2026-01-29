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

// AddonSpec defines the interface for addon-related configurations and secrets.
type AddonSpec interface {
	Namespacer

	GetManagedControlPlaneSpec() mcpv1alpha1.ManagedControlPlaneSpec
	GetManagedControlPlaneStatus() mcpv1alpha1.ManagedControlPlaneStatus

	Konnectivity() KonnectivityConfig

	Admin() AdminConfig
	ManagedAddons() ManagedAddonsConfig
}

// AdminConfig defines methods to retrieve administration-related secret names.
type AdminConfig interface {
	ClientSecret() string
	KubeconfigSecret() string
	KubeconfigDataKey() string
}

// ManagedAddonsConfig defines methods to retrieve managed addon configurations.
type ManagedAddonsConfig interface {
	KonnectivityAgentNamespace() string
}

// KonnectivityConfig defines methods to retrieve Konnectivity-related secret names and configurations.
type KonnectivityConfig interface {
	// Names (if you have them; include only if used)
	AgentName() string

	// Secret names used by multiple layers
	CASecret() string
	AgentTLSSecret() string
	ServerTLSSecret() string
}

var _ AdminConfig = admin{}
var _ ManagedAddonsConfig = managedAddons{}

func (cc ClusterContext) Admin() AdminConfig {
	return admin{cc: cc}
}

// admin is an internal struct that implements the AdminConfig interface.
type admin struct{ cc ClusterContext }

// Secret holding the admin client cert/key/ca (created by PKI)
func (a admin) ClientSecret() string {
	return a.cc.PKI().Certificate().AdminClient()
}

// Secret where you store generated kubeconfig
func (a admin) KubeconfigSecret() string {
	return "admin-kubeconfig" // keep backward-compatible with your old Names.AdminKubeconfigSecretName()
}

// Keys inside kubeconfig secret (choose ONE and use it consistently)
func (a admin) KubeconfigDataKey() string {
	return "kubeconfig" // keep backward-compatible with your old cc.Keys.AdminKubeconfigKey
}

func (cc ClusterContext) ManagedAddons() ManagedAddonsConfig {
	return managedAddons{cc: cc}
}

// managedAddons is an internal struct that implements the ManagedAddonsConfig interface.
type managedAddons struct{ cc ClusterContext }

func (m managedAddons) KonnectivityAgentNamespace() string {
	return "kube-system" // or whatever your old Names.KonnectivityAgentNamespace() returned
}

var _ KonnectivityConfig = konnectivity{}

// Konnectivity returns the KonnectivityConfig interface for managing Konnectivity-related configurations.
func (cc ClusterContext) Konnectivity() KonnectivityConfig {
	return konnectivity{cc: cc}
}

// konnectivity is an internal struct that implements the KonnectivityConfig interface.
type konnectivity struct {
	cc ClusterContext
}

// Secrets (names only, via PKI contract)
func (k konnectivity) CASecret() string {
	return k.cc.PKI().Certificate().KonnectivityCA()
}

func (k konnectivity) ServerTLSSecret() string {
	return k.cc.PKI().Certificate().KonnectivityServerTLS()
}

// AgentTLSSecret returns the name of the secret containing Konnectivity agent TLS certificates.
func (k konnectivity) AgentTLSSecret() string {
	return k.cc.PKI().Certificate().KonnectivityAgentTLS()
}

// Optional “name” (still just a name; ok to keep here if you want)
func (k konnectivity) AgentName() string {
	return "konnectivity-agent"
}
