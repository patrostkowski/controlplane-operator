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

package etcd

import (
	"testing"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildService_Etcd(t *testing.T) {
	cp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-mcp",
			Namespace: "mcp",
		},
	}

	svc := buildService(cp)

	// metadata
	if svc.Name != nameEtcd {
		t.Fatalf("expected service name %q, got %q", nameEtcd, svc.Name)
	}
	if svc.Namespace != cp.Namespace {
		t.Fatalf("expected service namespace %q, got %q", cp.Namespace, svc.Namespace)
	}

	// labels
	wantLabels := map[string]string{appLabelKey: appLabelVal}
	if svc.Labels == nil || svc.Labels[appLabelKey] != appLabelVal {
		t.Fatalf("expected labels %#v, got %#v", wantLabels, svc.Labels)
	}

	// selector should match labels (critical for endpoints)
	if svc.Spec.Selector == nil || svc.Spec.Selector[appLabelKey] != appLabelVal {
		t.Fatalf("expected selector %#v, got %#v", wantLabels, svc.Spec.Selector)
	}

	// headless service
	if svc.Spec.ClusterIP != "None" {
		t.Fatalf("expected ClusterIP %q (headless), got %q", "None", svc.Spec.ClusterIP)
	}

	// ports
	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d: %#v", len(svc.Spec.Ports), svc.Spec.Ports)
	}

	expectPort(t, svc.Spec.Ports, "client", clientPort)
	expectPort(t, svc.Spec.Ports, "peer", peerPort)

	// protocol defaults to TCP if empty, don't be strict here
	for _, p := range svc.Spec.Ports {
		if p.Protocol != "" && p.Protocol != corev1.ProtocolTCP {
			t.Fatalf("expected protocol TCP/empty-default, got %q for port %#v", p.Protocol, p)
		}
	}
}

func TestBuildStatefulSet_Etcd(t *testing.T) {
	cp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-mcp",
			Namespace: "mcp",
		},
	}

	p := pki.New(cp).ETCD()
	sts := buildStatefulSet(cp)
	if sts == nil {
		t.Fatalf("buildStatefulSet returned nil")
	}

	// metadata
	if sts.Name != nameEtcd {
		t.Fatalf("metadata.name: got %q want %q", sts.Name, nameEtcd)
	}
	if sts.Namespace != cp.Namespace {
		t.Fatalf("metadata.namespace: got %q want %q", sts.Namespace, cp.Namespace)
	}
	if sts.Labels == nil || sts.Labels[appLabelKey] != appLabelVal {
		t.Fatalf("metadata.labels: got %#v want app=%q", sts.Labels, appLabelVal)
	}

	// spec basics
	if sts.Spec.ServiceName != nameEtcd {
		t.Fatalf("spec.serviceName: got %q want %q", sts.Spec.ServiceName, nameEtcd)
	}
	if sts.Spec.Selector == nil || sts.Spec.Selector.MatchLabels[appLabelKey] != appLabelVal {
		t.Fatalf("spec.selector.matchLabels: got %#v want app=%q", sts.Spec.Selector, appLabelVal)
	}
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != 1 {
		t.Fatalf("spec.replicas: got %#v want 1", sts.Spec.Replicas)
	}

	// pod template
	pod := sts.Spec.Template.Spec
	if len(pod.Containers) != 1 {
		t.Fatalf("pod.spec.containers: got %d want 1", len(pod.Containers))
	}

	c := pod.Containers[0]
	if c.Name != nameEtcd {
		t.Fatalf("container.name: got %q want %q", c.Name, nameEtcd)
	}
	if c.Image == "" {
		t.Fatalf("container.image should not be empty")
	}

	// key args exist (keep these stable + important)
	mustContainArg(t, c.Args, "--name="+memberName)
	mustContainArg(t, c.Args, "--data-dir="+dataDir)

	mustContainArg(t, c.Args, "--trusted-ca-file="+p.CA.CAPath())
	mustContainArg(t, c.Args, "--cert-file="+p.Server.CertPath())
	mustContainArg(t, c.Args, "--key-file="+p.Server.KeyPath())

	mustContainArg(t, c.Args, "--peer-trusted-ca-file="+p.CA.CAPath())
	mustContainArg(t, c.Args, "--peer-cert-file="+p.Peer.CertPath())
	mustContainArg(t, c.Args, "--peer-key-file="+p.Peer.KeyPath())

	// probes
	if c.LivenessProbe == nil {
		t.Fatalf("expected livenessProbe to be set")
	}
	if c.ReadinessProbe == nil {
		t.Fatalf("expected readinessProbe to be set")
	}

	// pvc template
	if len(sts.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("expected 1 volumeClaimTemplate, got %d", len(sts.Spec.VolumeClaimTemplates))
	}
	vct := sts.Spec.VolumeClaimTemplates[0]
	if vct.Name != "etcd-data" {
		t.Fatalf("volumeClaimTemplate.name: got %q want %q", vct.Name, "etcd-data")
	}
	qty := vct.Spec.Resources.Requests[corev1.ResourceStorage]
	if qty.IsZero() {
		t.Fatalf("expected storage request to be set")
	}
	if qty.Cmp(resource.MustParse(defaultStorage)) != 0 {
		t.Fatalf("storage request: got %s want %s", qty.String(), defaultStorage)
	}

	// mounts
	expectMount(t, c.VolumeMounts, "etcd-data", dataDir, false) // data dir not readOnly
	expectMount(t, c.VolumeMounts, p.CA.SecretName, p.CA.MountDir, true)
	expectMount(t, c.VolumeMounts, p.Server.SecretName, p.Server.MountDir, true)
	expectMount(t, c.VolumeMounts, p.Peer.SecretName, p.Peer.MountDir, true)

	// volumes
	expectVolume(t, pod.Volumes, p.CA.SecretName, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != p.CA.SecretName {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", p.CA.SecretName, p.CA.SecretName, v)
		}
	})
	expectVolume(t, pod.Volumes, p.Server.SecretName, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != p.Server.SecretName {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", p.Server.SecretName, p.Server.SecretName, v)
		}
	})
	expectVolume(t, pod.Volumes, p.Peer.SecretName, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != p.Peer.SecretName {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", p.Peer.SecretName, p.Peer.SecretName, v)
		}
	})
}

func expectPort(t *testing.T, ports []corev1.ServicePort, name string, port int32) {
	t.Helper()
	for _, p := range ports {
		if p.Name == name {
			if p.Port != port {
				t.Fatalf("port %q: expected %d, got %d", name, port, p.Port)
			}
			return
		}
	}
	t.Fatalf("expected port %q not found; ports=%#v", name, ports)
}

func mustContainArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Fatalf("expected args to contain %q, got %#v", want, args)
}

func expectMount(t *testing.T, mounts []corev1.VolumeMount, name, path string, readOnly bool) {
	t.Helper()
	for _, m := range mounts {
		if m.Name == name {
			if m.MountPath != path {
				t.Fatalf("mount %q: expected path %q, got %q", name, path, m.MountPath)
			}
			if m.ReadOnly != readOnly {
				t.Fatalf("mount %q: expected readOnly=%v, got %v", name, readOnly, m.ReadOnly)
			}
			return
		}
	}
	t.Fatalf("expected volumeMount %q (path=%q) not found; mounts=%#v", name, path, mounts)
}

func expectVolume(t *testing.T, vols []corev1.Volume, name string, validate func(corev1.Volume)) {
	t.Helper()
	for _, v := range vols {
		if v.Name == name {
			validate(v)
			return
		}
	}
	t.Fatalf("expected volume %q not found; volumes=%#v", name, vols)
}
