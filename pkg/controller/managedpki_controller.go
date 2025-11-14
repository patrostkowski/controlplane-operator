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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/operator-template/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/operator-template/pkg/controlplane/pki"
	"github.com/patrostkowski/operator-template/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedPKIReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *ManagedPKIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedpki", req.NamespacedName)

	pkiObj := &mcpv1alpha1.ManagedPKI{}
	if err := r.Get(ctx, req.NamespacedName, pkiObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedPKI")

	resources := pki.Resources(pkiObj)

	for _, res := range resources {
		desired := res

		log.Info("reconciling PKI resource", "name", res.GetName(), "namespace", res.GetNamespace())

		err := utils.EnsureOwned(ctx, r.Client, r.Scheme, pkiObj, res, log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *certmanagerv1.Issuer:
				d := desired.(*certmanagerv1.Issuer)
				o.Spec = d.Spec
			case *certmanagerv1.Certificate:
				d := desired.(*certmanagerv1.Certificate)
				o.Spec = d.Spec
			}
			return nil
		})
		if err != nil {
			log.Error(err, "failed to reconcile PKI resource", "name", res.GetName(), "namespace", res.GetNamespace())
			return ctrl.Result{}, err
		}
	}

	if err := r.Status().Update(ctx, pkiObj); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling managedpki")
	return ctrl.Result{}, nil
}

func SetupManagedPKIReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedPKI{}).
		Owns(&certmanagerv1.Issuer{}).
		Owns(&certmanagerv1.Certificate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedPKIReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controller").WithName("ManagedPKI"),
			Recorder: mgr.GetEventRecorderFor("managedpki"),
			Scheme:   mgr.GetScheme(),
		})
}
