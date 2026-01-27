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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/controllermanager"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ControllerManagerComponent struct {
	r *ManagedControlPlaneReconciler
}

func (c *ControllerManagerComponent) Name() string {
	return "controller-manager"
}

func (c *ControllerManagerComponent) Reconcile(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	return c.r.reconcileControllerManager(ctx, mcp)
}

func (c *ControllerManagerComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageControllerManagerWaiting
}

func (c *ControllerManagerComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageControllerManagerFailed
}

func (r *ManagedControlPlaneReconciler) reconcileControllerManager(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("controllermanager", mcp.Namespace)

	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(mcp),
		controllermanager.Resources(mcp)...,
	); err != nil {
		log.Error(err, "failed to apply controller-manager resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
