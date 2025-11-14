// Copyright 2025 controlplane.patrostkowski.dev
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
)

// ManagedControlPlaneSpec defines the desired state of ManagedControlPlane.
type ManagedControlPlaneSpec struct {
	// Version is the desired Kubernetes control plane version, e.g. "v1.34.0".
	Version string `json:"version"`
}

// ManagedControlPlaneStatus defines the observed state of ManagedControlPlane.
type ManagedControlPlaneStatus struct {
	// Conditions represents the latest available observations of the control plane's state.
	// e.g. APIServerAvailable, EtcdHealthy, ControllersHealthy, Ready, etc.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedcontrolplanes,scope=Namespaced,shortName=mcp

// ManagedControlPlane is the root CR that “owns” the other managed components.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedControlPlaneSpec   `json:"spec,omitempty"`
	Status ManagedControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedControlPlaneList contains a list of ManagedControlPlane.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedControlPlane `json:"items"`
}

// ManagedPKISpec defines the desired state of ManagedPKI.
type ManagedPKISpec struct {
	// ControlPlaneRef can be used to explicitly link to a ManagedControlPlane
	ControlPlaneName string `json:"controlPlaneName,omitempty"`
}

// ManagedPKIStatus defines the observed state of ManagedPKI.
type ManagedPKIStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ManagedPKI manages the PKI resources (Certificate CRs, Secrets, etc.) for a control plane.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedpkis,scope=Namespaced,shortName=mpki
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedPKI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedPKISpec   `json:"spec,omitempty"`
	Status ManagedPKIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedPKIList contains a list of ManagedPKI.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedPKIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedPKI `json:"items"`
}

// ManagedETCDSpec defines the desired state of ManagedETCD.
type ManagedETCDSpec struct {
	ControlPlaneName string `json:"controlPlaneName,omitempty"`
}

// ManagedETCDStatus defines the observed state of ManagedETCD.
type ManagedETCDStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ManagedETCD manages the etcd StatefulSet and associated configuration.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedetcds,scope=Namespaced,shortName=metcd
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ManagedETCD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedETCDSpec   `json:"spec,omitempty"`
	Status ManagedETCDStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedETCDList contains a list of ManagedETCD.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedETCDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedETCD `json:"items"`
}

// ManagedAPIServerSpec defines the desired state of ManagedAPIServer.
type ManagedAPIServerSpec struct {
	ControlPlaneName string `json:"controlPlaneName,omitempty"`
}

// ManagedAPIServerStatus defines the observed state of ManagedAPIServer.
type ManagedAPIServerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ManagedAPIServer manages the kube-apiserver Deployment/Service and related config.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedapiservers,scope=Namespaced,shortName=mapisrv
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ManagedAPIServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedAPIServerSpec   `json:"spec,omitempty"`
	Status ManagedAPIServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedAPIServerList contains a list of ManagedAPIServer.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedAPIServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedAPIServer `json:"items"`
}

// ManagedControllerManagerSpec defines the desired state of ManagedControllerManager.
type ManagedControllerManagerSpec struct {
	ControlPlaneName string `json:"controlPlaneName,omitempty"`
}

// ManagedControllerManagerStatus defines the observed state of ManagedControllerManager.
type ManagedControllerManagerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ManagedControllerManager manages the kube-controller-manager Deployment.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedcontrollermanagers,scope=Namespaced,shortName=mcm
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ManagedControllerManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedControllerManagerSpec   `json:"spec,omitempty"`
	Status ManagedControllerManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedControllerManagerList contains a list of ManagedControllerManager.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedControllerManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedControllerManager `json:"items"`
}

// ManagedSchedulerSpec defines the desired state of ManagedScheduler.
type ManagedSchedulerSpec struct {
	ControlPlaneName string `json:"controlPlaneName,omitempty"`
}

// ManagedSchedulerStatus defines the observed state of ManagedScheduler.
type ManagedSchedulerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedschedulers,scope=Namespaced,shortName=msched

// ManagedScheduler manages the kube-scheduler Deployment.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedSchedulerSpec   `json:"spec,omitempty"`
	Status ManagedSchedulerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedSchedulerList contains a list of ManagedScheduler.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ManagedSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedScheduler `json:"items"`
}
