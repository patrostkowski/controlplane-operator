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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

const (
	KindManagedControlPlane     = "ManagedControlPlane"
	KindManagedControlPlaneList = "ManagedControlPlaneList"
)

// TODO: add kubebuilder tags for validations

// ManagedControlPlaneSpec defines the desired state of ManagedControlPlane.
type ManagedControlPlaneSpec struct {
	// Version is the desired Kubernetes control plane version, e.g. "v1.34.0".
	Kubernetes        KubernetesSpec         `json:"kubernetes"`
	Addons            *AddonsSpec            `json:"addons,omitempty"`
	APIServer         *APIServerSpec         `json:"apiserver,omitempty"`
	ControllerManager *ControllerManagerSpec `json:"controllerManager,omitempty"`
	Scheduler         *SchedulerSpec         `json:"scheduler,omitempty"`
	ETCD              *ETCDSpec              `json:"etcd,omitempty"`
}

// ManagedControlPlaneStatus defines the observed state of ManagedControlPlane.
type ManagedControlPlaneStatus struct {
	// Conditions represents the latest available observations of the control plane's state.
	// e.g. APIServerAvailable, EtcdHealthy, ControllersHealthy, Ready, etc.
	Conditions               []metav1.Condition `json:"conditions,omitempty"`
	Status                   `json:",inline,omitempty"`
	Address                  string                   `json:"address,omitempty"`
	AdminKubeconfigSecretRef AdminKubeconfigSecretRef `json:"adminKubeconfigSecretRef,omitempty"`
}

type APIServerSpec struct {
	AvailabilitySpec `json:",inline"`
}

type ControllerManagerSpec struct {
	AvailabilitySpec `json:",inline"`
}

type SchedulerSpec struct {
	AvailabilitySpec `json:",inline"`
}

type ETCDSpec struct {
	AvailabilitySpec `json:",inline"`
}

type AddonsSpec struct {
	CoreDNS      *CoreDNS      `json:"coredns,omitempty"`
	CSI          *CSI          `json:"csi,omitempty"`
	Flannel      *Flannel      `json:"flannel,omitempty"`
	Konnectivity *Konnectivity `json:"konnectivity,omitempty"`
	Kubeproxy    *Kubeproxy    `json:"kubeproxy,omitempty"`
}

type CoreDNS struct {
	Enabled *bool `json:"enabled"`
}

type CSI struct {
	Enabled *bool `json:"enabled"`
}

type Flannel struct {
	Enabled *bool `json:"enabled"`
}

type Konnectivity struct {
	Enabled *bool `json:"enabled"`
}

type Kubeproxy struct {
	Enabled *bool `json:"enabled"`
}

type KubernetesSpec struct {
	Version    string          `json:"version"`
	Networking *NetworkingSpec `json:"networking,omitempty"`
}

type NetworkingSpec struct {
	// +kubebuilder:validation:Items:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/([0-9]|[12][0-9]|3[0-2])$`
	// +optional
	PodCIDR string `json:"podCIDR,omitempty"`

	// +kubebuilder:validation:Items:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/([0-9]|[12][0-9]|3[0-2])$`
	// +optional
	ServiceCIDR string `json:"serviceCIDR,omitempty"`
}

// ManagedControlPlane is the root CR that “owns” the other managed components.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedcontrolplanes,scope=Namespaced,shortName=mcp
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Control plane ready"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message",description="Last status message"
// +kubebuilder:printcolumn:name="Address",type="string",JSONPath=".status.address",description="API server address"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age"
type ManagedControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedControlPlaneSpec   `json:"spec,omitempty"`
	Status ManagedControlPlaneStatus `json:"status,omitempty"`
}

// ManagedControlPlaneList contains a list of ManagedControlPlane.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedControlPlane `json:"items"`
}

type Status struct {
	Message string `json:"message"`
	Ready   *bool  `json:"ready"`
}

type AdminKubeconfigSecretRef struct {
	Name string `json:"name"`
}

type AvailabilitySpec struct {
	Replicas                 *int32                           `json:"replicas,omitempty"`
	Resources                *corev1.ResourceRequirements     `json:"resources,omitempty"`
	TopologySpreadConstraint *corev1.TopologySpreadConstraint `json:"topologySpreadConstraint,omitempty"`
	*corev1.Affinity         `json:",inline,omitempty"`
}

type (
	Condition string
	Reason    string
	Message   string
)

type Conditions struct {
	Type    Condition
	Status  metav1.ConditionStatus
	Reason  Reason
	Message Message
}
