// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"bytes"
	"context"
	"maps"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: find a way to watch & act on managed resources
func (r *ManagedControlPlaneReconciler) reconcileAddons(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	if err := apply(
		ctx,
		r.cp.Client,
		r.managedScheme(),
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

	if err := apply(
		ctx,
		r.cp.Client,
		r.managedScheme(),
		r.managedApplyOpts(),
		secrets...,
	); err != nil {
		return ctrl.Result{}, err
	}

	if err := apply(
		ctx,
		r.cp.Client,
		r.managedScheme(),
		r.managedApplyOpts(),
		addons.KonnectivityAgentResources(mcp)...,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) controlPlaneClient(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (*controlplane.ControlPlaneClient, error) {
	log := r.Log.WithValues("addons", mcp.GetObjectMeta().GetNamespace())
	cp, err := controlplane.NewFromKubeconfigSecret(ctx, r.Client, r.Scheme, mcp.Namespace)
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
func (r *ManagedControlPlaneReconciler) reconcileKubeletJoinResources(
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

	if err := apply(
		ctx,
		r.cp.Client,
		r.managedScheme(),
		r.managedApplyOpts(),
		resources...,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) getClusterCA(
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

// Still it doesnt look perfect
func (r *ManagedControlPlaneReconciler) reconcileAdminConfig(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	ns := mcp.Namespace

	if mcp.Status.Address == "" {
		r.Log.Info("API address not set yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	serverURL := "https://" + mcp.Status.Address + ":6443"

	p := pki.New(mcp).Admin()

	// get admin-client secret
	adminClient := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: p.Client.SecretName, Namespace: ns}, adminClient); err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, client.IgnoreNotFound(err)
	}

	ca := adminClient.Data[common.CACrtKey]
	crt := adminClient.Data[common.TLSCrtKey]
	key := adminClient.Data[common.TLSKeyKey]

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

	// ensure admin-config secret
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.AdminConfigName,
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
	}

	err = utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcp, s, r.Log, func(obj client.Object) error {
		sec := obj.(*corev1.Secret)
		if sec.Data == nil {
			sec.Data = map[string][]byte{}
		}
		if bytes.Equal(sec.Data[common.AdminConfigKubeconfigKey], kubeconfigBytes) {
			return nil
		}
		sec.Data[common.AdminConfigKubeconfigKey] = kubeconfigBytes

		return nil
	})
	if err != nil {
		r.Log.Error(err, "failed to ensure Admin config secret", "name", s.GetName())
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	// updating MCP status with sercet ref
	if err = r.UpdateMCPAdminSecretRef(ctx, mcp, common.AdminConfigName); err != nil {
		r.Log.Error(err, "failed to update admin config secret ref")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// todo: think how to rotate the token
func (r *ManagedControlPlaneReconciler) ensureBootstrapToken(
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
				*metav1.NewControllerRef(mcp, mcpv1alpha1.SchemeGroupVersion.WithKind("ManagedControlPlane")),
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

func (r *ManagedControlPlaneReconciler) ensureKonnectivityTLSData(
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
