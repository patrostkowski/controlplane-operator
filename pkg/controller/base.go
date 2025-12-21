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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func (r *BaseReconciler) UpdateMCPAddress(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	address string,
) error {
	key := client.ObjectKeyFromObject(mcp)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := mcp.DeepCopyObject().(*mcpv1alpha1.ManagedControlPlane)
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}

		latest.Status.Address = address

		return r.Status().Update(ctx, latest)
	})
}

func (r *BaseReconciler) UpdateMCPAdminSecretRef(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	secretName string,
	namespace string,
) error {
	key := client.ObjectKeyFromObject(mcp)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := mcp.DeepCopyObject().(*mcpv1alpha1.ManagedControlPlane)
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}

		latest.Status.AdminKubeconfigSecretRef.Name = secretName

		return r.Status().Update(ctx, latest)
	})
}

// func (r *BaseReconciler) IsDeploymentReady(dep *appsv1.Deployment) bool {
// 	desired := *dep.Spec.Replicas
// 	if dep.Status.ReadyReplicas < desired {
// 		return false
// 	}
// 	if dep.Generation > dep.Status.ObservedGeneration {
// 		return false
// 	}
// 	if dep.Status.UpdatedReplicas < desired {
// 		return false
// 	}
// 	if dep.Status.ReadyReplicas < desired || dep.Status.AvailableReplicas < desired {
// 		return false
// 	}
// 	return true
// }

// // UpdateCondition is kept for compatibility with your existing call sites.
// // It is now idempotent and only patches status when something actually changed.
// func (r *BaseReconciler) UpdateCondition(
// 	ctx context.Context,
// 	mcp *mcpv1alpha1.ManagedControlPlane,
// 	cond mcpv1alpha1.Conditions,
// 	st mcpv1alpha1.Status,
// ) error {
// 	_, err := r.SetConditionIfChanged(ctx, mcp, cond, st)
// 	return err
// }

// // SetConditionIfChanged patches status only if the condition or inline status fields differ.
// // Returns changed=true if it performed a status patch.
// func (r *BaseReconciler) SetConditionIfChanged(
// 	ctx context.Context,
// 	mcp *mcpv1alpha1.ManagedControlPlane,
// 	cond mcpv1alpha1.Conditions,
// 	st mcpv1alpha1.Status,
// ) (bool, error) {

// 	before := mcp.DeepCopy()

// 	// Build the new metav1.Condition
// 	newC := metav1.Condition{
// 		Type:               string(cond.Type),
// 		Status:             cond.Status,
// 		Reason:             string(cond.Reason),
// 		Message:            string(cond.Message),
// 		ObservedGeneration: mcp.GetGeneration(),
// 	}

// 	// Find existing condition of the same type
// 	oldC := meta.FindStatusCondition(mcp.Status.Conditions, newC.Type)

// 	condChanged := false
// 	if oldC == nil {
// 		condChanged = true
// 	} else if oldC.Status != newC.Status ||
// 		oldC.Reason != newC.Reason ||
// 		oldC.Message != newC.Message ||
// 		oldC.ObservedGeneration != newC.ObservedGeneration {
// 		condChanged = true
// 	}

// 	if condChanged {
// 		meta.SetStatusCondition(&mcp.Status.Conditions, newC)
// 	}

// 	// Inline status fields (your embedded Status struct)
// 	statusChanged := false
// 	if mcp.Status.Ready != st.Ready {
// 		mcp.Status.Ready = st.Ready
// 		statusChanged = true
// 	}
// 	if mcp.Status.Message != st.Message {
// 		mcp.Status.Message = st.Message
// 		statusChanged = true
// 	}

// 	// Nothing changed -> don’t write status
// 	if !condChanged && !statusChanged {
// 		return false, nil
// 	}

// 	// Patch status only (subresource)
// 	if err := r.Status().Patch(ctx, mcp, client.MergeFrom(before)); err != nil {
// 		// If the object disappeared between get and patch, treat as done.
// 		if apierrors.IsNotFound(err) {
// 			return false, nil
// 		}
// 		return false, err
// 	}
// 	return true, nil
// }
