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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	testutil "github.com/patrostkowski/controlplane-operator/internal/test"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildService_Etcd(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "example-mcp", Namespace: "mcp"},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	e := cc.Etcd()

	objs := NewBuilder(cc).Objects()
	svc := testutil.MustFindService(t, objs)

	if svc.Name != e.ServiceName() {
		t.Fatalf("service name=%q want %q", svc.Name, e.ServiceName())
	}
	if svc.Namespace != mcp.Namespace {
		t.Fatalf("service namespace=%q want %q", svc.Namespace, mcp.Namespace)
	}
	if svc.Spec.ClusterIP != "None" {
		t.Fatalf("expected headless ClusterIP=None, got %q", svc.Spec.ClusterIP)
	}
	testutil.ExpectPort(t, svc.Spec.Ports, "client", e.ClientPort())
	testutil.ExpectPort(t, svc.Spec.Ports, "peer", e.PeerPort())
}

func TestBuildStatefulSet_Etcd(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "example-mcp", Namespace: "mcp"},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	e := cc.Etcd()

	objs := NewBuilder(cc).Objects()
	sts := testutil.MustFindStatefulSet(t, objs)

	if sts.Name != e.StatefulSetName() {
		t.Fatalf("statefulset name=%q want %q", sts.Name, e.StatefulSetName())
	}
	if sts.Namespace != mcp.Namespace {
		t.Fatalf("statefulset namespace=%q want %q", sts.Namespace, mcp.Namespace)
	}
	if sts.Spec.ServiceName != e.ServiceName() {
		t.Fatalf("statefulset serviceName=%q want %q", sts.Spec.ServiceName, e.ServiceName())
	}

	if len(sts.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
	}
	c := sts.Spec.Template.Spec.Containers[0]
	if c.Name != e.ServiceName() {
		t.Fatalf("container name=%q want %q", c.Name, e.ServiceName())
	}

	// Important args should reference FQDNs + cert paths.
	testutil.MustContainArg(t, c.Args, "--name="+e.MemberName())
	testutil.MustContainArg(t, c.Args, "--data-dir="+e.DataDir())
	testutil.MustContainArg(t, c.Args, "--advertise-client-urls=https://"+e.MemberFQDNClient())
	testutil.MustContainArg(t, c.Args, "--initial-advertise-peer-urls=https://"+e.MemberFQDNPeer())
	testutil.MustContainArg(t, c.Args, "--trusted-ca-file="+e.CAPath())
	testutil.MustContainArg(t, c.Args, "--cert-file="+e.ServerCertPath())
	testutil.MustContainArg(t, c.Args, "--key-file="+e.ServerKeyPath())

	// Volumes must include three secrets.
	testutil.ExpectVolume(t, sts.Spec.Template.Spec.Volumes, e.CASecret(), func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != e.CASecret() {
			t.Fatalf("expected CA volume secret %q, got %#v", e.CASecret(), v.Secret)
		}
	})
	testutil.ExpectVolume(t, sts.Spec.Template.Spec.Volumes, e.ServerTLSSecret(), func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != e.ServerTLSSecret() {
			t.Fatalf("expected server TLS volume secret %q, got %#v", e.ServerTLSSecret(), v.Secret)
		}
	})
	testutil.ExpectVolume(t, sts.Spec.Template.Spec.Volumes, e.PeerTLSSecret(), func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != e.PeerTLSSecret() {
			t.Fatalf("expected peer TLS volume secret %q, got %#v", e.PeerTLSSecret(), v.Secret)
		}
	})

	// There must be one PVC named after StatefulSetName with default storage.
	if got := len(sts.Spec.VolumeClaimTemplates); got != 1 {
		t.Fatalf("expected 1 PVC template, got %d", got)
	}
	pvc := sts.Spec.VolumeClaimTemplates[0]
	if pvc.Name != e.StatefulSetName() {
		t.Fatalf("pvc name=%q want %q", pvc.Name, e.StatefulSetName())
	}
	q := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if q.String() != e.DefaultStorage() {
		t.Fatalf("pvc storage=%q want %q", q.String(), e.DefaultStorage())
	}

	// dataDir mount should exist and be RW.
	found := false
	for _, m := range c.VolumeMounts {
		if m.MountPath == dataDir {
			found = true
			if m.ReadOnly {
				t.Fatalf("expected data dir mount to be readWrite")
			}
		}
	}
	if !found {
		t.Fatalf("expected volumeMount for dataDir %q", dataDir)
	}

	// Sanity: initial-cluster contains the member name.
	ok := false
	for _, a := range c.Args {
		if strings.HasPrefix(a, "--initial-cluster=") && strings.Contains(a, e.ServiceName()) {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("expected --initial-cluster arg to include %q; args=%#v", e.ServiceName(), c.Args)
	}
}
