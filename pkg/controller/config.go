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
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AdminConfigComponent reconciles the admin kubeconfig secret for the control plane.
type AdminConfigComponent struct {
	r *ManagedControlPlaneReconciler
}

// Name returns the name of the admin kubeconfig component.
func (c *AdminConfigComponent) Name() string {
	return "admin-kubeconfig"
}

// Reconcile reconciles the admin kubeconfig secret.
func (c *AdminConfigComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileAdminConfig(ctx, cc)
}

// WaitingMessage returns the waiting message for the admin kubeconfig.
func (c *AdminConfigComponent) WaitingMessage() mcpv1alpha1.Message {
	return mcpv1alpha1.MessageAdminKubeconfigWaiting
}

// FailedMessage returns the failed message for the admin kubeconfig.
func (c *AdminConfigComponent) FailedMessage() mcpv1alpha1.Message {
	return mcpv1alpha1.MessageAdminKubeconfigFailed
}

// reconcileAdminConfig reconciles the admin kubeconfig secret for the managed control plane.
func (r *ManagedControlPlaneReconciler) reconcileAdminConfig(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	mcp := cc.MCP()
	ns := cc.Namespace()

	status := cc.GetManagedControlPlaneStatus()
	if status.Address == "" {
		r.Log.Info("API address not set yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	serverURL := "https://" + status.Address + ":6443"

	adminSecretName := cc.Admin().ClientSecret()

	// get admin-client secret
	adminClient := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: adminSecretName, Namespace: ns}, adminClient); err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, client.IgnoreNotFound(err)
	}

	// cert-manager standard keys
	ca := adminClient.Data["ca.crt"]
	crt := adminClient.Data["tls.crt"]
	key := adminClient.Data["tls.key"]

	if len(ca) == 0 || len(crt) == 0 || len(key) == 0 {
		r.Log.Info("admin config secret not ready yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// build kubeconfig (data-embedded)
	cfg := utils.BuildKubeconfigWithCertData(serverURL, "local", ca, crt, key)

	kubeconfigBytes, err := clientcmd.Write(*cfg)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	kubeconfigSecretName := cc.Admin().KubeconfigSecret()
	kubeconfigKey := cc.Admin().KubeconfigDataKey()

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeconfigSecretName,
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			kubeconfigKey: kubeconfigBytes,
		},
	}

	if err := r.apply(ctx, r.Client, r.applyOpts(cc.Owner()), s); err != nil {
		r.Log.Error(err, "failed to apply Admin config secret", "name", s.GetName())
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	// update MCP status with secret ref
	if err := r.updateMCPAdminSecretRef(ctx, mcp, kubeconfigSecretName); err != nil {
		r.Log.Error(err, "failed to update admin config secret ref")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
