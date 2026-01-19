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
