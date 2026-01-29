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

package addons

const (
	keyCACrt  = "ca.crt"
	keyTLSCrt = "tls.crt"
	keyTLSKey = "tls.key"

	KubeSystemNamespace = "kube-system"

	BootstrapTokenMgmtSecretName = "bootstrap-token"
	BootstrapTokenIDKey          = "token-id"
	BootstrapTokenSecretKey      = "token-secret"
	BootstrapTokenDescription    = "Bootstrap token for kubelet workers"
	BootstrapAuthExtraGroups     = "system:bootstrappers:kubelet-bootstrap"

	BootstrapUsageAuth = "true"
	BootstrapUsageSign = "true"

	GroupBootstrappers = "system:bootstrappers"
	GroupNodes         = "system:nodes"

	CRBKubeadmNodeAutoapproveBootstrap = "kubeadm:node-autoapprove-bootstrap"
	CRBKubeadmNodeAutoapproveRotation  = "kubeadm:node-autoapprove-rotation"
	CRBNodeBootstrapperName            = "kubelet-bootstrap"

	RoleNodeClientCSRApprove = "system:certificates.k8s.io:certificatesigningrequests:nodeclient"
	RoleSelfNodeClientCSR    = "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient"
	RoleNodeBootstrapper     = "system:node-bootstrapper"

	konnectivityAgentName            = "konnectivity-agent"
	konnectivityServerPort     int32 = 8132
	konnectivityAgentDSName          = "konnectivity-agent"
	konnectivityAgentNamespace       = "kube-system"
)
