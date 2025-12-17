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

package apiserver

const (
	KubeAPIServerSvcName = apiServerName

	apiServerName = "kube-apiserver"
	appLabelKey   = "app"
	appLabelVal   = "kube-apiserver"

	securePort int32 = 6443

	// All certs are mounted under this root, with subdirs per secret/volume.
	mountRoot = "/var/run/k8s"

	// cert-manager Secret keys (your certs are created with tls.crt/tls.key; CA secret also uses tls.crt here)
	tlsCrt = "tls.crt"
	tlsKey = "tls.key"
)
