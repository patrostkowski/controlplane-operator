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
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CSINamespaceName = "local-path-storage"

	CSIServiceAccountName = "local-path-provisioner-service-account"

	CSIRoleName        = "local-path-provisioner-role"
	CSIClusterRoleName = "local-path-provisioner-role"

	CSIRoleBindingName        = "local-path-provisioner-bind"
	CSIClusterRoleBindingName = "local-path-provisioner-bind"

	CSIDeploymentName = "local-path-provisioner"

	CSIStorageClassName  = "local-path"
	CSIProvisionerName   = "rancher.io/local-path"
	CSIReclaimPolicy     = corev1.PersistentVolumeReclaimDelete
	CSIVolumeBindingMode = storagev1.VolumeBindingWaitForFirstConsumer

	CSIConfigMapName = "local-path-config"

	CSIProvisionerImage = "rancher/local-path-provisioner:v0.0.32"
	CSIHelperPodImage   = "busybox"
)

var CSILabels = map[string]string{
	"app": "local-path-provisioner",
}

func buildCSI() []client.Object {
	return []client.Object{
		buildCSINamespace(),
		buildCSIServiceAccount(),
		buildCSIClusterRole(),
		buildCSIClusterRoleBinding(),
		buildCSIRole(),
		buildCSIRoleBinding(),
		buildCSIConfigMap(),
		buildCSIStorageClass(),
		buildCSIDeployment(),
	}
}

func buildCSINamespace() *corev1.Namespace {
	return builders.NewNamespace().
		WithName(CSINamespaceName).
		Build()
}

func buildCSIServiceAccount() *corev1.ServiceAccount {
	return builders.NewServiceAccount().
		WithName(CSIServiceAccountName).
		WithNamespace(CSINamespaceName).
		Build()
}

func buildCSIRole() *rbacv1.Role {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete"},
		},
	}

	return builders.NewRole().
		WithName(CSIRoleName).
		WithNamespace(CSINamespaceName).
		WithRules(rules).
		Build()
}

func buildCSIClusterRole() *rbacv1.ClusterRole {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes", "persistentvolumeclaims", "configmaps", "pods", "pods/log"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create", "patch"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}

	return builders.NewClusterRole().
		WithName(CSIClusterRoleName).
		WithRules(rules).
		Build()
}

func buildCSIRoleBinding() *rbacv1.RoleBinding {
	roleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     CSIRoleName,
	}
	subjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      CSIServiceAccountName,
			Namespace: CSINamespaceName,
		},
	}

	return builders.NewRoleBinding().
		WithName(CSIRoleBindingName).
		WithNamespace(CSINamespaceName).
		WithRefs(subjects, roleRef).
		Build()
}

func buildCSIClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	roleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     CSIClusterRoleName,
	}
	subjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      CSIServiceAccountName,
			Namespace: CSINamespaceName,
		},
	}

	return builders.NewClusterRoleBinding().
		WithName(CSIClusterRoleName).
		WithRefs(subjects, roleRef).
		Build()
}

func buildCSIStorageClass() *storagev1.StorageClass {
	reclaim := CSIReclaimPolicy
	mode := CSIVolumeBindingMode

	return builders.NewStorageClass().
		WithName(CSIStorageClassName).
		WithProvisioner(CSIProvisionerName).
		WithPolicy(reclaim).
		WithBindingMode(mode).
		Build()
}

func buildCSIConfigMap() *corev1.ConfigMap {
	// upstream default path
	// creates PV dirs under /opt/local-path-provisioner on the node
	configJSON := `{
  "nodePathMap":[
    {
      "node":"DEFAULT_PATH_FOR_NON_LISTED_NODES",
      "paths":["/opt/local-path-provisioner"]
    }
  ]
}`

	setup := `#!/bin/sh
set -eu
mkdir -m 0777 -p "$VOL_DIR"
`

	teardown := `#!/bin/sh
set -eu
rm -rf "$VOL_DIR"
`

	helperPodYAML := `apiVersion: v1
kind: Pod
metadata:
  name: helper-pod
spec:
  priorityClassName: system-node-critical
  tolerations:
    - key: node.kubernetes.io/disk-pressure
      operator: Exists
      effect: NoSchedule
  containers:
    - name: helper-pod
      image: ` + CSIHelperPodImage + `
      imagePullPolicy: IfNotPresent
`
	return builders.NewConfigMap().
		WithName(CSIConfigMapName).
		WithNamespace(CSINamespaceName).
		Put("config.json", configJSON).
		Put("setup", setup).
		Put("teardown", teardown).
		Put("helperPod.yaml", helperPodYAML).
		Build()
}

func buildCSIDeployment() *appsv1.Deployment {
	var replicas int32 = 1

	c := corev1.Container{
		Name:            "local-path-provisioner",
		Image:           CSIProvisionerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"local-path-provisioner",
			"--debug",
			"start",
			"--config",
			"/etc/config/config.json",
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{Name: "CONFIG_MOUNT_PATH", Value: "/etc/config/"},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{Name: "config-volume", MountPath: "/etc/config/"},
	}
	volumes := []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: CSIConfigMapName},
				},
			},
		},
	}

	return builders.NewDeployment().
		WithName(CSIDeploymentName).
		WithNamespace(CSINamespaceName).
		WithLabels(CSILabels).
		WithSelector(CSILabels).
		WithServiceAccount(CSIServiceAccountName).
		WithReplicas(replicas).
		WithContainer(c).
		AddVolumes(volumes...).
		AddVolumeMounts(c.Name, volumeMounts...).
		Build()
}
