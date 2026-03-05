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

import mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"

const (
	keyCACrt  = "ca.crt"
	keyTLSCrt = "tls.crt"
	keyTLSKey = "tls.key"

	kubeSystemNamespace = "kube-system"

	bootstrapTokenDescription = "Bootstrap token for kubelet workers"

	groupBootstrappers = "system:bootstrappers"
	groupNodes         = "system:nodes"

	crbKubeadmNodeAutoapproveBootstrap = "kubeadm:node-autoapprove-bootstrap"
	crbKubeadmNodeAutoapproveRotation  = "kubeadm:node-autoapprove-rotation"
	crbNodeBootstrapperName            = "kubelet-bootstrap"

	roleNodeClientCSRApprove = "system:certificates.k8s.io:certificatesigningrequests:nodeclient"
	roleSelfNodeClientCSR    = "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient"
	roleNodeBootstrapper     = "system:node-bootstrapper"

	konnectivityAgentName            = "konnectivity-agent"
	konnectivityServerPort     int32 = 8132
	konnectivityAgentDSName          = "konnectivity-agent"
	konnectivityAgentNamespace       = "kube-system"
)

var partOf = mcpv1alpha1.PartOf
