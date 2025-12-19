// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// todo: match how other things reconcile
func (r *ManagedControlPlaneReconciler) reconcileAddon(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	c *controlplane.ControlPlaneClient,
) (ctrl.Result, error) {
	log := r.Log.WithValues("addons", mcp.Namespace)
	log.Info("Reconciling Addons")

	resources := addons.Resources(mcp)

	if err := c.ApplySSA(ctx, "mcp", resources...); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling Addons")

	return ctrl.Result{}, nil
}

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

	resources := addons.BootstrapKubeletJoinResources(tok)

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
) (addons.BootstrapToken, error) {

	sec := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: mcp.Namespace,
		Name:      addons.BootstrapTokenMgmtSecretName,
	}

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
