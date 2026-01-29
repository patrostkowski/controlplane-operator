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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func (e addonsBuilder) buildFlannel() []client.Object {
	return []client.Object{
		e.buildFlannelNamespace(),
		e.buildFlannelServiceAccount(),
		e.buildFlannelClusterRole(),
		e.buildFlannelClusterRoleBinding(),
		e.buildFlannelConfigMap(),
		e.buildFlannelDaemonSet(),
	}
}

func defFlannelLabels() map[string]string {
	return map[string]string{
		"app":     "flannel",
		"k8s-app": "flannel",
	}
}

func defFlannelNSLabels() map[string]string {
	return map[string]string{
		"k8s-app":                            "flannel",
		"pod-security.kubernetes.io/enforce": "privileged",
	}
}

func (e addonsBuilder) buildFlannelNamespace() *corev1.Namespace {
	labels := defFlannelNSLabels()
	return builders.NewNamespace().
		WithName(FlannelNamespaceName).
		WithLabels(labels).
		Build()
}

func (e addonsBuilder) buildFlannelClusterRole() *rbacv1.ClusterRole {
	labels := defFlannelLabels()
	return builders.NewClusterRole().
		WithName(FlannelClusterRoleName).
		WithLabels(labels).
		WithRules(
			rbacv1.PolicyRule{
				APIGroups: []string{coreAPIGroup},
				Resources: []string{ResourcePods},
				Verbs:     []string{VerbGet},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{coreAPIGroup},
				Resources: []string{ResourceNodes},
				Verbs: []string{
					VerbGet,
					VerbList,
					VerbWatch,
				},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{coreAPIGroup},
				Resources: []string{ResourceNodesStatus},
				Verbs:     []string{VerbPatch},
			},
		).
		Build()
}

func (e addonsBuilder) buildFlannelClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	labels := defFlannelLabels()
	return builders.NewClusterRoleBinding().
		WithName(FlannelClusterRoleBindingName).
		WithLabels(labels).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacAPIGroup,
				Kind:     KindClusterRole,
				Name:     FlannelClusterRoleName,
			},
			rbacv1.Subject{
				Kind:      KindServiceAccount,
				Name:      FlannelServiceAccountName,
				Namespace: FlannelNamespaceName,
			},
		).
		Build()
}

func (e addonsBuilder) buildFlannelServiceAccount() *corev1.ServiceAccount {
	labels := defFlannelNSLabels()
	return builders.NewServiceAccount().
		WithName(FlannelServiceAccountName).
		WithNamespace(FlannelNamespaceName).
		WithLabels(labels).
		Build()
}

func (e addonsBuilder) buildFlannelConfigMap() *corev1.ConfigMap {
	labels := defFlannelLabels()
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
  "Network": "` + e.cc.GetManagedControlPlaneSpec().Kubernetes.Networking.PodCIDR + `",
  "EnableNFTables": false,
  "Backend": {
    "Type": "` + FlannelBackendType + `"
  }
}`

	return builders.NewConfigMap().
		WithName(FlannelConfigMapName).
		WithNamespace(FlannelNamespaceName).
		WithLabels(labels).
		Put("cni-conf.json", cniConf).
		Put("net-conf.json", netConf).
		Build()
}

func (e addonsBuilder) buildFlannelDaemonSet() *appsv1.DaemonSet {
	fileOrCreate := corev1.HostPathFileOrCreate
	labels := defFlannelLabels()
	priorityClass := "system-node-critical"

	affinity := corev1.Affinity{
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
	}

	toleration := corev1.Toleration{
		Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule,
	}

	initContainers := []corev1.Container{
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
	}

	c := corev1.Container{
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
	}

	volumeMounts := []corev1.VolumeMount{
		{Name: "run", MountPath: "/run/flannel"},
		{Name: "flannel-cfg", MountPath: "/etc/kube-flannel/"},
		{Name: "xtables-lock", MountPath: "/run/xtables.lock"},
	}
	volumes := []corev1.Volume{
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
	}

	return builders.NewDaemonSet().
		WithName(FlannelDaemonSetName).
		WithNamespace(FlannelNamespaceName).
		WithLabels(labels).
		WithSelector(labels).
		WithPodLabels(labels).
		WithServiceAccount(FlannelServiceAccountName).
		WithHostNetwork().
		WithPriorityClass(priorityClass).
		WithAffinity(affinity).
		WithTolerations(toleration).
		WithInitContainers(initContainers...).
		WithContainer(c).
		AddVolumes(volumes...).
		AddVolumeMounts(c.Name, volumeMounts...).
		Build()
}
