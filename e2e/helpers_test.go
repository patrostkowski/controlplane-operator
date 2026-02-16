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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	setupCloudProviderWait = 120 * time.Second
	setupOperatorWait      = 60 * time.Second

	resourceExistsPollInterval = 2 * time.Second

	cleanupAttempts = 3
	cleanupDelay    = 2 * time.Second

	podsPollInterval = 3 * time.Second
)

var (
	cmdTask    = "task"
	cmdKind    = "kind"
	cmdKubectl = "kubectl"
	cmdHelm    = "helm"
	cmdDocker  = "docker"
	cmdBash    = "bash"

	pathCRDs              = "../config/crd/"
	pathDeployManifests   = "../config/deploy/manifests.yaml"
	pathCloudProviderKind = "./testdata/cloud-provider-kind.yaml"
	pathKindConfig        = "./testdata/kind.yaml"

	certManagerCommandArgs = []string{
		"upgrade",
		"--install",
		"cert-manager",
		"oci://quay.io/jetstack/charts/cert-manager",
		"--namespace", "cert-manager",
		"--create-namespace",
		"--set", "crds.enabled=true",
		"--wait",
	}
)

type WorkloadKind string

const (
	WorkloadDeployment  WorkloadKind = "deployment"
	WorkloadStatefulSet WorkloadKind = "statefulset"
)

type WorkloadRef struct {
	Kind WorkloadKind
	NS   string
	Name string

	ExistsTimeout time.Duration
	ReadyTimeout  time.Duration
}

func WaitForWorkloads(ctx context.Context, refs []WorkloadRef) error {
	for _, r := range refs {
		if err := waitResourceExists(ctx, string(r.Kind), r.NS, r.Name, r.ExistsTimeout); err != nil {
			return err
		}
		switch r.Kind {
		case WorkloadDeployment:
			if err := waitDeployment(ctx, r.NS, r.Name, r.ReadyTimeout); err != nil {
				return err
			}
		case WorkloadStatefulSet:
			if err := waitStatefulSet(ctx, r.NS, r.Name, r.ReadyTimeout); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown workload kind: %q", r.Kind)
		}
	}
	return nil
}

func (tc *TestContext) Setup(ctx context.Context) error {
	tc.log.Info("running codegen")
	if err := run(ctx, cmdTask, "dev:codegen"); err != nil {
		return fmt.Errorf("make manifests failed: %w", err)
	}

	tc.log.Info("building ctl binary")
	if err := run(ctx, cmdTask, "dev:build-tesseractl"); err != nil {
		return fmt.Errorf("ctl build failed: %w", err)
	}

	tc.log.Info("building docker image")
	if err := run(ctx, cmdTask, "dev:docker-build"); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	tc.log.Info("creating kind cluster")
	if err := ensureKindCluster(ctx); err != nil {
		return fmt.Errorf("kind create failed: %w", err)
	}

	tc.log.Info("loading docker image to kind")
	if err := run(ctx, cmdKind, "load", "docker-image", imageName, "--name", clusterName); err != nil {
		return fmt.Errorf("kind load failed: %w", err)
	}

	tc.log.Info("applying opreator CRDs")
	if err := run(ctx, cmdKubectl, "apply", "-f", pathCRDs); err != nil {
		return fmt.Errorf("failed: %w", err)
	}

	tc.log.Info("installing cert manager dependency")
	if err := run(ctx, cmdHelm, certManagerCommandArgs...); err != nil {
		return fmt.Errorf("failed: %w", err)
	}

	tc.log.Info("creating kind cloud provider")
	if err := run(ctx, cmdKubectl, "apply", "-f", pathCloudProviderKind); err != nil {
		return fmt.Errorf("failed: %w", err)
	}

	tc.log.Info("waiting for cloud provider to become healthy")
	if err := waitDeployment(ctx, "kube-system", "cloud-provider-kind", setupCloudProviderWait); err != nil {
		return fmt.Errorf("failed: %w", err)
	}

	tc.log.Info("deploying opreator manifests")
	if err := run(ctx, cmdKubectl, "apply", "-f", pathDeployManifests); err != nil {
		return fmt.Errorf("failed: %w", err)
	}

	tc.log.Info("waiting for operator to become healthy")
	if err := waitDeployment(ctx, "controlplane-system", "controlplane-operator-controller-manager", setupOperatorWait); err != nil {
		return fmt.Errorf("failed: %w", err)
	}

	return nil
}

