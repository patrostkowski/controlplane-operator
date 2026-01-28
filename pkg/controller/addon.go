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
	"fmt"
	"maps"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: find a way to watch & act on managed resources
func (r *ManagedAddonsReconciler) reconcileAddons(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	secrets, err := r.ensureKonnectivityTLSData(ctx, cc)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}
	if len(secrets) == 0 {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}
	if err := r.apply(
		ctx,
		r.cp.Client,
		r.managedApplyOpts(),
		secrets...,
	); err != nil {
		return ctrl.Result{}, err
	}

	b := addons.NewAddonsBuilder(cc)
	if err := r.apply(ctx, r.cp.Client, r.managedApplyOpts(), b.Objects()...); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedAddonsReconciler) getControlPlaneClient(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (*ControlPlaneClient, error) {
	log := r.Log.WithValues("addons", cc.MCP.GetObjectMeta().GetNamespace())
	cp, err := r.newFromKubeconfigSecret(ctx, r.Client, cc)
	if err != nil {
		return nil, err
	}

	v, err := cp.Discovery.ServerVersion()
	if err != nil {
		log.Error(err, "discovery failed, will retry", "after", RequeueAfterFailure)
		return nil, err
	}

	log.Info("Obtained managed cluster config", "version", v)

	svc := &corev1.Service{}
	err = cp.Get(ctx, client.ObjectKey{
		Namespace: "default",
		Name:      "kubernetes",
	}, svc)
	if err != nil {
		log.Error(err, "managed client failed, will retry", "after", RequeueAfterFailure)
		return nil, err
	}

	log.Info("Got kubernetes service",
		"clusterIPs", svc.Spec.ClusterIPs,
		"type", svc.Spec.Type,
	)
	return cp, nil
}

// TODO: decide separate it from addons or
// bundle them together
func (r *ManagedAddonsReconciler) reconcileKubeletJoinResources(
	ctx context.Context,
	cc *cluster.ClusterContext,
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

	if err := r.apply(ctx, r.cp.Client, r.managedApplyOpts(), join.Objects()...); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedAddonsReconciler) getClusterCA(
	ctx context.Context,
	cc *cluster.ClusterContext,
) ([]byte, error) {
	mcp := cc.MCP
	caSecretName := cc.Names.SecretManagedCAName()

	sec := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: mcp.Namespace, Name: caSecretName}, sec); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	ca := sec.Data[cc.Keys.CACrt]
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
	mcp := cc.MCP

	sec := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: mcp.Namespace,
		Name:      addons.BootstrapTokenMgmtSecretName,
	}

	// basically check token already exists if exists
	// if true, return whatever was generated
	err := r.Get(ctx, key, sec)
	if err == nil {
		return addons.BootstrapToken{
			ID:     string(sec.Data[addons.BootstrapTokenIDKey]),
			Secret: string(sec.Data[addons.BootstrapTokenSecretKey]),
		}, nil
	}

	if !apierrors.IsNotFound(err) {
		return addons.BootstrapToken{}, err
	}

	tok, err := addons.NewBootstrapToken()
	if err != nil {
		return addons.BootstrapToken{}, err
	}

	sec = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addons.BootstrapTokenMgmtSecretName,
			Namespace: mcp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mcp, mcpv1alpha1.SchemeGroupVersion.WithKind(mcpv1alpha1.KindManagedControlPlane)),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			addons.BootstrapTokenIDKey:     []byte(tok.ID),
			addons.BootstrapTokenSecretKey: []byte(tok.Secret),
		},
	}

	if err := r.Create(ctx, sec); err != nil {
		return addons.BootstrapToken{}, err
	}

	return tok, nil
}

func (r *ManagedAddonsReconciler) ensureKonnectivityTLSData(
	ctx context.Context,
	cc *cluster.ClusterContext,
) ([]client.Object, error) {
	mcp := cc.MCP
	mgmtNS := mcp.Namespace

	agentTLSName := cc.Names.SecretKonnectivityAgentTLSName()
	caName := cc.Names.SecretKonnectivityCAName()

	agentTLS := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: agentTLSName, Namespace: mgmtNS}, agentTLS); err != nil {
		return nil, client.IgnoreNotFound(err) // if not found -> (nil, nil) -> requeue by caller
	}
	cc.Info("got agent TLS secret", "name", agentTLS.Name)

	caSec := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: caName, Namespace: mgmtNS}, caSec); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	cc.Info("got konnectivity CA secret", "name", caSec.Name)

	targetNS := cc.Names.KonnectivityAgentNamespace()

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

	cc.Info("copying konnectivity secrets into manageds cluster", "namespace", targetNS, "agentTLS", agentTLSName, "ca", caName)
	return []client.Object{workloadAgentTLS, workloadCA}, nil
}

type ControlPlaneClient struct {
	client.Client
	Discovery discovery.DiscoveryInterface
	REST      *rest.Config
}

func New(c client.Client, d discovery.DiscoveryInterface, r *rest.Config) *ControlPlaneClient {
	return &ControlPlaneClient{
		Client:    c,
		Discovery: d,
		REST:      r,
	}
}

func (r *ManagedAddonsReconciler) newFromKubeconfigSecret(
	ctx context.Context,
	mgmt client.Reader,
	cc *cluster.ClusterContext,
) (*ControlPlaneClient, error) {
	mcp := cc.MCP

	secName := mcp.Status.AdminKubeconfigSecretRef.Name
	if secName == "" {
		return nil, fmt.Errorf("%s/%s has empty status.adminKubeconfigSecretRef.name", mcp.Namespace, mcp.Name)
	}

	sec := &corev1.Secret{}
	if err := mgmt.Get(ctx, client.ObjectKey{Namespace: mcp.Namespace, Name: secName}, sec); err != nil {
		return nil, err
	}

	kubeconfigBytes, ok := sec.Data[cc.Keys.AdminConfigKubeconfigKey]
	if !ok || len(kubeconfigBytes) == 0 {
		return nil, fmt.Errorf("secret %s/%s missing key %q or it is empty", mcp.Namespace, secName, cc.Keys.AdminConfigKubeconfigKey)
	}

	restCfg, err := restConfigFromKubeconfigBytes(kubeconfigBytes)
	if err != nil {
		return nil, err
	}

	restCfg.QPS = 20
	restCfg.Burst = 40

	c, err := client.New(restCfg, client.Options{})
	if err != nil {
		return nil, err
	}

	d, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return New(c, d, restCfg), nil
}

func restConfigFromKubeconfigBytes(kubeconfig []byte) (*rest.Config, error) {
	// clientcmd handles clusters/users/contexts in kubeconfig properly.
	cfg, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}
	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	return restCfg, nil
}
