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

package controlplane

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionReady              Condition = "Ready"
	ConditionReconciling        Condition = "Reconciling"
	ConditionStalled            Condition = "Stalled"
	ConditionResourcesReady     Condition = "ResourcesReady"
	ConditionDeploymentReady    Condition = "DeploymentReady"
	ConditionWaitingForResource Condition = "WaitingForResource"

	ReasonDeploymentReady      Reason = "DeploymentReady"
	ReasonDeploymentNotReady   Reason = "DeploymentNotReady"
	ReasonWaitingForDeployment Reason = "WaitingForDeployment"
	ReasonWaitingForService    Reason = "WaitingForService"
	ReasonResourceCreated      Reason = "ResourceCreated"
	ReasonResourceUpdated      Reason = "ResourceUpdated"
	ReasonResourceNotReady     Reason = "ResourceNotReady"
	ReasonAllResourcesReady    Reason = "ResourcesReady"
	ReasonWaitingForResources  Reason = "ResourcesNotReady"
	ReasonControlPlaneReady    Reason = "ControlPlaneReady"
	ReasonReconciling          Reason = "ResourcesReconciling"

	MessageDeploymentReady                  Message = "Deployment is ready"
	MessageDeploymentNotReady               Message = "Deployment is not ready yet"
	MessageWaitingForDeployment             Message = "Waiting for Deployment to become Ready"
	MessageWaitingForService                Message = "Waiting for Service to be created"
	MessageAllResourcesReady                Message = "All resources are Ready"
	MessageNotAllResourcesReady             Message = "Some resources are not Ready"
	MessageWaitingForResources              Message = "Waiting for resources to become Ready"
	MessageControlPlaneAllReady             Message = "All control plane components are Ready"
	MessageWaitingForPKI                    Message = "Waiting for ManagedPKI to become Ready"
	MessageWaitingForETCD                   Message = "Waiting for ManagedETCD to become Ready"
	MessageWaitingForAPIServer              Message = "Waiting for ManagedAPIServer to become Ready"
	MessageWaitingForControllerMgr          Message = "Waiting for ManagedControllerManager to become Ready"
	MessageWaitingForScheduler              Message = "Waiting for ManagedScheduler to become Ready"
	MessageAPIServerDeploymentReady         Message = "kube-apiserver Deployment is Ready"
	MessageAPIServerWaitingForDeployment    Message = "Waiting for kube-apiserver Deployment to become Ready"
	MessageControllerManagerDeploymentReady Message = "kube-controller-manager Deployment is Ready"
	MessageControllerManagerWaiting         Message = "Waiting for kube-controller-manager Deployment to become Ready"
	MessageSchedulerDeploymentReady         Message = "kube-scheduler Deployment is Ready"
	MessageSchedulerWaiting                 Message = "Waiting for kube-scheduler Deployment to become Ready"
	MessagePKIAllResourcesReady             Message = "All PKI resources are Ready"
	MessagePKIWaitingResources              Message = "Waiting for PKI resources to become Ready"
	MessageETCDAllResourcesReady            Message = "etcd StatefulSet is Ready"
	MessageETCDWaitingResources             Message = "Waiting for etcd StatefulSet to become Ready"
	MessageMCPAllReady                      Message = "All control plane components are Ready"
	MessageMCPWaitingResources              Message = "Waiting for control plane components to become Ready"
	MessageReconciling                      Message = "Awaiting reconciliation of resources"
)

type Condition string
type Reason string
type Message string

type Conditions struct {
	Type    Condition
	Status  metav1.ConditionStatus
	Reason  Reason
	Message Message
}

type Status struct {
	Message string `json:"message,omitempty"`
	Ready   bool   `json:"ready,omitempty"`
}

func (r *BaseReconciler) UpdateCondition(
	ctx context.Context,
	obj ObjectHelper,
	conditions Conditions,
	status Status,
) error {
	key := client.ObjectKeyFromObject(obj)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := obj.DeepCopyObject().(ObjectHelper)
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}

		cond := metav1.Condition{
			Message:            string(conditions.Message),
			Reason:             string(conditions.Reason),
			Status:             conditions.Status,
			Type:               string(conditions.Type),
			ObservedGeneration: latest.GetGeneration(),
		}

		conds := latest.GetConditions()
		if !meta.SetStatusCondition(conds, cond) {
			return nil
		}

		if st := latest.GetStatus(); st != nil {
			st.Message = string(conditions.Message)

			if conditions.Type == ConditionReady {
				st.Ready = (conditions.Status == metav1.ConditionTrue)
			}
		}

		r.Log.Info("Updating status condition",
			"kind", latest.GetObjectKind().GroupVersionKind().Kind,
			"name", latest.GetName(),
			"type", conditions.Type,
			"status", conditions.Status,
			"reason", conditions.Reason,
			"message", conditions.Message,
		)
		return r.Status().Update(ctx, latest)
	})
}

// func (r *BaseReconciler) UpdateStatus(
// 	ctx context.Context,
// 	obj ObjectHelper,
// 	status Status,
// ) error {
// 	key := client.ObjectKeyFromObject(obj)
// 	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
// 		latest := obj.DeepCopyObject().(ObjectHelper)
// 		if err := r.Get(ctx, key, latest); err != nil {
// 			return err
// 		}

// 		st := latest.GetStatus()
// 		st.Message = status.Message
// 		st.Ready = status.Ready

// 		r.Log.Info("Updating resource status",
// 			"kind", latest.GetObjectKind().GroupVersionKind().Kind,
// 			"name", latest.GetName(),
// 			"message", st.Message,
// 			"ready", st.Ready,
// 		)
// 		return r.Status().Update(ctx, latest)
// 	})
// }

// Separate helper for the top-level MCP itself (if you want different reasons)
func ReadyConditionsForMCP(allReady bool) Conditions {
	if allReady {
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonControlPlaneReady,
			Message: MessageMCPAllReady,
		}
	}
	return Conditions{
		Type:    ConditionReady,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonWaitingForResources,
		Message: MessageMCPWaitingResources,
	}
}
