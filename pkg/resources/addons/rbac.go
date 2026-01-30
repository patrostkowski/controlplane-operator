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

	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
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

const (
	coreAPIGroup      = ""
	rbacAPIGroup      = rbacv1.GroupName
	storageAPIGroup   = storagev1.GroupName
	discoveryAPIGroup = "discovery.k8s.io"

	ResourceNodes          = "nodes"
	ResourceNodesStatus    = "nodes/status"
	ResourcePodsLogs       = "pods/logs"
	ResourceEndpoints      = "endpoints"
	ResourceEndpointSlices = "endpointslices"
	ResourceNamespaces     = "namespaces"
	ResourcePVs            = "persistentvolumes"
	ResourceStorageClasses = "storageclasses"
	ResourceEvents         = "events"

	VerbGet    = "get"
	VerbList   = "list"
	VerbWatch  = "watch"
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbPatch  = "patch"
	VerbDelete = "delete"

	KindGroup = rbacv1.GroupKind
	KindUser  = rbacv1.UserKind
)

var (
	ResourcePods           = corev1.ResourcePods.String()
	ResourceServices       = corev1.ResourceServices.String()
	ResourceConfigMaps     = corev1.ResourceConfigMaps.String()
	ResourcePVCs           = corev1.ResourcePersistentVolumeClaims.String()
	KindClusterRole        = rbacv1.SchemeGroupVersion.WithKind("ClusterRole").Kind
	KindClusterRoleBinding = rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding").Kind
	KindRole               = rbacv1.SchemeGroupVersion.WithKind("Role").Kind
	KindServiceAccount     = rbacv1.ServiceAccountKind
	KindRoleBinding        = rbacv1.SchemeGroupVersion.WithKind("RoleBinding").Kind
)

const DefaultNodeBootstrapperGroup = "system:bootstrappers:kubeadm:default-node-token"

type kubeletJoinBuilder struct {
	cc    cluster.AddonSpec
	tok   BootstrapToken
	caPEM []byte
}

func NewKubeletJoinBuilder(cc cluster.AddonSpec, tok BootstrapToken, caPEM []byte) cluster.ObjectProducer {
	return kubeletJoinBuilder{cc: cc, tok: tok, caPEM: caPEM}
}

func (b kubeletJoinBuilder) Objects() []client.Object {
	return []client.Object{
		b.bootstrapTokenSecret(),
		b.nodeAutoapproveRotation(),
		b.nodeAutoapproveBootstrap(),
		b.nodeBootstrapper(),

		b.clusterInfo(),

		b.clusterInfoRole(),
		b.clusterInfoRoleBindingAnon(),

		b.kubeletConfigVersioned(),
		b.kubeletConfigUnversioned(),
		b.kubeletConfig(),

		b.kubeadmKubeletConfigRole(),
		b.kubeadmKubeletConfigRoleBinding(),
		b.kubeadmNodesKubeadmConfigRole(),
		b.kubeadmNodesKubeadmConfigRoleBinding(),
		b.kubeadmGetNodesClusterRole(),
		b.kubeadmGetNodesClusterRoleBinding(),

		b.kubeadmConfigConfigMap(),
	}
}

func (b kubeletJoinBuilder) bootstrapTokenSecret() *corev1.Secret {
	name := bootstraphandle.BootstrapTokenSecretName(b.tok.ID)
	return builders.NewSecret().
		WithName(name).
		WithNamespace(kubeSystemNamespace).
		WithType(bootstrapapi.SecretTypeBootstrapToken).
		Put(bootstrapapi.BootstrapTokenDescriptionKey, bootstrapTokenDescription).
		Put(bootstrapapi.BootstrapTokenIDKey, b.tok.ID).
		Put(bootstrapapi.BootstrapTokenSecretKey, b.tok.Secret).
		Put(bootstrapapi.BootstrapTokenUsageAuthentication, "true").
		Put(bootstrapapi.BootstrapTokenUsageSigningKey, "true").
		Put(bootstrapapi.BootstrapTokenExtraGroupsKey, DefaultNodeBootstrapperGroup).
		Build()
}

func (b kubeletJoinBuilder) nodeAutoapproveBootstrap() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(crbKubeadmNodeAutoapproveBootstrap).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     KindClusterRole,
				Name:     roleNodeClientCSRApprove,
			},
			rbacv1.Subject{
				APIGroup: rbacAPIGroup,
				Kind:     KindGroup,
				Name:     groupBootstrappers,
			},
		).
		Build()
}

