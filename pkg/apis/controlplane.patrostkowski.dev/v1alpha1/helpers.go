package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
