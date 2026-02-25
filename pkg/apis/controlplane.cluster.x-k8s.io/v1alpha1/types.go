// Copyright 2026
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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
)

const (
	KindManagedControlPlane         = "ManagedControlPlane"
	KindManagedControlPlaneList     = "ManagedControlPlaneList"
	KindManagedControlPlaneTemplate = "ManagedControlPlaneTemplate"
)

// ManagedControlPlaneSpec defines the desired state of ManagedControlPlane.
// This schema follows the Cluster API ControlPlane provider contract (v1beta2).
type ManagedControlPlaneSpec struct {
	// clusterName is the name of the Cluster this ControlPlane belongs to.
	//
	// NOTE: This field is not part of the contract itself, but it is a common provider pattern.
	// Cluster API core controllers set this via topology/patching and labels/ownerrefs.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// replicas represent the number of desired replicas.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// version defines the desired Kubernetes version for the control plane.
	// The value must be a valid semantic version; if the value provided by the user does not start with the v prefix, it
	// must be added by the controller.
	//
	// NOTE: This is mandatory for ClusterClass support.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Version string `json:"version"`

	// templateRef optionally points to an ManagedControlPlaneTemplate used to generate this ManagedControlPlane.
	// This is not used by the contract directly but can help with debugging/traceability.
	// +optional
	TemplateRef *corev1.ObjectReference `json:"templateRef,omitempty"`

	// providerSpec contains controlplane-operator specific configuration.
	// This is used as the base to create/update the backing ManagedControlPlane.
	// +optional
	ProviderSpec *mcpv1alpha1.ManagedControlPlaneSpec `json:"providerSpec,omitempty"`
}

// ManagedControlPlaneInitializationStatus provides observations of the ManagedControlPlane initialization process.
// +kubebuilder:validation:MinProperties=1
type ManagedControlPlaneInitializationStatus struct {
	// controlPlaneInitialized is true when the control plane provider reports that the Kubernetes control plane is initialized.
	// NOTE: this field is part of the Cluster API contract.
	// +optional
	ControlPlaneInitialized *bool `json:"controlPlaneInitialized,omitempty"`
}

// ManagedControlPlaneStatus defines the observed state of ManagedControlPlane.
type ManagedControlPlaneStatus struct {
	// initialization provides observations of the ManagedControlPlane initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Cluster provisioning.
	// +optional
	Initialization ManagedControlPlaneInitializationStatus `json:"initialization,omitempty,omitzero"`

	// selector is the label selector in string format to avoid introspection by clients.
	// Used by the scale subresource.
	// +optional
	Selector string `json:"selector,omitempty"`

	// replicas is the total number of machines/instances targeted by this control plane.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// readyReplicas is the number of ready replicas for this ControlPlane.
	// +optional
	ReadyReplicas *int32 `json:"readyReplicas,omitempty"`

	// availableReplicas is the number of available replicas for this ControlPlane.
	// +optional
	AvailableReplicas *int32 `json:"availableReplicas,omitempty"`

	// upToDateReplicas is the number of up-to-date replicas targeted by this ControlPlane.
	// +optional
	UpToDateReplicas *int32 `json:"upToDateReplicas,omitempty"`

	// version represents the minimum Kubernetes version for the control plane.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Version string `json:"version,omitempty"`

	// conditions define current service state of the ControlPlane.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// observedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=managedcontrolplanes,shortName=mcp,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.initialization.controlPlaneInitialized",description="Control plane initialized"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="Kubernetes version"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age"
// +kubebuilder:scale:labelSelectorPath=.status.selector,specReplicasPath=.spec.replicas,statusReplicasPath=.status.replicas

// ManagedControlPlane is the Schema for ManagedControlPlanes.
// It is a ControlPlane resource compliant with the Cluster API contract.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:categories=cluster-api
// +kubebuilder:metadata:labels="cluster.x-k8s.io/v1beta2=v1alpha1"
type ManagedControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedControlPlaneSpec   `json:"spec,omitempty"`
	Status ManagedControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedControlPlaneList contains a list of ManagedControlPlane.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:categories=cluster-api
type ManagedControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedControlPlane `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=managedcontrolplanetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// ManagedControlPlaneTemplate is the Schema for ManagedControlPlaneTemplates.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:categories=cluster-api
// +kubebuilder:metadata:labels="cluster.x-k8s.io/v1beta2=v1alpha1"
type ManagedControlPlaneTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ManagedControlPlaneTemplateSpec `json:"spec,omitempty"`
}

// ManagedControlPlaneTemplateSpec defines the desired state of ManagedControlPlaneTemplate.
type ManagedControlPlaneTemplateSpec struct {
	Template ManagedControlPlaneTemplateResource `json:"template"`
}

// ManagedControlPlaneTemplateResource defines the template to create ManagedControlPlanes.
type ManagedControlPlaneTemplateResource struct {
	// Standard object's metadata.
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	Spec ManagedControlPlaneSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// ManagedControlPlaneTemplateList contains a list of ManagedControlPlaneTemplate.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:categories=cluster-api
type ManagedControlPlaneTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedControlPlaneTemplate `json:"items"`
}
