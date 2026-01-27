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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
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
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	if err := r.apply(
		ctx,
		r.cp.Client,
		r.managedApplyOpts(),
		addons.Resources(mcp)...,
	); err != nil {
		return ctrl.Result{}, err
	}

	secrets, err := r.ensureKonnectivityTLSData(ctx, mcp)
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

	if err := r.apply(
		ctx,
		r.cp.Client,
		r.managedApplyOpts(),
		addons.KonnectivityAgentResources(mcp)...,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedAddonsReconciler) getControlPlaneClient(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (*ControlPlaneClient, error) {
	log := r.Log.WithValues("addons", mcp.GetObjectMeta().GetNamespace())
	cp, err := r.newFromKubeconfigSecret(ctx, r.Client, mcp.Namespace)
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
	return cp, err
}

// TODO: decide separate it from addons or
// bundle them together
func (r *ManagedAddonsReconciler) reconcileKubeletJoinResources(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	tok, err := r.ensureBootstrapToken(ctx, mcp)
	if err != nil {
		return ctrl.Result{}, err
	}

	caPEM, err := r.getClusterCA(ctx, mcp)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(caPEM) == 0 {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	resources := addons.BootstrapKubeletJoinResources(mcp, tok, caPEM)

	if err := r.apply(
		ctx,
		r.cp.Client,
		r.managedApplyOpts(),
		resources...,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedAddonsReconciler) getClusterCA(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) ([]byte, error) {
	caSecretName := pki.New(mcp).APIServer().ClientCA.SecretName

	sec := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: mcp.Namespace, Name: caSecretName}, sec); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	ca := sec.Data[common.CACrtKey]
	if len(ca) == 0 {
		return nil, nil // not ready yet
	}
	return ca, nil
}

// todo: think how to rotate the token
func (r *ManagedAddonsReconciler) ensureBootstrapToken(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (addons.BootstrapToken, error) {
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
	mcp *mcpv1alpha1.ManagedControlPlane,
) ([]client.Object, error) {
	ns := mcp.Namespace

	controlPlaneAgentTLSSecret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: common.KonnectivityAgentTLSSecretName, Namespace: ns}, controlPlaneAgentTLSSecret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	controlPlaneCASecret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: common.KonnectivityCASecretName, Namespace: ns}, controlPlaneCASecret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	workloadAgentTLSSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        controlPlaneAgentTLSSecret.Name,
			Namespace:   common.KonnectivityAgentNamespace,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Type: controlPlaneAgentTLSSecret.Type,
		Data: maps.Clone(controlPlaneAgentTLSSecret.Data),
	}

	workloadCASecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        controlPlaneCASecret.Name,
			Namespace:   common.KonnectivityAgentNamespace,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Type: controlPlaneCASecret.Type,
		Data: maps.Clone(controlPlaneCASecret.Data),
	}

	objs := []client.Object{workloadAgentTLSSecret, workloadCASecret}

	return objs, nil
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
	secretNS string,
) (*ControlPlaneClient, error) {
	sec := &corev1.Secret{}
	if err := mgmt.Get(ctx, client.ObjectKey{Namespace: secretNS, Name: common.AdminConfigName}, sec); err != nil {
		return nil, err
	}

	kubeconfigBytes, ok := sec.Data[common.AdminConfigKubeconfigKey]
	if !ok || len(kubeconfigBytes) == 0 {
		return nil, fmt.Errorf("secret %s/%s missing key %q or it is empty", secretNS, common.AdminConfigName, common.AdminConfigKubeconfigKey)
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
