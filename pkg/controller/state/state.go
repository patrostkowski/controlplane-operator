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

package state

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
)

// Condition defines the type of condition for the ManagedControlPlane status.
const (

	// ConditionReconciling indicates that the control plane is currently reconciling.
	ConditionReconciling mcpv1alpha1.Condition = "Reconciling"

	// ConditionReady indicates that the control plane is ready.
	ConditionReady mcpv1alpha1.Condition = "Ready"
)

// Reason defines the reason for a ManagedControlPlane condition.
const (
	// ReasonReconciling indicates that the control plane is actively reconciling.
	ReasonReconciling mcpv1alpha1.Reason = "Reconciling"
	// ReasonWaitingForResources indicates that the reconciliation is waiting for resources to become ready.
	ReasonWaitingForResources mcpv1alpha1.Reason = "WaitingForResources"
	// ReasonComponentFailed indicates that a specific component failed during reconciliation.
	ReasonComponentFailed mcpv1alpha1.Reason = "ComponentFailed"
	// ReasonAllResourcesReady indicates that all required resources are ready.
	ReasonAllResourcesReady mcpv1alpha1.Reason = "AllResourcesReady"
)

// Message defines detailed messages for ManagedControlPlane conditions.
const (
	// MessageReconciling indicates that the control plane is currently reconciling.
	MessageReconciling mcpv1alpha1.Message = "Reconciling control plane"
	// MessageWaitingForResources indicates that the control plane is waiting for resources to become ready.
	MessageWaitingForResources mcpv1alpha1.Message = "Waiting for resources"
	// MessageAllResourcesReady indicates that all control plane resources are ready.
	MessageAllResourcesReady mcpv1alpha1.Message = "All resources are Ready"

	// MessageAPIServiceFailed indicates that the API Service reconciliation failed.
	MessageAPIServiceFailed mcpv1alpha1.Message = "Failed to reconcile API Service"
	// MessageAPIServiceWaiting indicates that the control plane is waiting for the API Service.
	MessageAPIServiceWaiting mcpv1alpha1.Message = "Waiting for API Service"

	// MessageAPIServiceSvcFailed indicates that the API Service service IP address reconciliation failed.
	MessageAPIServiceSvcFailed mcpv1alpha1.Message = "Failed to reconcile API Service service IP address"
	// MessageAPIServiceSvcWaiting indicates that the control plane is waiting for the API Service IP address.
	MessageAPIServiceSvcWaiting mcpv1alpha1.Message = "Waiting for API Service IP address"

	// MessagePKIFailed indicates that the PKI reconciliation failed.
	MessagePKIFailed mcpv1alpha1.Message = "Failed to reconcile PKI"
	// MessagePKIWaiting indicates that the control plane is waiting for PKI.
	MessagePKIWaiting mcpv1alpha1.Message = "Waiting for PKI"

	// MessageAdminKubeconfigFailed indicates that the admin kubeconfig reconciliation failed.
	MessageAdminKubeconfigFailed mcpv1alpha1.Message = "Failed to reconcile admin kubeconfig"
	// MessageAdminKubeconfigWaiting indicates that the control plane is waiting for the admin kubeconfig.
	MessageAdminKubeconfigWaiting mcpv1alpha1.Message = "Waiting for admin kubeconfig"

	// MessageETCDFailed indicates that the etcd reconciliation failed.
	MessageETCDFailed mcpv1alpha1.Message = "Failed to reconcile ETCD"
	// MessageETCDWaiting indicates that the control plane is waiting for etcd.
	MessageETCDWaiting mcpv1alpha1.Message = "Waiting for ETCD"

	// MessageAPIServerSvcFailed indicates that the API server service IP address reconciliation failed.
	MessageAPIServerSvcFailed mcpv1alpha1.Message = "Failed to reconcile API Server service IP address"
	// MessageAPIServerSvcWaiting indicates that the control plane is waiting for the API server IP address.
	MessageAPIServerSvcWaiting mcpv1alpha1.Message = "Waiting for API Server IP address"

	// MessageKonnectivityServerSvcFailed indicates that the Konnectivity Server service IP address reconciliation failed.
	MessageKonnectivityServerSvcFailed mcpv1alpha1.Message = "Failed to reconcile Konnectivity Server service IP address"
	// MessageKonnectivityServerSvcWaiting indicates that the control plane is waiting for the Konnectivity Server IP address.
	MessageKonnectivityServerSvcWaiting mcpv1alpha1.Message = "Waiting for Konnectivity Server IP address"

	// MessageAPIServerFailed indicates that the API server reconciliation failed.
	MessageAPIServerFailed mcpv1alpha1.Message = "Failed to reconcile API Server"
	// MessageAPIServerWaiting indicates that the control plane is waiting for the API server.
	MessageAPIServerWaiting mcpv1alpha1.Message = "Waiting for API Server"

	// MessageControllerManagerFailed indicates that the Controller Manager reconciliation failed.
	MessageControllerManagerFailed mcpv1alpha1.Message = "Failed to reconcile Controller Manager"
	// MessageControllerManagerWaiting indicates that the control plane is waiting for the Controller Manager.
	MessageControllerManagerWaiting mcpv1alpha1.Message = "Waiting for Controller Manager"

	// MessageSchedulerFailed indicates that the Scheduler reconciliation failed.
	MessageSchedulerFailed mcpv1alpha1.Message = "Failed to reconcile Scheduler"
	// MessageSchedulerWaiting indicates that the control plane is waiting for the Scheduler.
	MessageSchedulerWaiting mcpv1alpha1.Message = "Waiting for Scheduler"

	// MessageKubeResourcesFailed indicates that the Kubernetes resources reconciliation failed.
	MessageKubeResourcesFailed mcpv1alpha1.Message = "Failed to reconcile Kubernetes resources"
	// MessageKubeResourcesWaiting indicates that the control plane is waiting for Kubernetes resources.
	MessageKubeResourcesWaiting mcpv1alpha1.Message = "Waiting for Kubernetes resources"

	// MessageAddonsFailed indicates that the Addons reconciliation failed.
	MessageAddonsFailed mcpv1alpha1.Message = "Failed to reconcile Addons"
	// MessageAddonsWaiting indicates that the control plane is waiting for Addons.
	MessageAddonsWaiting mcpv1alpha1.Message = "Waiting for Addons"
)
