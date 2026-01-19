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
	"bytes"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraphandle "k8s.io/cluster-bootstrap/token/util"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	kubeadmv1beta4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KubePublicNamespace                = "kube-public"
	RoleBootstrapSignerClusterInfoName = "kubeadm:bootstrap-signer-clusterinfo"

	RoleKubeadmKubeletConfig      = "kubeadm:kubelet-config"
	RoleKubeadmNodesKubeadmConfig = "kubeadm:nodes-kubeadm-config"

	CMKubeletConfigBase = "kubelet-config"
	CMKubeadmConfig     = "kubeadm-config"

	KubeadmConfigCMName = "kubeadm-config"
)

const DefaultNodeBootstrapperGroup = "system:bootstrappers:kubeadm:default-node-token"

func BootstrapKubeletJoinResources(mcp *mcpv1alpha1.ManagedControlPlane, token BootstrapToken, ca []byte) []client.Object {
	return []client.Object{
		bootstrapTokenSecret(token),
		nodeAutoapproveRotation(),
		nodeAutoapproveBootstrap(),
		nodeBootstrapper(),

		clusterInfo(mcp, ca),

		clusterInfoRole(),
		clusterInfoRoleBindingAnon(),

		kubeletConfigVersioned(mcp),
		kubeletConfigUnversioned(mcp),
		kubeletConfig(),

		kubeadmKubeletConfigRole(),
		kubeadmKubeletConfigRoleBinding(),
		kubeadmNodesKubeadmConfigRole(),
		kubeadmNodesKubeadmConfigRoleBinding(),
		kubeadmGetNodesClusterRole(),
		kubeadmGetNodesClusterRoleBinding(),

		kubeadmConfigConfigMap(mcp),
	}
}

func bootstrapTokenSecret(token BootstrapToken) *corev1.Secret {
	name := bootstraphandle.BootstrapTokenSecretName(token.ID)
	return builders.NewSecret().
		WithName(name).
		WithNamespace(KubeSystemNamespace).
		WithType(bootstrapapi.SecretTypeBootstrapToken).
		Put(bootstrapapi.BootstrapTokenDescriptionKey, BootstrapTokenDescription).
		Put(bootstrapapi.BootstrapTokenIDKey, token.ID).
		Put(bootstrapapi.BootstrapTokenSecretKey, token.Secret).
		Put(bootstrapapi.BootstrapTokenUsageAuthentication, "true").
		Put(bootstrapapi.BootstrapTokenUsageSigningKey, "true").
		Put(bootstrapapi.BootstrapTokenExtraGroupsKey, DefaultNodeBootstrapperGroup).
		Build()
}

func nodeAutoapproveBootstrap() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(CRBKubeadmNodeAutoapproveBootstrap).
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

func nodeAutoapproveRotation() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(CRBKubeadmNodeAutoapproveRotation).
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

