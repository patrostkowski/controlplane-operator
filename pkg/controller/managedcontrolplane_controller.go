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
	"bytes"
	"context"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/controllermanager"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/etcd"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/scheduler"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const RequeueAfterFailure = 10 * time.Second

const ManagedControlPlaneFinalizer = "controlplane.patrostkowski.dev/finalizer"

type ManagedControlPlaneReconciler struct {
	BaseReconciler
}

func (r *ManagedControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedcontrolplane", req.NamespacedName)

	mcpObj := &mcpv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, mcpObj); err != nil {
		if apierrors.IsNotFound(err) {
			// MCP already gone
			return ctrl.Result{}, nil
		}
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Version)

	if !mcpObj.ObjectMeta.DeletionTimestamp.IsZero() {
		// If finalizer not present, nothing to do
		if !controllerutil.ContainsFinalizer(mcpObj, ManagedControlPlaneFinalizer) {
			return ctrl.Result{}, nil
		}

		log.Info("ManagedControlPlane is being deleted, deleting child resources")

		// Actively delete child CRs
		// stillExists, err := r.deleteChildren(ctx, req)
		// if err != nil {
		// 	log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		// return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
		// }

		// if stillExists {
		// 	// Some children are still terminating; requeue and wait
		// 	return ctrl.Result{Requeue: true}, nil
		// }

		log.Info("All child resources deleted, removing finalizer")

		controllerutil.RemoveFinalizer(mcpObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, mcpObj); err != nil {
			log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
			return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
		}

		log.Info("Reconcile finished with deleted resources", "resource", mcpObj.GetName())
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(mcpObj, ManagedControlPlaneFinalizer) {
		log.Info("Adding finalizer to ManagedControlPlane")
		controllerutil.AddFinalizer(mcpObj, ManagedControlPlaneFinalizer)
		if err := r.Update(ctx, mcpObj); err != nil {
			log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
			return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// API Server Service
	if res, err := r.reconcileAPIServiceSvc(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageAPIServerSvcFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageAPIServerSvcWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// PKI
	if res, err := r.reconcilePKI(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessagePKIFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessagePKIWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// ETCD
	if res, err := r.reconcileETCD(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageETCDFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageETCDWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// APIServer
	if res, err := r.reconcileAPIServer(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageAPIServerFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageAPIServerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// ControllerManager
	if res, err := r.reconcileControllerManager(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageControllerManagerFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageControllerManagerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// Scheduler
	if res, err := r.reconcileScheduler(ctx, mcpObj); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageSchedulerFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageSchedulerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileAdminConfig(ctx, mcpObj, mcpObj.Namespace); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessagePKIFailed)
		log.Error(err, "failed to ensure admin kubeconfig")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageSchedulerWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	cp, err := r.controlPlaneClient(ctx, mcpObj)
	if err != nil {
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	v, err := cp.Discovery.ServerVersion()
	if err != nil {
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	log.Info("Obtained child cluster config", "version", v)

	svc := &corev1.Service{}
	err = cp.Get(ctx, client.ObjectKey{
		Namespace: "default",
		Name:      "kubernetes",
	}, svc)
	if err != nil {
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	log.Info("Got kubernetes service",
		"clusterIPs", svc.Spec.ClusterIPs,
		"type", svc.Spec.Type,
	)

	if res, err := r.reconcileKubeletJoinResources(ctx, mcpObj, cp); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageKubeResourcesFailed)
		log.Error(err, "component failed, will retry", "after", RequeueAfterFailure)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageKubeResourcesWaiting)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if res, err := r.reconcileAddon(ctx, mcpObj, cp); err != nil {
		_ = r.statusFailed(ctx, mcpObj, state.MessageAddonsFailed)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	} else if !res.IsZero() {
		_ = r.statusWaiting(ctx, mcpObj, state.MessageAddonsWaiting)
		return res, nil
	}

	_ = r.statusReady(ctx, mcpObj)
	log.Info("Finished reconciling ManagedControlPlane")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) controlPlaneClient(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (*controlplane.ControlPlaneClient, error) {
	return controlplane.NewFromKubeconfigSecret(ctx, r.Client, r.Scheme, mcp.Namespace)
}

func (r *ManagedControlPlaneReconciler) reconcileAPIServiceSvc(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {

	log := r.Log.WithValues("api-endpoint", mcp.Namespace)
	api := NewAPIServer(mcp, r.Client, r.Scheme, log)

	if err := api.Ensure(ctx, api.endpointManifests()); err != nil {
		return ctrl.Result{}, err
	}

	addr, err := api.tryEndpointAddress(ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to read API Service address (best-effort)")
		return ctrl.Result{}, err
	}
	if addr != "" && mcp.Status.Address != addr {
		if err := r.UpdateMCPAddress(ctx, mcp, addr); err != nil {
			log.Error(err, "failed to update address")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileAPIServer(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {

	log := r.Log.WithValues("apiserver", mcp.Namespace)
	api := NewAPIServer(mcp, r.Client, r.Scheme, log)

	if err := api.Ensure(ctx, api.workloadManifests()); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileETCD(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("etcd", mcp.GetObjectMeta().GetNamespace())
	e := NewETCD(mcp, r.Client, r.Scheme, log)

	resources := etcd.Resources(mcp)

	if err := e.Ensure(ctx, resources); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileControllerManager(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (ctrl.Result, error) {
	log := r.Log.WithValues("controllermanager", mcp.GetObjectMeta().GetNamespace())
	cm := NewControllerManager(mcp, r.Client, r.Scheme, log)

	resources := controllermanager.Resources(mcp)

	if err := cm.Ensure(ctx, resources); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcileScheduler(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("scheduler", mcp.GetObjectMeta().GetNamespace())
	s := NewScheduler(mcp, r.Client, r.Scheme, log)

	resources := scheduler.Resources(mcp)

	if err := s.Ensure(ctx, resources); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) reconcilePKI(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("pki", mcp.GetObjectMeta().GetNamespace())
	p := NewPKI(mcp, r.Client, r.Scheme, log)

	resources := pki.Resources(mcp)

	if err := p.Ensure(ctx, resources); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Still it doesnt look perfect
func (r *ManagedControlPlaneReconciler) reconcileAdminConfig(
	ctx context.Context,
	mcpObj *mcpv1alpha1.ManagedControlPlane,
	ns string,
) (ctrl.Result, error) {
	serverURL := "https://" + mcpObj.Status.Address + ":6443"

	if mcpObj.Status.Address == "" {
		r.Log.Info("API address not set yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// get admin-client secret
	adminClient := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Name: "admin-client", Namespace: ns}, adminClient); err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, client.IgnoreNotFound(err)
	}

	ca := adminClient.Data["ca.crt"]
	crt := adminClient.Data["tls.crt"]
	key := adminClient.Data["tls.key"]

	if len(ca) == 0 || len(crt) == 0 || len(key) == 0 {
		r.Log.Info("admin config secret not ready yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// build kubecfg
	cfg := clientcmdapi.NewConfig()
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:                   serverURL,
		CertificateAuthorityData: ca,
	}
	cfg.AuthInfos["local"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: crt,
		ClientKeyData:         key,
	}
	cfg.Contexts["local"] = &clientcmdapi.Context{Cluster: "local", AuthInfo: "local"}
	cfg.CurrentContext = "local"

	kubeconfigBytes, err := clientcmd.Write(*cfg)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-config",
			Namespace: ns,
		},
		Type: corev1.SecretTypeOpaque,
	}

	err = utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcpObj, s, r.Log, func(obj client.Object) error {
		sec := obj.(*corev1.Secret)
		if sec.Data == nil {
			sec.Data = map[string][]byte{}
		}
		if bytes.Equal(sec.Data["config"], kubeconfigBytes) {
			return nil
		}
		sec.Data["config"] = kubeconfigBytes
		return nil
	})
	if err != nil {
		r.Log.Error(err, "failed to ensure Admin config secret", "name", s.GetName())
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	return ctrl.Result{}, nil
}

func SetupManagedControlPlaneController(mgr ctrl.Manager) error {
	certPred := predicate.GenerationChangedPredicate{}
	issuerPred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedControlPlane{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&certmanagerv1.Certificate{}, builder.WithPredicates(certPred)).
		Owns(&certmanagerv1.Issuer{}, builder.WithPredicates(issuerPred)).
		WithOptions(
			controller.Options{
				// MaxConcurrentReconciles: 1,
				RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
					5*time.Second,
					60*time.Second,
				),
			},
		).
		Complete(&ManagedControlPlaneReconciler{
			BaseReconciler: BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedControlPlane"),
				Recorder: mgr.GetEventRecorderFor("managedcontrolplane"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
