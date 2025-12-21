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

package mcp

import (
	"fmt"
	"os"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	mcpclient "github.com/patrostkowski/controlplane-operator/pkg/client"
	"go.yaml.in/yaml/v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/spf13/cobra"
)

func NewMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP object",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(
		newKubeconfigCommand(),
	)

	return cmd
}

func newKubeconfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Print kubeconfig of child cluster ",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath, _ := cmd.InheritedFlags().GetString("kubeconfig")
			ctx, _ := cmd.InheritedFlags().GetString("context")
			ns, _ := cmd.InheritedFlags().GetString("namespace")

			k := mcpclient.NewClient(ctx, kubeconfigPath)
			c, err := k.BuildClient()
			if err != nil {
				return err
			}

			name := args[0]
			mcp := &mcpv1alpha1.ManagedControlPlane{}
			if err := c.Get(cmd.Context(), types.NamespacedName{
				Namespace: ns,
				Name:      name,
			}, mcp); err != nil {
				return err
			}

			sec := &corev1.Secret{}
			secName := mcp.Status.AdminKubeconfigSecretRef.Name
			if err := c.Get(cmd.Context(), types.NamespacedName{
				Name:      secName,
				Namespace: ns,
			}, sec); err != nil {
				return err
			}

			printKubeconfigFromSecret(sec, "config")

			return nil
		},
	}

	return cmd
}

func printKubeconfigFromSecret(sec *corev1.Secret, key string) error {
	b, ok := sec.Data[key]
	if !ok {
		return fmt.Errorf("secret %s/%s missing data[%q]", sec.Namespace, sec.Name, key)
	}

	var obj any
	if err := yaml.Unmarshal(b, &obj); err != nil {
		return nil
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, string(out))
	return nil
}
