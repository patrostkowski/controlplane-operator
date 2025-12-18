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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedControlPlane", "version", mcpObj.Spec.Version)

	if err := r.UpdateCondition(ctx, mcpObj,
		mcpv1alpha1.Conditions{
			Type:    state.ConditionReconciling,
			Status:  metav1.ConditionTrue,
			Reason:  state.ReasonReconciling,
			Message: state.MessageReconciling,
		},
		mcpv1alpha1.Status{
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
		// stillExists, err := r.deleteChildren(ctx, req)
		// if err != nil {
		// 	return ctrl.Result{}, err
		// }

		// if stillExists {
		// 	// Some children are still terminating; requeue and wait
		// 	return ctrl.Result{Requeue: true}, nil
		// }

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

	if res, err := r.reconcileAPIServiceSvc(ctx, mcpObj); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed API Service address",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for API Service address",
			},
		)
		return res, nil
	}

	// PKI
	if res, err := r.reconcilePKI(ctx, mcpObj); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed PKI",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for PKI",
			},
		)
		return res, nil
	}

	// ETCD
	if res, err := r.reconcileETCD(ctx, mcpObj); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed ETCD",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for ETCD",
			},
		)
		return res, nil
	}

	// APIServer
	if res, err := r.reconcileAPIServer(ctx, mcpObj); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed APIServer",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for APIServer",
			},
		)
		return res, nil
	}

	// ControllerManager
	if res, err := r.reconcileControllerManager(ctx, mcpObj); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed ControllerManager",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for ControllerManager",
			},
		)
		return res, nil
	}

	// Scheduler
	if res, err := r.reconcileScheduler(ctx, mcpObj); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed Scheduler",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for Scheduler",
			},
		)
		return res, nil
	}

	// if res, err := r.reconcileAddon(ctx, mcpObj); !res.IsZero() || err != nil {
	// 	return res, err
	// }

	cp, err := r.controlPlaneClient(ctx, mcpObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	v, err := cp.Discovery.ServerVersion()
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Obtained child cluster config", "version", v)

	svc := &corev1.Service{}
	err = cp.Get(ctx, client.ObjectKey{
		Namespace: "default",
		Name:      "kubernetes",
	}, svc)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Got kubernetes service",
		"clusterIPs", svc.Spec.ClusterIPs,
		"type", svc.Spec.Type,
	)

	if res, err := r.reconcileKubeletJoinResources(ctx, mcpObj, cp); err != nil {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonFailed,
				Message: state.MessageFailed,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "failed Kubernetes resources",
			},
		)
		return res, err
	} else if !res.IsZero() {
		_ = r.UpdateCondition(ctx, mcpObj,
			mcpv1alpha1.Conditions{
				Type:    state.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  state.ReasonWaitingForResources,
				Message: state.MessageWaitingForResources,
			},
			mcpv1alpha1.Status{
				Ready:   false,
				Message: "waiting for resources",
			},
		)
		return res, nil
	}

	if err := r.UpdateCondition(ctx, mcpObj,
		mcpv1alpha1.Conditions{
			Type:    state.ConditionReady,
			Status:  metav1.ConditionTrue,
			Reason:  state.ReasonAllResourcesReady,
			Message: state.MessageAllResourcesReady,
		},
		mcpv1alpha1.Status{
			Ready:   true,
			Message: "all ready",
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciling ManagedControlPlane")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) controlPlaneClient(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
) (*controlplane.ControlPlaneClient, error) {
	return controlplane.NewFromKubeconfigSecret(ctx, r.Client, r.Scheme, mcp.Namespace)
}

func SetupManagedControlPlaneController(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedControlPlane{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&certmanagerv1.Issuer{}).
		Owns(&certmanagerv1.Certificate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedControlPlaneReconciler{
			BaseReconciler: BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedControlPlane"),
				Recorder: mgr.GetEventRecorderFor("managedcontrolplane"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
