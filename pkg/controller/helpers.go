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
	"fmt"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// setStatus updates the status of the ManagedControlPlane object with the given condition and message.
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

	mcp.Status.Ready = ptr.To(ready)
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

// statusWaiting sets the ManagedControlPlane status to waiting with a specific message.
func (r *ManagedControlPlaneReconciler) statusWaiting(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane, msg mcpv1alpha1.Message) error {
	return r.setStatus(ctx, mcp,
		mcpv1alpha1.ConditionReady,
		metav1.ConditionFalse,
		mcpv1alpha1.ReasonWaitingForResources,
		msg,
		false,
	)
}

// statusFailed sets the ManagedControlPlane status to failed with a specific message.
func (r *ManagedControlPlaneReconciler) statusFailed(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane, msg mcpv1alpha1.Message) error {
	return r.setStatus(ctx, mcp,
		mcpv1alpha1.ConditionReady,
		metav1.ConditionFalse,
		mcpv1alpha1.ReasonComponentFailed,
		msg,
		false,
	)
}

// statusReady sets the ManagedControlPlane status to ready.
func (r *ManagedControlPlaneReconciler) statusReady(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) error {
	return r.setStatus(ctx, mcp,
		mcpv1alpha1.ConditionReady,
		metav1.ConditionTrue,
		mcpv1alpha1.ReasonAllResourcesReady,
		mcpv1alpha1.MessageAllResourcesReady,
		true,
	)
}

// applyOpts returns the ApplyOptions with field owner and owner reference set for a given owner object.
func (r *BaseReconciler) applyOpts(owner client.Object) ApplyOptions {
	return ApplyOptions{
		FieldOwner:  fieldOwner,
		Force:       true,
		Owner:       owner,
		SetOwnerRef: true,
	}
}

// ApplyOptions defines options for server-side apply operations.
type ApplyOptions struct {
	FieldOwner  string
	Force       bool
	Owner       client.Object
	SetOwnerRef bool
	Log         logr.Logger
}

// apply performs a server-side apply operation for a list of Kubernetes objects.
func (r *BaseReconciler) apply(ctx context.Context, c client.Client, opts ApplyOptions, objs ...client.Object) error {
	if r.Scheme == nil {
		return fmt.Errorf("apply: scheme is nil")
	}
	if opts.SetOwnerRef && opts.Owner == nil {
		return fmt.Errorf("apply: SetOwnerRef=true but Owner is nil")
	}

	for _, obj := range objs {
		if opts.SetOwnerRef && opts.Owner != nil {
			if err := controllerutil.SetControllerReference(opts.Owner, obj, r.Scheme); err != nil {
				return fmt.Errorf("set owner ref for %T/%s: %w", obj, obj.GetName(), err)
			}
		}

		gvk, err := apiutil.GVKForObject(obj, r.Scheme)
		if err != nil {
			return fmt.Errorf("resolve gvk for %T/%s: %w", obj, obj.GetName(), err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)

		if opts.Log.GetSink() != nil {
			opts.Log.V(5).Info("Applying",
				"gvk", gvk.String(),
				"ns", obj.GetNamespace(),
				"name", obj.GetName(),
				"fieldOwner", opts.FieldOwner,
			)
		}

		patchOpts := []client.PatchOption{client.FieldOwner(opts.FieldOwner)}
		if opts.Force {
			patchOpts = append(patchOpts, client.ForceOwnership)
		}

		if err := c.Patch(ctx, obj, client.Apply, patchOpts...); err != nil {
			return fmt.Errorf("apply %s %s/%s: %w", gvk.Kind, obj.GetNamespace(), obj.GetName(), err)
		}

	}
	return nil
}

// updateMCPAddress updates the address field in the ManagedControlPlane's status.
func (r *ManagedControlPlaneReconciler) updateMCPAddress(
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

// updateMCPAdminSecretRef updates the admin kubeconfig secret reference in the ManagedControlPlane's status.
func (r *ManagedControlPlaneReconciler) updateMCPAdminSecretRef(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	secretName string,
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
