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
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedPKIReconciler struct {
	controlplane.BaseReconciler
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

	if err := r.UpdateCondition(ctx, pkiObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReconciling,
			Status:  metav1.ConditionFalse,
			Reason:  controlplane.ReasonReconciling,
			Message: controlplane.MessageReconciling,
		},
		controlplane.Status{
			Ready:   false,
			Message: "reconciling",
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	resources := pki.Resources(pkiObj)

	if err := r.ensureResources(ctx, pkiObj, resources); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		_ = r.UpdateCondition(ctx, pkiObj,
			controlplane.Conditions{
				Type:    controlplane.ConditionWaitingForResource,
				Status:  metav1.ConditionFalse,
				Reason:  controlplane.ReasonWaitingForResources,
				Message: controlplane.MessageWaitingForResources,
			},
			controlplane.Status{
				Ready:   false,
				Message: "awaiting all ready",
			},
		)
		log.Info("requeueing reconcile for PKI controller")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.UpdateCondition(ctx, pkiObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  controlplane.ReasonAllResourcesReady,
			Message: controlplane.MessageAllResourcesReady,
		},
		controlplane.Status{
			Ready:   true,
			Message: "all ready",
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	// We wait for certManger resources ensured
	// so it shouldn't fail
	if err := r.ensureAdminConfig(ctx, req.Namespace); err != nil {
		return ctrl.Result{}, nil
	}

	log.Info("Finished reconciling ManagedPKI")

	return ctrl.Result{}, nil
}

func (r *ManagedPKIReconciler) ensureResources(
	ctx context.Context,
	pkiObj *mcpv1alpha1.ManagedPKI,
	resources []client.Object,
) error {
	for _, desired := range resources {
		r.Log.Info("Ensuring PKI resource", "name", desired.GetName(), "namespace", desired.GetNamespace(), "kind", desired.GetObjectKind())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, pkiObj, desired, r.Log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *certmanagerv1.Issuer:
				o.Spec = desired.(*certmanagerv1.Issuer).Spec
			case *certmanagerv1.Certificate:
				o.Spec = desired.(*certmanagerv1.Certificate).Spec
			}
			return nil
		})
		if err != nil {
			r.Log.Error(err, "failed to ensure PKI resource", "name", desired.GetName())
			return err
		}
	}

	return nil
}

func (r *ManagedPKIReconciler) checkResourcesReady(
	ctx context.Context,
	resources []client.Object,
) (bool, error) {
	allReady := true

	for _, desired := range resources {
		r.Log.Info("Checking PKI resource if ready", "name", desired.GetName(), "namespace", desired.GetNamespace(), "kind", desired.GetObjectKind())

		key := client.ObjectKey{Namespace: desired.GetNamespace(), Name: desired.GetName()}

		switch desired.(type) {

		case *certmanagerv1.Issuer:
			iss := &certmanagerv1.Issuer{}
			if err := r.Get(ctx, key, iss); err != nil {
				r.Log.Error(err, "failed to get Issuer", "name", key.Name)
				allReady = false
				continue
			}
			if !r.isIssuerReady(iss) {
				r.Log.Info("Issuer not ready", "name", iss.Name)
				allReady = false
			}

		case *certmanagerv1.Certificate:
			cert := &certmanagerv1.Certificate{}
			if err := r.Get(ctx, key, cert); err != nil {
				r.Log.Error(err, "failed to get Certificate", "name", key.Name)
				allReady = false
				continue
			}
			if !r.isCertificateReady(cert) {
				r.Log.Info("Certificate not ready", "name", cert.Name)
				allReady = false
			}
		}
	}

	return allReady, nil
}

// Still it doesnt look perfect
func (r *ManagedPKIReconciler) ensureAdminConfig(ctx context.Context, ns string) error {
	// hardcoded until we find a way to pass
	// API server addr
	serverURL := "https://172.30.0.250:6443"
	// get admin-client secret
	adminClient := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: "admin-client", Namespace: ns}, adminClient); err != nil {
		return client.IgnoreNotFound(err)
	}

	ca := adminClient.Data["ca.crt"]
	crt := adminClient.Data["tls.crt"]
	key := adminClient.Data["tls.key"]

	// build kubecfg
	cfg := clientcmdapi.NewConfig()
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:                   serverURL,
		CertificateAuthorityData: ca,
	}
	cfg.AuthInfos["local"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: crt,
		ClientKeyData:         key,
	}
	cfg.Contexts["local"] = &clientcmdapi.Context{Cluster: "local", AuthInfo: "local"}
	cfg.CurrentContext = "local"

	kubeconfigBytes, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}

	s := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-config",
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"config": kubeconfigBytes},
	}

	if err := r.Patch(ctx, s, client.Apply, client.FieldOwner("managedpki-controller")); err != nil {
		return err
	}

	return nil
}

func (r *ManagedPKIReconciler) isIssuerReady(iss *certmanagerv1.Issuer) bool {
	for _, cond := range iss.Status.Conditions {
		if cond.Type == certmanagerv1.IssuerConditionReady &&
			cond.Status == certmanagermeta.ConditionTrue {
			r.Log.Info("Issuer ready", "issuer", iss.Name)
			return true
		}
	}
	r.Log.Info("Issuer not ready", "issuer", iss.Name)
	return false
}

func (r *ManagedPKIReconciler) isCertificateReady(cert *certmanagerv1.Certificate) bool {
	for _, cond := range cert.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady &&
			cond.Status == certmanagermeta.ConditionTrue {
			r.Log.Info("Certificate ready", "issuer", cert.Name)
			return true
		}
	}
	r.Log.Info("Certificate not ready", "issuer", cert.Name)
	return false
}

func SetupManagedPKIReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedPKI{}).
		Owns(&certmanagerv1.Issuer{}).
		Owns(&certmanagerv1.Certificate{}).
		Owns(&corev1.ConfigMap{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedPKIReconciler{
			BaseReconciler: controlplane.BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedPKI"),
				Recorder: mgr.GetEventRecorderFor("managedpki"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
