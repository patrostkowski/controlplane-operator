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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FlannelNamespaceName          = "kube-flannel"
	FlannelServiceAccountName     = "flannel"
	FlannelConfigMapName          = "kube-flannel-cfg"
	FlannelDaemonSetName          = "kube-flannel-ds"
	FlannelClusterRoleName        = "flannel"
	FlannelClusterRoleBindingName = "flannel"
	FlannelImage                  = "ghcr.io/flannel-io/flannel:v0.27.4"
	CNIPluginImage                = "ghcr.io/flannel-io/flannel-cni-plugin:v1.8.0-flannel1"
	FlannelBackendType            = "vxlan"
)

func buildFlannel(ma *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildFlannelNamespace(),
		buildFlannelServiceAccount(),
		buildFlannelClusterRole(),
		buildFLannelClusterRoleBinding(),
		buildFlannelConfigMap(ma),
		buildFlannelDaemonSet(),
	}
}

var FlannelLabels = map[string]string{
	"app":     "flannel",
	"k8s-app": "flannel",
}

func buildFlannelNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: FlannelNamespaceName,
			Labels: map[string]string{
				"k8s-app":                            "flannel",
				"pod-security.kubernetes.io/enforce": "privileged",
			},
		},
	}
}

func buildFlannelClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   FlannelClusterRoleName,
			Labels: FlannelLabels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes/status"},
				Verbs:     []string{"patch"},
			},
		},
	}
}

func buildFLannelClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   FlannelClusterRoleBindingName,
			Labels: FlannelLabels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     FlannelClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      FlannelServiceAccountName,
				Namespace: FlannelNamespaceName,
			},
		},
	}
}

func buildFlannelServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      FlannelServiceAccountName,
			Namespace: FlannelNamespaceName,
			Labels:    FlannelLabels,
		},
	}
}

func buildFlannelConfigMap(ma *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	cniConf := `{
  "name": "cbr0",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "flannel",
      "delegate": {
        "hairpinMode": true,
        "isDefaultGateway": true
      }
    },
    {
      "type": "portmap",
      "capabilities": {
        "portMappings": true
      }
    }
  ]
}`

	netConf := `{
  "Network": "` + ma.Spec.Networking.PodCIDR + `",
  "EnableNFTables": false,
  "Backend": {
    "Type": "` + FlannelBackendType + `"
  }
}`

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      FlannelConfigMapName,
			Namespace: FlannelNamespaceName,
			Labels:    FlannelLabels,
		},
		Data: map[string]string{
			"cni-conf.json": cniConf,
			"net-conf.json": netConf,
		},
	}
}

func buildFlannelDaemonSet() *appsv1.DaemonSet {
	fileOrCreate := corev1.HostPathFileOrCreate

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      FlannelDaemonSetName,
			Namespace: FlannelNamespaceName,
			Labels:    FlannelLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: FlannelLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: FlannelLabels},
				Spec: corev1.PodSpec{
					HostNetwork:        true,
					PriorityClassName:  "system-node-critical",
					ServiceAccountName: FlannelServiceAccountName,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/os",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"linux"},
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "install-cni-plugin",
							Image:   CNIPluginImage,
							Command: []string{"cp"},
							Args:    []string{"-f", "/flannel", "/opt/cni/bin/flannel"},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "cni-plugin", MountPath: "/opt/cni/bin"},
							},
						},
						{
							Name:    "install-cni",
							Image:   FlannelImage,
							Command: []string{"cp"},
							Args:    []string{"-f", "/etc/kube-flannel/cni-conf.json", "/etc/cni/net.d/10-flannel.conflist"},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "cni", MountPath: "/etc/cni/net.d"},
								{Name: "flannel-cfg", MountPath: "/etc/kube-flannel/"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "kube-flannel",
							Image:   FlannelImage,
							Command: []string{"/opt/bin/flanneld"},
							Args:    []string{"--ip-masq", "--kube-subnet-mgr"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("50Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN", "NET_RAW"},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
									},
								},
								{Name: "EVENT_QUEUE_DEPTH", Value: "5000"},
								{Name: "CONT_WHEN_CACHE_NOT_READY", Value: "false"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "run", MountPath: "/run/flannel"},
								{Name: "flannel-cfg", MountPath: "/etc/kube-flannel/"},
								{Name: "xtables-lock", MountPath: "/run/xtables.lock"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "run",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/run/flannel"},
							},
						},
						{
							Name: "cni-plugin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/opt/cni/bin"},
							},
						},
						{
							Name: "cni",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/etc/cni/net.d"},
							},
						},
						{
							Name: "flannel-cfg",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: FlannelConfigMapName},
								},
							},
						},
						{
							Name: "xtables-lock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/xtables.lock",
									Type: &fileOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}
}
