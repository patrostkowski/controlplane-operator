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
	ctrl "sigs.k8s.io/controller-runtime"
)

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
