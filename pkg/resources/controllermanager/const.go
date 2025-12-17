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

package controllermanager

const (
	// Component identity
	componentName = "kube-controller-manager"

	labelValApp = "kcm"

	containerName = "kcm"

	// Kubeconfig ConfigMap
	cmKubeconfigName     = "controller-kubeconfig"
	cmKubeconfigKey      = "controller-manager.conf"
	cmKubeconfigFileName = "controller-manager.conf"

	// Mounts/paths
	kubeconfigMountDir = "/etc/kubernetes"
	kubeconfigPath     = kubeconfigMountDir + "/" + cmKubeconfigFileName

	// Controller-manager secure port
	securePort int32 = 10257
	healthPath       = "/healthz"
)
