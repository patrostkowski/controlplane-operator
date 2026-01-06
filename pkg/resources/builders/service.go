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

package builders

import (
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ServiceTemplate struct {
	*corev1.Service
	meta MetaMutator
}

func NewService() *ServiceTemplate {
	obj := &corev1.Service{}
	b := &ServiceTemplate{Service: obj}
	b.meta = MetaMutator{obj: obj}
	b.Spec = corev1.ServiceSpec{
		Ports:    []corev1.ServicePort{},
		Selector: map[string]string{},
		Type:     corev1.ServiceTypeClusterIP,
	}
	return b
}

func (s *ServiceTemplate) GetMeta() *metav1.ObjectMeta {
	return &s.Service.ObjectMeta
}

func (s *ServiceTemplate) WithLabels(labels map[string]string) *ServiceTemplate {
	s.meta.WithLabels(labels)
	return s
}

func (s *ServiceTemplate) WithAnnotations(ann map[string]string) *ServiceTemplate {
	s.meta.WithAnnotations(ann)
	return s
}

func (s *ServiceTemplate) WithName(name string) *ServiceTemplate {
	s.meta.WithName(name)
	return s
}

func (s *ServiceTemplate) WithNamespace(ns string) *ServiceTemplate {
	s.meta.WithNamespace(ns)
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

func (s *ServiceTemplate) WithClusterIP(ipaddr net.IP) *ServiceTemplate {
	s.Spec.ClusterIP = ipaddr.String()
	return s
}

// Useful for etcd headless Service
func (s *ServiceTemplate) Headless() *ServiceTemplate {
	s.Spec.ClusterIP = corev1.ClusterIPNone
	// K8s typically also sets ClusterIPs=["None"] in newer APIs; that’s handled server-side.
	return s
}

func (s *ServiceTemplate) AddPorts(ports []corev1.ServicePort) *ServiceTemplate {
	s.Spec.Ports = append(s.Spec.Ports, ports...)
	return s
}

// Convenience: port with targetPort as int
func (s *ServiceTemplate) AddPort(name string, port int32, targetPort int32, protocol corev1.Protocol) *ServiceTemplate {
	return s.AddPorts([]corev1.ServicePort{
		{
			Name:       name,
			Port:       port,
			TargetPort: intstr.FromInt(int(targetPort)),
			Protocol:   protocol,
		},
	})
}

func (s *ServiceTemplate) Build() *corev1.Service {
	return s.Service.DeepCopy()
}
