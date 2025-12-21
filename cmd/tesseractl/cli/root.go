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
	"strings"
	"time"

	"github.com/patrostkowski/controlplane-operator/cmd/tesseractl/cli/mcp"
	agentv1alpha1 "github.com/patrostkowski/controlplane-operator/proto/agent/v1alpha1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	node     string
	endpoint string
	token    string
	timeout  time.Duration
)

func NewTesseractCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tesseractl",
		Short: "tesseract control CLI",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.PersistentFlags().StringP("kubeconfig", "k", "", "override kubeconfig default path ~/.kube/config")
	cmd.PersistentFlags().StringP("namespace", "n", "default", "namespace")
	cmd.PersistentFlags().StringP("context", "c", "", "kubernetes context to use (defaults to current-context from kubeconfig)")

	cmd.AddCommand(
		newJoinCommand(),
		mcp.NewMCPCommand(),
	)

	return cmd
}

func newJoinCommand() *cobra.Command {
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
