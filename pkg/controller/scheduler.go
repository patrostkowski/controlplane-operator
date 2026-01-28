// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/controller/state"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/scheduler"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SchedulerComponent struct {
	r *ManagedControlPlaneReconciler
}

func (c *SchedulerComponent) Name() string {
	return "scheduler"
}

func (c *SchedulerComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileScheduler(ctx, cc)
}

func (c *SchedulerComponent) WaitingMessage() mcpv1alpha1.Message {
	return state.MessageSchedulerWaiting
}

func (c *SchedulerComponent) FailedMessage() mcpv1alpha1.Message {
	return state.MessageSchedulerFailed
}

func (r *ManagedControlPlaneReconciler) reconcileScheduler(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	mcp := cc.MCP
	log := r.Log.WithValues("scheduler", mcp.Namespace)

	s := scheduler.NewBuilder(cc)
	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(mcp),
		s.Objects()...,
	); err != nil {
		log.Error(err, "failed to apply scheduler resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
