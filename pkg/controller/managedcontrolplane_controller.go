// Copyright 2025 Patryk Rostkowski
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
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/etcd"
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

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Version)

	if err := r.UpdateCondition(ctx, mcpObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReconciling,
			Status:  metav1.ConditionFalse,
			Reason:  controlplane.ReasonReconciling,
			Message: controlplane.MessageReconciling,
		},
		controlplane.Status{
			Ready:   false,
			Message: "reconciling",
		},
	); err != nil {
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

	// if res, err := r.reconcileAddon(ctx, mcpObj, log); !res.IsZero() || err != nil {
	// 	return res, err
	// }

	if err := r.UpdateCondition(ctx, mcpObj,
		controlplane.Conditions{
			Type:    controlplane.ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  controlplane.ReasonAllResourcesReady,
			Message: controlplane.MessageAllResourcesReady,
		},
		controlplane.Status{
			Ready:   true,
			Message: "all ready",
		},
	); err != nil {
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
		&mcpv1alpha1.ManagedAddon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      baseName + "-addon",
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
			// err = r.updateCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if !r.isReadyForLatestSpec(current, current.Status.Conditions) {
		log.Info("ManagedPKI not Ready yet, waiting",
			"child", current.GetName(),
			"generation", current.GetGeneration(),
		)
		// if err := r.updateCondition(ctx, mcp, false); err != nil {
		// 	return ctrl.Result{}, err
		// }
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

	etcdVersion, err := r.resolveEtcdImageForKubeVersion(mcp.Spec.Version)
	if err != nil {
		log.Error(err, "failed to resolve etcd version from Kubernetes version",
			"kubeVersion", mcp.Spec.Version)
		// _ = r.updateCondition(ctx, mcp, false)
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateOwned(ctx, mcp, etcd, func() error {
		etcd.Spec.ControlPlaneName = mcp.Name
		etcd.Spec.Version = etcdVersion
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedETCD")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedETCD{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(etcd), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedETCD not yet visible, requeueing")
			// err = r.updateCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if !r.isReadyForLatestSpec(current, current.Status.Conditions) {
		log.Info("ManagedETCD not Ready yet, waiting",
			"child", current.GetName(),
			"generation", current.GetGeneration(),
		)
		// if err := r.updateCondition(ctx, mcp, false); err != nil {
		// 	return ctrl.Result{}, err
		// }
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
		api.Spec.Version = mcp.Spec.Version
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedAPIServer")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedAPIServer{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(api), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedAPIServer not yet visible, requeueing")
			// err = r.updateCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if !r.isReadyForLatestSpec(current, current.Status.Conditions) {
		log.Info("ManagedAPIServer not Ready yet, waiting",
			"child", current.GetName(),
			"generation", current.GetGeneration(),
		)
		// if err := r.updateCondition(ctx, mcp, false); err != nil {
		// 	return ctrl.Result{}, err
		// }
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
		cm.Spec.Version = mcp.Spec.Version
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedControllerManager")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedControllerManager{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(cm), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedControllerManager not yet visible, requeueing")
			// err = r.updateCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if !r.isReadyForLatestSpec(current, current.Status.Conditions) {
		log.Info("ManagedControllerManager not Ready yet, waiting",
			"child", current.GetName(),
			"generation", current.GetGeneration(),
		)
		// if err := r.updateCondition(ctx, mcp, false); err != nil {
		// 	return ctrl.Result{}, err
		// }
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
		sched.Spec.Version = mcp.Spec.Version
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedScheduler")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedScheduler{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(sched), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedScheduler not yet visible, requeueing")
			// err = r.updateCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if !r.isReadyForLatestSpec(current, current.Status.Conditions) {
		log.Info("ManagedScheduler not Ready yet, waiting",
			"child", current.GetName(),
			"generation", current.GetGeneration(),
		)
		// if err := r.updateCondition(ctx, mcp, false); err != nil {
		// 	return ctrl.Result{}, err
		// }
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("ManagedScheduler Ready")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileAddon(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	log logr.Logger,
) (ctrl.Result, error) {
	baseName := mcp.Name
	addonName := baseName + "-addon"

	addon := &mcpv1alpha1.ManagedAddon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addonName,
			Namespace: mcp.Namespace,
		},
	}

	if err := r.createOrUpdateOwned(ctx, mcp, addon, func() error {
		addon.Spec.ControlPlaneName = mcp.Name
		// temporary set to Helm
		addon.Spec.Type = "Helm"
		addon.Spec.Version = mcp.Spec.Version
		return nil
	}); err != nil {
		log.Error(err, "failed to reconcile ManagedAddon")
		return ctrl.Result{}, err
	}

	current := &mcpv1alpha1.ManagedAddon{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(addon), current); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ManagedAddon not yet visible, requeueing")
			// err = r.updateCondition(ctx, mcp, false)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, err
	}

	if !r.isReadyForLatestSpec(current, current.Status.Conditions) {
		log.Info("ManagedAddon not Ready yet, waiting",
			"child", current.GetName(),
			"generation", current.GetGeneration(),
		)
		// if err := r.updateCondition(ctx, mcp, false); err != nil {
		// 	return ctrl.Result{}, err
		// }
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

func (r *ManagedControlPlaneReconciler) resolveEtcdImageForKubeVersion(kubeVersion string) (string, error) {
	// kubeVersion is expected like "v1.31.0" or "1.31.0"
	v := strings.TrimPrefix(kubeVersion, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("unable to parse Kubernetes version %q", kubeVersion)
	}

	minorKey := parts[0] + "." + parts[1]

	ver, ok := etcd.EtcdVersionByKubeMinor[minorKey]
	if !ok {
		return "", fmt.Errorf("no etcd version mapping for Kubernetes minor %q", minorKey)
	}

	r.Log.Info("Resolved etcd version for kube version",
		"kubeVersion", kubeVersion,
		"etcdVersion", ver,
	)

	return ver, nil
}

func (r *ManagedControlPlaneReconciler) isReadyForLatestSpec(obj client.Object, conds []metav1.Condition) bool {
	cond := apimeta.FindStatusCondition(conds, string(controlplane.ConditionReady))
	if cond == nil {
		return false
	}
	if cond.Status != metav1.ConditionTrue {
		return false
	}
	// Critical upgrade-serialization check:
	if cond.ObservedGeneration != obj.GetGeneration() {
		return false
	}
	return true
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
