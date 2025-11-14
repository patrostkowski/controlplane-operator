// Copyright 2025 controlplane.patrostkowski.dev
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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ManagedSchedulerReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *ManagedSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedscheduler", req.NamespacedName)

	schedObj := &mcpv1alpha1.ManagedScheduler{}
	if err := r.Get(ctx, req.NamespacedName, schedObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !schedObj.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("ManagedScheduler is being deleted")
		controllerutil.RemoveFinalizer(schedObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, schedObj); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("ManagedScheduler is deleted")
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(schedObj, ManagedControlPlaneFinalizer) {
		log.Info("Adding finalizer to ManagedScheduler")
		controllerutil.AddFinalizer(schedObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, schedObj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling ManagedScheduler")

	resources := scheduler.Resources(schedObj)

	if err := r.ensureResources(ctx, schedObj, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateReadyCondition(ctx, schedObj, allReady); err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for ManagedScheduler until Deployment is Ready")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("Finished reconciling ManagedScheduler")
	return reconcile.Result{}, nil
}

func (r *ManagedSchedulerReconciler) ensureResources(
	ctx context.Context,
	schedObj *mcpv1alpha1.ManagedScheduler,
	resources []client.Object,
	log logr.Logger,
) error {
	for _, desired := range resources {
		log.Info("Ensuring Scheduler resource",
			"kind", desired.GetObjectKind().GroupVersionKind().Kind,
			"name", desired.GetName(),
		)

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, schedObj, desired, log, func(obj client.Object) error {
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

func (r *ManagedSchedulerReconciler) checkResourcesReady(
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
			if !isDeploymentReady(deploy) {
				log.Info("Deployment not ready", "name", deploy.Name, "readyReplicas", deploy.Status.ReadyReplicas)
				allReady = false
			}
		}
	}

	return allReady, nil
}

func (r *ManagedSchedulerReconciler) updateReadyCondition(
	ctx context.Context,
	schedObj *mcpv1alpha1.ManagedScheduler,
	allReady bool,
) error {
	var cond metav1.Condition

	if allReady {
		cond = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DeploymentReady",
			Message:            "kube-scheduler Deployment is Ready",
			ObservedGeneration: schedObj.Generation,
		}
	} else {
		cond = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "WaitingForDeployment",
			Message:            "Waiting for kube-scheduler Deployment to become Ready",
			ObservedGeneration: schedObj.Generation,
		}
	}

	if apimeta.SetStatusCondition(&schedObj.Status.Conditions, cond) {
		return r.Status().Update(ctx, schedObj)
	}
	return nil
}

func SetupManagedSchedulerReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedScheduler{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedSchedulerReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controller").WithName("ManagedScheduler"),
			Recorder: mgr.GetEventRecorderFor("managedscheduler"),
			Scheme:   mgr.GetScheme(),
		})
}
