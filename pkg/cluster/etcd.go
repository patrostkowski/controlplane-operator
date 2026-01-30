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
	"fmt"

	"github.com/patrostkowski/controlplane-operator/pkg/utils"
)

// ETCDSpec defines the interface for etcd-related configurations and secrets.
type ETCDSpec interface {
	Namespacer
	MountLayout

	Etcd() EtcdConfig
}

type EtcdConfig interface {
	// Naming / identity
	ServiceName() string
	StatefulSetName() string
	MemberName() string

	// Ports
	ClientPort() int32
	PeerPort() int32

	// DNS helpers
	MemberFQDNClient() string
	MemberFQDNPeer() string

	// Data / storage defaults
	DataDir() string
	DefaultStorage() string

	// Secret names (used by resources to mount secrets)
	CASecret() string
	ServerTLSSecret() string
	PeerTLSSecret() string

	// Paths (resources use these directly for etcd flags)
	CAPath() string
	ServerCertPath() string
	ServerKeyPath() string
	PeerCertPath() string
	PeerKeyPath() string
}

var _ EtcdConfig = etcd{}

func (cc ClusterContext) Etcd() EtcdConfig {
	return etcd{cc: cc}
}

// etcd is an internal struct that implements the EtcdConfig interface.
type etcd struct {
	cc ClusterContext
}

// ServiceName returns the name of the etcd service.
func (e etcd) ServiceName() string { return "etcd" }

// StatefulSetName returns the name of the etcd StatefulSet.
func (e etcd) StatefulSetName() string { return "etcd" }

// MemberName returns the name of an etcd member.
func (e etcd) MemberName() string { return "etcd-0" }

// ClientPort returns the client port for etcd.
func (e etcd) ClientPort() int32 { return 2379 }

// PeerPort returns the peer port for etcd.
func (e etcd) PeerPort() int32 { return 2380 }

// ServiceFQDN returns the FQDN of the etcd service.
func (e etcd) ServiceFQDN() string {
	ns := e.cc.Namespace()
	return fmt.Sprintf("%s.%s.svc.cluster.local", e.ServiceName(), ns)
}

// MemberFQDNClient returns the client FQDN for an etcd member.
func (e etcd) MemberFQDNClient() string {
	ns := e.cc.Namespace()
	svc := e.ServiceName()
	return e.MemberName() + "." + svc + "." + ns + ".svc:" + utils.PortString(e.ClientPort())
}

// MemberFQDNPeer returns the peer FQDN for an etcd member.
func (e etcd) MemberFQDNPeer() string {
	ns := e.cc.Namespace()
	svc := e.ServiceName()
	return e.MemberName() + "." + svc + "." + ns + ".svc:" + utils.PortString(e.PeerPort())
}

// DataDir returns the data directory for etcd.
func (e etcd) DataDir() string { return "/var/lib/etcd" }

// DefaultStorage returns the default storage size for etcd.
func (e etcd) DefaultStorage() string { return "10Gi" }

// CASecret returns the name of the secret containing the etcd CA certificate.
func (e etcd) CASecret() string { return e.cc.PKI().Certificate().EtcdCA() }

// ServerTLSSecret returns the name of the secret containing the etcd server TLS certificates.
func (e etcd) ServerTLSSecret() string { return e.cc.PKI().Certificate().EtcdServerTLS() }

// PeerTLSSecret returns the name of the secret containing the etcd peer TLS certificates.
func (e etcd) PeerTLSSecret() string { return e.cc.PKI().Certificate().EtcdPeerTLS() }

// CAPath returns the path to the etcd CA certificate.
func (e etcd) CAPath() string { return e.cc.CAPath(e.CASecret()) }

// ServerCertPath returns the path to the etcd server certificate.
func (e etcd) ServerCertPath() string { return e.cc.CertPath(e.ServerTLSSecret()) }

// ServerKeyPath returns the path to the etcd server key.
func (e etcd) ServerKeyPath() string { return e.cc.KeyPath(e.ServerTLSSecret()) }

// PeerCertPath returns the path to the etcd peer certificate.
func (e etcd) PeerCertPath() string { return e.cc.CertPath(e.PeerTLSSecret()) }

// PeerKeyPath returns the path to the etcd peer key.
func (e etcd) PeerKeyPath() string { return e.cc.KeyPath(e.PeerTLSSecret()) }
