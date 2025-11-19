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
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const ManagedControlPlaneFinalizer = "controlplane.patrostkowski.dev/finalizer"

type ManagedControlPlaneReconciler struct {
	controlplane.BaseReconciler
}

func (r *ManagedControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedcontrolplane", req.NamespacedName)

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			// MCP already gone
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !mcpObj.ObjectMeta.DeletionTimestamp.IsZero() {
		// If finalizer not present, nothing to do
		if !controllerutil.ContainsFinalizer(mcpObj, ManagedControlPlaneFinalizer) {
			return ctrl.Result{}, nil
		}

		log.Info("ManagedControlPlane is being deleted, deleting child resources")

		// Actively delete child CRs
		stillExists, err := r.deleteChildren(ctx, mcpObj, log)
		if err != nil {
			return ctrl.Result{}, err
		}

		if stillExists {
			// Some children are still terminating; requeue and wait
			return ctrl.Result{Requeue: true}, nil
		}

		// All child CRs are gone → safe to drop finalizer
		log.Info("All child resources deleted, removing finalizer")

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
		// Finalizer updated → requeue so we don't continue in same reconcile
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Version)

	if res, err := r.reconcilePKI(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	if res, err := r.reconcileETCD(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	if res, err := r.reconcileAPIServer(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	if res, err := r.reconcileControllerManager(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	if res, err := r.reconcileScheduler(ctx, mcpObj, log); !res.IsZero() || err != nil {
		return res, err
	}

	if err := r.setMCPReadyCondition(ctx, mcpObj, true); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling ManagedControlPlane")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) deleteChildren(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (bool, error) {
	baseName := mcp.Name
	ns := mcp.Namespace

	children := []client.Object{
		&mcpv1alpha1.ManagedPKI{
			ObjectMeta: metav1.ObjectMeta{
				Name:      baseName + "-pki",
				Namespace: ns,
			},
		},
		&mcpv1alpha1.ManagedETCD{
			ObjectMeta: metav1.ObjectMeta{
				Name:      baseName + "-etcd",
				Namespace: ns,
			},
		},
		&mcpv1alpha1.ManagedAPIServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      baseName + "-apiserver",
				Namespace: ns,
			},
		},
		&mcpv1alpha1.ManagedControllerManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      baseName + "-controller-manager",
				Namespace: ns,
			},
		},
		&mcpv1alpha1.ManagedScheduler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      baseName + "-scheduler",
				Namespace: ns,
			},
		},
	}

	stillExists := false

	for _, child := range children {
		// Try to Get to see if it exists / is terminating
		err := r.Get(ctx, client.ObjectKeyFromObject(child), child)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Already gone
				continue
			}
			// Transient error – be conservative, say "still exists" so we requeue
			log.Error(err, "error getting child during deletion", "name", child.GetName())
			stillExists = true
			continue
		}

		// If it's not being deleted yet, issue a Delete
		if child.GetDeletionTimestamp().IsZero() {
			log.Info("Deleting child resource", "name", child.GetName(), "kind", child.GetObjectKind().GroupVersionKind().Kind)
			if err := r.Delete(ctx, child); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to delete child resource", "name", child.GetName())
					stillExists = true
					continue
				}
				// NotFound after delete is fine
				continue
			}
		}

		// Either in deletion or just deleted; consider it "still exists" until
		// next reconcile when Get will 404.
		stillExists = true
	}

	return stillExists, nil
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
			_ = r.setMCPReadyCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedPKI not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, false); err != nil {
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
			_ = r.setMCPReadyCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedETCD not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, false); err != nil {
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
			_ = r.setMCPReadyCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedAPIServer not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, false); err != nil {
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
			_ = r.setMCPReadyCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedControllerManager not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, false); err != nil {
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
			_ = r.setMCPReadyCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if !isReady(current.Status.Conditions) {
		log.Info("ManagedScheduler not Ready yet, waiting")
		if err := r.setMCPReadyCondition(ctx, mcp, false); err != nil {
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

// isReady checks whether the Ready condition is true on a child status.
func isReady(conds []metav1.Condition) bool {
	return apimeta.IsStatusConditionTrue(conds, string(controlplane.ConditionReady))
}

// setMCPReadyCondition uses the shared controlplane.ReadyConditionsForMCP
// helper to set the top-level Ready condition based on the aggregated state.
func (r *ManagedControlPlaneReconciler) setMCPReadyCondition(
	ctx context.Context,
	mcpObj *mcpv1alpha1.ManagedControlPlane,
	allReady bool,
) error {
	conds := controlplane.ReadyConditionsForMCP(allReady)
	return r.UpdateCondition(ctx, mcpObj, conds)
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
			BaseReconciler: controlplane.BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedControlPlane"),
				Recorder: mgr.GetEventRecorderFor("managedcontrolplane"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