func nodeBootstrapper() *rbacv1.ClusterRoleBinding {
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

func clusterInfo(mcp *mcpv1alpha1.ManagedControlPlane, ca []byte) *corev1.ConfigMap {
	cfg := api.NewConfig()

	cfg.Clusters[""] = &api.Cluster{
		Server:                   "https://" + mcp.Status.Address + ":6443",
		CertificateAuthorityData: ca,
	}
	cfg.Contexts = map[string]*api.Context{}
	cfg.AuthInfos = map[string]*api.AuthInfo{}

	kubeconfigBytes, _ := clientcmd.Write(*cfg)

	return builders.NewConfigMap().
		WithName(bootstrapapi.ConfigMapClusterInfo).
		WithNamespace("kube-public").
		Put(bootstrapapi.KubeConfigKey, string(kubeconfigBytes)).
		Build()
}

// func clusterInfoAnon() *rbacv1.ClusterRoleBinding {
// 	return builders.NewClusterRoleBinding().
// 		WithName("kubeadm:cluster-info-anon").
// 		WithRefs(
// 			rbacv1.RoleRef{
// 				APIGroup: rbacv1.GroupName,
// 				Kind:     common.KindClusterRole,
// 				Name:     "system:public-info-viewer",
// 			},
// 			rbacv1.Subject{
// 				APIGroup: rbacv1.GroupName,
// 				Kind:     common.KindUser,
// 				Name:     "system:anonymous",
// 			},
// 		).
// 		Build()
// }

func kubeletConfigUnversioned(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	clusterDNS, _ := utils.IPAtOffset(mcp.Spec.Networking.ServiceCIDR, 10)

	kc := &kubeletconfigv1beta1.KubeletConfiguration{
		RotateCertificates: true,
		ServerTLSBootstrap: true,
		ClusterDNS:         []string{clusterDNS.String()},
		ClusterDomain:      "cluster.local",
		Authentication: kubeletconfigv1beta1.KubeletAuthentication{
			Anonymous: kubeletconfigv1beta1.KubeletAnonymousAuthentication{Enabled: ptr.To(false)},
			Webhook:   kubeletconfigv1beta1.KubeletWebhookAuthentication{Enabled: ptr.To(true)},
		},
		Authorization: kubeletconfigv1beta1.KubeletAuthorization{
			Mode: kubeletconfigv1beta1.KubeletAuthorizationModeWebhook,
		},
		FailSwapOn: ptr.To(false),
	}
	kc.TypeMeta = metav1.TypeMeta{
		APIVersion: "kubelet.config.k8s.io/v1beta1",
		Kind:       "KubeletConfiguration",
	}

	yamlStr := mustEncodeKubeletConfigYAML(kc)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubelet-config",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"kubelet": yamlStr,
		},
	}
}

func kubeletConfigVersioned(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	clusterDNS, _ := utils.IPAtOffset(mcp.Spec.Networking.ServiceCIDR, 10)

	kc := &kubeletconfigv1beta1.KubeletConfiguration{
		RotateCertificates: true,
		ServerTLSBootstrap: true,
		ClusterDNS:         []string{clusterDNS.String()},
		ClusterDomain:      "cluster.local",
		Authentication: kubeletconfigv1beta1.KubeletAuthentication{
			Anonymous: kubeletconfigv1beta1.KubeletAnonymousAuthentication{Enabled: ptr.To(false)},
			Webhook:   kubeletconfigv1beta1.KubeletWebhookAuthentication{Enabled: ptr.To(true)},
		},
		Authorization: kubeletconfigv1beta1.KubeletAuthorization{
			Mode: kubeletconfigv1beta1.KubeletAuthorizationModeWebhook,
		},
		FailSwapOn: ptr.To(false),
	}

	kc.TypeMeta = metav1.TypeMeta{
		APIVersion: "kubelet.config.k8s.io/v1beta1",
		Kind:       "KubeletConfiguration",
	}

	yamlStr := mustEncodeKubeletConfigYAML(kc)

	return builders.NewConfigMap().
		WithName("kubelet-config-"+utils.GetMajorMinorString(mcp.Spec.Version)).
		WithNamespace("kube-system").
		Put("kubelet", yamlStr).
		Build()
}

func kubeletConfig() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName("kubeadm:kubelet-config").
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindClusterRole,
				Name:     "system:kubelet-config-reader",
			},
			rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     common.KindGroup,
				Name:     "system:nodes",
			},
		).
		Build()
}

func clusterInfoRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleBootstrapSignerClusterInfoName,
			Namespace: KubePublicNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{bootstrapapi.ConfigMapClusterInfo},
				Verbs:         []string{"get"},
			},
		},
	}
}

func clusterInfoRoleBindingAnon() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleBootstrapSignerClusterInfoName,
			Namespace: KubePublicNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     RoleBootstrapSignerClusterInfoName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "User",
				Name:     "system:anonymous",
			},
		},
	}
}

// kubeadmKubeletConfigRole allows nodes to GET the "kubelet-config" ConfigMap in kube-system.
func kubeadmKubeletConfigRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmKubeletConfig,
			Namespace: KubeSystemNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{CMKubeletConfigBase},
				Verbs:         []string{"get"},
			},
		},
	}
}

