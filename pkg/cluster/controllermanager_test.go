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
	"testing"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestControllerManagerConfig_NamesAndSecrets(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ns"},
	}
	cc := NewClusterContext(mcp, logr.Logger{})
	cm := cc.ControllerManager()

	// resource names
	if got, want := cm.DeploymentName(), "demo-controller-manager"; got != want {
		t.Fatalf("DeploymentName()=%q want %q", got, want)
	}
	if got, want := cm.KubeconfigConfigMapName(), "demo-controller-manager-kubeconfig"; got != want {
		t.Fatalf("KubeconfigConfigMapName()=%q want %q", got, want)
	}

	// PKI secret names
	if got, want := cm.ClusterCASecret(), "demo-managed-ca"; got != want {
		t.Fatalf("ClusterCASecret()=%q want %q", got, want)
	}
	if got, want := cm.ClientCertSecret(), "demo-cm-client"; got != want {
		t.Fatalf("ClientCertSecret()=%q want %q", got, want)
	}
	if got, want := cm.SASignerSecret(), "demo-sa-signer"; got != want {
		t.Fatalf("SASignerSecret()=%q want %q", got, want)
	}
}
