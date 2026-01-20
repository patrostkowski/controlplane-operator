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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ManagedControlPlaneReconciler) reconcilePKI(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("pki", mcp.Namespace)

	if err := apply(
		ctx,
		r.Client,
		r.Scheme,
		r.applyOpts(mcp),
		pki.Resources(mcp)...,
	); err != nil {
		log.Error(err, "failed to apply PKI resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
