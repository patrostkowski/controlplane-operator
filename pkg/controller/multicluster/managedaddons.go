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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewManagedAddon return CR ManagedAddon
func NewManagedAddon() client.Object {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   mcpv1alpha1.ManagedAddonGroupName,
		Version: mcpv1alpha1.ManagedAddonVersion,
		Kind:    mcpv1alpha1.ManagedAddonKind,
	})
	u.SetName(mcpv1alpha1.ManagedAddonCRName)
	u.Object["spec"] = map[string]any{}
	return u
}

// newManagedAddonCRD return CRD ManagedAddon
func newManagedAddonsCRD() *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcpv1alpha1.APIExtensionsGV,
			Kind:       mcpv1alpha1.APIExtensionsKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: mcpv1alpha1.ManagedAddonCRDName,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: mcpv1alpha1.ManagedAddonGroupName,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind:     mcpv1alpha1.ManagedAddonKind,
				ListKind: mcpv1alpha1.ManagedAddonList,
				Plural:   mcpv1alpha1.ManagedAddonPlural,
				Singular: mcpv1alpha1.ManagedAddonSingular,
				ShortNames: []string{
					mcpv1alpha1.ManagedAddonShortName,
				},
			},
			Scope: mcpv1alpha1.ManagedAddonCRDScope,
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:    mcpv1alpha1.ManagedAddonVersion,
					Served:  true,
					Storage: true,
					Schema: &apiextv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextv1.JSONSchemaProps{
							Description: "ManagedAddons is a dummy CR used only as a reconciliation anchor for addons.",
							Type:        "object",
							Required:    []string{"spec"},
							Properties: map[string]apiextv1.JSONSchemaProps{
								"apiVersion": {
									Type: "string",
									Description: "APIVersion defines the versioned schema of this representation of an object.\n" +
										"Servers should convert recognized schemas to the latest internal value, and\n" +
										"may reject unrecognized values.\n" +
										"More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
								},
								"kind": {
									Type: "string",
									Description: "Kind is a string value representing the REST resource this object represents.\n" +
										"Servers may infer this from the endpoint the client submits requests to.\n" +
										"Cannot be updated.\n" +
										"In CamelCase.\n" +
										"More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
								},
								"metadata": {
									Type: "object",
								},
								"spec": {
									Type:        "object",
									Description: "CR is used only as a reconciliation anchor.",
								},
							},
						},
					},
				},
			},
		},
	}
}
