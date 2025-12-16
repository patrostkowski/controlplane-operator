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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CSINamespaceName,
		},
	}
}

func buildCSIServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CSIServiceAccountName,
			Namespace: CSINamespaceName,
		},
	}
}

func buildCSIRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CSIRoleName,
			Namespace: CSINamespaceName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete"},
			},
		},
	}
}

func buildCSIClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CSIClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
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
		},
	}
}

func buildCSIRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CSIRoleBindingName,
			Namespace: CSINamespaceName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     CSIRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      CSIServiceAccountName,
				Namespace: CSINamespaceName,
			},
		},
	}
}

func buildCSIClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CSIClusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     CSIClusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      CSIServiceAccountName,
				Namespace: CSINamespaceName,
			},
		},
	}
}

func buildCSIDeployment() *appsv1.Deployment {
	var replicas int32 = 1

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CSIDeploymentName,
			Namespace: CSINamespaceName,
			Labels:    CSILabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: CSILabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: CSILabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: CSIServiceAccountName,
					Containers: []corev1.Container{
						{
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
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/config/"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: CSIConfigMapName},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildCSIStorageClass() *storagev1.StorageClass {
	reclaim := CSIReclaimPolicy
	mode := CSIVolumeBindingMode

	return &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CSIStorageClassName,
		},
		Provisioner:       CSIProvisionerName,
		ReclaimPolicy:     &reclaim,
		VolumeBindingMode: &mode,
	}
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

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CSIConfigMapName,
			Namespace: CSINamespaceName,
		},
		Data: map[string]string{
			"config.json":    configJSON,
			"setup":          setup,
			"teardown":       teardown,
			"helperPod.yaml": helperPodYAML,
		},
	}
}
