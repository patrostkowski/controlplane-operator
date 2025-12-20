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
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	agentv1alpha1 "github.com/patrostkowski/controlplane-operator/proto/agent/v1alpha1"
)

const listenAddr = ":32137"

var kubeletScheme = kuberuntime.NewScheme()
var kubeletCodecs = serializer.NewCodecFactory(kubeletScheme)

func init() {
	utilruntime.Must(kubeletconfigv1beta1.AddToScheme(kubeletScheme))
}

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

	kubeletConfigPath := "/etc/kubernetes/conifg.yaml"
	caPath := "/etc/kubernetes/pki/ca.crt"
	initKubeconfigPath := "/etc/kubernetes/kubeconfig"
	pkiDir := "/etc/kubernetes/pki"

	if err := validateCACertBundle(req.CACert); err != nil {
		klog.Error(err, "Failed to validate CA cert bundle")
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}
	if err := validateKubeconfigBytes(req.InitKubeconfig); err != nil {
		klog.Error(err, "Failed to validate kubeconfig")
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}
	if _, err := validateKubeletConfigBytes(req.KubeletConfig); err != nil {
		klog.Error(err, "Failed to validate kubelet config")
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Ensuring PKI directory exists", "dir", pkiDir)
	if err := os.MkdirAll(pkiDir, 0o755); err != nil {
		klog.Error(err, "Failed to ensure dir", pkiDir)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Creating file", "file", kubeletConfigPath)
	if err := writeToFile(kubeletConfigPath, req.KubeletConfig); err != nil {
		klog.Error(err, "Failed to write to file", kubeletConfigPath)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Creating file", "file", caPath)
	if err := writeToFile(caPath, req.CACert); err != nil {
		klog.Error(err, "Failed to write to file", caPath)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Creating file", "file", initKubeconfigPath)
	if err := writeToFile(initKubeconfigPath, req.InitKubeconfig); err != nil {
		klog.Error(err, "Failed to write to file", initKubeconfigPath)
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	klog.InfoS("Starting kubelet")
	if err := startKubelet(ctx); err != nil {
		klog.Error(err, "starting kubelet failed")
		return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_NOK}, err
	}

	return &agentv1alpha1.JoinResponse{Code: agentv1alpha1.StatusCode_STATUS_CODE_OK}, nil
}

func writeToFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("Could not open file %q: %s", path, err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("Could not write to file %q: %s", path, err)
	}

	return nil
}

func startKubelet(ctx context.Context) error {
	if _, err := exec.LookPath("systemctl"); err == nil {
		cmd := exec.CommandContext(ctx, "systemctl", "start", "kubelet")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemctl start kubelet failed: %w (output: %s)", err, string(out))
		}
		return nil
	}
	return fmt.Errorf("systemctl not found; cannot start kubelet")
}

func validateCACertBundle(pemBytes []byte) error {
	certs, err := cert.ParseCertsPEM(pemBytes)
	if err != nil {
		return fmt.Errorf("parse CA bundle PEM: %w", err)
	}
	if len(certs) == 0 {
		return fmt.Errorf("no certificates found in CA bundle")
	}

	hasCA := false
	for _, c := range certs {
		if c.IsCA {
			hasCA = true
			break
		}
	}
	if !hasCA {
		return fmt.Errorf("no CA certificates (IsCA=true) found in bundle")
	}

	for _, c := range certs {
		if _, err := x509.ParseCertificate(c.Raw); err != nil {
			return fmt.Errorf("x509 parse failed unexpectedly: %w", err)
		}
	}

	return nil
}

func validateKubeconfigBytes(kc []byte) error {
	cfg, err := clientcmd.Load(kc)
	if err != nil {
		return fmt.Errorf("parse kubeconfig: %w", err)
	}

	if len(cfg.Clusters) == 0 {
		return fmt.Errorf("kubeconfig has no clusters")
	}
	if cfg.CurrentContext == "" {
		return fmt.Errorf("kubeconfig has empty current-context")
	}

	cc := cfg.Contexts[cfg.CurrentContext]
	if cc == nil {
		return fmt.Errorf("current-context %q not found", cfg.CurrentContext)
	}
	cluster := cfg.Clusters[cc.Cluster]
	if cluster == nil {
		return fmt.Errorf("cluster %q not found", cc.Cluster)
	}
	if cluster.Server == "" {
		return fmt.Errorf("cluster.server is empty")
	}

	// Require CA either embedded or file path (since you write ca.crt)
	if len(cluster.CertificateAuthorityData) == 0 && cluster.CertificateAuthority == "" {
		return fmt.Errorf("cluster has neither certificate-authority-data nor certificate-authority")
	}

	// If file path is used, make sure it matches what you write
	if cluster.CertificateAuthority != "" && cluster.CertificateAuthority != "/etc/kubernetes/pki/ca.crt" {
		return fmt.Errorf("cluster.certificate-authority=%q (expected /etc/kubernetes/pki/ca.crt or embed certificate-authority-data)", cluster.CertificateAuthority)
	}

	return nil
}

func validateKubeletConfigBytes(b []byte) (*kubeletconfigv1beta1.KubeletConfiguration, error) {
	decoder := kubeletCodecs.UniversalDecoder(kubeletconfigv1beta1.SchemeGroupVersion)

	obj, _, err := decoder.Decode(b, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decode kubelet config: %w", err)
	}

	cfg, ok := obj.(*kubeletconfigv1beta1.KubeletConfiguration)
	if !ok {
		return nil, fmt.Errorf("decoded object is %T, expected *kubeletconfigv1beta1.KubeletConfiguration", obj)
	}

	if cfg.Authentication.X509.ClientCAFile == "" {
		return nil, fmt.Errorf("kubelet config: authentication.x509.clientCAFile is empty")
	}
	if cfg.Authorization.Mode == "" {
		return nil, fmt.Errorf("kubelet config: authorization.mode is empty")
	}
	if cfg.Authentication.X509.ClientCAFile != "/etc/kubernetes/pki/ca.crt" {
		return nil, fmt.Errorf("kubelet config: authentication.x509.clientCAFile=%q (expected /etc/kubernetes/pki/ca.crt)", cfg.Authentication.X509.ClientCAFile)
	}

	return cfg, nil
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