func (tc *TestContext) CleanUp(ctx context.Context) error {
	var errs []error

	if err := retry(ctx, cleanupAttempts, cleanupDelay, func() error {
		tc.log.Info("removing kind cluster")
		return run(ctx, cmdKind, "delete", "cluster", "--name", clusterName)
	}); err != nil {
		errs = append(errs, fmt.Errorf("kind delete cluster: %w", err))
	}

	if err := retry(ctx, cleanupAttempts, cleanupDelay, func() error {
		tc.log.Info("removing conainers using kindest/node image")
		return dockerRemoveByAncestor(ctx, "kindest/node")
	}); err != nil {
		errs = append(errs, fmt.Errorf("docker remove ancestor kindest/node: %w", err))
	}

	if err := retry(ctx, cleanupAttempts, cleanupDelay, func() error {
		tc.log.Info("removing kind cloud provider envoy containers")
		return dockerRemoveByFilter(ctx, "name=^kindccm-")
	}); err != nil {
		errs = append(errs, fmt.Errorf("docker remove kindccm containers: %w", err))
	}

	if err := retry(ctx, cleanupAttempts, cleanupDelay, func() error {
		tc.log.Info("removing mcp-* containers")
		return dockerRemoveByNamePrefix(ctx, "mcp-")
	}); err != nil {
		errs = append(errs, fmt.Errorf("docker remove mcp-* containers: %w", err))
	}

	if err := retry(ctx, cleanupAttempts, cleanupDelay, func() error {
		tc.log.Info("removing mcp-* volumes")
		return dockerVolumeRemoveByNamePrefix(ctx, "mcp-")
	}); err != nil {
		errs = append(errs, fmt.Errorf("docker remove mcp-* volumes: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup failed: %w", errors.Join(errs...))
	}
	return nil
}

func retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	var lastErr error

	for i := 1; i <= attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}

func dockerRemoveByNamePrefix(ctx context.Context, prefix string) error {
	out, err := runOutput(ctx, cmdDocker, "ps", "-aq", "--filter", "name=^"+prefix)
	if err != nil {
		return fmt.Errorf("docker ps (name prefix=%q) failed: %w", prefix, err)
	}

	ids := strings.Fields(strings.TrimSpace(out))
	if len(ids) == 0 {
		return nil
	}

	return run(ctx, cmdDocker, append([]string{"rm", "-f"}, ids...)...)
}

func dockerVolumeRemoveByNamePrefix(ctx context.Context, prefix string) error {
	out, err := runOutput(ctx, cmdDocker, "volume", "ls", "-q")
	if err != nil {
		return fmt.Errorf("docker volume ls failed: %w", err)
	}

	var names []string
	for _, v := range strings.Fields(out) {
		if strings.HasPrefix(v, prefix) {
			names = append(names, v)
		}
	}
	if len(names) == 0 {
		return nil
	}

	return run(ctx, cmdDocker, append([]string{"volume", "rm", "-f"}, names...)...)
}

func dockerRemoveByAncestor(ctx context.Context, image string) error {
	out, err := runOutput(ctx,
		cmdDocker, "ps", "-aq",
		"--filter", "ancestor="+image,
	)
	if err != nil {
		return fmt.Errorf("docker ps failed: %w", err)
	}

	ids := strings.Fields(strings.TrimSpace(out))
	if len(ids) == 0 {
		return nil
	}

	return run(ctx, cmdDocker, append([]string{"rm", "-f"}, ids...)...)
}

func dockerRemoveByFilter(ctx context.Context, filters ...string) error {
	args := []string{"ps", "-aq"}
	for _, f := range filters {
		args = append(args, "--filter", f)
	}

	out, err := runOutput(ctx, cmdDocker, args...)
	if err != nil {
		return fmt.Errorf("docker ps (filters=%v) failed: %w", filters, err)
	}

	ids := strings.Fields(strings.TrimSpace(out))
	if len(ids) == 0 {
		return nil
	}

	return run(ctx, cmdDocker, append([]string{"rm", "-f"}, ids...)...)
}