// kubeadmNodesKubeadmConfigRole allows bootstrappers/nodes to GET the "kubeadm-config" ConfigMap in kube-system.
func kubeadmNodesKubeadmConfigRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmNodesKubeadmConfig,
			Namespace: KubeSystemNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{CMKubeadmConfig},
				Verbs:         []string{"get"},
			},
		},
	}
}

// kubeadmKubeletConfigRoleBinding binds system:nodes to Role kubeadm:kubelet-config.
func kubeadmKubeletConfigRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmKubeletConfig,
			Namespace: KubeSystemNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     RoleKubeadmKubeletConfig,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     "system:nodes",
			},
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     DefaultNodeBootstrapperGroup,
			},
		},
	}
}

func kubeadmGetNodesClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeadm:get-nodes",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get"},
			},
		},
	}
}

func kubeadmGetNodesClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeadm:get-nodes",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "kubeadm:get-nodes",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     DefaultNodeBootstrapperGroup,
			},
		},
	}
}

// kubeadmNodesKubeadmConfigRoleBinding binds system:bootstrappers to Role kubeadm:nodes-kubeadm-config.
// This is the identity kubeadm uses during join: system:bootstrap:<token-id>.
func kubeadmNodesKubeadmConfigRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmNodesKubeadmConfig,
			Namespace: KubeSystemNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     RoleKubeadmNodesKubeadmConfig,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     DefaultNodeBootstrapperGroup,
			},
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     "system:nodes",
			},
		},
	}
}

// kubeadmConfigConfigMap creates kube-system/kubeadm-config with typed ClusterConfiguration (v1beta4)
func kubeadmConfigConfigMap(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	cc := &kubeadmv1beta4.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta4",
			Kind:       "ClusterConfiguration",
		},
		ClusterName:       mcp.Name,
		KubernetesVersion: mcp.Spec.Version,
		Networking: kubeadmv1beta4.Networking{
			DNSDomain: "cluster.local",
		},
	}

	if mcp.Status.Address != "" {
		cc.ControlPlaneEndpoint = mcp.Status.Address + ":6443"
	}

	if mcp.Spec.Networking != nil {
		cc.Networking.ServiceSubnet = mcp.Spec.Networking.ServiceCIDR
		cc.Networking.PodSubnet = mcp.Spec.Networking.PodCIDR
	}

	yamlStr := mustEncodeKubeadmClusterConfigYAML(cc)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeadmConfigCMName,
			Namespace: KubeSystemNamespace,
		},
		Data: map[string]string{
			"ClusterConfiguration": yamlStr,
		},
	}
}

func mustEncodeKubeadmClusterConfigYAML(
	cc *kubeadmv1beta4.ClusterConfiguration,
) string {
	scheme := runtime.NewScheme()

	// Best-effort registration
	_ = kubeadmv1beta4.AddToScheme(scheme)

	gv := schema.GroupVersion{
		Group:   "kubeadm.k8s.io",
		Version: "v1beta4",
	}
	cc.GetObjectKind().SetGroupVersionKind(
		gv.WithKind("ClusterConfiguration"),
	)

	ser := json.NewYAMLSerializer(
		json.DefaultMetaFactory,
		scheme,
		scheme,
	)

	var buf bytes.Buffer
	ser.Encode(cc, &buf)
	return buf.String()
}

func mustEncodeKubeletConfigYAML(kc *kubeletconfigv1beta1.KubeletConfiguration) string {
	scheme := runtime.NewScheme()
	_ = kubeletconfigv1beta1.AddToScheme(scheme)

	gv := schema.GroupVersion{Group: "kubelet.config.k8s.io", Version: "v1beta1"}
	kc.GetObjectKind().SetGroupVersionKind(gv.WithKind("KubeletConfiguration"))

	ser := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme)

	var buf bytes.Buffer
	ser.Encode(kc, &buf)
	return buf.String()
}
