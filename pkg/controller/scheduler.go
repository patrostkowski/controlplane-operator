// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/scheduler"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SchedulerComponent struct {
	r *ManagedControlPlaneReconciler
}

func (c *SchedulerComponent) Name() string {
	return "scheduler"
}

func (c *SchedulerComponent) Reconcile(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	return c.r.reconcileScheduler(ctx, mcp)
}

func (c *SchedulerComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageSchedulerWaiting
}

func (c *SchedulerComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageSchedulerFailed
}

func (r *ManagedControlPlaneReconciler) reconcileScheduler(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("scheduler", mcp.Namespace)

	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(mcp),
		scheduler.Resources(mcp)...,
	); err != nil {
		log.Error(err, "failed to apply scheduler resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
