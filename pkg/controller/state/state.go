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

const (
	ConditionReconciling mcpv1alpha1.Condition = "Reconciling"
	ConditionReady       mcpv1alpha1.Condition = "Ready"

	ReasonReconciling         mcpv1alpha1.Reason = "Reconciling"
	ReasonWaitingForResources mcpv1alpha1.Reason = "WaitingForResources"
	ReasonComponentFailed     mcpv1alpha1.Reason = "ComponentFailed"
	ReasonAllResourcesReady   mcpv1alpha1.Reason = "AllResourcesReady"

	MessageReconciling         mcpv1alpha1.Message = "Reconciling control plane"
	MessageWaitingForResources mcpv1alpha1.Message = "Waiting for resources"
	MessageAllResourcesReady   mcpv1alpha1.Message = "All resources are Ready"

	MessageAPIServiceFailed  mcpv1alpha1.Message = "Failed to reconcile API Service"
	MessageAPIServiceWaiting mcpv1alpha1.Message = "Waiting for API Service"

	MessageAPIServiceSvcFailed  mcpv1alpha1.Message = "Failed to reconcile API Service service IP address"
	MessageAPIServiceSvcWaiting mcpv1alpha1.Message = "Waiting for API Service IP address"

	MessagePKIFailed  mcpv1alpha1.Message = "Failed to reconcile PKI"
	MessagePKIWaiting mcpv1alpha1.Message = "Waiting for PKI"

	MessageAdminKubeconfigFailed  mcpv1alpha1.Message = "Failed to reconcile admin kubeconfig"
	MessageAdminKubeconfigWaiting mcpv1alpha1.Message = "Waiting for admin kubeconfig"

	MessageETCDFailed  mcpv1alpha1.Message = "Failed to reconcile ETCD"
	MessageETCDWaiting mcpv1alpha1.Message = "Waiting for ETCD"

	MessageAPIServerSvcFailed  mcpv1alpha1.Message = "Failed to reconcile API Server service IP address"
	MessageAPIServerSvcWaiting mcpv1alpha1.Message = "Waiting for API Server IP address"

	MessageKonnectivityServerSvcFailed  mcpv1alpha1.Message = "Failed to reconcile Konnectivity Server service IP address"
	MessageKonnectivityServerSvcWaiting mcpv1alpha1.Message = "Waiting for Konnectivity Server IP address"

	MessageAPIServerFailed  mcpv1alpha1.Message = "Failed to reconcile API Server"
	MessageAPIServerWaiting mcpv1alpha1.Message = "Waiting for API Server"

	MessageControllerManagerFailed  mcpv1alpha1.Message = "Failed to reconcile Controller Manager"
	MessageControllerManagerWaiting mcpv1alpha1.Message = "Waiting for Controller Manager"

	MessageSchedulerFailed  mcpv1alpha1.Message = "Failed to reconcile Scheduler"
	MessageSchedulerWaiting mcpv1alpha1.Message = "Waiting for Scheduler"

	MessageKubeResourcesFailed  mcpv1alpha1.Message = "Failed to reconcile Kubernetes resources"
	MessageKubeResourcesWaiting mcpv1alpha1.Message = "Waiting for Kubernetes resources"

	MessageAddonsFailed  mcpv1alpha1.Message = "Failed to reconcile Addons"
	MessageAddonsWaiting mcpv1alpha1.Message = "Waiting for Addons"
)
