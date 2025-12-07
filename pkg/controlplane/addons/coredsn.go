package addons

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CoreDNSNamespaceName          = "kube-system"
	CoreDNSServiceAccountName     = "coredns"
	CoreDNSConfigMapName          = "coredns"
	CoreDNSDeploymentName         = "coredns"
	CoreDNSServiceName            = "kube-dns"
	CoreDNSClusterRoleName        = "system:coredns"
	CoreDNSClusterRoleBindingName = "system:coredns"

	// TODO: make a check to be equal to k8s minor version
	CoreDNSImage = "registry.k8s.io/coredns/coredns:v1.11.3"

	CoreDNSReplicas int32 = 2

	ClusterDNSServiceIP = "10.200.0.10"

	CoreDNSPort        = 53
	CoreDNSMetricsPort = 9153
)

var CoreDNSLabels = map[string]string{
	"k8s-app": "kube-dns",
}

func buildCoreDNS() []client.Object {
	return []client.Object{
		buildCoreDNSServiceAccount(),
		buildCoreDNSClusterRole(),
		buildCoreDNSClusterRoleBinding(),
		buildCoreDNSConfigMap(),
		buildCoreDNSService(),
		buildCoreDNSDeployment(),
	}
}

func buildCoreDNSServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSServiceAccountName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSLabels,
		},
	}
}

func buildCoreDNSClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   CoreDNSClusterRoleName,
			Labels: CoreDNSLabels,
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
			Name:   CoreDNSClusterRoleBindingName,
			Labels: CoreDNSLabels,
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

func buildCoreDNSConfigMap() *corev1.ConfigMap {
	corefile := `.:53 {
    errors
    health
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}`

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSConfigMapName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSLabels,
		},
		Data: map[string]string{
			"Corefile": corefile,
		},
	}
}

func buildCoreDNSService() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSServiceName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSLabels,
			Annotations: map[string]string{
				"prometheus.io/port":   "9153",
				"prometheus.io/scrape": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector:  CoreDNSLabels,
			ClusterIP: ClusterDNSServiceIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "dns",
					Port:       CoreDNSPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt(CoreDNSPort),
				},
				{
					Name:       "dns-tcp",
					Port:       CoreDNSPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(CoreDNSPort),
				},
				{
					Name:       "metrics",
					Port:       CoreDNSMetricsPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(CoreDNSMetricsPort),
				},
			},
		},
	}
}

func buildCoreDNSDeployment() *appsv1.Deployment {
	replicas := CoreDNSReplicas

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CoreDNSDeploymentName,
			Namespace: CoreDNSNamespaceName,
			Labels:    CoreDNSLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: CoreDNSLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: CoreDNSLabels,
				},
				Spec: corev1.PodSpec{
					PriorityClassName:  "system-cluster-critical",
					ServiceAccountName: CoreDNSServiceAccountName,
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
					},
					Tolerations: []corev1.Toleration{
						{Operator: corev1.TolerationOpExists},
					},
					Containers: []corev1.Container{
						{
							Name:  "coredns",
							Image: CoreDNSImage,
							Args:  []string{"-conf", "/etc/coredns/Corefile"},
							Ports: []corev1.ContainerPort{
								{Name: "dns", ContainerPort: CoreDNSPort, Protocol: corev1.ProtocolUDP},
								{Name: "dns-tcp", ContainerPort: CoreDNSPort, Protocol: corev1.ProtocolTCP},
								{Name: "metrics", ContainerPort: CoreDNSMetricsPort, Protocol: corev1.ProtocolTCP},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("70Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("170Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								ReadOnlyRootFilesystem:   ptr.To(true),
								Capabilities: &corev1.Capabilities{
									Add:  []corev1.Capability{"NET_BIND_SERVICE"},
									Drop: []corev1.Capability{"ALL"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/coredns", ReadOnly: true},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
								SuccessThreshold:    1,
								FailureThreshold:    5,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromInt(8181),
									},
								},
								PeriodSeconds:    10,
								TimeoutSeconds:   1,
								SuccessThreshold: 1,
								FailureThreshold: 3,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: CoreDNSConfigMapName},
									Items: []corev1.KeyToPath{
										{Key: "Corefile", Path: "Corefile"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
