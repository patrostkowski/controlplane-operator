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

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) setStatus(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	condType mcpv1alpha1.Condition,
	condStatus metav1.ConditionStatus,
	reason mcpv1alpha1.Reason,
	message mcpv1alpha1.Message,
	ready bool,
) error {
	before := mcp.DeepCopy()

	mcp.Status.Ready = ready
	mcp.Status.Message = string(message)

	newC := metav1.Condition{
		Type:               string(condType),
		Status:             condStatus,
		Reason:             string(reason),
		Message:            string(message),
		ObservedGeneration: mcp.GetGeneration(),
	}
	meta.SetStatusCondition(&mcp.Status.Conditions, newC)

	oldC := meta.FindStatusCondition(before.Status.Conditions, newC.Type)
	sameCond := oldC != nil &&
		oldC.Status == newC.Status &&
		oldC.Reason == newC.Reason &&
		oldC.Message == newC.Message &&
		oldC.ObservedGeneration == newC.ObservedGeneration

	if before.Status.Ready == mcp.Status.Ready &&
		before.Status.Message == mcp.Status.Message &&
		sameCond {
		return nil
	}

	return r.Status().Patch(ctx, mcp, client.MergeFrom(before))
}

func (r *ManagedControlPlaneReconciler) statusWaiting(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane, msg mcpv1alpha1.Message) error {
	return r.setStatus(ctx, mcp,
		state.ConditionReady,
		metav1.ConditionFalse,
		state.ReasonWaitingForResources,
		msg,
		false,
	)
}

func (r *ManagedControlPlaneReconciler) statusFailed(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane, msg mcpv1alpha1.Message) error {
	return r.setStatus(ctx, mcp,
		state.ConditionReady,
		metav1.ConditionFalse,
		state.ReasonComponentFailed,
		msg,
		false,
	)
}

func (r *ManagedControlPlaneReconciler) statusReady(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) error {
	return r.setStatus(ctx, mcp,
		state.ConditionReady,
		metav1.ConditionTrue,
		state.ReasonAllResourcesReady,
		state.MessageAllResourcesReady,
		true,
	)
}
