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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BootstrapTokenLabel = "controlplane.patrostkowski.dev/bootstrap-token"
	BootstrapTokenValue = "true"
)

func BootstrapKubeletJoinResources(token BootstrapToken) []client.Object {
	return []client.Object{
		bootstrapTokenSecret(token),
		crbNodeClientAutoApprove(),
		crbSelfNodeClientRotationAutoApprove(),
		crbNodeBootstrapper(),
		crbKubeletBootstrapper(),
	}
}

func bootstrapTokenSecret(token BootstrapToken) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      BootstrapTokenMgmtSecretName + "-" + token.ID,
			Namespace: KubeSystemNamespace,
			Labels: map[string]string{
				BootstrapTokenLabel: BootstrapTokenValue,
			},
		},
		Type: corev1.SecretTypeBootstrapToken,
		StringData: map[string]string{
			"description":                    BootstrapTokenDescription,
			BootstrapTokenIDKey:              token.ID,
			BootstrapTokenSecretKey:          token.Secret,
			"auth-extra-groups":              BootstrapAuthExtraGroups,
			"usage-bootstrap-authentication": BootstrapUsageAuth,
			"usage-bootstrap-signing":        BootstrapUsageSign,
		},
	}
}

func crbNodeClientAutoApprove() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CRBNodeClientAutoApproveName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     RoleNodeClientCSRApprove,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     GroupBootstrappers,
			},
		},
	}
}

func crbSelfNodeClientRotationAutoApprove() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CRBSelfNodeClientRotationName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     RoleSelfNodeClientCSR,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     GroupNodes,
			},
		},
	}
}

func crbNodeBootstrapper() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CRBNodeBootstrapperName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     RoleNodeBootstrapper,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     GroupBootstrappers,
			},
		},
	}
}

func crbKubeletBootstrapper() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: CRBNodeCertAutoApprover,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     RoleSelfNodeServer,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     GroupNodes,
			},
		},
	}
}
