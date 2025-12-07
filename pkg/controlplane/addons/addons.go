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

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterCIDR = "10.244.0.0/16"
)

type AddonsBuilder func() []client.Object

func buildDefaultAddons(builders ...AddonsBuilder) []client.Object {
	var addonObjs []client.Object

	for _, b := range builders {
		addonObjs = append(addonObjs, b()...)
	}
	return addonObjs
}

func Resources(ma *mcpv1alpha1.ManagedAddon) []client.Object {
	return buildDefaultAddons(
		func() []client.Object { return buildKubeproxy(ma) },
		func() []client.Object { return buildFlannel() },
		func() []client.Object { return buildCoreDNS() },
		func() []client.Object { return buildCSI() },
	)
}
