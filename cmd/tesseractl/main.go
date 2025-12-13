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

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	agentv1alpha1 "github.com/patrostkowski/controlplane-operator/proto/agent/v1alpha1"
)

func main() {
	var (
		node     string
		endpoint string
		token    string
		timeout  time.Duration
	)

	rootCmd := &cobra.Command{
		Use:   "tesseractl",
		Short: "tesseract control CLI",
	}

	joinCmd := &cobra.Command{
		Use:   "join",
		Short: "Join a node to the control plane",
		RunE: func(cmd *cobra.Command, args []string) error {
			node = strings.TrimSpace(node)
			endpoint = strings.TrimSpace(endpoint)
			token = strings.TrimSpace(token)

			if node == "" {
				return errors.New("--node is required")
			}
			if endpoint == "" {
				return errors.New("--endpoint is required")
			}
			if token == "" {
				return errors.New("--token is required")
			}

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
				return fmt.Errorf("dial %s: %w", node, err)
			}
			defer conn.Close()

			client := agentv1alpha1.NewAgentServiceClient(conn)
			resp, err := client.Join(ctx, &agentv1alpha1.JoinRequest{
				Endpoint: endpoint,
				Token:    token,
			})
			if err != nil {
				return fmt.Errorf("join rpc failed: %w", err)
			}

			// Print a simple result
			fmt.Printf("code=%s\n", resp.GetCode().String())
			return nil
		},
	}

	joinCmd.Flags().StringVar(&node, "node", "", "Agent node address (ip or ip:port). Default port is 32137")
	joinCmd.Flags().StringVar(&endpoint, "endpoint", "", "Kubernetes API endpoint IP to map as 'kubernetes' in /etc/hosts")
	joinCmd.Flags().StringVar(&token, "token", "", "Bearer token used in kubeconfig")
	joinCmd.Flags().DurationVar(&timeout, "timeout", 20*time.Second, "Request timeout")

	_ = joinCmd.MarkFlagRequired("node")
	_ = joinCmd.MarkFlagRequired("endpoint")
	_ = joinCmd.MarkFlagRequired("token")

	rootCmd.AddCommand(joinCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
