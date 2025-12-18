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

package addons

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CoreDNSNamespaceName = "kube-system"

	CoreDNSServiceAccountName = "coredns"

	CoreDNSConfigMapName = "coredns-config"

	CoreDNSDeploymentName = "coredns"

	CoreDNSServiceName = "kube-dns"

	CoreDNSClusterRoleName        = "system:coredns"
	CoreDNSClusterRoleBindingName = "system:coredns"

	// NEW: configmap read RBAC
	CoreDNSRoleName        = "coredns"
	CoreDNSRoleBindingName = "coredns"

	CoreDNSImage = "registry.k8s.io/coredns/coredns:v1.11.3"

	CoreDNSReplicas int32 = 2

	CoreDNSPort        = 53
	CoreDNSMetricsPort = 9153
)

// labels used by SA + Deployment + Pod template
var CoreDNSPodLabels = map[string]string{
	"k8s-app": "coredns",
}

// service labels (kube-dns)
var CoreDNSServiceLabels = map[string]string{
	"k8s-app": "kube-dns",
}

var CoreDNSLabels = map[string]string{
	"k8s-app": "kube-dns",
}

func buildCoreDNS(ma *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildCoreDNSServiceAccount(),
		buildCoreDNSClusterRole(),
		buildCoreDNSClusterRoleBinding(),
		buildCoreDNSRole(),
		buildCoreDNSRoleBinding(),
		buildCoreDNSConfigMap(),
		buildCoreDNSService(ma),
		buildCoreDNSDeployment(),
	}
}

func buildCoreDNSServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSServiceAccountName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSPodLabels,
		},
	}
}

func buildCoreDNSClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CoreDNSClusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     CoreDNSClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      CoreDNSServiceAccountName,
				Namespace: CoreDNSNamespaceName,
			},
		},
	}
}

func buildCoreDNSClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{
			Name: CoreDNSClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints", "services", "pods", "namespaces"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func buildCoreDNSRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSRoleName,
			Namespace: CoreDNSNamespaceName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{CoreDNSConfigMapName},
				Verbs:         []string{"get", "list", "watch"},
			},
		},
	}
}

func buildCoreDNSRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSRoleBindingName,
			Namespace: CoreDNSNamespaceName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     CoreDNSRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      CoreDNSServiceAccountName,
				Namespace: CoreDNSNamespaceName,
			},
		},
	}
}

func buildCoreDNSConfigMap() *corev1.ConfigMap {
	corefile := `.:53 {
        errors
        health
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        prometheus :9153
        forward . 8.8.8.8 1.1.1.1
        cache 30
        loop
        reload
        loadbalance
    }`

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSConfigMapName,
			Namespace: CoreDNSNamespaceName,
		},
		Data: map[string]string{"Corefile": corefile},
	}
}

func buildCoreDNSService(ma *mcpv1alpha1.ManagedControlPlane) *corev1.Service {
	ip, _ := utils.IPAtOffset(ma.Spec.Networking.ServiceCIDR, 10)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSServiceName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSServiceLabels,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: ip.String(),
			Selector:  CoreDNSPodLabels,
			Ports: []corev1.ServicePort{
				{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP, TargetPort: intstr.FromInt(53)},
				{Name: "dns-tcp", Port: 53, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(53)},
			},
		},
	}
}

func buildCoreDNSDeployment() *appsv1.Deployment {
	replicas := CoreDNSReplicas

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSDeploymentName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSPodLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: CoreDNSPodLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: CoreDNSPodLabels},
				Spec: corev1.PodSpec{
					ServiceAccountName: CoreDNSServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:            "coredns",
							Image:           CoreDNSImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            []string{"-conf", "/etc/coredns/Corefile"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 53, Protocol: corev1.ProtocolUDP},
								{ContainerPort: 53, Protocol: corev1.ProtocolTCP},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/coredns"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: CoreDNSConfigMapName},
								},
							},
						},
					},
				},
			},
		},
	}
}
