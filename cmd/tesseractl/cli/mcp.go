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
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	mcpclient "github.com/patrostkowski/controlplane-operator/pkg/client"
	"go.yaml.in/yaml/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

const (
	adminConfigSecretKey = "kubeconfig"

	kubeconfigFlagName = "kubeconfig"
	namespaceFlagName  = "namespace"
	contextFlagName    = "context"
)

const (
	outputFlagName = "output"
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
		c.newMCPCommand(),
	)

	return &c
}

func (c *CLI) Run() error {
	return c.cmd.Execute()
}

func (c *CLI) newJoinCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeadm-join",
		Short: "Print kuebadm join command",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			kcfgBytes, ok := sec.Data[adminConfigSecretKey]
			if !ok {
				return fmt.Errorf("secret %s/%s missing data[%q]", sec.Namespace, sec.Name, adminConfigSecretKey)
			}

			childRestCfg, err := clientcmd.RESTConfigFromKubeConfig(kcfgBytes)
			if err != nil {
				return err
			}
			child, err := kubernetes.NewForConfig(childRestCfg)
			if err != nil {
				return err
			}

			token, err := c.getLatestBootstrapToken(cmd.Context(), child)
			if err != nil {
				return err
			}

			caHash, err := c.getDiscoveryCAHash(cmd.Context(), child)
			if err != nil {
				return err
			}

			endpoint, err := c.getAPIEndpoint(cmd.Context(), child, childRestCfg.Host)
			if err != nil {
				return err
			}

			// TODO: support kubeadm flags from join command
			fmt.Fprintf(os.Stdout, "kubeadm join %s --token %s --discovery-token-ca-cert-hash %s --ignore-preflight-errors=SystemVerification\n",
				endpoint, token, caHash,
			)
			return nil
		},
	}
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
		c.newJoinCommand(),
		c.newGetCommand(),
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

func (c *CLI) getLatestBootstrapToken(ctx context.Context, cs kubernetes.Interface) (string, error) {
	secs, err := cs.CoreV1().Secrets("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	var tokens []corev1.Secret
	for _, s := range secs.Items {
		if strings.HasPrefix(s.Name, "bootstrap-token-") {
			tokens = append(tokens, s)
		}
	}
	if len(tokens) == 0 {
		return "", fmt.Errorf("no bootstrap-token-* secrets found in kube-system")
	}

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].CreationTimestamp.Time.Before(tokens[j].CreationTimestamp.Time)
	})
	latest := tokens[len(tokens)-1]

	id := strings.TrimSpace(string(latest.Data["token-id"]))
	sec := strings.TrimSpace(string(latest.Data["token-secret"]))
	if id == "" || sec == "" {
		return "", fmt.Errorf("bootstrap token secret %q missing token-id/token-secret", latest.Name)
	}
	return id + "." + sec, nil
}

func (c *CLI) getDiscoveryCAHash(ctx context.Context, cs kubernetes.Interface) (string, error) {
	cm, err := cs.CoreV1().ConfigMaps("kube-public").Get(ctx, "cluster-info", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	kcfg := cm.Data["kubeconfig"]
	if strings.TrimSpace(kcfg) == "" {
		return "", fmt.Errorf("kube-public/cluster-info missing .data.kubeconfig")
	}

	type kc struct {
		Clusters []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data"`
			} `yaml:"cluster"`
		} `yaml:"clusters"`
	}
	var obj kc
	if err := yaml.Unmarshal([]byte(kcfg), &obj); err != nil {
		return "", err
	}
	if len(obj.Clusters) == 0 || obj.Clusters[0].Cluster.CertificateAuthorityData == "" {
		return "", fmt.Errorf("cluster-info kubeconfig missing certificate-authority-data")
	}

	caDER, err := base64.StdEncoding.DecodeString(obj.Clusters[0].Cluster.CertificateAuthorityData)
	if err != nil {
		return "", err
	}

	cert, err := x509.ParseCertificate(caDER)
	if err != nil {
		if block, _ := pem.Decode(caDER); block != nil {
			cert, err = x509.ParseCertificate(block.Bytes)
		}
		if err != nil {
			return "", err
		}
	}

	spkiDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(spkiDER)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (c *CLI) getAPIEndpoint(ctx context.Context, cs kubernetes.Interface, fallbackServer string) (string, error) {
	svc, err := cs.CoreV1().Services("default").Get(ctx, "kube-apiserver", metav1.GetOptions{})
	if err == nil {
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			ing := svc.Status.LoadBalancer.Ingress[0]
			host := ing.IP
			if host == "" {
				host = ing.Hostname
			}
			if host != "" {
				return host + ":6443", nil
			}
		}
	}

	u, err := url.Parse(fallbackServer)
	if err != nil {
		return "", fmt.Errorf("cannot parse kubeconfig server %q: %w", fallbackServer, err)
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("kubeconfig server missing host: %q", fallbackServer)
	}

	if !strings.Contains(host, ":") {
		host = host + ":6443"
	}
	return host, nil
}

func (c *CLI) newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [mcp-name]",
		Short: "Get ManagedControlPlane(s) in a namespace",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eff, err := c.effectiveKubeconfigFlags()
			if err != nil {
				return err
			}

			c.k8s, err = mcpclient.NewClient(eff.Context, eff.KubeconfigPath)
			if err != nil {
				return err
			}

			outFmt, _ := cmd.Flags().GetString(outputFlagName)

			if len(args) == 1 {
				name := args[0]
				obj := &mcpv1alpha1.ManagedControlPlane{}
				if err := c.k8s.Get(cmd.Context(), types.NamespacedName{
					Namespace: eff.Namespace,
					Name:      name,
				}, obj); err != nil {
					return err
				}

				if outFmt == "" {
					list := &mcpv1alpha1.ManagedControlPlaneList{
						Items: []mcpv1alpha1.ManagedControlPlane{*obj},
					}
					printMCPTable(list)
					return nil
				}

				obj.APIVersion = mcpv1alpha1.SchemeGroupVersion.String()
				obj.Kind = mcpv1alpha1.KindManagedControlPlane
				return printObject(obj, outFmt)
			}

			list := &mcpv1alpha1.ManagedControlPlaneList{}
			if err := c.k8s.List(cmd.Context(), list, client.InNamespace(eff.Namespace)); err != nil {
				return err
			}

			if outFmt == "" {
				printMCPTable(list)
				return nil
			}

			list.APIVersion = mcpv1alpha1.SchemeGroupVersion.String()
			list.Kind = "ManagedControlPlaneList"
			for i := range list.Items {
				list.Items[i].APIVersion = mcpv1alpha1.SchemeGroupVersion.String()
				list.Items[i].Kind = mcpv1alpha1.KindManagedControlPlane
			}
			return printObject(list, outFmt)
		},
	}

	cmd.Flags().StringP(outputFlagName, "o", "", "output format: yaml|json (default: table)")
	return cmd
}
