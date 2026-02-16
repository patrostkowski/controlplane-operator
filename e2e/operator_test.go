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

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	mcpControlPlaneName      = "e2e"
	mcpControlPlaneNamespace = "default"

	deployAPIServer  = mcpControlPlaneName + "-" + "apiserver"
	deployController = mcpControlPlaneName + "-" + "controller-manager"
	deployScheduler  = mcpControlPlaneName + "-" + "scheduler"
	stsEtcd          = mcpControlPlaneName + "-" + "etcd"

	applyCRTestTimeout    = 8 * time.Minute
	versionTestTimeout    = 3 * time.Minute
	joinWorkerTestTimeout = 15 * time.Minute
	healthyTestTimeout    = 5 * time.Minute

	workloadExistsTimeout = 2 * time.Minute
	workloadReadyTimeout  = 5 * time.Minute

	workerNodeReadyTimeout = 5 * time.Minute
	workerNodePollInterval = 5 * time.Second

	allPodsHealthyTimeout = 5 * time.Minute

	workerName = "mcp-worker1"
)

var (
	pathWorkerScript = "../hack/worker.sh"
	pathTesseractl   = "../bin/tesseractl"
)

type MCPInfo struct {
	Name      string
	Namespace string
}

func discoverMCPName(ctx context.Context) (string, error) {
	name, err := kubectlJSONPath(ctx, []string{"get", "mcp"}, "{.items[0].metadata.name}")
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("no MCP objects found")
	}
	return name, nil
}

func discoverMCPNamespace(ctx context.Context, mcpName string) string {
	ns, err := kubectlJSONPath(ctx, []string{"get", "mcp", mcpName}, "{.metadata.namespace}")
	if err == nil && strings.TrimSpace(ns) != "" {
		return strings.TrimSpace(ns)
	}

	ns, err = kubectlJSONPath(ctx, []string{"get", "mcp", mcpName}, "{.spec.namespace}")
	if err == nil && strings.TrimSpace(ns) != "" {
		return strings.TrimSpace(ns)
	}

	return "mcp"
}

func WaitForControlPlaneReady(ctx context.Context, info MCPInfo) error {
	refs := []WorkloadRef{
		{Kind: WorkloadStatefulSet, NS: info.Namespace, Name: stsEtcd, ExistsTimeout: workloadExistsTimeout, ReadyTimeout: workloadReadyTimeout},
		{Kind: WorkloadDeployment, NS: info.Namespace, Name: deployAPIServer, ExistsTimeout: workloadExistsTimeout, ReadyTimeout: workloadReadyTimeout},
		{Kind: WorkloadDeployment, NS: info.Namespace, Name: deployController, ExistsTimeout: workloadExistsTimeout, ReadyTimeout: workloadReadyTimeout},
		{Kind: WorkloadDeployment, NS: info.Namespace, Name: deployScheduler, ExistsTimeout: workloadExistsTimeout, ReadyTimeout: workloadReadyTimeout},
	}
	return WaitForWorkloads(ctx, refs)
}

func generateMCPKubeconfig(ctx context.Context, mcpName, kubeconfigPath string) error {
	cmd := fmt.Sprintf(`%s mcp kubeconfig %s > %s`, pathTesseractl, mcpName, kubeconfigPath)
	if err := run(ctx, cmdBash, "-lc", cmd); err != nil {
		return err
	}
	_, err := os.Stat(kubeconfigPath)
	return err
}

func waitForNodeReady(ctx context.Context, kubeconfigPath, nodeName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, _ := runOutput(ctx, cmdKubectl,
			"--kubeconfig", kubeconfigPath,
			"get", "node", nodeName,
			"--no-headers",
		)
		out = strings.TrimSpace(out)
		if out != "" && (strings.Contains(out, " Ready ") || strings.HasSuffix(out, " Ready") || strings.Contains(out, "\tReady\t")) {
			return nil
		}
		time.Sleep(workerNodePollInterval)
	}
	return fmt.Errorf("node %q not Ready within %s", nodeName, timeout.String())
}

