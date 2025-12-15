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
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/addons"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedAddonReconciler struct {
	controlplane.BaseReconciler
}

func (r *ManagedAddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedaddon", req.NamespacedName)

	addonObj := &mcpv1alpha1.ManagedAddon{}
	if err := r.Get(ctx, req.NamespacedName, addonObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedAddon")

	resources := addons.Resources(addonObj)

	if err := r.ensureAddons(ctx, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	// TODO: properly update conditions
	// based on installed stuff(addons)
	if err := r.UpdateCondition(ctx, addonObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  controlplane.ReasonAllResourcesReady,
			Message: controlplane.MessageAllResourcesReady,
		},
		controlplane.Status{
			Ready:   true,
			Message: "all ready",
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling ManagedAddon")
	// TODO: properly manage child addons status changes
	// currently just requeue after 60 seconds
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// TODO: properly handle all scenarios
func (r *ManagedAddonReconciler) ensureAddons(
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
		client.FieldOwner("managedaddon-controller"),
	)
}

// Get rest config from secrets
func (r *ManagedAddonReconciler) getChildConfig(
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

func SetupManagedAddonReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedAddon{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedAddonReconciler{
			BaseReconciler: controlplane.BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedAddon"),
				Recorder: mgr.GetEventRecorderFor("managedaddon"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
