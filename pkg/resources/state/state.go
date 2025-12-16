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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionReady              mcpv1alpha1.Condition = "Ready"
	ConditionReconciling        mcpv1alpha1.Condition = "Reconciling"
	ConditionStalled            mcpv1alpha1.Condition = "Stalled"
	ConditionResourcesReady     mcpv1alpha1.Condition = "ResourcesReady"
	ConditionDeploymentReady    mcpv1alpha1.Condition = "DeploymentReady"
	ConditionWaitingForResource mcpv1alpha1.Condition = "WaitingForResource"

	ReasonDeploymentReady      mcpv1alpha1.Reason = "DeploymentReady"
	ReasonDeploymentNotReady   mcpv1alpha1.Reason = "DeploymentNotReady"
	ReasonWaitingForDeployment mcpv1alpha1.Reason = "WaitingForDeployment"
	ReasonWaitingForService    mcpv1alpha1.Reason = "WaitingForService"
	ReasonResourceCreated      mcpv1alpha1.Reason = "ResourceCreated"
	ReasonResourceUpdated      mcpv1alpha1.Reason = "ResourceUpdated"
	ReasonResourceNotReady     mcpv1alpha1.Reason = "ResourceNotReady"
	ReasonAllResourcesReady    mcpv1alpha1.Reason = "ResourcesReady"
	ReasonWaitingForResources  mcpv1alpha1.Reason = "ResourcesNotReady"
	ReasonControlPlaneReady    mcpv1alpha1.Reason = "ControlPlaneReady"
	ReasonReconciling          mcpv1alpha1.Reason = "ResourcesReconciling"
	ReasonFailed               mcpv1alpha1.Reason = "ReasonFailed"

	MessageDeploymentReady                  mcpv1alpha1.Message = "Deployment is ready"
	MessageDeploymentNotReady               mcpv1alpha1.Message = "Deployment is not ready yet"
	MessageWaitingForDeployment             mcpv1alpha1.Message = "Waiting for Deployment to become Ready"
	MessageWaitingForService                mcpv1alpha1.Message = "Waiting for Service to be created"
	MessageAllResourcesReady                mcpv1alpha1.Message = "All resources are Ready"
	MessageNotAllResourcesReady             mcpv1alpha1.Message = "Some resources are not Ready"
	MessageWaitingForResources              mcpv1alpha1.Message = "Waiting for resources to become Ready"
	MessageControlPlaneAllReady             mcpv1alpha1.Message = "All control plane components are Ready"
	MessageWaitingForPKI                    mcpv1alpha1.Message = "Waiting for ManagedPKI to become Ready"
	MessageWaitingForETCD                   mcpv1alpha1.Message = "Waiting for ManagedControlPlane to become Ready"
	MessageWaitingForAPIServer              mcpv1alpha1.Message = "Waiting for ManagedControlPlane to become Ready"
	MessageWaitingForControllerMgr          mcpv1alpha1.Message = "Waiting for ManagedControlPlane to become Ready"
	MessageWaitingForScheduler              mcpv1alpha1.Message = "Waiting for ManagedControlPlane to become Ready"
	MessageAPIServerDeploymentReady         mcpv1alpha1.Message = "kube-apiserver Deployment is Ready"
	MessageAPIServerWaitingForDeployment    mcpv1alpha1.Message = "Waiting for kube-apiserver Deployment to become Ready"
	MessageControllerManagerDeploymentReady mcpv1alpha1.Message = "kube-controller-manager Deployment is Ready"
	MessageControllerManagerWaiting         mcpv1alpha1.Message = "Waiting for kube-controller-manager Deployment to become Ready"
	MessageSchedulerDeploymentReady         mcpv1alpha1.Message = "kube-scheduler Deployment is Ready"
	MessageSchedulerWaiting                 mcpv1alpha1.Message = "Waiting for kube-scheduler Deployment to become Ready"
	MessagePKIAllResourcesReady             mcpv1alpha1.Message = "All PKI resources are Ready"
	MessagePKIWaitingResources              mcpv1alpha1.Message = "Waiting for PKI resources to become Ready"
	MessageETCDAllResourcesReady            mcpv1alpha1.Message = "etcd StatefulSet is Ready"
	MessageETCDWaitingResources             mcpv1alpha1.Message = "Waiting for etcd StatefulSet to become Ready"
	MessageMCPAllReady                      mcpv1alpha1.Message = "All control plane components are Ready"
	MessageMCPWaitingResources              mcpv1alpha1.Message = "Waiting for control plane components to become Ready"
	MessageReconciling                      mcpv1alpha1.Message = "Awaiting reconciliation of resources"
	MessageFailed                           mcpv1alpha1.Message = "Step failed"
)

// Separate helper for the top-level MCP itself (if you want different reasons)
func ReadyConditionsForMCP(allReady bool) mcpv1alpha1.Conditions {
	if allReady {
		return mcpv1alpha1.Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonControlPlaneReady,
			Message: MessageMCPAllReady,
		}
	}
	return mcpv1alpha1.Conditions{
		Type:    ConditionReady,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonWaitingForResources,
		Message: MessageMCPWaitingResources,
	}
}
