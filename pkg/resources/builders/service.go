package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ServiceTemplate struct {
	*corev1.Service
}

func NewService(ns, name string) *ServiceTemplate {
	return &ServiceTemplate{
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: corev1.ServiceSpec{
				Ports:    []corev1.ServicePort{},
				Selector: map[string]string{},
				Type:     corev1.ServiceTypeClusterIP,
			},
		},
	}
}

func (s *ServiceTemplate) WithLabels(labels map[string]string) *ServiceTemplate {
	if s.Labels == nil {
		s.Labels = map[string]string{}
	}
	for k, v := range labels {
		s.Labels[k] = v
	}
	return s
}

func (s *ServiceTemplate) WithAnnotations(ann map[string]string) *ServiceTemplate {
	if s.Annotations == nil {
		s.Annotations = map[string]string{}
	}
	for k, v := range ann {
		s.Annotations[k] = v
	}
	return s
}

func (s *ServiceTemplate) WithSelector(sel map[string]string) *ServiceTemplate {
	if s.Spec.Selector == nil {
		s.Spec.Selector = map[string]string{}
	}
	for k, v := range sel {
		s.Spec.Selector[k] = v
	}
	return s
}

func (s *ServiceTemplate) WithType(t corev1.ServiceType) *ServiceTemplate {
	s.Spec.Type = t
	return s
}

// Useful for etcd headless Service
func (s *ServiceTemplate) Headless() *ServiceTemplate {
	s.Spec.ClusterIP = corev1.ClusterIPNone
	// K8s typically also sets ClusterIPs=["None"] in newer APIs; that’s handled server-side.
	return s
}

func (s *ServiceTemplate) AddPorts(ports ...corev1.ServicePort) *ServiceTemplate {
	s.Spec.Ports = append(s.Spec.Ports, ports...)
	return s
}

// Convenience: port with targetPort as int
func (s *ServiceTemplate) AddPort(name string, port int32, targetPort int32, protocol corev1.Protocol) *ServiceTemplate {
	s.Spec.Ports = append(s.Spec.Ports, corev1.ServicePort{
		Name:       name,
		Port:       port,
		TargetPort: intstr.FromInt(int(targetPort)),
		Protocol:   protocol,
	})
	return s
}

func (s *ServiceTemplate) Build() *corev1.Service {
	return s.Service.DeepCopy()
}
