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
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/operator-template/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/operator-template/pkg/controlplane/pki"
	"github.com/patrostkowski/operator-template/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	allReady := true

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

		// Now fetch a fresh object from the API to inspect status
		switch desired.(type) {
		case *certmanagerv1.Issuer:
			iss := &certmanagerv1.Issuer{}
			if err := r.Get(ctx, client.ObjectKey{
				Namespace: res.GetNamespace(),
				Name:      res.GetName(),
			}, iss); err != nil {
				allReady = false
				log.Error(err, "failed to get Issuer for readiness", "name", res.GetName())
				continue
			}
			log.Info("Issuer status", "name", iss.Name, "conditions", iss.Status.Conditions)
			if !isIssuerReady(iss) {
				allReady = false
			}

		case *certmanagerv1.Certificate:
			cert := &certmanagerv1.Certificate{}
			if err := r.Get(ctx, client.ObjectKey{
				Namespace: res.GetNamespace(),
				Name:      res.GetName(),
			}, cert); err != nil {
				allReady = false
				log.Error(err, "failed to get Certificate for readiness", "name", res.GetName())
				continue
			}
			log.Info("Certificate status", "name", cert.Name, "conditions", cert.Status.Conditions)
			if !isCertificateReady(cert) {
				allReady = false
			}
		}
	}

	var condStatus metav1.ConditionStatus
	var reason, msg string

	if allReady {
		log.Info("All PKI resources Ready")
		condStatus = metav1.ConditionTrue
		reason = "AllResourcesReady"
		msg = "All PKI resources are Ready"
	} else {
		log.Info("PKI resources not yet Ready")
		condStatus = metav1.ConditionFalse
		reason = "WaitingForResources"
		msg = "Waiting for PKI resources to become Ready"
	}

	cond := metav1.Condition{
		Type:               "Ready",
		Status:             condStatus,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: pkiObj.Generation,
	}

	changed := apimeta.SetStatusCondition(&pkiObj.Status.Conditions, cond)
	if changed {
		if err := r.Status().Update(ctx, pkiObj); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Finished reconciling managedpki")
	return ctrl.Result{}, nil
}

func isIssuerReady(iss *certmanagerv1.Issuer) bool {
	for _, cond := range iss.Status.Conditions {
		if cond.Type == certmanagerv1.IssuerConditionReady &&
			cond.Status == certmanagermeta.ConditionTrue {
			return true
		}
	}
	return false
}

func isCertificateReady(cert *certmanagerv1.Certificate) bool {
	for _, cond := range cert.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady &&
			cond.Status == certmanagermeta.ConditionTrue {
			return true
		}
	}
	return false
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
