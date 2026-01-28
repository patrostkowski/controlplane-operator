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
	"github.com/patrostkowski/controlplane-operator/pkg/controller/state"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AdminConfigComponent struct {
	r *ManagedControlPlaneReconciler
}

func (c *AdminConfigComponent) Name() string {
	return "admin-kubeconfig"
}

func (c *AdminConfigComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileAdminConfig(ctx, cc)
}

func (c *AdminConfigComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageAdminKubeconfigWaiting
}

func (c *AdminConfigComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageAdminKubeconfigFailed
}

func (r *ManagedControlPlaneReconciler) reconcileAdminConfig(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	mcp := cc.MCP
	ns := mcp.Namespace

	if mcp.Status.Address == "" {
		r.Log.Info("API address not set yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	serverURL := "https://" + mcp.Status.Address + ":6443"

	adminSecretName := cc.Names.SecretAdminClientName()

	// get admin-client secret
	adminClient := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: adminSecretName, Namespace: ns}, adminClient); err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, client.IgnoreNotFound(err)
	}

	ca := adminClient.Data[cc.Keys.CACrt]
	crt := adminClient.Data[cc.Keys.TLSCrt]
	key := adminClient.Data[cc.Keys.TLSKey]

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

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cc.Names.AdminKubeconfigSecretName(),
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			cc.Keys.AdminKubeconfigKey: kubeconfigBytes,
		},
	}

	err = r.apply(ctx, r.Client, r.applyOpts(mcp), s)
	if err != nil {
		r.Log.Error(err, "failed to apply Admin config secret", "name", s.GetName())
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	// updating MCP status with sercet ref
	if err = r.updateMCPAdminSecretRef(ctx, mcp, cc.Names.AdminKubeconfigSecretName()); err != nil {
		r.Log.Error(err, "failed to update admin config secret ref")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
