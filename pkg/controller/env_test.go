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
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// SchemeForControllerTests returns a scheme with all APIs used by controllers.
func SchemeForControllerTests(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	if err := mcpv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("add MCP scheme failed: %v", err)
	}
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("add corev1 scheme failed: %v", err)
	}
	if err := appsv1.AddToScheme(s); err != nil {
		t.Fatalf("add appsv1 scheme failed: %v", err)
	}
	if err := certmanagerv1.AddToScheme(s); err != nil {
		t.Fatalf("add cert-manager scheme failed: %v", err)
	}

	return s
}

// BaseReconcilerForTests returns a BaseReconciler suitable for unit tests.
func BaseReconcilerForTests(t *testing.T, scheme *runtime.Scheme) BaseReconciler {
	t.Helper()

	return BaseReconciler{
		Log:    ctrl.Log.WithName("test"),
		Scheme: scheme,
	}
}

// FakeClientForTests creates a controller-runtime fake client.
func FakeClientForTests(
	t *testing.T,
	scheme *runtime.Scheme,
	objects ...client.Object,
) client.Client {
	t.Helper()

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objects) > 0 {
		builder = builder.WithObjects(objects...)
	}
	return builder.Build()
}

// StartEnvTestForControllers starts envtest and returns a real API client.
func StartEnvTestForControllers(t *testing.T) (*envtest.Environment, *runtime.Scheme, client.Client) {
	t.Helper()

	scheme := SchemeForControllerTests(t)

	env := &envtest.Environment{}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("envtest start failed: %v", err)
	}

	t.Cleanup(func() {
		_ = env.Stop()
	})

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("creating envtest client failed: %v", err)
	}

	return env, scheme, c
}
