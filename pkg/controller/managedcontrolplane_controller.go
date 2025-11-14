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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
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
		log.Info("ManagedControlPlane is deleted")
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

	// PHASE 1: PKI
	if res, err := r.reconcilePKI(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	// PHASE 2: ETCD (requires PKI Ready)
	if res, err := r.reconcileETCD(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	// PHASE 3: API server (requires ETCD Ready)
	if res, err := r.reconcileAPIServer(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	// PHASE 4: ControllerManager (requires APIServer Ready)
	if res, err := r.reconcileControllerManager(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	// PHASE 5: Scheduler (requires ControllerManager Ready)
	if res, err := r.reconcileScheduler(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	// All components ready → mark MCP Ready=True
	if err := r.setMCPReadyCondition(
		ctx,
		mcpObj,
		metav1.ConditionTrue,
		"ControlPlaneReady",
		"All control plane components are Ready",
	); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling ManagedControlPlane")
	return ctrl.Result{}, nil
}
func (r *ManagedControlPlaneReconciler) reconcilePKI(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (ctrl.Result, error) {
	baseName := mcp.Name
	pkiName := baseName + "-pki"

	pki := &mcpv1alpha1.ManagedPKI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkiName,
			Namespace: mcp.Namespace,
		},
	}

	if err := r.createOrUpdateOwned(ctx, mcp, pki, func() error {
		pki.Spec.ControlPlaneName = mcp.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedPKI")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedPKI{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(pki), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedPKI not yet visible, requeueing")
			_ = r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForPKI", "Waiting for ManagedPKI to become Ready")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedPKI not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForPKI", "Waiting for ManagedPKI to become Ready"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("ManagedPKI Ready")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileETCD(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (ctrl.Result, error) {
	baseName := mcp.Name
	etcdName := baseName + "-etcd"

	etcd := &mcpv1alpha1.ManagedETCD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdName,
			Namespace: mcp.Namespace,
		},
	}

	if err := r.createOrUpdateOwned(ctx, mcp, etcd, func() error {
		etcd.Spec.ControlPlaneName = mcp.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedETCD")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedETCD{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(etcd), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedETCD not yet visible, requeueing")
			_ = r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForETCD", "Waiting for ManagedETCD to become Ready")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedETCD not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForETCD", "Waiting for ManagedETCD to become Ready"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("ManagedETCD Ready")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileAPIServer(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (ctrl.Result, error) {
	baseName := mcp.Name
	apiName := baseName + "-apiserver"

	api := &mcpv1alpha1.ManagedAPIServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiName,
			Namespace: mcp.Namespace,
		},
	}

	if err := r.createOrUpdateOwned(ctx, mcp, api, func() error {
		api.Spec.ControlPlaneName = mcp.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedAPIServer")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedAPIServer{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(api), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedAPIServer not yet visible, requeueing")
			_ = r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForAPIServer", "Waiting for ManagedAPIServer to become Ready")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedAPIServer not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForAPIServer", "Waiting for ManagedAPIServer to become Ready"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("ManagedAPIServer Ready")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileControllerManager(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (ctrl.Result, error) {
	baseName := mcp.Name
	cmName := baseName + "-controller-manager"

	cm := &mcpv1alpha1.ManagedControllerManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: mcp.Namespace,
		},
	}

	if err := r.createOrUpdateOwned(ctx, mcp, cm, func() error {
		cm.Spec.ControlPlaneName = mcp.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedControllerManager")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedControllerManager{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(cm), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedControllerManager not yet visible, requeueing")
			_ = r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForControllerManager", "Waiting for ManagedControllerManager to become Ready")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedControllerManager not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForControllerManager", "Waiting for ManagedControllerManager to become Ready"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("ManagedControllerManager Ready")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileScheduler(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (ctrl.Result, error) {
	baseName := mcp.Name
	schedName := baseName + "-scheduler"

	sched := &mcpv1alpha1.ManagedScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedName,
			Namespace: mcp.Namespace,
		},
	}

	if err := r.createOrUpdateOwned(ctx, mcp, sched, func() error {
		sched.Spec.ControlPlaneName = mcp.Name
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedScheduler")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedScheduler{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(sched), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedScheduler not yet visible, requeueing")
			_ = r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForScheduler", "Waiting for ManagedScheduler to become Ready")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedScheduler not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, metav1.ConditionFalse, "WaitingForScheduler", "Waiting for ManagedScheduler to become Ready"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("ManagedScheduler Ready")
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

func isReady(conds []metav1.Condition) bool {
	return apimeta.IsStatusConditionTrue(conds, "Ready")
}

// setMCPReadyCondition sets/updates the top-level Ready condition on ManagedControlPlane.
func (r *ManagedControlPlaneReconciler) setMCPReadyCondition(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	status metav1.ConditionStatus,
	reason, message string,
) error {
	cond := metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: mcp.Generation,
	}

	if apimeta.SetStatusCondition(&mcp.Status.Conditions, cond) {
		return r.Status().Update(ctx, mcp)
	}
	return nil
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
