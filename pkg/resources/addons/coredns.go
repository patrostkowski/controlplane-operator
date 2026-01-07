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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	return builders.NewServiceAccount().
		WithName(CoreDNSServiceAccountName).
		WithNamespace(CoreDNSNamespaceName).
		WithLabels(CoreDNSPodLabels).
		Build()
}

func buildCoreDNSClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(CoreDNSClusterRoleBindingName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: common.RBACAPIGroup,
				Kind:     common.KindClusterRole,
				Name:     CoreDNSClusterRoleName,
			},

			rbacv1.Subject{
				Kind:      common.KindServiceAccount,
				Name:      CoreDNSServiceAccountName,
				Namespace: CoreDNSNamespaceName,
			},
		).
		Build()
}

func buildCoreDNSClusterRole() *rbacv1.ClusterRole {
	return builders.NewClusterRole().
		WithName(CoreDNSClusterRoleName).
		WithRules(
			rbacv1.PolicyRule{
				APIGroups: []string{common.CoreAPIGroup},
				Resources: []string{
					common.ResourceEndpoints,
					common.ResourceServices,
					common.ResourcePods,
					common.ResourceNamespaces,
				},
				Verbs: []string{
					common.VerbList,
					common.VerbWatch,
				},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{common.DiscoveryAPIGroup},
				Resources: []string{common.ResourceEndpointSlices},
				Verbs: []string{
					common.VerbList,
					common.VerbWatch,
				},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{common.CoreAPIGroup},
				Resources: []string{common.ResourceNodes},
				Verbs: []string{
					common.VerbGet,
					common.VerbList,
					common.VerbWatch,
				},
			},
		).
		Build()
}

func buildCoreDNSRole() *rbacv1.Role {
	return builders.NewRole().
		WithName(CoreDNSRoleName).
		WithNamespace(CoreDNSNamespaceName).
		WithRules(
			rbacv1.PolicyRule{
				APIGroups:     []string{common.CoreAPIGroup},
				Resources:     []string{common.ResourceConfigMaps},
				ResourceNames: []string{CoreDNSConfigMapName},
				Verbs: []string{
					common.VerbGet,
					common.VerbList,
					common.VerbWatch,
				},
			},
		).
		Build()
}

func buildCoreDNSRoleBinding() *rbacv1.RoleBinding {
	return builders.NewRoleBinding().
		WithName(CoreDNSRoleBindingName).
		WithNamespace(CoreDNSNamespaceName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: common.RBACAPIGroup,
				Kind:     common.KindRole,
				Name:     CoreDNSRoleName,
			},
			rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      CoreDNSServiceAccountName,
				Namespace: CoreDNSNamespaceName,
			},
		).
		Build()
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

	return builders.NewConfigMap().
		WithName(CoreDNSConfigMapName).
		WithNamespace(CoreDNSNamespaceName).
		Put("Corefile", corefile).
		Build()
}

func buildCoreDNSService(ma *mcpv1alpha1.ManagedControlPlane) *corev1.Service {
	ports := []corev1.ServicePort{
		{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP, TargetPort: intstr.FromInt(53)},
		{Name: "dns-tcp", Port: 53, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(53)},
	}
	ip, _ := utils.IPAtOffset(ma.Spec.Networking.ServiceCIDR, 10)
	return builders.NewService().
		WithName(CoreDNSServiceName).
		WithNamespace(CoreDNSNamespaceName).
		WithLabels(CoreDNSServiceLabels).
		WithSelector(CoreDNSPodLabels).
		WithType(corev1.ServiceTypeClusterIP).
		WithClusterIP(ip).
		AddPorts(ports).
		Build()
}

func buildCoreDNSDeployment() *appsv1.Deployment {
	c := corev1.Container{
		Name:            "coredns",
		Image:           CoreDNSImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"-conf", "/etc/coredns/Corefile"},
		Ports: []corev1.ContainerPort{
			{ContainerPort: 53, Protocol: corev1.ProtocolUDP},
			{ContainerPort: 53, Protocol: corev1.ProtocolTCP},
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: CoreDNSConfigMapName},
				},
			},
		},
	}
	volumeMounts := corev1.VolumeMount{
		Name: "config-volume", MountPath: "/etc/coredns",
	}

	return builders.NewDeployment().
		WithName(CoreDNSDeploymentName).
		WithNamespace(CoreDNSNamespaceName).
		WithLabels(CoreDNSPodLabels).
		WithSelector(CoreDNSPodLabels).
		WithReplicas(CoreDNSReplicas).
		WithContainer(c).
		AddVolumes(volumes...).
		AddVolumeMounts(c.Name, volumeMounts).
		Build()
}
