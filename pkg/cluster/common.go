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
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectProducer is implemented by components that produce one or more Kubernetes objects to be applied.
type ObjectProducer interface {
	Objects() []client.Object
}

// Namespacer provides a stable name and namespace for generated resources.
type Namespacer interface {
	Namespace() string
	Name() string
}

// MountLayout defines filesystem paths and volume helpers for mounting Secret-based PKI material.
type MountLayout interface {
	CertPath(secretName string) string
	KeyPath(secretName string) string
	CAPath(secretName string) string

	SecretVolume(secretName string) corev1.Volume
	SecretMount(secretName string, readOnly bool) corev1.VolumeMount
}
