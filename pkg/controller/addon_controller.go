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

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	mcctrl "github.com/patrostkowski/controlplane-operator/pkg/controller/multicluster"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// ManagedAddonReconciler reconciles addon objects.
type ManagedAddonsReconciler struct {
	BaseReconciler
	client.Client
	mcmanager.Manager
}

// Reconcile performs the main reconciliation loop for addon objects.
func (r *ManagedAddonsReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	// req.ClusterName is basically ns/name from mcp
	log := r.Log.WithValues("addons", req.ClusterName)

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	ns, name, _ := cache.SplitMetaNamespaceKey(req.ClusterName)
	if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			// MCP already gone
			return ctrl.Result{}, nil
		}
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	}

	cc := cluster.NewClusterContext(mcpObj, r.Log)
	cl, err := r.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, err
	}

	mc := cl.GetClient()

	ma := mcctrl.NewManagedAddon()
	if err := mc.Get(ctx, req.NamespacedName, ma); err != nil {
		if apierrors.IsNotFound(err) {
			// ManagedAddon already gone
			return ctrl.Result{}, nil
		}
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	}

	log.Info("reconciling kubeadm resources")
	if res, err := r.reconcileKubeletJoinResources(ctx, cc, mc, ma); err != nil {
		log.Error(err, "reconciling kubeadm resources failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	log.Info("reconciling addons resources")

	if res, err := r.reconcileAddons(ctx, cc, mc, ma); err != nil {
		log.Error(err, "reconciling addons failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{}, err
	} else if !res.IsZero() {
		return res, nil
	}

	log.Info("Finished reconciling addon controller")
	return ctrl.Result{}, nil
}

// SetupManagedAddonController sets up the ManagedAddon controller with the Kubernetes manager.
func SetupManagedAddonController(mgr mcmanager.Manager) error {
	u := mcctrl.NewManagedAddon()
	return mcbuilder.ControllerManagedBy(mgr).
		Named("addons-controller").
		For(u, mcbuilder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&storagev1.StorageClass{}).
		Complete(&ManagedAddonsReconciler{
			Manager: mgr,
			BaseReconciler: BaseReconciler{
				Log:      ctrl.Log.WithName("addon").WithName(mcpv1alpha1.KindManagedControlPlane),
				Recorder: mgr.GetLocalManager().GetEventRecorderFor("managedaddon"),
				Scheme:   mgr.GetLocalManager().GetScheme(),
			},
			Client: mgr.GetLocalManager().GetClient(),
		})
}
