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
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectProducer interface {
	Objects() []client.Object
}

// ClusterContext is instantiated ONCE per reconcile and passed everywhere.
type ClusterContext struct {
	MCP *mcpv1alpha1.ManagedControlPlane
	logr.Logger
	Names  Names
	Labels map[string]string
	Owner  metav1.OwnerReference

	Keys      Keys
	Layout    Layout
	Volumes   Volumes
	WellKnown WellKnown
	Contract  Contract
}

// Names is the single naming contract used by controllers + resource builders.
type Names struct {
	Namespace string
	Name      string
}

func NewClusterContext(mcp *mcpv1alpha1.ManagedControlPlane, log logr.Logger) *ClusterContext {
	n := NewNames(mcp)
	cc := &ClusterContext{
		MCP:       mcp,
		Logger:    log.WithName("ClusterContext"),
		Names:     n,
		Owner:     *metav1.NewControllerRef(mcp, mcpv1alpha1.SchemeGroupVersion.WithKind(mcpv1alpha1.KindManagedControlPlane)),
		Keys:      DefaultKeys(),
		Layout:    DefaultLayout(),
		Volumes:   DefaultVolumes(),
		WellKnown: DefaultWellKnown(),
		Contract:  DefaultContract(),
	}
	return cc
}

func NewNames(mcp *mcpv1alpha1.ManagedControlPlane) Names {
	return Names{
		Namespace: mcp.Namespace,
		Name:      mcp.Name,
	}
}

type Keys struct {
	TLSCrt string // "tls.crt"
	TLSKey string // "tls.key"
	CACrt  string // "ca.crt"

	LivezPath  string // "/livez"
	ReadyzPath string // "/readyz"
	HealthPath string // "/healthz"

	AdminKubeconfigKey       string // "config"
	AdminConfigKubeconfigKey string
	LabelKeyApp              string
}

type Layout struct {
	PKIRoot string // "/var/run/k8s"
}

type Volumes struct {
	Kubeconfig string // "kubeconfig"
}

type WellKnown struct {
	// keep only those that are truly global
	KonnectivityAgentNamespace string // "kube-system"
	CoreAPIGroup               string
	DiscoveryAPIGroup          string // "discovery.k8s.io"
}

type Contract struct {
	APIServer    APIServerContract
	Konnectivity KonnectivityContract
	RBAC         RBACContract
}

type APIServerContract struct {
	ServiceNameLegacy string // "kube-apiserver" (legacy)
	SecurePort        int32  // 6443

	AppLabelKey string // "app"
	AppLabelVal string // "kube-apiserver"
}

type KonnectivityContract struct {
	ServerName string // "konnectivity-server"
	AgentName  string // "konnectivity-agent"

	ServerPort int32 // 8132

	EgressSelectorKind       string // "EgressSelectorConfiguration"
	EgressSelectorAPIVersion string // "apiserver.k8s.io/v1beta1"

	ConfigMapName    string // "konnectivity-egress-selector"
	ConfigVolumeName string // "egress-selector"
	ConfigMapKey     string // "konnectivity-egress-selector.yaml"
	ServerUDSVolume  string // "konnectivity-uds"
	UDSFileName      string // "konnectivity-uds.sock"

}

type RBACContract struct {
	RBACAPIGroup    string
	StorageAPIGroup string

	ResourceNodes string
	VerbGet       string
}
