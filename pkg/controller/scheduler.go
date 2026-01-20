// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/scheduler"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ManagedControlPlaneReconciler) reconcileScheduler(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("scheduler", mcp.Namespace)

	if err := apply(
		ctx,
		r.Client,
		r.Scheme,
		r.applyOpts(mcp),
		scheduler.Resources(mcp)...,
	); err != nil {
		log.Error(err, "failed to apply scheduler resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
