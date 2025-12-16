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
	"bytes"
	"context"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcilePKI(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("pki", mcp.GetObjectMeta().GetNamespace())

	log.Info("Reconciling PKI")

	resources := pki.Resources(mcp)

	if err := r.ensurePKIResources(ctx, mcp, resources); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkPKIResourcesReady(ctx, resources)
	if err != nil {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for PKI")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// We wait for certManger resources ensured
	// so it shouldn't fail
	if err := r.ensureAdminConfig(ctx, mcp, mcp.GetObjectMeta().GetNamespace()); err != nil {
		return ctrl.Result{}, nil
	}

	log.Info("Finished reconciling PKI")

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) ensurePKIResources(
	ctx context.Context,
	pkiObj *mcpv1alpha1.ManagedControlPlane,
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

func (r *ManagedControlPlaneReconciler) checkPKIResourcesReady(
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
func (r *ManagedControlPlaneReconciler) ensureAdminConfig(
	ctx context.Context,
	pkiObj *mcpv1alpha1.ManagedControlPlane,
	ns string,
) error {
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

	if len(ca) == 0 || len(crt) == 0 || len(key) == 0 {
		r.Log.Info("admin config secret not ready yet")
		return nil
	}

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
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-config",
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
	}

	err = utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, pkiObj, s, r.Log, func(obj client.Object) error {
		sec := obj.(*corev1.Secret)
		if sec.Data == nil {
			sec.Data = map[string][]byte{}
		}
		if bytes.Equal(sec.Data["config"], kubeconfigBytes) {
			return nil
		}
		sec.Data["config"] = kubeconfigBytes
		return nil
	})
	if err != nil {
		r.Log.Error(err, "failed to ensure Admin config secret", "name", s.GetName())
		return err
	}

	return nil
}

func (r *ManagedControlPlaneReconciler) isIssuerReady(iss *certmanagerv1.Issuer) bool {
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

func (r *ManagedControlPlaneReconciler) isCertificateReady(cert *certmanagerv1.Certificate) bool {
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
