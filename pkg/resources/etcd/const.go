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

const (
	appLabelKey = "app"
	appLabelVal = "etcd"

	nameEtcd = "etcd"

	clientPort int32 = 2379
	peerPort   int32 = 2380

	dataDir = "/var/lib/etcd"

	mountRoot = "/etc/etcd/pki"
	dirCA     = "ca"
	dirServer = "server"
	dirPeer   = "peer"

	caCrt  = "ca.crt"
	tlsCrt = "tls.crt"
	tlsKey = "tls.key"

	defaultStorage = "10Gi"

	// Member config (single node for now)
	memberName  = "etcd-0"
	clusterName = "etcd-0"
)
