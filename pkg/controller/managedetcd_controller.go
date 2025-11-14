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

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/operator-template/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedETCDReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *ManagedETCDReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedetcd", req.NamespacedName)
	log.Info("Reconciling managedetcd")
	log.Info("Finished reconciling managedetcd")
	return ctrl.Result{}, nil
}

func SetupManagedETCDReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedETCD{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedETCDReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controller").WithName("ManagedETCD"),
			Recorder: mgr.GetEventRecorderFor("managedetcd"),
			Scheme:   mgr.GetScheme(),
		})
}
