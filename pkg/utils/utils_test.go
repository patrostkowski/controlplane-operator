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

package utils

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/go-logr/logr"
)

type testEnv struct {
	ctx    context.Context
	scheme *runtime.Scheme
	c      client.Client
	log    logr.Logger
}

func newTestEnv(t *testing.T, initObjs ...client.Object) testEnv {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}

	b := fake.NewClientBuilder().WithScheme(scheme)
	if len(initObjs) > 0 {
		b = b.WithObjects(initObjs...)
	}

	return testEnv{
		ctx:    context.Background(),
		scheme: scheme,
		c:      b.Build(),
		log:    logr.Discard(),
	}
}

func newOwnerPod(ns, name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

func newConfigMapTemplate(ns, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

func TestEnsureCreatedAndOwned_CreatesAndSetsOwnerRefAndMutates(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	owner := newOwnerPod("ns", "owner-pod")
	template := newConfigMapTemplate("ns", "cm")

	err := EnsureCreatedAndOwned(env.ctx, env.c, env.scheme, owner, template, env.log, func(obj client.Object) error {
		cm, ok := obj.(*corev1.ConfigMap)
		if !ok {
			t.Fatalf("expected *corev1.ConfigMap, got %T", obj)
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["k"] = "v1"
		return nil
	})
	if err != nil {
		t.Fatalf("EnsureCreatedAndOwned: %v", err)
	}

	var got corev1.ConfigMap
	if err := env.c.Get(env.ctx, types.NamespacedName{Name: "cm", Namespace: "ns"}, &got); err != nil {
		t.Fatalf("Get created configmap: %v", err)
	}
	if got.Data["k"] != "v1" {
		t.Fatalf("expected Data[k]=v1, got %q", got.Data["k"])
	}

	if len(got.OwnerReferences) != 1 {
		t.Fatalf("expected 1 ownerReference, got %d", len(got.OwnerReferences))
	}
	ref := got.OwnerReferences[0]
	if ref.Name != owner.Name {
		t.Fatalf("expected ownerReference name %q, got %q", owner.Name, ref.Name)
	}
	if ref.Kind != "Pod" {
		t.Fatalf("expected ownerReference kind %q, got %q", "Pod", ref.Kind)
	}
	if ref.Controller == nil || !*ref.Controller {
		t.Fatalf("expected ownerReference controller=true, got %#v", ref.Controller)
	}
}

func TestEnsureCreatedAndOwned_UpdatesOnSecondCall(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	owner := newOwnerPod("ns", "owner")
	template := newConfigMapTemplate("ns", "cm")

	// First create
	if err := EnsureCreatedAndOwned(env.ctx, env.c, env.scheme, owner, template, env.log, func(obj client.Object) error {
		cm := obj.(*corev1.ConfigMap)
		cm.Data = map[string]string{"k": "v1"}
		return nil
	}); err != nil {
		t.Fatalf("first EnsureCreatedAndOwned: %v", err)
	}

	// Second call should update
	if err := EnsureCreatedAndOwned(env.ctx, env.c, env.scheme, owner, template, env.log, func(obj client.Object) error {
		cm := obj.(*corev1.ConfigMap)
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["k"] = "v2"
		return nil
	}); err != nil {
		t.Fatalf("second EnsureCreatedAndOwned: %v", err)
	}

	var got corev1.ConfigMap
	if err := env.c.Get(env.ctx, types.NamespacedName{Name: "cm", Namespace: "ns"}, &got); err != nil {
		t.Fatalf("Get updated configmap: %v", err)
	}
	if got.Data["k"] != "v2" {
		t.Fatalf("expected Data[k]=v2, got %q", got.Data["k"])
	}
}

func TestEnsureCreatedAndOwned_MutateNilDoesNotError(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)

	owner := newOwnerPod("ns", "owner")
	template := newConfigMapTemplate("ns", "cm")

	if err := EnsureCreatedAndOwned(env.ctx, env.c, env.scheme, owner, template, env.log, nil); err != nil {
		t.Fatalf("EnsureCreatedAndOwned with nil mutate: %v", err)
	}

	var got corev1.ConfigMap
	if err := env.c.Get(env.ctx, types.NamespacedName{Name: "cm", Namespace: "ns"}, &got); err != nil {
		t.Fatalf("Get created configmap: %v", err)
	}
}

func TestIPAtOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cidr    string
		offset  uint32
		want    string
		wantErr bool
	}{
		{"basic", "10.0.0.0/24", 1, "10.0.0.1", false},
		{"networkAddr", "192.168.1.0/24", 0, "192.168.1.0", false},
		{"lastInRange", "192.168.1.0/30", 3, "192.168.1.3", false},
		{"outOfRange", "192.168.1.0/30", 4, "", true},
		{"badCIDR", "not-a-cidr", 1, "", true},
		{"ipv6NotSupported", "2001:db8::/64", 1, "", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ip, err := IPAtOffset(tt.cidr, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ip.String() != tt.want {
				t.Fatalf("got %q want %q", ip.String(), tt.want)
			}
		})
	}
}

func TestMergeStringMap(t *testing.T) {
	t.Parallel()

	t.Run("bothNil", func(t *testing.T) {
		t.Parallel()
		if got := MergeStringMap(nil, nil); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("dstNil", func(t *testing.T) {
		t.Parallel()
		got := MergeStringMap(nil, map[string]string{"a": "1"})
		if got == nil || got["a"] != "1" {
			t.Fatalf("unexpected result: %#v", got)
		}
	})

	t.Run("srcNil", func(t *testing.T) {
		t.Parallel()
		dst := map[string]string{"a": "1"}
		got := MergeStringMap(dst, nil)
		if got["a"] != "1" {
			t.Fatalf("unexpected result: %#v", got)
		}
	})

	t.Run("overwrites", func(t *testing.T) {
		t.Parallel()
		dst := map[string]string{"a": "1", "b": "2"}
		src := map[string]string{"b": "22", "c": "3"}
		got := MergeStringMap(dst, src)
		if got["a"] != "1" || got["b"] != "22" || got["c"] != "3" {
			t.Fatalf("unexpected merge: %#v", got)
		}
	})
}

func TestPortString(t *testing.T) {
	t.Parallel()

	if got := PortString(80); got != "80" {
		t.Fatalf("got %q want %q", got, "80")
	}
	if got := PortString(0); got != "0" {
		t.Fatalf("got %q want %q", got, "0")
	}
}

func TestIntstrFromInt(t *testing.T) {
	t.Parallel()

	got := IntstrFromInt(8080)
	if got.Type != intstr.Int || got.IntValue() != 8080 {
		t.Fatalf("unexpected IntOrString: %#v", got)
	}
}

func TestEnsureCreatedAndOwned_InvalidSchemeFails(t *testing.T) {
	t.Parallel()

	// Intentionally no corev1.AddToScheme here.
	scheme := runtime.NewScheme()
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	owner := newOwnerPod("ns", "owner")
	template := newConfigMapTemplate("ns", "cm")

	err := EnsureCreatedAndOwned(context.Background(), c, scheme, owner, template, logr.Discard(), nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	_ = apierrors.IsInvalid(err)
}
