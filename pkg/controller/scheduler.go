// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/scheduler"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SchedulerComponent reconciles the Kubernetes scheduler.
type SchedulerComponent struct {
	r *ManagedControlPlaneReconciler
}

// Name returns the name of the scheduler component.
func (c *SchedulerComponent) Name() string {
	return "scheduler"
}

// Reconcile reconciles the scheduler deployment.
func (c *SchedulerComponent) Reconcile(ctx context.Context, cc *cluster.ClusterContext) (ctrl.Result, error) {
	return c.r.reconcileScheduler(ctx, cc)
}

// WaitingMessage returns the waiting message for the scheduler component.
func (c *SchedulerComponent) WaitingMessage() mcpv1alpha1.Message {
	return mcpv1alpha1.MessageSchedulerWaiting
}

// FailedMessage returns the failed message for the scheduler component.
func (c *SchedulerComponent) FailedMessage() mcpv1alpha1.Message {
	return mcpv1alpha1.MessageSchedulerFailed
}

// reconcileScheduler reconciles the Kubernetes scheduler deployment.
func (r *ManagedControlPlaneReconciler) reconcileScheduler(
	ctx context.Context,
	cc *cluster.ClusterContext,
) (ctrl.Result, error) {
	log := r.Log.WithValues("scheduler", cc.Namespace())

	s := scheduler.NewBuilder(cc)
	if err := r.apply(
		ctx,
		r.Client,
		r.applyOpts(cc.Owner()),
		s.Objects()...,
	); err != nil {
		log.Error(err, "failed to apply scheduler resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
