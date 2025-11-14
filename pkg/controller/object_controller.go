// Copyright 2025 controlplane.patrostkowski.dev
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

	objectv1alpha1 "github.com/patrostkowski/operator-template/pkg/apis/example.com/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type VMReconciler struct {
	client.Client
}

func (r *VMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	object := &objectv1alpha1.Object{}
	if err := r.Get(ctx, req.NamespacedName, object); err != nil {
		// object deleted or not found
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("reconciling", "name", object.Name)

	// update status if needed
	// _ = r.Status().Update(ctx, vm)

	return ctrl.Result{}, nil
}

func SetupVMController(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&objectv1alpha1.Object{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&VMReconciler{Client: mgr.GetClient()})
}
