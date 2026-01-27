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
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

func (cc *ClusterContext) SecretMountDir(secretName string) string {
	return filepath.Join(cc.Layout.PKIRoot, secretName)
}
func (cc *ClusterContext) CertPath(secretName string) string {
	return filepath.Join(cc.SecretMountDir(secretName), cc.Keys.TLSCrt)
}
func (cc *ClusterContext) KeyPath(secretName string) string {
	return filepath.Join(cc.SecretMountDir(secretName), cc.Keys.TLSKey)
}
func (cc *ClusterContext) CAPath(secretName string) string {
	return filepath.Join(cc.SecretMountDir(secretName), cc.Keys.CACrt)
}

func (cc *ClusterContext) SecretVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	}
}

func (cc *ClusterContext) SecretMount(secretName string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      secretName,
		ReadOnly:  readOnly,
		MountPath: cc.SecretMountDir(secretName),
	}
}
