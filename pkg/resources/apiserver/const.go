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

package apiserver

import apiserverv1beta "k8s.io/apiserver/pkg/apis/apiserver/v1beta1"

const (
	securePort = int32(6443)
	grpcPort   = int32(8132)

	livezPath   = "/livez"
	readyzPath  = "/readyz"
	healthzPath = "/healthz"

	appLabelKey = "app"
	appLabelVal = "kube-apiserver"

	egressSelectorKind       = "EgressSelectorConfiguration"
	egressSelectorAPIVersion = "apiserver.k8s.io/v1beta1"

	konnectivityServerName             = "konnectivity-server"
	konnectivityServerPort       int32 = 8132
	konnectivityConfigVolumeName       = "egress-selector"
	konnectivityConfigMapKey           = "konnectivity-egress-selector.yaml"
	konnectivityConfFileName           = "konnectivity-egress-selector.yaml"
	konnectivityServerMountDir         = "/etc/konnectivity"
	konnectivityConfFilePath           = konnectivityServerMountDir + "/" + konnectivityConfFileName
	konnectivityServerUDS              = "konnectivity-uds"

	egressConnectionNameCluster      = "cluster"
	egressConnectionNameControlPlane = "controlplane"
	egressConnectionGRPCType         = apiserverv1beta.ProtocolGRPC
	egressConnectionDirectType       = apiserverv1beta.ProtocolDirect
	konnectivityUDSFile              = "konnectivity-uds.sock"
)
