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

package provider

import (
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	ManagedAddonCRDName   = "managedaddons.controlplane.patrostkowski.dev"
	ManagedAddonGroupName = "controlplane.patrostkowski.dev"
	ManagedAddonKind      = "ManagedAddon"
	ManagedAddonVersion   = "v1alpha1"
	ManagedAddonList      = "ManagedAddonList"
	ManagedAddonPlural    = "managedaddons"
	ManagedAddonSingular  = "managedaddon"
	ManagedAddonShortName = "ma"
	ManagedAddonCRName    = "addonset"
	APIExtensionsKind     = "CustomResourceDefinition"

	ManagedAddonCRDScope = apiextv1.ClusterScoped
)

var APIExtensionsGV = apiextv1.SchemeGroupVersion.Group + "/" + apiextv1.SchemeGroupVersion.Version
