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

package common

import corev1 "k8s.io/api/core/v1"

type EmptyDirMount struct {
	EmptyDirName string
	MountDir     string
}

func (e EmptyDirMount) Volume() corev1.Volume {
	return corev1.Volume{
		Name:         e.EmptyDirName,
		VolumeSource: corev1.VolumeSource{},
	}
}

func (e EmptyDirMount) Mount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      e.EmptyDirName,
		MountPath: e.MountDir,
	}
}
