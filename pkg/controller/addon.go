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
	"maps"

	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedAddonsReconciler) reconcileAddons(
	ctx context.Context,
	cc *cluster.ClusterContext,
	cl client.Client,
	obj client.Object,
) (ctrl.Result, error) {
	b := addons.NewAddonsBuilder(cc)

	secrets, err := r.ensureKonnectivityTLSData(ctx, cc)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}
	if len(secrets) == 0 {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if err := r.apply(
		ctx,
		cl,
		r.applyOpts(obj),
		secrets...,
	); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.apply(ctx, cl, r.applyOpts(obj), b.Objects()...); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedAddonsReconciler) reconcileKubeletJoinResources(
	ctx context.Context,
	cc *cluster.ClusterContext,
	cl client.Client,
	obj client.Object,
) (ctrl.Result, error) {
	tok, err := r.ensureBootstrapToken(ctx, cc)
	if err != nil {
		return ctrl.Result{}, err
	}

	caPEM, err := r.getClusterCA(ctx, cc)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(caPEM) == 0 {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	join := addons.NewKubeletJoinBuilder(cc, tok, caPEM)

	if err := r.apply(ctx, cl, r.applyOpts(obj), join.Objects()...); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedAddonsReconciler) getClusterCA(
	ctx context.Context,
	cc *cluster.ClusterContext,
) ([]byte, error) {
	ns := cc.Namespace()

	caSecretName := cc.PKI().Certificate().ManagedCA() // formerly cc.Names.SecretManagedCAName()

	sec := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: caSecretName}, sec); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// cert-manager standard key
	ca := sec.Data["ca.crt"]
	if len(ca) == 0 {
		return nil, nil // not ready yet
	}
	return ca, nil
}

// todo: think how to rotate the token
func (r *ManagedAddonsReconciler) ensureBootstrapToken(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (addons.BootstrapToken, error) {
	ns := cc.Namespace()
	bootstrapTokenSecretName := cc.ManagedAddons().BootstrapTokenMgmtSecretName()

	sec := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: ns,
		Name:      bootstrapTokenSecretName,
	}

	// If token already exists, return it.
	err := r.Get(ctx, key, sec)
	if err == nil {
		return addons.BootstrapToken{
			ID:     string(sec.Data[bootstrapapi.BootstrapTokenIDKey]),
			Secret: string(sec.Data[bootstrapapi.BootstrapTokenSecretKey]),
		}, nil
	}
	if !apierrors.IsNotFound(err) {
		return addons.BootstrapToken{}, err
	}

	// Otherwise generate a new one and persist it.
	tok, err := addons.NewBootstrapToken()
	if err != nil {
		return addons.BootstrapToken{}, err
	}

	sec = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapTokenSecretName,
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			bootstrapapi.BootstrapTokenIDKey:     []byte(tok.ID),
			bootstrapapi.BootstrapTokenSecretKey: []byte(tok.Secret),
		},
	}

	// Use SSA apply so it's consistent + sets controller ref via applyOpts.
	if err := r.apply(ctx, r.Client, r.applyOpts(cc.Owner()), sec); err != nil {
		return addons.BootstrapToken{}, err
	}

	return tok, nil
}

func (r *ManagedAddonsReconciler) ensureKonnectivityTLSData(
	ctx context.Context,
	cc *cluster.ClusterContext,
) ([]client.Object, error) {
	ns := cc.Namespace()

	agentTLSName := cc.Konnectivity().AgentTLSSecret()
	caName := cc.Konnectivity().CASecret()

	agentTLS := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: agentTLSName, Namespace: ns}, agentTLS); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	r.Log.Info("got agent TLS secret", "name", agentTLS.Name)

	caSec := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: caName, Namespace: ns}, caSec); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	r.Log.Info("got konnectivity CA secret", "name", caSec.Name)

	targetNS := cc.ManagedAddons().KonnectivityAgentNamespace()

	workloadAgentTLS := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentTLSName,
			Namespace: targetNS,
		},
		Type: agentTLS.Type,
		Data: maps.Clone(agentTLS.Data),
	}

	workloadCA := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caName,
			Namespace: targetNS,
		},
		Type: caSec.Type,
		Data: maps.Clone(caSec.Data),
	}

	r.Log.Info("copying konnectivity secrets into managed cluster",
		"namespace", targetNS,
		"agentTLS", agentTLSName,
		"ca", caName,
	)
	return []client.Object{workloadAgentTLS, workloadCA}, nil
}
