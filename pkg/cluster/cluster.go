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

package cluster

import (
	"path/filepath"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterContext is a per-reconcile helper that exposes the ManagedControlPlane
// identity plus common ownership and secret-mount utilities.
type ClusterContext struct {
	mcp *mcpv1alpha1.ManagedControlPlane
	Log logr.Logger
}

// NewClusterContext constructs a ClusterContext for a single reconcile, scoping the logger to "ClusterContext".
func NewClusterContext(mcp *mcpv1alpha1.ManagedControlPlane, log logr.Logger) *ClusterContext {
	cc := &ClusterContext{
		mcp: mcp,
		Log: log.WithName("ClusterContext"),
	}
	return cc
}

// Name returns the ManagedControlPlane name.
func (cc ClusterContext) Name() string {
	return cc.mcp.GetName()
}

// Namespace returns the ManagedControlPlane namespace.
func (cc ClusterContext) Namespace() string {
	return cc.mcp.GetNamespace()
}

// GetManagedControlPlaneSpec returns the ManagedControlPlane spec snapshot.
func (cc ClusterContext) GetManagedControlPlaneSpec() mcpv1alpha1.ManagedControlPlaneSpec {
	return cc.mcp.Spec
}

// GetManagedControlPlaneStatus returns the ManagedControlPlane status snapshot.
func (cc ClusterContext) GetManagedControlPlaneStatus() mcpv1alpha1.ManagedControlPlaneStatus {
	return cc.mcp.Status
}

// Owner returns the ManagedControlPlane object to use for owner references.
func (cc ClusterContext) Owner() client.Object {
	return cc.mcp
}

// MCP returns the underlying ManagedControlPlane pointer.
func (cc ClusterContext) MCP() *mcpv1alpha1.ManagedControlPlane {
	return cc.mcp
}

// prefix applies a naming prefix for generated resources.
func (cc ClusterContext) prefix(s string) string {
	return cc.Name() + "-" + s
}

// SecretMountDir returns the filesystem mount directory for a secret under the PKI root.
func (cc ClusterContext) SecretMountDir(secretName string) string {
	return filepath.Join(pkiRoot, secretName)
}

// CertPath returns the path to tls.crt for a mounted secret.
func (cc ClusterContext) CertPath(secretName string) string {
	return filepath.Join(cc.SecretMountDir(secretName), tlsCrt)
}

// KeyPath returns the path to tls.key for a mounted secret.
func (cc ClusterContext) KeyPath(secretName string) string {
	return filepath.Join(cc.SecretMountDir(secretName), tlsKey)
}

// CAPath returns the path to ca.crt for a mounted secret.
func (cc ClusterContext) CAPath(secretName string) string {
	return filepath.Join(cc.SecretMountDir(secretName), caCrt)
}

// SecretVolume returns a corev1.Volume that sources data from the given Secret name.
func (cc ClusterContext) SecretVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	}
}

// SecretMount returns a corev1.VolumeMount that mounts the given Secret at its computed mount directory.
func (cc ClusterContext) SecretMount(secretName string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      secretName,
		ReadOnly:  readOnly,
		MountPath: cc.SecretMountDir(secretName),
	}
}