func (b kubeletJoinBuilder) nodeAutoapproveRotation() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(crbKubeadmNodeAutoapproveRotation).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     KindClusterRole,
				Name:     roleSelfNodeClientCSR,
			},
			rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     KindGroup,
				Name:     groupNodes,
			},
		).
		Build()
}

func (b kubeletJoinBuilder) nodeBootstrapper() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(crbNodeBootstrapperName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     KindClusterRole,
				Name:     roleNodeBootstrapper,
			},
			rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     KindGroup,
				Name:     groupBootstrappers,
			},
		).
		Build()
}

func (b kubeletJoinBuilder) clusterInfo() *corev1.ConfigMap {
	mcpStatus := b.cc.GetManagedControlPlaneStatus()
	cfg := api.NewConfig()

	cfg.Clusters[""] = &api.Cluster{
		Server:                   "https://" + mcpStatus.Address + ":6443",
		CertificateAuthorityData: b.caPEM,
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

func (b kubeletJoinBuilder) kubeletConfigUnversioned() *corev1.ConfigMap {
	mcpSpec := b.cc.GetManagedControlPlaneSpec()
	clusterDNS, _ := utils.IPAtOffset(mcpSpec.Kubernetes.Networking.ServiceCIDR, 10)

	kc := &kubeletconfigv1beta1.KubeletConfiguration{
		RotateCertificates: true,
		ServerTLSBootstrap: false,
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

func (b kubeletJoinBuilder) kubeletConfigVersioned() *corev1.ConfigMap {
	mcpSpec := b.cc.GetManagedControlPlaneSpec()
	clusterDNS, _ := utils.IPAtOffset(mcpSpec.Kubernetes.Networking.ServiceCIDR, 10)

	kc := &kubeletconfigv1beta1.KubeletConfiguration{
		RotateCertificates: true,
		ServerTLSBootstrap: false,
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
		WithName("kubelet-config-"+utils.GetMajorMinorString(mcpSpec.Kubernetes.Version)).
		WithNamespace("kube-system").
		Put("kubelet", yamlStr).
		Build()
}

func (b kubeletJoinBuilder) kubeletConfig() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName("kubeadm:kubelet-config").
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     KindClusterRole,
				Name:     "system:kubelet-config-reader",
			},
			rbacv1.Subject{
				APIGroup: rbacv1.GroupName,
				Kind:     KindGroup,
				Name:     "system:nodes",
			},
		).
		Build()
}

func (b kubeletJoinBuilder) clusterInfoRole() *rbacv1.Role {
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

func (b kubeletJoinBuilder) clusterInfoRoleBindingAnon() *rbacv1.RoleBinding {
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
func (b kubeletJoinBuilder) kubeadmKubeletConfigRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmKubeletConfig,
			Namespace: kubeSystemNamespace,
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
func (b kubeletJoinBuilder) kubeadmNodesKubeadmConfigRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmNodesKubeadmConfig,
			Namespace: kubeSystemNamespace,
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
func (b kubeletJoinBuilder) kubeadmKubeletConfigRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmKubeletConfig,
			Namespace: kubeSystemNamespace,
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

func (b kubeletJoinBuilder) kubeadmGetNodesClusterRole() *rbacv1.ClusterRole {
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

func (b kubeletJoinBuilder) kubeadmGetNodesClusterRoleBinding() *rbacv1.ClusterRoleBinding {
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
func (b kubeletJoinBuilder) kubeadmNodesKubeadmConfigRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RoleKubeadmNodesKubeadmConfig,
			Namespace: kubeSystemNamespace,
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
func (b kubeletJoinBuilder) kubeadmConfigConfigMap() *corev1.ConfigMap {
	mcpSpec := b.cc.GetManagedControlPlaneSpec()
	mcpStatus := b.cc.GetManagedControlPlaneStatus()
	cc := &kubeadmv1beta4.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubeadm.k8s.io/v1beta4",
			Kind:       "ClusterConfiguration",
		},
		ClusterName:       b.cc.Name(),
		KubernetesVersion: mcpSpec.Kubernetes.Version,
		Networking: kubeadmv1beta4.Networking{
			DNSDomain: "cluster.local",
		},
	}

	if mcpStatus.Address != "" {
		cc.ControlPlaneEndpoint = mcpStatus.Address + ":6443"
	}

	if mcpSpec.Kubernetes.Networking != nil {
		cc.Networking.ServiceSubnet = mcpSpec.Kubernetes.Networking.ServiceCIDR
		cc.Networking.PodSubnet = mcpSpec.Kubernetes.Networking.PodCIDR
	}

	yamlStr := mustEncodeKubeadmClusterConfigYAML(cc)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeadmConfigCMName,
			Namespace: kubeSystemNamespace,
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
