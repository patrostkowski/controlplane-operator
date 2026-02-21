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

package cluster

import (
	"testing"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCC() *ClusterContext {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "demo-ns"},
		Spec:       mcpv1alpha1.ManagedControlPlaneSpec{Kubernetes: mcpv1alpha1.KubernetesSpec{Version: "v1.34.0"}},
		Status:     mcpv1alpha1.ManagedControlPlaneStatus{Address: "192.0.2.10"},
	}
	return NewClusterContext(mcp, logr.Logger{})
}

func TestClusterContext_IdentityAndOwner(t *testing.T) {
	cc := newCC()

	if cc.Name() != "demo" {
		t.Fatalf("Name()=%q want %q", cc.Name(), "demo")
	}
	if cc.Namespace() != "demo-ns" {
		t.Fatalf("Namespace()=%q want %q", cc.Namespace(), "demo-ns")
	}

	if cc.Owner() == nil {
		t.Fatalf("Owner() must not be nil")
	}
	if cc.MCP() == nil {
		t.Fatalf("MCP() must not be nil")
	}

	if got := cc.GetManagedControlPlaneSpec().Kubernetes.Version; got != "v1.34.0" {
		t.Fatalf("spec.kubernetes.version=%q want %q", got, "v1.34.0")
	}
	if got := cc.GetManagedControlPlaneStatus().Address; got != "192.0.2.10" {
		t.Fatalf("status.address=%q want %q", got, "192.0.2.10")
	}
}

func TestClusterContext_SecretMountPaths(t *testing.T) {
	cc := newCC()
	sec := "my-secret"

	if got, want := cc.SecretMountDir(sec), "/var/run/k8s/my-secret"; got != want {
		t.Fatalf("SecretMountDir()=%q want %q", got, want)
	}
	if got, want := cc.CertPath(sec), "/var/run/k8s/my-secret/tls.crt"; got != want {
		t.Fatalf("CertPath()=%q want %q", got, want)
	}
	if got, want := cc.KeyPath(sec), "/var/run/k8s/my-secret/tls.key"; got != want {
		t.Fatalf("KeyPath()=%q want %q", got, want)
	}
	if got, want := cc.CAPath(sec), "/var/run/k8s/my-secret/ca.crt"; got != want {
		t.Fatalf("CAPath()=%q want %q", got, want)
	}
}

func TestClusterContext_SecretVolumeAndMount(t *testing.T) {
	cc := newCC()
	sec := "pki-material"

	v := cc.SecretVolume(sec)
	if v.Name != sec {
		t.Fatalf("volume.name=%q want %q", v.Name, sec)
	}
	if v.VolumeSource.Secret == nil || v.VolumeSource.Secret.SecretName != sec {
		t.Fatalf("volume.secret.secretName=%v want %q", v.VolumeSource.Secret, sec)
	}

	m := cc.SecretMount(sec, true)
	if m.Name != sec {
		t.Fatalf("mount.name=%q want %q", m.Name, sec)
	}
	if !m.ReadOnly {
		t.Fatalf("mount.readOnly=false want true")
	}
	if got, want := m.MountPath, "/var/run/k8s/pki-material"; got != want {
		t.Fatalf("mount.mountPath=%q want %q", got, want)
	}

	// Ensure API stays stable wrt corev1 types
	_ = corev1.Volume(v)
	_ = corev1.VolumeMount(m)
}
