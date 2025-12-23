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

package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	mcpclient "github.com/patrostkowski/controlplane-operator/pkg/client"
	agentv1alpha1 "github.com/patrostkowski/controlplane-operator/proto/agent/v1alpha1"
	"go.yaml.in/yaml/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

const (
	adminConfigSecretKey = "config"

	kubeconfigFlagName = "kubeconfig"
	namespaceFlagName  = "namespace"
	contextFlagName    = "context"
)

var (
	node     string
	endpoint string
	token    string
	timeout  time.Duration
)

func New() *CLI {
	c := CLI{}
	c.cmd = &cobra.Command{
		Use:   "tesseractl",
		Short: "tesseract CLI",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	c.cmd.SilenceErrors = true
	c.cmd.SilenceUsage = true

	c.cmd.PersistentFlags().StringP(kubeconfigFlagName, "k", "", "override kubeconfig default path ~/.kube/config")
	c.cmd.PersistentFlags().StringP(namespaceFlagName, "n", "", "namespace (defaults to kubeconfig context namespace; else 'default')")
	c.cmd.PersistentFlags().StringP(contextFlagName, "c", "", "kubernetes context to use (defaults to current-context from kubeconfig)")

	c.cmd.AddCommand(
		c.newJoinCommand(),
		c.newMCPCommand(),
	)

	return &c
}

func (c *CLI) Run() error {
	return c.cmd.Execute()
}

func (c *CLI) newJoinCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Join a node to the control plane",
		RunE: func(cmd *cobra.Command, args []string) error {
			node = strings.TrimSpace(node)
			endpoint = strings.TrimSpace(endpoint)
			token = strings.TrimSpace(token)

			// Accept either "<ip>" or "<ip>:32137"
			if !strings.Contains(node, ":") {
				node = net.JoinHostPort(node, "32137")
			}

			// Optional: validate endpoint is IP (matches your server expectations)
			if ip := net.ParseIP(endpoint); ip == nil {
				return fmt.Errorf("--endpoint must be an IP address, got %q", endpoint)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			conn, err := grpc.NewClient(node, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}
			defer conn.Close()

			client := agentv1alpha1.NewAgentServiceClient(conn)
			resp, err := client.Join(ctx, &agentv1alpha1.JoinRequest{
				Endpoint: endpoint,
				Token:    token,
			})
			if err != nil {
				return err
			}

			// Print a simple result
			fmt.Printf("code=%s\n", resp.GetCode().String())
			return nil
		},
	}

	cmd.Flags().StringVar(&node, "node", "", "Agent node address (ip or ip:port). Default port is 32137")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Kubernetes API endpoint IP to map as 'kubernetes' in /etc/hosts")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token used in kubeconfig")
	cmd.Flags().DurationVar(&timeout, "timeout", 20*time.Second, "Request timeout")

	_ = cmd.MarkFlagRequired("node")
	_ = cmd.MarkFlagRequired("endpoint")
	_ = cmd.MarkFlagRequired("token")

	return cmd
}

func (c *CLI) newMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP object",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(
		c.newKubeconfigCommand(),
	)

	return cmd
}

func (c *CLI) newKubeconfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Print kubeconfig of child cluster ",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf(
					"missing ManagedControlPlane name\n\n" +
						"Usage:\n" +
						"  tesseractl mcp kubeconfig <mcp-name>\n\n" +
						"Example:\n" +
						"  tesseractl mcp kubeconfig my-cluster",
				)
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("expected cluster name as positional argument, got %d arguments", len(args))
			}
			name := args[0]

			eff, err := c.effectiveKubeconfigFlags()
			if err != nil {
				return err
			}

			c.k8s, err = mcpclient.NewClient(eff.Context, eff.KubeconfigPath)
			if err != nil {
				return err
			}

			mcp := &mcpv1alpha1.ManagedControlPlane{}
			if err := c.k8s.Get(cmd.Context(), types.NamespacedName{
				Namespace: eff.Namespace,
				Name:      name,
			}, mcp); err != nil {
				return err
			}

			sec := &corev1.Secret{}
			secName := mcp.Status.AdminKubeconfigSecretRef.Name
			if err := c.k8s.Get(cmd.Context(), types.NamespacedName{
				Name:      secName,
				Namespace: eff.Namespace,
			}, sec); err != nil {
				return err
			}

			return c.printKubeconfigFromSecret(sec, adminConfigSecretKey)
		},
	}

	return cmd
}

func (c *CLI) printKubeconfigFromSecret(sec *corev1.Secret, key string) error {
	b, ok := sec.Data[key]
	if !ok {
		return fmt.Errorf("secret %s/%s missing data[%q]", sec.Namespace, sec.Name, key)
	}

	var obj any
	if err := yaml.Unmarshal(b, &obj); err != nil {
		return err
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, string(out))
	return nil
}

func (c *CLI) effectiveKubeconfigFlags() (effectiveFlags, error) {
	kubeconfigPath, _ := c.cmd.PersistentFlags().GetString(kubeconfigFlagName)
	contextName, _ := c.cmd.PersistentFlags().GetString(contextFlagName)
	namespaceName, _ := c.cmd.PersistentFlags().GetString(namespaceFlagName)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	rawCfg, err := cc.RawConfig()
	if err != nil {
		return effectiveFlags{}, err
	}

	ctx := contextName
	if ctx == "" {
		ctx = rawCfg.CurrentContext
	}

	ns := namespaceName
	if ns == "" {
		if cctx, ok := rawCfg.Contexts[ctx]; ok && cctx != nil && cctx.Namespace != "" {
			ns = cctx.Namespace
		} else {
			ns = "default"
		}
	}

	return effectiveFlags{
		KubeconfigPath: kubeconfigPath,
		Context:        ctx,
		Namespace:      ns,
	}, nil
}
