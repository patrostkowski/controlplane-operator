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
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/rbac"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcileKubeletJoinResources(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	c *controlplane.ControlPlaneClient,
) (ctrl.Result, error) {

	log := r.Log.WithValues("rbac", mcp.Namespace)

	log.Info("Reconciling kubelet join resources")

	tok, err := r.ensureBootstrapToken(ctx, mcp)
	if err != nil {
		return ctrl.Result{}, err
	}

	resources := rbac.BootstrapKubeletJoinResources(tok)

	if err := c.ApplySSA(ctx, "mcp", resources...); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling kubelet join resources",
		"tokenID", tok.ID,
	)

	return ctrl.Result{}, nil
}

// todo: think how to rotate the token
func (r *ManagedControlPlaneReconciler) ensureBootstrapToken(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (rbac.BootstrapToken, error) {

	sec := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: mcp.Namespace,
		Name:      rbac.BootstrapTokenMgmtSecretName,
	}

	err := r.Get(ctx, key, sec)
	if err == nil {
		return rbac.BootstrapToken{
			ID:     string(sec.Data[rbac.BootstrapTokenIDKey]),
			Secret: string(sec.Data[rbac.BootstrapTokenSecretKey]),
		}, nil
	}

	if !apierrors.IsNotFound(err) {
		return rbac.BootstrapToken{}, err
	}

	tok, err := rbac.NewBootstrapToken()
	if err != nil {
		return rbac.BootstrapToken{}, err
	}

	sec = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbac.BootstrapTokenMgmtSecretName,
			Namespace: mcp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mcp, mcpv1alpha1.SchemeGroupVersion.WithKind("ManagedControlPlane")),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			rbac.BootstrapTokenIDKey:     []byte(tok.ID),
			rbac.BootstrapTokenSecretKey: []byte(tok.Secret),
		},
	}

	if err := r.Create(ctx, sec); err != nil {
		return rbac.BootstrapToken{}, err
	}

	return tok, nil
}