func DumpDebug(ctx context.Context) error {
	log.Info("Dumping cluster state")

	_ = run(ctx, cmdKubectl, "get", "all", "-A")
	_ = run(ctx, cmdKubectl, "get", "events", "-A", "--sort-by=.lastTimestamp")
	_ = run(ctx, cmdKubectl, "-n", "system", "logs", "deploy/controller-manager", "--all-containers=true", "--tail=200")

	return nil
}

func ensureKindCluster(ctx context.Context) error {
	out, _ := runOutput(ctx, cmdKind, "get", "clusters")
	if strings.Contains(out, clusterName) {
		return nil
	}
	return run(ctx, cmdKind, "create", "cluster", "--name", clusterName, "--config", pathKindConfig, "--wait", "60s")
}

func waitResourceExists(ctx context.Context, kind, ns, name string, timeout time.Duration) error {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(resourceExistsPollInterval)
	defer ticker.Stop()

	for {
		args := []string{}
		if ns != "" {
			args = append(args, "-n", ns)
		}
		args = append(args, "get", kind, name)

		if _, err := runOutput(cctx, cmdKubectl, args...); err == nil {
			return nil
		}

		select {
		case <-cctx.Done():
			return fmt.Errorf("timed out waiting for %s/%s to exist in ns=%q: %w", kind, name, ns, cctx.Err())
		case <-ticker.C:
		}
	}
}

func waitDeployment(ctx context.Context, ns, name string, timeout time.Duration) error {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(
		cctx,
		cmdKubectl,
		"-n", ns,
		"rollout", "status",
		"deployment/"+name,
		"--timeout", timeout.String(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitStatefulSet(ctx context.Context, ns, name string, timeout time.Duration) error {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(
		cctx,
		cmdKubectl,
		"-n", ns,
		"rollout", "status",
		"statefulset/"+name,
		"--timeout", timeout.String(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func kubectlJSONPath(ctx context.Context, args []string, jsonPath string) (string, error) {
	full := append([]string{}, args...)
	full = append(full, "-o", "jsonpath="+jsonPath)
	out, err := runOutput(ctx, cmdKubectl, full...)
	return strings.TrimSpace(out), err
}

var imageTagRe = regexp.MustCompile(`.+:(.+)$`)

func extractImageTag(image string) string {
	if strings.Contains(image, "@sha256:") {
		return ""
	}
	m := imageTagRe.FindStringSubmatch(image)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func assertAllPodsRunning(ctx context.Context, t *testing.T, kubeconfigPath string) {
	t.Helper()

	out, err := runOutput(ctx, cmdKubectl,
		"--kubeconfig", kubeconfigPath,
		"get", "pods", "-A",
		"--no-headers",
	)
	if err != nil {
		t.Fatalf("kubectl get pods failed: %v", err)
	}

	var bad []string
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		status := parts[3]
		switch status {
		case "Running", "Completed", "Succeeded":
		default:
			bad = append(bad, line)
		}
	}

	if len(bad) > 0 {
		var b strings.Builder
		b.WriteString("found pods not Running/Completed:\n")
		for _, line := range bad {
			b.WriteString(" - ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		t.Fatalf("%s\n", b.String())
	}
}

func waitAllPodsRunning(ctx context.Context, t *testing.T, kubeconfigPath string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastBad string

	for time.Now().Before(deadline) {
		out, err := runOutput(ctx, cmdKubectl,
			"--kubeconfig", kubeconfigPath,
			"get", "pods", "-A",
			"--no-headers",
		)
		if err != nil {
			lastBad = fmt.Sprintf("kubectl get pods failed: %v", err)
			time.Sleep(podsPollInterval)
			continue
		}

		var bad []string
		lines := strings.Split(strings.TrimSpace(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 4 {
				continue
			}

			status := parts[3]
			switch status {
			case "Running", "Completed", "Succeeded":
			default:
				bad = append(bad, line)
			}
		}

		if len(bad) == 0 {
			return
		}

		var b strings.Builder
		b.WriteString("pods not Running/Completed yet:\n")
		for _, line := range bad {
			b.WriteString(" - ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		lastBad = b.String()
		log.Info("pods not ready", "pods", lastBad)
		time.Sleep(podsPollInterval)
	}

	t.Fatalf("timed out after %s waiting for pods to become Running/Completed.\nLast observed:\n%s",
		timeout.String(), lastBad)
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return b.String(), err
}

