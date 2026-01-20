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

import "github.com/patrostkowski/controlplane-operator/pkg/resources/common"

const (
	apiServerName = common.KubeAPIServerName
	appLabelKey   = common.KubeAPIAppLabelKey
	appLabelVal   = common.KubeAPIAppLabelVal

	konnectivityServerName = common.KonnectivityServerName

	securePort = common.KubeAPISecurePort

	konnectivityServerPort       = common.KonnectivityServerPort
	konnectivityConfigMapName    = common.KonnectivityConfigMapName
	konnectivityConfigVolumeName = common.KonnectivityConfigVolumeName
	konnectivityConfigMapKey     = common.KonnectivityConfigMapKey
	konnectivityConfFileName     = "konnectivity-egress-selector.yaml"

	konnectivityServerMountDir = "/etc/konnectivity"
	konnectivityConfFilePath   = konnectivityServerMountDir + "/" + konnectivityConfFileName
	konnectivityServerUDS      = common.KonnectivityServerUDS
)