func TestE2E(t *testing.T) {
	var (
		mcpOnce sync.Once
		mcpInfo MCPInfo
		mcpErr  error

		kcfgOnce sync.Once
		kcfgPath string
		kcfgErr  error
	)

	baseTmpDir := t.TempDir()
	kubeconfigFile := filepath.Join(baseTmpDir, "mcp.kubeconfig")

	getMCP := func(ctx context.Context) (MCPInfo, error) {
		mcpOnce.Do(func() {
			name, err := discoverMCPName(ctx)
			if err != nil {
				mcpErr = err
				return
			}
			ns := discoverMCPNamespace(ctx, name)
			mcpInfo = MCPInfo{Name: name, Namespace: ns}
		})
		return mcpInfo, mcpErr
	}

	getKubeconfig := func(ctx context.Context, mcpName string) (string, error) {
		kcfgOnce.Do(func() {
			if err := generateMCPKubeconfig(ctx, mcpName, kubeconfigFile); err != nil {
				kcfgErr = err
				return
			}
			kcfgPath = kubeconfigFile
		})
		return kcfgPath, kcfgErr
	}

	t.Run("ApplyCR", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), applyCRTestTimeout)
		defer cancel()

		if err := run(ctx, cmdKubectl, "apply", "-f", "./testdata/mcp.yaml"); err != nil {
			t.Fatalf("failed applying CR: %v", err)
		}

		info, err := getMCP(ctx)
		if err != nil {
			t.Fatalf("failed discovering MCP name: %v", err)
		}

		if err := WaitForControlPlaneReady(ctx, info); err != nil {
			t.Fatalf("control plane not ready (ns=%s): %v", info.Namespace, err)
		}
	})

	t.Run("VersionMatches", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), versionTestTimeout)
		defer cancel()

		info, err := getMCP(ctx)
		if err != nil {
			t.Fatalf("failed discovering MCP name: %v", err)
		}

		version, err := kubectlJSONPath(ctx, []string{"get", "mcp", info.Name}, "{.spec.kubernetes.version}")
		if err != nil {
			t.Fatalf("failed reading spec.kubernetes.version: %v", err)
		}
		if version == "" {
			t.Fatalf("spec.kubernetes.version is empty")
		}

		image, err := kubectlJSONPath(ctx,
			[]string{"-n", info.Namespace, "get", "deploy", deployAPIServer},
			"{.spec.template.spec.containers[0].image}",
		)
		if err != nil {
			t.Fatalf("failed reading apiserver image: %v", err)
		}
		if image == "" {
			t.Fatalf("apiserver image is empty")
		}

		tag := extractImageTag(image)
		if tag == "" {
			t.Fatalf("could not extract image tag from %q", image)
		}

		normalized := strings.TrimPrefix(version, "v")
		if tag != version && strings.TrimPrefix(tag, "v") != normalized {
			t.Fatalf("version mismatch: mcp spec.kubernetes.version=%q, apiserver image=%q (tag=%q)", version, image, tag)
		}
	})

	t.Run("JoinWorker", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), joinWorkerTestTimeout)
		defer cancel()

		info, err := getMCP(ctx)
		if err != nil {
			t.Fatalf("failed discovering MCP name: %v", err)
		}

		if err := run(ctx, cmdBash, "-lc",
			fmt.Sprintf(`%s --namespace %s --name %s create %s`, pathWorkerScript, info.Namespace, info.Name, workerName),
		); err != nil {
			t.Fatalf("worker bootstrap script failed: %v", err)
		}

		kubeconfigPath, err := getKubeconfig(ctx, info.Name)
		if err != nil {
			t.Fatalf("failed generating kubeconfig: %v", err)
		}

		if err := waitForNodeReady(ctx, kubeconfigPath, workerName, workerNodeReadyTimeout); err != nil {
			nodes, _ := runOutput(ctx, cmdKubectl, "--kubeconfig", kubeconfigPath, "get", "nodes", "-o", "wide")
			t.Fatalf("worker node %q did not become Ready in time. Nodes:\n%s", workerName, nodes)
		}
	})

	t.Run("ClusterHealthy", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), healthyTestTimeout)
		defer cancel()

		info, err := getMCP(ctx)
		if err != nil {
			t.Fatalf("failed discovering MCP name: %v", err)
		}

		kubeconfigPath, err := getKubeconfig(ctx, info.Name)
		if err != nil {
			t.Fatalf("failed generating kubeconfig: %v", err)
		}

		waitAllPodsRunning(ctx, t, kubeconfigPath, allPodsHealthyTimeout)
	})
}

