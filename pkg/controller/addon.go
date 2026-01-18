// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	applier "github.com/patrostkowski/controlplane-operator/pkg/controller/apply"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Addons struct {
	*applier.Applier
	mcp *mcpv1alpha1.ManagedControlPlane
	log logr.Logger
}

func NewAddons(mcp *mcpv1alpha1.ManagedControlPlane, k8s client.Client, scheme *runtime.Scheme, log logr.Logger) *Addons {
	return &Addons{
		Applier: applier.NewApplier(
			k8s,
			scheme,
			log,
			fieldOwner,
			applier.WithOwnerRef(false),
		),
		mcp: mcp,
		log: log.WithName("addons"),
	}
}

func (a *Addons) Ensure(ctx context.Context, resources []client.Object) error {
	return a.Apply(ctx, a.mcp, resources...)
}

func (a *Addons) konnectivityAgentManifests() []client.Object {
	return addons.KonnectivityAgentResources(a.mcp)
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
