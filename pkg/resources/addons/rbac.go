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
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BootstrapKubeletJoinResources(token BootstrapToken) []client.Object {
	return []client.Object{
		bootstrapTokenSecret(token),
		crbNodeClientAutoApprove(),
		crbSelfNodeClientRotationAutoApprove(),
		crbNodeBootstrapper(),
	}
}

func bootstrapTokenSecret(token BootstrapToken) *corev1.Secret {
	name := BootstrapTokenMgmtSecretName + "-" + token.ID
	return builders.NewSecret().
		WithName(name).
		WithNamespace(KubeSystemNamespace).
		WithType(corev1.SecretTypeBootstrapToken).
		Put("description", BootstrapTokenDescription).
		Put(BootstrapTokenIDKey, token.ID).
		Put(BootstrapTokenSecretKey, token.Secret).
		Put("auth-extra-groups", BootstrapAuthExtraGroups).
		Put("usage-bootstrap-authentication", BootstrapUsageAuth).
		Put("usage-bootstrap-signing", BootstrapUsageSign).
		Build()
}

func crbNodeClientAutoApprove() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(CRBNodeClientAutoApproveName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindClusterRole,
				Name:     RoleNodeClientCSRApprove,
			},
			rbacv1.Subject{
				APIGroup: common.RBACAPIGroup,
				Kind:     common.KindGroup,
				Name:     GroupBootstrappers,
			},
		).
		Build()
}

func crbSelfNodeClientRotationAutoApprove() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(CRBSelfNodeClientRotationName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindClusterRole,
				Name:     RoleSelfNodeClientCSR,
			},
			rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindGroup,
				Name:     GroupNodes,
			},
		).
		Build()
}

func crbNodeBootstrapper() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(CRBNodeBootstrapperName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindClusterRole,
				Name:     RoleNodeBootstrapper,
			},
			rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindGroup,
				Name:     GroupBootstrappers,
			},
		).
		Build()
}
