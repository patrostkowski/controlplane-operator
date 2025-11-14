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
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	if err := r.ensureResources(ctx, pkiObj, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updatePKIReadyCondition(ctx, pkiObj, allReady); err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for PKI controller")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Finished reconciling ManagedPKI")

	return reconcile.Result{}, nil
}

func (r *ManagedPKIReconciler) ensureResources(
	ctx context.Context,
	pkiObj *mcpv1alpha1.ManagedPKI,
	resources []client.Object,
	log logr.Logger,
) error {

	for _, desired := range resources {
		log.Info("Ensuring PKI resource", "name", desired.GetName(), "namespace", desired.GetNamespace())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, pkiObj, desired, log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *certmanagerv1.Issuer:
				o.Spec = desired.(*certmanagerv1.Issuer).Spec
			case *certmanagerv1.Certificate:
				o.Spec = desired.(*certmanagerv1.Certificate).Spec
			}
			return nil
		})
		if err != nil {
			log.Error(err, "failed to ensure PKI resource", "name", desired.GetName())
			return err
		}
	}

	return nil
}

func (r *ManagedPKIReconciler) checkResourcesReady(
	ctx context.Context,
	resources []client.Object,
	log logr.Logger,
) (bool, error) {

	allReady := true

	for _, desired := range resources {
		key := client.ObjectKey{Namespace: desired.GetNamespace(), Name: desired.GetName()}

		switch desired.(type) {

		case *certmanagerv1.Issuer:
			iss := &certmanagerv1.Issuer{}
			if err := r.Get(ctx, key, iss); err != nil {
				log.Error(err, "failed to get Issuer", "name", key.Name)
				allReady = false
				continue
			}
			if !isIssuerReady(iss) {
				log.Info("Issuer not ready", "name", iss.Name)
				allReady = false
			}

		case *certmanagerv1.Certificate:
			cert := &certmanagerv1.Certificate{}
			if err := r.Get(ctx, key, cert); err != nil {
				log.Error(err, "failed to get Certificate", "name", key.Name)
				allReady = false
				continue
			}
			if !isCertificateReady(cert) {
				log.Info("Certificate not ready", "name", cert.Name)
				allReady = false
			}
		}
	}

	return allReady, nil
}

func (r *ManagedPKIReconciler) updatePKIReadyCondition(
	ctx context.Context,
	pkiObj *mcpv1alpha1.ManagedPKI,
	allReady bool,
) error {

	var cond metav1.Condition

	if allReady {
		cond = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "AllResourcesReady",
			Message:            "All PKI resources are Ready",
			ObservedGeneration: pkiObj.Generation,
		}
	} else {
		cond = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "WaitingForResources",
			Message:            "Waiting for PKI resources to become Ready",
			ObservedGeneration: pkiObj.Generation,
		}
	}

	if apimeta.SetStatusCondition(&pkiObj.Status.Conditions, cond) {
		return r.Status().Update(ctx, pkiObj)
	}

	return nil
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
