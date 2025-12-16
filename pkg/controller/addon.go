// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/addons"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcileAddon(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("addons", mcp.GetObjectMeta().GetNamespace())

	log.Info("Reconciling Addons")

	resources := addons.Resources(mcp)

	if err := r.ensureAddons(ctx, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling Addons")
	// TODO: properly manage child addons status changes
	// currently just requeue after 60 seconds
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// TODO: properly handle all scenarios
func (r *ManagedControlPlaneReconciler) ensureAddons(
	ctx context.Context,
	resources []client.Object,
	log logr.Logger,
) error {
	childClientSet, err := r.getChildConfig(ctx)
	if err != nil {
		return err
	}

	for _, addon := range resources {
		log.Info("Ensuring Addon resource", "kind", addon.GetObjectKind().GroupVersionKind().Kind, "name", addon.GetName())

		err := ServerSideApply(ctx, childClientSet, addon)
		if err != nil {
			log.Error(err, "failed to ensure Addon resource", "name", addon.GetName())
			return err
		}
	}

	return nil
}

func ServerSideApply(
	ctx context.Context,
	ctrlclient client.Client,
	obj client.Object,
) error {
	return ctrlclient.Patch(
		ctx,
		obj,
		client.Apply,
		client.FieldOwner("ManagedControlPlane-controller"),
	)
}

// Get rest config from secrets
func (r *ManagedControlPlaneReconciler) getChildConfig(
	ctx context.Context,
) (client.Client, error) {
	// caSecret := corev1.Secret{}
	adminSecret := corev1.Secret{}

	// if err := r.Get(ctx,
	// 	client.ObjectKey{Name: "managed-ca", Namespace: "mcp"},
	// 	&caSecret); err != nil {
	// 	return err
	// }
	if err := r.Get(ctx,
		client.ObjectKey{Name: "admin-client", Namespace: "mcp"},
		&adminSecret); err != nil {
		return nil, err
	}

	// caData := caSecret.Data["ca.crt"]
	certData := adminSecret.Data["tls.crt"]
	keyData := adminSecret.Data["tls.key"]

	RESTKubeConf := &rest.Config{
		// when testing with dev:run target
		// it works only with port forwarding
		Host: "https://172.30.0.250:6443", // TODO get actual API address
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
			// comment as with insecure flag it won't work
			// CAData:   caData,
			CertData: certData,
			KeyData:  keyData,
		},
	}

	RESTKubeConf.QPS = 20
	RESTKubeConf.Burst = 40

	ctrlClient, err := client.New(RESTKubeConf, client.Options{Scheme: r.Scheme})
	if err != nil {
		return nil, err
	}

	return ctrlClient, nil
}
