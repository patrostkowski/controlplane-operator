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
	"time"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/etcd"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ManagedControlPlaneReconciler) reconcileETCD(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) (ctrl.Result, error) {
	log := r.Log.WithValues("etcd", mcp.GetObjectMeta().GetNamespace())

	log.Info("Reconciling ETCD")

	resources := etcd.Resources(mcp)

	if err := r.ensureETCDResources(ctx, mcp, resources, log); err != nil {
		return ctrl.Result{}, err
	}

	allReady, err := r.checkETCDResourcesReady(ctx, resources, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		log.Info("requeueing reconcile for ETCD")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Finished reconciling ETCD")
	return ctrl.Result{}, nil
}

func (r *ManagedControlPlaneReconciler) ensureETCDResources(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	resources []client.Object,
	log logr.Logger,
) error {
	for _, desired := range resources {
		log.Info("Ensuring etcd resource", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", desired.GetName())

		err := utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcp, desired, log, func(obj client.Object) error {
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

func (r *ManagedControlPlaneReconciler) checkETCDResourcesReady(
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
