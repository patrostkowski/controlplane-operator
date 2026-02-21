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
	"testing"
	"time"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestManagedControlPlaneReconciler_Reconcile_NotFound(t *testing.T) {
	t.Parallel()

	scheme := SchemeForControllerTests(t)
	client := FakeClientForTests(t, scheme)

	r := &ManagedControlPlaneReconciler{
		BaseReconciler: BaseReconcilerForTests(t, scheme),
		Client:         client,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "missing",
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !res.IsZero() {
		t.Fatalf("expected zero result, got %#v", res)
	}
}

func TestManagedControlPlaneReconciler_AddsFinalizer(t *testing.T) {
	t.Parallel()

	scheme := SchemeForControllerTests(t)

	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{Version: "v1.34.0"},
		},
	}

	client := FakeClientForTests(t, scheme, mcp)

	r := &ManagedControlPlaneReconciler{
		BaseReconciler: BaseReconcilerForTests(t, scheme),
		Client:         client,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "demo",
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Fatalf("expected requeue=true, got %#v", res)
	}

	got := &mcpv1alpha1.ManagedControlPlane{}
	if err := client.Get(context.Background(), req.NamespacedName, got); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if !controllerutil.ContainsFinalizer(got, ManagedControlPlaneFinalizer) {
		t.Fatalf("finalizer %q not added", ManagedControlPlaneFinalizer)
	}
}

func TestManagedControlPlaneReconciler_RemovesFinalizerOnDelete(t *testing.T) {
	t.Parallel()

	scheme := SchemeForControllerTests(t)
	now := metav1.NewTime(time.Now())

	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "demo",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{ManagedControlPlaneFinalizer},
		},
	}

	client := FakeClientForTests(t, scheme, mcp)

	r := &ManagedControlPlaneReconciler{
		BaseReconciler: BaseReconcilerForTests(t, scheme),
		Client:         client,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "demo",
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsZero() {
		t.Fatalf("expected zero result, got %#v", res)
	}

	got := &mcpv1alpha1.ManagedControlPlane{}
	err = client.Get(context.Background(), req.NamespacedName, got)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Acceptable: object finalized and removed.
			return
		}
		t.Fatalf("get failed: %v", err)
	}

	if controllerutil.ContainsFinalizer(got, ManagedControlPlaneFinalizer) {
		t.Fatalf("finalizer was not removed")
	}
}
