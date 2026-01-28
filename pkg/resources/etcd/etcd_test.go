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

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBuildService_Etcd(t *testing.T) {
	cp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-mcp",
			Namespace: "mcp",
		},
	}

	cc := cluster.NewClusterContext(cp, logr.Logger{})
	objs := NewBuilder(cc).Objects()

	svc := mustFindService(t, objs)

	if svc.Name != nameEtcd {
		t.Fatalf("expected service name %q, got %q", nameEtcd, svc.Name)
	}
	if svc.Namespace != cp.Namespace {
		t.Fatalf("expected service namespace %q, got %q", cp.Namespace, svc.Namespace)
	}

	wantLabels := map[string]string{appLabelKey: appLabelVal}
	if svc.Labels == nil || svc.Labels[appLabelKey] != appLabelVal {
		t.Fatalf("expected labels %#v, got %#v", wantLabels, svc.Labels)
	}

	if svc.Spec.Selector == nil || svc.Spec.Selector[appLabelKey] != appLabelVal {
		t.Fatalf("expected selector %#v, got %#v", wantLabels, svc.Spec.Selector)
	}

	if svc.Spec.ClusterIP != "None" {
		t.Fatalf("expected ClusterIP %q (headless), got %q", "None", svc.Spec.ClusterIP)
	}

	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d: %#v", len(svc.Spec.Ports), svc.Spec.Ports)
	}

	expectPort(t, svc.Spec.Ports, "client", clientPort)
	expectPort(t, svc.Spec.Ports, "peer", peerPort)

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

	cc := cluster.NewClusterContext(cp, logr.Logger{})
	objs := NewBuilder(cc).Objects()

	sts := mustFindStatefulSet(t, objs)

	// --- PKI secret names from cc ---
	etcdCA := cc.Names.SecretEtcdCAName()
	etcdServer := cc.Names.SecretEtcdServerTLSName()
	etcdPeer := cc.Names.SecretEtcdPeerTLSName()

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

	// PKI args must match cc paths now
	mustContainArg(t, c.Args, "--trusted-ca-file="+cc.CAPath(etcdCA))
	mustContainArg(t, c.Args, "--cert-file="+cc.CertPath(etcdServer))
	mustContainArg(t, c.Args, "--key-file="+cc.KeyPath(etcdServer))

	mustContainArg(t, c.Args, "--peer-trusted-ca-file="+cc.CAPath(etcdCA))
	mustContainArg(t, c.Args, "--peer-cert-file="+cc.CertPath(etcdPeer))
	mustContainArg(t, c.Args, "--peer-key-file="+cc.KeyPath(etcdPeer))

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

	// your previous tests expected this pvc name
	if vct.Name != cc.Names.EtcdStatefulSetName() {
		t.Fatalf("volumeClaimTemplate.name: got %q want %q", vct.Name, cc.Names.EtcdStatefulSetName())
	}

	qty := vct.Spec.Resources.Requests[corev1.ResourceStorage]
	if qty.IsZero() {
		t.Fatalf("expected storage request to be set")
	}
	if qty.Cmp(resource.MustParse(defaultStorage)) != 0 {
		t.Fatalf("storage request: got %s want %s", qty.String(), defaultStorage)
	}

	// mounts
	expectMount(t, c.VolumeMounts, cc.Names.EtcdStatefulSetName(), dataDir, false) // data dir not readOnly
	expectMount(t, c.VolumeMounts, etcdCA, cc.SecretMountDir(etcdCA), true)
	expectMount(t, c.VolumeMounts, etcdServer, cc.SecretMountDir(etcdServer), true)
	expectMount(t, c.VolumeMounts, etcdPeer, cc.SecretMountDir(etcdPeer), true)

	// volumes
	expectVolume(t, pod.Volumes, etcdCA, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != etcdCA {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", etcdCA, etcdCA, v)
		}
	})
	expectVolume(t, pod.Volumes, etcdServer, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != etcdServer {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", etcdServer, etcdServer, v)
		}
	})
	expectVolume(t, pod.Volumes, etcdPeer, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != etcdPeer {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", etcdPeer, etcdPeer, v)
		}
	})
}

func mustFindService(t *testing.T, objs []client.Object) *corev1.Service {
	t.Helper()
	for _, o := range objs {
		if s, ok := o.(*corev1.Service); ok {
			return s
		}
	}
	t.Fatalf("expected *corev1.Service in objects, got %#v", objs)
	return nil
}

func mustFindStatefulSet(t *testing.T, objs []client.Object) *appsv1.StatefulSet {
	t.Helper()
	for _, o := range objs {
		if s, ok := o.(*appsv1.StatefulSet); ok {
			return s
		}
	}
	t.Fatalf("expected *appsv1.StatefulSet in objects, got %#v", objs)
	return nil
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
