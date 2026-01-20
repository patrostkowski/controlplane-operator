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
