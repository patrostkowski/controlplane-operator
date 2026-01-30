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

package scheduler

import (
	"strings"
	"testing"

	"github.com/go-logr/logr"
	testutil "github.com/patrostkowski/controlplane-operator/internal/test"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildConfigMap_Scheduler(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-cp", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
			},
		},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	s := cc.Scheduler()

	objs := NewBuilder(cc).Objects()
	cfg := testutil.MustFindConfigMap(t, objs)

	if cfg.Name != s.KubeconfigConfigMapName() {
		t.Fatalf("configmap name=%q want %q", cfg.Name, s.KubeconfigConfigMapName())
	}
	if cfg.Namespace != mcp.Namespace {
		t.Fatalf("configmap namespace=%q want %q", cfg.Namespace, mcp.Namespace)
	}

	val := strings.TrimSpace(cfg.Data[schedulerKubeconfigKey])
	if val == "" {
		t.Fatalf("expected configmap data[%q] to be non-empty", schedulerKubeconfigKey)
	}
	if !strings.Contains(val, "apiVersion: v1") {
		t.Fatalf("expected kubeconfig to contain apiVersion: v1, got:\n%s", val)
	}
}

func TestBuildDeployment_KubeScheduler(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-cp", Namespace: "mcp"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{Version: "v1.34.1"},
		},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	s := cc.Scheduler()

	objs := NewBuilder(cc).Objects()
	dep := testutil.MustFindDeployment(t, objs)

	if dep.Name != s.DeploymentName() {
		t.Fatalf("deployment name=%q want %q", dep.Name, s.DeploymentName())
	}
	if dep.Namespace != mcp.Namespace {
		t.Fatalf("deployment namespace=%q want %q", dep.Namespace, mcp.Namespace)
	}

	if len(dep.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected exactly 1 container, got %d", len(dep.Spec.Template.Spec.Containers))
	}
	c := dep.Spec.Template.Spec.Containers[0]

	clusterCA := s.ClusterCASecret()
	client := s.ClientCertSecret()

	testutil.MustContainArg(t, c.Args, "--kubeconfig="+kubeconfigPath)
	testutil.MustContainArg(t, c.Args, "--authentication-kubeconfig="+kubeconfigPath)
	testutil.MustContainArg(t, c.Args, "--authorization-kubeconfig="+kubeconfigPath)

	testutil.ExpectVolume(t, dep.Spec.Template.Spec.Volumes, clusterCA, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != clusterCA {
			t.Fatalf("volume %q expected secret %q, got %#v", clusterCA, clusterCA, v.Secret)
		}
	})
	testutil.ExpectVolume(t, dep.Spec.Template.Spec.Volumes, client, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != client {
			t.Fatalf("volume %q expected secret %q, got %#v", client, client, v.Secret)
		}
	})

	testutil.ExpectMount(t, c.VolumeMounts, kubeconfigVolumeName, kubeconfigMountDir, true)
	testutil.ExpectMount(t, c.VolumeMounts, clusterCA, cc.SecretMountDir(clusterCA), true)
	testutil.ExpectMount(t, c.VolumeMounts, client, cc.SecretMountDir(client), true)
}
