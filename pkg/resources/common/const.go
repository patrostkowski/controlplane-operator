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

package common

const (
	TLSCrtKey = "tls.crt"
	TLSKeyKey = "tls.key"
	CACrtKey  = "ca.crt"

	PKIMountRoot = "/var/run/k8s"

	LabelKeyApp = "app"

	// Probes
	LivezPath  = "/livez"
	ReadyzPath = "/readyz"
	HealthPath = "/healthz"

	KubeconfigVolumeName = "kubeconfig"

	AdminConfigName          = "admin-config"
	AdminConfigKubeconfigKey = "config"
)

// API server vars
const (
	KubeAPIServerName  = "kube-apiserver"
	KubeAPIAppLabelKey = "app"
	KubeAPIAppLabelVal = "kube-apiserver"

	KonnectivityServerName = "konnectivity-server"
	KonnectivityAgentName  = "konnectivity-agent"

	KubeAPISecurePort int32 = 6443
)

// Konnectivity vars
const (
	EgressSelectorKind             = "EgressSelectorConfiguration"
	EgressSelectorAPIVersion       = "apiserver.k8s.io/v1beta1"
	KonnectivityCASecretName       = "konnectivity-ca"
	KonnectivityTLSSecretName      = "konnectivity-server-tls"
	KonnectivityAgentTLSSecretName = "konnectivity-agent-tls"

	KonnectivityCACN     = "konnectivity-ca"
	KonnectivityServerCN = "konnectivity-server"
	KonnectivityAgentCN  = "konnectivity-agent"

	KonnectivityServerPort       int32 = 8132
	KonnectivityAgentNamespace         = "kube-system"
	KonnectivityConfigMapName          = "konnectivity-egress-selector"
	KonnectivityConfigVolumeName       = "egress-selector"
	KonnectivityConfigMapKey           = "konnectivity-egress-selector.yaml"
	KonnectivityAgentDSName            = "konnectivity-agent"

	KonnectivityServerUDS = "konnectivity-uds"
)
