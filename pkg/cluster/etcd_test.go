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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEtcdConfig_NamesPortsFQDNAndPaths(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"}}
	cc := NewClusterContext(mcp, logr.Logger{})
	e := cc.Etcd()

	if got, want := e.ServiceName(), "demo-etcd"; got != want {
		t.Fatalf("ServiceName()=%q want %q", got, want)
	}
	if got, want := e.StatefulSetName(), "demo-etcd"; got != want {
		t.Fatalf("StatefulSetName()=%q want %q", got, want)
	}
	if got, want := e.MemberName(), "demo-etcd-0"; got != want {
		t.Fatalf("MemberName()=%q want %q", got, want)
	}
	if got, want := e.ClientPort(), int32(2379); got != want {
		t.Fatalf("ClientPort()=%d want %d", got, want)
	}
	if got, want := e.PeerPort(), int32(2380); got != want {
		t.Fatalf("PeerPort()=%d want %d", got, want)
	}

	// DNS helpers are part of the public contract:
	if got, want := e.MemberFQDNClient(), "demo-etcd-0.demo-etcd.ns.svc:2379"; got != want {
		t.Fatalf("MemberFQDNClient()=%q want %q", got, want)
	}
	if got, want := e.MemberFQDNPeer(), "demo-etcd-0.demo-etcd.ns.svc:2380"; got != want {
		t.Fatalf("MemberFQDNPeer()=%q want %q", got, want)
	}

	// Paths should be composed from mount layout + secret names
	if got, want := e.CAPath(), "/var/run/k8s/demo-etcd-ca/ca.crt"; got != want {
		t.Fatalf("CAPath()=%q want %q", got, want)
	}
	if got, want := e.ServerCertPath(), "/var/run/k8s/demo-etcd-server-tls/tls.crt"; got != want {
		t.Fatalf("ServerCertPath()=%q want %q", got, want)
	}
	if got, want := e.PeerKeyPath(), "/var/run/k8s/demo-etcd-peer-tls/tls.key"; got != want {
		t.Fatalf("PeerKeyPath()=%q want %q", got, want)
	}
}
