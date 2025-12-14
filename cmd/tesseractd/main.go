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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"

	agentv1alpha1 "github.com/patrostkowski/controlplane-operator/proto/agent/v1alpha1"
)

const listenAddr = ":32137"

type agentServer struct {
	agentv1alpha1.UnimplementedAgentServiceServer
	mu sync.Mutex
}

func main() {
	klog.InfoS(
		"Hello from tesseractd!",
		"go", goInfo(),
		"kernel", kernelInfo(),
	)

	listener, err := net.Listen("tcp", ":32137")
	if err != nil {
		klog.Error(err, "failed to listen")
		panic(err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	agentv1alpha1.RegisterAgentServiceServer(grpcServer, &agentServer{})
	reflection.Register(grpcServer)

	klog.InfoS("gRPC server listening", "addr", listenAddr)

	if err := grpcServer.Serve(listener); err != nil {
		klog.Error(err, "gRPC server stopped")
		panic(err)
	}
}

func (s *agentServer) Status(ctx context.Context, _ *agentv1alpha1.StatusRequest) (*agentv1alpha1.StatusResponse, error) {
	return &agentv1alpha1.StatusResponse{
		Code:       agentv1alpha1.StatusCode_STATUS_CODE_OK,
		Message:    "ok",
		Timestamp:  time.Now().Unix(),
		Kubelet:    &agentv1alpha1.ComponentStatus{},
		Containerd: &agentv1alpha1.ComponentStatus{},
	}, nil
}

func (s *agentServer) Join(ctx context.Context, req *agentv1alpha1.JoinRequest) (*agentv1alpha1.JoinResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := strings.TrimSpace(req.GetEndpoint())
	token := strings.TrimSpace(req.GetToken())

	klog.InfoS("Join requested", "endpoint", endpoint)

	if endpoint == "" {
		err := errors.New("endpoint is required")
		klog.Error(err, "Join validation failed")
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}
	if token == "" {
		err := errors.New("token is required")
		klog.Error(err, "Join validation failed", "endpoint", endpoint)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Validating endpoint", "endpoint", endpoint)
	if ip := net.ParseIP(endpoint); ip == nil {
		err := fmt.Errorf("endpoint must be an IP address, got %q", endpoint)
		klog.Error(err, "Join validation failed", "endpoint", endpoint)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Ensuring /etc/hosts contains kubernetes entry", "endpoint", endpoint)
	if err := ensureHostsHasKubernetes(endpoint); err != nil {
		wrapped := fmt.Errorf("updating /etc/hosts: %w", err)
		klog.Error(wrapped, "Join step failed", "step", "ensureHostsHasKubernetes", "endpoint", endpoint)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, wrapped
	}

	pkiDir := "/etc/kubernetes/pki"
	klog.InfoS("Ensuring PKI directory exists", "dir", pkiDir)
	if err := os.MkdirAll(pkiDir, 0o755); err != nil {
		wrapped := fmt.Errorf("creating %s: %w", pkiDir, err)
		klog.Error(wrapped, "Join step failed", "step", "MkdirAll", "dir", pkiDir)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, wrapped
	}

	caPath := filepath.Join(pkiDir, "ca.crt")
	klog.InfoS("Fetching Kubernetes API certificate", "server", "kubernetes:6443", "out", caPath)
	if err := fetchFirstPeerCertToFile(ctx, "kubernetes:6443", caPath); err != nil {
		wrapped := fmt.Errorf("fetching api cert: %w", err)
		klog.Error(wrapped, "Join step failed", "step", "fetchFirstPeerCertToFile", "server", "kubernetes:6443", "out", caPath)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, wrapped
	}

	kubeconfigPath := "/etc/kubernetes/kubeconfig"
	klog.InfoS("Writing kubeconfig", "path", kubeconfigPath)
	if err := writeKubeconfig(kubeconfigPath, token); err != nil {
		wrapped := fmt.Errorf("writing kubeconfig: %w", err)
		klog.Error(wrapped, "Join step failed", "step", "writeKubeconfig", "path", kubeconfigPath)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, wrapped
	}

	klog.InfoS("Starting kubelet")
	if err := startKubelet(ctx); err != nil {
		wrapped := fmt.Errorf("starting kubelet: %w", err)
		klog.Error(wrapped, "Join step failed", "step", "startKubelet")
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, wrapped
	}

	klog.InfoS("Join completed successfully", "endpoint", endpoint)
	return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_OK}, nil
}

func ensureHostsHasKubernetes(endpoint string) error {
	const hostsPath = "/etc/hosts"

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return err
	}

	// If ANY non-comment line contains hostname "kubernetes", do nothing.
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, h := range fields[1:] {
			if h == "kubernetes" {
				return nil
			}
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	// Append new line
	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	// Ensure we append with a leading newline if file doesn't end with one
	needsNL := len(data) > 0 && data[len(data)-1] != '\n'
	if needsNL {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(fmt.Sprintf("%s kubernetes\n", endpoint))
	return err
}

func fetchFirstPeerCertToFile(ctx context.Context, hostport, outPath string) error {
	// Make sure openssl exists
	if _, err := exec.LookPath("openssl"); err != nil {
		return fmt.Errorf("openssl not found in PATH: %w", err)
	}

	// Equivalent-ish to: echo '' | openssl s_client -connect kubernetes:6443 -showcerts
	cmd := exec.CommandContext(ctx, "openssl", "s_client", "-connect", hostport, "-showcerts")
	cmd.Stdin = strings.NewReader("\n")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openssl s_client failed: %w (output: %s)", err, string(out))
	}

	pem, err := firstPEMCert(out)
	if err != nil {
		return err
	}

	// Write atomically
	tmp := outPath + ".tmp"
	if err := os.WriteFile(tmp, pem, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, outPath)
}

func firstPEMCert(opensslOutput []byte) ([]byte, error) {
	const begin = "-----BEGIN CERTIFICATE-----"
	const end = "-----END CERTIFICATE-----"

	s := string(opensslOutput)
	i := strings.Index(s, begin)
	if i < 0 {
		return nil, errors.New("no BEGIN CERTIFICATE found in openssl output")
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return nil, errors.New("no END CERTIFICATE found in openssl output")
	}
	j = i + j + len(end)

	block := s[i:j]
	// ensure trailing newline
	if !strings.HasSuffix(block, "\n") {
		block += "\n"
	}
	return []byte(block), nil
}

func writeKubeconfig(path string, token string) error {
	// Ensure /etc/kubernetes exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`apiVersion: v1
kind: Config

clusters:
- name: default
  cluster:
    certificate-authority: /etc/kubernetes/pki/ca.crt
    server: https://kubernetes:6443

contexts:
- name: default
  context:
    cluster: default
    user: default

current-context: default

users:
- name: default
  user:
    token: %s
`, token)

	tmp := path + ".tmp"
	// kubeconfig should be private
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func startKubelet(ctx context.Context) error {
	// Prefer systemctl if present
	if _, err := exec.LookPath("systemctl"); err == nil {
		// "enable --now" is handy, but "start" is closer to your requirement.
		cmd := exec.CommandContext(ctx, "systemctl", "start", "kubelet")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemctl start kubelet failed: %w (output: %s)", err, string(out))
		}
		return nil
	}

	// Fallback to service
	if _, err := exec.LookPath("service"); err == nil {
		cmd := exec.CommandContext(ctx, "service", "kubelet", "start")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("service kubelet start failed: %w (output: %s)", err, string(out))
		}
		return nil
	}

	return errors.New("neither systemctl nor service found; cannot start kubelet")
}

func goInfo() string {
	return fmt.Sprintf(
		"go=%s os=%s arch=%s cpu=%d",
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
	)
}

func kernelInfo() string {
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		return "kernel=unknown"
	}

	return fmt.Sprintf(
		"release=%s machine=%s",
		byteSliceToString(uts.Release[:]),
		byteSliceToString(uts.Machine[:]),
	)
}

func byteSliceToString(b []byte) string {
	n := 0
	for ; n < len(b); n++ {
		if b[n] == 0 {
			break
		}
	}
	return string(b[:n])
}
