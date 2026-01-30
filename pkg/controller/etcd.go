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
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/controller/state"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/etcd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ETCDComponent reconciles the etcd cluster.
type ETCDComponent struct {
	r *ManagedControlPlaneReconciler
}

// Name returns the name of the etcd component.
func (c *ETCDComponent) Name() string {
	return "etcd"
}

// Reconcile reconciles the etcd cluster.
func (c *ETCDComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileETCD(ctx, cc)
}

// WaitingMessage returns the waiting message for the etcd component.
func (c *ETCDComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageETCDWaiting
}

// FailedMessage returns the failed message for the etcd component.
func (c *ETCDComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageETCDFailed
}

// reconcileETCD reconciles the etcd cluster resources.
func (r *ManagedControlPlaneReconciler) reconcileETCD(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	log := r.Log.WithValues("etcd", cc.Namespace())

	e := etcd.NewBuilder(cc)
	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(cc.Owner()),
		e.Objects()...,
	); err != nil {
		log.Error(err, "failed to apply etcd resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
