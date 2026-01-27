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
	"sync"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controller/state"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (r *BaseReconciler) applyOpts(owner client.Object) ApplyOptions {
	return ApplyOptions{
		FieldOwner:  fieldOwner,
		Force:       true,
		Owner:       owner,
		SetOwnerRef: true,
	}
}

func (r *BaseReconciler) managedApplyOpts() ApplyOptions {
	return ApplyOptions{
		FieldOwner:  fieldOwner, // or "controlplane-operator-managed" if you want to separate
		Force:       true,
		Owner:       nil,
		SetOwnerRef: false,
	}
}

var (
	managedSchemeOnce sync.Once
	managedSchemeInst *runtime.Scheme
)

func (r *BaseReconciler) managedScheme() *runtime.Scheme {
	managedSchemeOnce.Do(func() {
		s := runtime.NewScheme()
		_ = corev1.AddToScheme(s)
		_ = appsv1.AddToScheme(s)
		_ = rbacv1.AddToScheme(s)
		_ = storagev1.AddToScheme(s)
		managedSchemeInst = s
	})
	return managedSchemeInst
}

type ApplyOptions struct {
	FieldOwner  string
	Force       bool
	Owner       client.Object
	SetOwnerRef bool
	Log         logr.Logger
}

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
