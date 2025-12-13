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

package v1alpha1

import (
	"github.com/patrostkowski/controlplane-operator/pkg/controlplane"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *ManagedControlPlane) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedPKI) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedETCD) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedAPIServer) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedControllerManager) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedScheduler) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedAddon) GetConditions() *[]metav1.Condition {
	return &m.Status.Conditions
}

func (m *ManagedPKI) GetStatus() *controlplane.Status {
	return &m.Status.Status
}

func (m *ManagedETCD) GetStatus() *controlplane.Status {
	return &m.Status.Status
}

func (m *ManagedAPIServer) GetStatus() *controlplane.Status {
	return &m.Status.Status
}

func (m *ManagedControllerManager) GetStatus() *controlplane.Status {
	return &m.Status.Status
}

func (m *ManagedScheduler) GetStatus() *controlplane.Status {
	return &m.Status.Status
}

func (m *ManagedControlPlane) GetStatus() *controlplane.Status {
	return &m.Status.Status
}

func (m *ManagedAddon) GetStatus() *controlplane.Status {
	return &m.Status.Status
}
