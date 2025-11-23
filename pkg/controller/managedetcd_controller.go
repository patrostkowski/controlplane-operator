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
	"time"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/etcd"
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type ManagedETCDReconciler struct {
	controlplane.BaseReconciler
}

func (r *ManagedETCDReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("managedetcd", req.NamespacedName)

	etcdObj := &mcpv1alpha1.ManagedETCD{}
	if err := r.Get(ctx, req.NamespacedName, etcdObj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ManagedETCD")

	if err := r.UpdateCondition(ctx, etcdObj,
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

	resources := etcd.Resources(etcdObj)

	if err := r.ensureResources(ctx, etcdObj, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		_ = r.UpdateCondition(ctx, etcdObj,
			controlplane.Conditions{
				Type:    controlplane.ConditionWaitingForResource,
				Status:  metav1.ConditionFalse,
				Reason:  controlplane.ReasonWaitingForResources,
				Message: controlplane.MessageWaitingForResources,
			},
			controlplane.Status{
				Ready:   false,
				Message: "awaiting all ready",
			},
		)
		log.Info("requeueing reconcile for ETCD controller")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.UpdateCondition(ctx, etcdObj,
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

	log.Info("Finished reconciling ManagedETCD")
	return ctrl.Result{}, nil
}

func (r *ManagedETCDReconciler) ensureResources(
	ctx context.Context,
	etcdObj *mcpv1alpha1.ManagedETCD,
	resources []client.Object,
	log logr.Logger,
) error {
	for _, desired := range resources {
		log.Info("Ensuring etcd resource", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", desired.GetName())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, etcdObj, desired, log, func(obj client.Object) error {
			switch o := obj.(type) {
			case *corev1.Service:
				d := desired.(*corev1.Service)
				// Preserve clusterIP on updates
				clusterIP := o.Spec.ClusterIP
				o.Spec = d.Spec
				if clusterIP != "" {
					o.Spec.ClusterIP = clusterIP
				}
			case *appsv1.StatefulSet:
				d := desired.(*appsv1.StatefulSet)
				// Keep existing status, only spec/labels/annotations etc.
				o.Spec = d.Spec
				o.Labels = mergeStringMap(o.Labels, d.Labels)
				o.Annotations = mergeStringMap(o.Annotations, d.Annotations)
			}
			return nil
		})
		if err != nil {
			log.Error(err, "failed to ensure etcd resource", "name", desired.GetName())
			return err
		}
	}
	return nil
}

func (r *ManagedETCDReconciler) checkResourcesReady(
	ctx context.Context,
	resources []client.Object,
	log logr.Logger,
) (bool, error) {
	allReady := true

	for _, desired := range resources {
		switch desired.(type) {

		case *appsv1.StatefulSet:
			key := client.ObjectKey{Namespace: desired.GetNamespace(), Name: desired.GetName()}

			sts := &appsv1.StatefulSet{}
			if err := r.Get(ctx, key, sts); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "failed to get StatefulSet", "name", key.Name)
					return false, err
				}
				log.Info("StatefulSet not found yet", "name", key.Name)
				allReady = false
				continue
			}

			desiredReplicas := int32(1)
			if sts.Spec.Replicas != nil {
				desiredReplicas = *sts.Spec.Replicas
			}

			if sts.Status.ReadyReplicas < desiredReplicas {
				log.Info("StatefulSet not ready", "name", sts.Name, "readyReplicas", sts.Status.ReadyReplicas, "desiredReplicas", desiredReplicas)
				allReady = false
			} else {
				log.Info("StatefulSet ready", "name", sts.Name)
			}
		}
	}

	return allReady, nil
}

func mergeStringMap(dst, src map[string]string) map[string]string {
	if dst == nil && src == nil {
		return nil
	}
	if dst == nil {
		dst = map[string]string{}
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func SetupManagedETCDReconciler(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpv1alpha1.ManagedETCD{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(&ManagedETCDReconciler{
			BaseReconciler: controlplane.BaseReconciler{
				Client:   mgr.GetClient(),
				Log:      ctrl.Log.WithName("controller").WithName("ManagedETCD"),
				Recorder: mgr.GetEventRecorderFor("managedetcd"),
				Scheme:   mgr.GetScheme(),
			},
		})
}
