// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Scheduler struct {
	*Applier
	mcp *mcpv1alpha1.ManagedControlPlane
	log logr.Logger
}

func NewScheduler(mcp *mcpv1alpha1.ManagedControlPlane, k8s client.Client, scheme *runtime.Scheme, log logr.Logger) *Scheduler {
	return &Scheduler{
		Applier: NewApplier(k8s, scheme, log, fieldOwner),
		mcp:     mcp,
		log:     log.WithName("scheduler"),
	}
}

func (a *Scheduler) Ensure(ctx context.Context, resources []client.Object) error {
	return a.Apply(ctx, a.mcp, resources...)
}
