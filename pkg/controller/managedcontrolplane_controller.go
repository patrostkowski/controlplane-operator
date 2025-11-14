// Copyright 2025 controlplane.patrostkowski.dev
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const ManagedControlPlaneFinalizer = "controlplane.patrostkowski.dev/finalizer"

type ManagedControlPlaneReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *ManagedControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedcontrolplane", req.NamespacedName)

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !mcpObj.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("ManagedControlPlane is being deleted")
		controllerutil.RemoveFinalizer(mcpObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, mcpObj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(mcpObj, ManagedControlPlaneFinalizer) {
		log.Info("Adding finalizer to ManagedControlPlane")
		controllerutil.AddFinalizer(mcpObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, mcpObj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Version)

	baseName := mcpObj.Name
	pkiName := baseName + "-pki"
	etcdName := baseName + "-etcd"
	apiName := baseName + "-apiserver"
	cmName := baseName + "-controller-manager"
	schedName := baseName + "-scheduler"

	pki := &mcpv1alpha1.ManagedPKI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkiName,
			Namespace: mcpObj.Namespace,
		},
	}
	if err := r.createOrUpdateOwned(ctx, mcpObj, pki, func() error {
		pki.Spec.ControlPlaneName = mcpObj.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedPKI")
		return ctrl.Result{}, err
	}

	etcd := &mcpv1alpha1.ManagedETCD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdName,
			Namespace: mcpObj.Namespace,
		},
	}
	if err := r.createOrUpdateOwned(ctx, mcpObj, etcd, func() error {
		etcd.Spec.ControlPlaneName = mcpObj.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedETCD")
		return ctrl.Result{}, err
	}

	api := &mcpv1alpha1.ManagedAPIServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiName,
			Namespace: mcpObj.Namespace,
		},
	}
	if err := r.createOrUpdateOwned(ctx, mcpObj, api, func() error {
		api.Spec.ControlPlaneName = mcpObj.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedAPIServer")
		return ctrl.Result{}, err
	}

	cm := &mcpv1alpha1.ManagedControllerManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: mcpObj.Namespace,
		},
	}
	if err := r.createOrUpdateOwned(ctx, mcpObj, cm, func() error {
		cm.Spec.ControlPlaneName = mcpObj.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedControllerManager")
		return ctrl.Result{}, err
	}

	sched := &mcpv1alpha1.ManagedScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: mcpObj.Namespace,
		},
	}
	if err := r.createOrUpdateOwned(ctx, mcpObj, sched, func() error {
		sched.Spec.ControlPlaneName = mcpObj.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedScheduler")
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling ManagedControlPlane")

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) createOrUpdateOwned(
	ctx context.Context,
	owner *mcpv1alpha1.ManagedControlPlane,
	obj client.Object,
	mutate func() error,
) error {
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, obj, func() error {
		if err := controllerutil.SetControllerReference(owner, obj, r.Scheme); err != nil {
			return err
		}
		return mutate()
	})
	return err
}

func SetupManagedControlPlaneController(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedControlPlane{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Owns(&mcpv1alpha1.ManagedPKI{}).
		Owns(&mcpv1alpha1.ManagedETCD{}).
		Owns(&mcpv1alpha1.ManagedAPIServer{}).
		Owns(&mcpv1alpha1.ManagedControllerManager{}).
		Owns(&mcpv1alpha1.ManagedScheduler{}).
		Complete(&ManagedControlPlaneReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controller").WithName("ManagedControlPlane"),
			Recorder: mgr.GetEventRecorderFor("managedcontrolplane"),
			Scheme:   mgr.GetScheme(),
		})
}
