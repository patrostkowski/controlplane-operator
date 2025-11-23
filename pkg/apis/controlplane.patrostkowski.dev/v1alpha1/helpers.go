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
