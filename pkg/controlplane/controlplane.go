package controlplane

import (
	"context"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BaseReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *BaseReconciler) GetOrIgnoreNotFound(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
) error {
	if err := r.Get(ctx, key, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

type ObjectHelper interface {
	client.Object
	GetConditions() *[]metav1.Condition
}

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
