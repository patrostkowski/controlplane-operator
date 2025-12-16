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

package controller

import (
	"context"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	appsv1 "k8s.io/api/apps/v1"
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

type ObjectHelper interface {
	client.Object
	GetConditions() *[]metav1.Condition
	GetStatus() *mcpv1alpha1.Status
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

func (r *BaseReconciler) IsDeploymentReady(dep *appsv1.Deployment) bool {
	desired := *dep.Spec.Replicas
	if dep.Status.ReadyReplicas < desired {
		return false
	}
	if dep.Generation > dep.Status.ObservedGeneration {
		return false
	}
	if dep.Status.UpdatedReplicas < desired {
		return false
	}
	if dep.Status.ReadyReplicas < desired || dep.Status.AvailableReplicas < desired {
		return false
	}
	return true
}

func (r *BaseReconciler) UpdateCondition(
	ctx context.Context,
	obj ObjectHelper,
	conditions mcpv1alpha1.Conditions,
	status mcpv1alpha1.Status,
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

			if conditions.Type == state.ConditionReady {
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

func (r *BaseReconciler) UpdateMCPAddress(
	ctx context.Context,
	obj ObjectHelper,
	address string,
) error {
	key := client.ObjectKeyFromObject(obj)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := obj.DeepCopyObject().(*mcpv1alpha1.ManagedControlPlane)
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}

		latest.Status.Address = address

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
