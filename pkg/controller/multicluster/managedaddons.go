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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewManagedAddon return CR ManagedAddon
func NewManagedAddon() client.Object {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   ManagedAddonGroupName,
		Version: ManagedAddonVersion,
		Kind:    ManagedAddonKind,
	})
	u.SetName(ManagedAddonCRName)
	u.Object["spec"] = map[string]any{}
	return u
}

// newManagedAddonCRD return CRD ManagedAddon
func newManagedAddonsCRD() *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIExtensionsGV,
			Kind:       APIExtensionsKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ManagedAddonCRDName,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: ManagedAddonGroupName,
			Names: apiextv1.CustomResourceDefinitionNames{
				Kind:     ManagedAddonKind,
				ListKind: ManagedAddonList,
				Plural:   ManagedAddonPlural,
				Singular: ManagedAddonSingular,
				ShortNames: []string{
					ManagedAddonShortName,
				},
			},
			Scope: ManagedAddonCRDScope,
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:    ManagedAddonVersion,
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
