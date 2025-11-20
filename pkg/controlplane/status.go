package controlplane

import (
	"context"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionReady              ConditionType = "Ready"
	ConditionReconciling        ConditionType = "Reconciling"
	ConditionStalled            ConditionType = "Stalled"
	ConditionResourcesReady     ConditionType = "ResourcesReady"
	ConditionDeploymentReady    ConditionType = "DeploymentReady"
	ConditionWaitingForResource ConditionType = "WaitingForResource"

	ReasonDeploymentReady      ConditionReason = "DeploymentReady"
	ReasonDeploymentNotReady   ConditionReason = "DeploymentNotReady"
	ReasonWaitingForDeployment ConditionReason = "WaitingForDeployment"
	ReasonWaitingForService    ConditionReason = "WaitingForService"
	ReasonResourceCreated      ConditionReason = "ResourceCreated"
	ReasonResourceUpdated      ConditionReason = "ResourceUpdated"
	ReasonResourceNotReady     ConditionReason = "ResourceNotReady"
	ReasonAllResourcesReady    ConditionReason = "ResourcesReady"
	ReasonWaitingForResources  ConditionReason = "ResourcesNotReady"
	ReasonControlPlaneReady    ConditionReason = "ControlPlaneReady"

	MessageDeploymentReady                  ConditionMessage = "Deployment is ready"
	MessageDeploymentNotReady               ConditionMessage = "Deployment is not ready yet"
	MessageWaitingForDeployment             ConditionMessage = "Waiting for Deployment to become Ready"
	MessageWaitingForService                ConditionMessage = "Waiting for Service to be created"
	MessageAllResourcesReady                ConditionMessage = "All resources are Ready"
	MessageNotAllResourcesReady             ConditionMessage = "Some resources are not Ready"
	MessageWaitingForResources              ConditionMessage = "Waiting for resources to become Ready"
	MessageControlPlaneAllReady             ConditionMessage = "All control plane components are Ready"
	MessageWaitingForPKI                    ConditionMessage = "Waiting for ManagedPKI to become Ready"
	MessageWaitingForETCD                   ConditionMessage = "Waiting for ManagedETCD to become Ready"
	MessageWaitingForAPIServer              ConditionMessage = "Waiting for ManagedAPIServer to become Ready"
	MessageWaitingForControllerMgr          ConditionMessage = "Waiting for ManagedControllerManager to become Ready"
	MessageWaitingForScheduler              ConditionMessage = "Waiting for ManagedScheduler to become Ready"
	MessageAPIServerDeploymentReady         ConditionMessage = "kube-apiserver Deployment is Ready"
	MessageAPIServerWaitingForDeployment    ConditionMessage = "Waiting for kube-apiserver Deployment to become Ready"
	MessageControllerManagerDeploymentReady ConditionMessage = "kube-controller-manager Deployment is Ready"
	MessageControllerManagerWaiting         ConditionMessage = "Waiting for kube-controller-manager Deployment to become Ready"
	MessageSchedulerDeploymentReady         ConditionMessage = "kube-scheduler Deployment is Ready"
	MessageSchedulerWaiting                 ConditionMessage = "Waiting for kube-scheduler Deployment to become Ready"
	MessagePKIAllResourcesReady             ConditionMessage = "All PKI resources are Ready"
	MessagePKIWaitingResources              ConditionMessage = "Waiting for PKI resources to become Ready"
	MessageETCDAllResourcesReady            ConditionMessage = "etcd StatefulSet is Ready"
	MessageETCDWaitingResources             ConditionMessage = "Waiting for etcd StatefulSet to become Ready"
	MessageMCPAllReady                      ConditionMessage = "All control plane components are Ready"
	MessageMCPWaitingResources              ConditionMessage = "Waiting for control plane components to become Ready"
)

type ConditionType string
type ConditionReason string
type ConditionMessage string

type Conditions struct {
	Type    ConditionType
	Status  metav1.ConditionStatus
	Reason  ConditionReason
	Message ConditionMessage
}

func (r *BaseReconciler) UpdateCondition(
	ctx context.Context,
	obj ObjectHelper,
	conditions Conditions,
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

// Per-type configs for *child* objects
func ReadyConditionsForChild(obj ObjectHelper, allReady bool) Conditions {
	switch obj.(type) {
	case *mcpv1alpha1.ManagedAPIServer:
		if allReady {
			return Conditions{
				Type:    ConditionReady,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonDeploymentReady,
				Message: MessageAPIServerDeploymentReady,
			}
		}
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonWaitingForDeployment,
			Message: MessageAPIServerWaitingForDeployment,
		}

	case *mcpv1alpha1.ManagedControllerManager:
		if allReady {
			return Conditions{
				Type:    ConditionReady,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonDeploymentReady,
				Message: MessageControllerManagerDeploymentReady,
			}
		}
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonWaitingForDeployment,
			Message: MessageControllerManagerWaiting,
		}

	case *mcpv1alpha1.ManagedScheduler:
		if allReady {
			return Conditions{
				Type:    ConditionReady,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonDeploymentReady,
				Message: MessageSchedulerDeploymentReady,
			}
		}
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonWaitingForDeployment,
			Message: MessageSchedulerWaiting,
		}

	case *mcpv1alpha1.ManagedPKI:
		if allReady {
			return Conditions{
				Type:    ConditionReady,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonAllResourcesReady,
				Message: MessagePKIAllResourcesReady,
			}
		}
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonWaitingForResources,
			Message: MessagePKIWaitingResources,
		}

	case *mcpv1alpha1.ManagedETCD:
		if allReady {
			return Conditions{
				Type:    ConditionReady,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonAllResourcesReady,
				Message: MessageETCDAllResourcesReady,
			}
		}
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonWaitingForResources,
			Message: MessageETCDWaitingResources,
		}
	}

	// fallback generic
	if allReady {
		return Conditions{
			Type:    ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonAllResourcesReady,
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
