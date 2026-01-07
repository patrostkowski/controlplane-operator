package common

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	storagev1 "k8s.io/api/storage/v1"
)

// TODO: add functions to get specific stuff like:
// AllVerbs, ReadVerbs, WriteVerbs

const (
	CoreAPIGroup      = ""
	RBACAPIGroup      = rbacv1.GroupName
	StorageAPIGroup   = storagev1.GroupName
	DiscoveryAPIGroup = "discovery.k8s.io"

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
