// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// ...

package controller

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/scheduler"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcileScheduler(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("controllermanager", mcp.GetObjectMeta().GetNamespace())

	log.Info("Reconciling Scheduler")

	resources := scheduler.Resources(mcp)

	if err := r.ensureSchedulerResources(ctx, mcp, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkSchedulerResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for Scheduler")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Finished reconciling Scheduler")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) ensureSchedulerResources(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	resources []client.Object,
	log logr.Logger,
) error {
	for _, desired := range resources {
		log.Info("Ensuring Scheduler resource",
			"kind", desired.GetObjectKind().GroupVersionKind().Kind,
			"name", desired.GetName(),
		)

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcp, desired, log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *corev1.ConfigMap:
				d := desired.(*corev1.ConfigMap)
				o.Data = d.Data
			case *appsv1.Deployment:
				d := desired.(*appsv1.Deployment)
				o.Spec = d.Spec
			}
			return nil
		})
		if err != nil {
			log.Error(err, "failed to ensure Scheduler resource", "name", desired.GetName())
			return err
		}
	}

	return nil
}

func (r *ManagedControlPlaneReconciler) checkSchedulerResourcesReady(
	ctx context.Context,
	resources []client.Object,
	log logr.Logger,
) (bool, error) {
	allReady := true

	for _, desired := range resources {
		key := client.ObjectKey{Namespace: desired.GetNamespace(), Name: desired.GetName()}

		switch desired.(type) {
		case *corev1.ConfigMap:
			cm := &corev1.ConfigMap{}
			if err := r.Get(ctx, key, cm); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to get ConfigMap", "name", key.Name)
				}
				allReady = false
			}

		case *appsv1.Deployment:
			deploy := &appsv1.Deployment{}
			if err := r.Get(ctx, key, deploy); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to get Deployment", "name", key.Name)
				}
				allReady = false
				continue
			}
			if !r.IsDeploymentReady(deploy) {
				log.Info("Deployment not ready", "name", deploy.Name, "readyReplicas", deploy.Status.ReadyReplicas)
				allReady = false
			}
		}
	}

	return allReady, nil
}
