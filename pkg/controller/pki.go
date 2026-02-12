// Copyright 2025 mcpv1alpha1.patrostkowski.dev
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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	ctrl "sigs.k8s.io/controller-runtime"
)

// PKIComponent reconciles the Public Key Infrastructure (PKI) resources.
type PKIComponent struct {
	r *ManagedControlPlaneReconciler
}

// Name returns the name of the PKI component.
func (c *PKIComponent) Name() string {
	return "pki"
}

// Reconcile reconciles the PKI resources.
func (c *PKIComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcilePKI(ctx, cc)
}

// WaitingMessage returns the waiting message for the PKI component.
func (c *PKIComponent) WaitingMessage() mcpv1alpha1.Message {
	return mcpv1alpha1.MessagePKIWaiting
}

// FailedMessage returns the failed message for the PKI component.
func (c *PKIComponent) FailedMessage() mcpv1alpha1.Message {
	return mcpv1alpha1.MessagePKIFailed
}

// reconcilePKI reconciles the PKI resources for the control plane.
func (r *ManagedControlPlaneReconciler) reconcilePKI(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	log := r.Log.WithValues(cc.Namespace(), cc.Name())

	p := pki.NewBuilder(cc)
	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(cc.Owner()),
		p.Objects()...,
	); err != nil {
		log.Error(err, "failed to apply PKI resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
