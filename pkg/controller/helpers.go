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

package controller

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/state"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type Applier struct {
	k8s         client.Client
	scheme      *runtime.Scheme
	fieldOwner  string
	log         logr.Logger
	setOwnerRef bool
}

type ApplierOption func(*Applier)

func WithOwnerRef(enabled bool) ApplierOption {
	return func(a *Applier) { a.setOwnerRef = enabled }
}

func NewApplier(k8s client.Client, scheme *runtime.Scheme, log logr.Logger, fieldOwner string, opts ...ApplierOption) *Applier {
	a := &Applier{
		k8s:         k8s,
		scheme:      scheme,
		fieldOwner:  fieldOwner,
		log:         log.WithName("applier"),
		setOwnerRef: true,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func (a *Applier) Apply(ctx context.Context, owner client.Object, objs ...client.Object) error {
	for _, obj := range objs {
		if a.setOwnerRef {
			if err := controllerutil.SetControllerReference(owner, obj, a.scheme); err != nil {
				return fmt.Errorf("set owner ref for %T/%s: %w", obj, obj.GetName(), err)
			}
		}

		gvk, err := apiutil.GVKForObject(obj, a.scheme)
		if err != nil {
			return fmt.Errorf("resolve gvk for %T/%s: %w", obj, obj.GetName(), err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)

		a.log.Info("Applying",
			"gvk", gvk.String(),
			"ns", obj.GetNamespace(),
			"name", obj.GetName(),
			"fieldOwner", a.fieldOwner,
		)

		if err := a.k8s.Patch(
			ctx,
			obj,
			client.Apply,
			client.FieldOwner(a.fieldOwner),
			client.ForceOwnership,
		); err != nil {
			return fmt.Errorf("apply %s %s/%s: %w", gvk.Kind, obj.GetNamespace(), obj.GetName(), err)
		}
	}
	return nil
}

func (r *ManagedControlPlaneReconciler) setStatus(
	ctx context.Context,
	mcp *mcpv1alpha1.ManagedControlPlane,
	condType mcpv1alpha1.Condition,
	condStatus metav1.ConditionStatus,
	reason mcpv1alpha1.Reason,
	message mcpv1alpha1.Message,
	ready bool,
) error {
	before := mcp.DeepCopy()

	mcp.Status.Ready = ptr.To(ready)
	mcp.Status.Message = string(message)

	newC := metav1.Condition{
		Type:               string(condType),
		Status:             condStatus,
		Reason:             string(reason),
		Message:            string(message),
		ObservedGeneration: mcp.GetGeneration(),
	}
	meta.SetStatusCondition(&mcp.Status.Conditions, newC)

	oldC := meta.FindStatusCondition(before.Status.Conditions, newC.Type)
	sameCond := oldC != nil &&
		oldC.Status == newC.Status &&
		oldC.Reason == newC.Reason &&
		oldC.Message == newC.Message &&
		oldC.ObservedGeneration == newC.ObservedGeneration

	if before.Status.Ready == mcp.Status.Ready &&
		before.Status.Message == mcp.Status.Message &&
		sameCond {
		return nil
	}

	return r.Status().Patch(ctx, mcp, client.MergeFrom(before))
}

func (r *ManagedControlPlaneReconciler) statusWaiting(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane, msg mcpv1alpha1.Message) error {
	return r.setStatus(ctx, mcp,
		state.ConditionReady,
		metav1.ConditionFalse,
		state.ReasonWaitingForResources,
		msg,
		false,
	)
}

func (r *ManagedControlPlaneReconciler) statusFailed(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane, msg mcpv1alpha1.Message) error {
	return r.setStatus(ctx, mcp,
		state.ConditionReady,
		metav1.ConditionFalse,
		state.ReasonComponentFailed,
		msg,
		false,
	)
}

func (r *ManagedControlPlaneReconciler) statusReady(ctx context.Context, mcp *mcpv1alpha1.ManagedControlPlane) error {
	return r.setStatus(ctx, mcp,
		state.ConditionReady,
		metav1.ConditionTrue,
		state.ReasonAllResourcesReady,
		state.MessageAllResourcesReady,
		true,
	)
}

const (
	defaultMCName             = "default"
	defaultJoinTokenNS        = "kube-system"
	defaultJoinTokenName      = "bootstrap-token"
	defaultJoinTokenIDKey     = "token-id"
	defaultJoinTokenSecretKey = "token-secret"

	caPathOnNode = "/etc/kubernetes/pki/ca.crt"
)

const (
	// todo remove
	BootstrapTokenLabel = "controlplane.patrostkowski.dev/bootstrap-token"
	BootstrapTokenValue = "true"
)

func (r *ManagedControlPlaneReconciler) reconcileDefaultMachineConfiguration(
	ctx context.Context,
	mcpObj *mcpv1alpha1.ManagedControlPlane,
	cp client.Client, // <— pass the child cluster client you already build in Reconcile
) (ctrl.Result, error) {
	if mcpObj.Status.Address == "" {
		r.Log.Info("API address not set yet")
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	// 1) CA SecretRef from same ns (local)
	apiPKI := pki.New(mcpObj).APIServer()
	// Adjust this field name to whatever you actually have in your PKI model:
	// In your cluster it looks like "managed-ca".
	caSecretName := apiPKI.ClientCA.SecretName

	// 2) Join token Secret from child/controlplane cluster (kube-system)
	var tokenSecrets corev1.SecretList
	if err := cp.List(
		ctx,
		&tokenSecrets,
		client.InNamespace(defaultJoinTokenNS),
		client.MatchingLabels{
			BootstrapTokenLabel: BootstrapTokenValue,
		},
	); err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	if len(tokenSecrets.Items) == 0 {
		r.Log.Info("No bootstrap token found yet",
			"namespace", defaultJoinTokenNS,
			"label", BootstrapTokenLabel,
		)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, nil
	}

	if len(tokenSecrets.Items) > 1 {
		return ctrl.Result{}, fmt.Errorf(
			"multiple bootstrap tokens found (%d); expected exactly one",
			len(tokenSecrets.Items),
		)
	}

	tokenSecret := &tokenSecrets.Items[0]

	tokenIDBytes, _ := tokenSecret.Data[defaultJoinTokenIDKey]
	tokenSecretBytes, _ := tokenSecret.Data[defaultJoinTokenSecretKey]

	joinToken := string(tokenIDBytes) + "." + string(tokenSecretBytes)

	// 3) Build init kubeconfig (file path CA + token)
	serverURL := "https://" + mcpObj.Status.Address + ":6443"
	initCfg := &clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			"default": {
				Server:               serverURL,
				CertificateAuthority: caPathOnNode,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"default": {
				Cluster:  "default",
				AuthInfo: "default",
			},
		},
		CurrentContext: "default",
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"default": {
				Token: joinToken,
			},
		},
	}

	initKubeconfigBytes, err := clientcmd.Write(*initCfg)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, fmt.Errorf("write init kubeconfig: %w", err)
	}

	// 4) Build kubelet config YAML (you can keep it as YAML bytes blob)
	// If you already have this YAML template somewhere, you can just format+[]byte it.
	kubeletYAML := []byte(`apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cgroupDriver: systemd
containerRuntimeEndpoint: "unix:///run/crio/crio.sock"
failSwapOn: false
clusterDomain: cluster.local
clusterDNS:
  - 10.96.0.10
resolvConf: "/run/systemd/resolve/resolv.conf"
serverTLSBootstrap: true
rotateCertificates: true
authentication:
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
  anonymous:
    enabled: false
  webhook:
    enabled: true
authorization:
  mode: Webhook
readOnlyPort: 0
healthzBindAddress: 127.0.0.1
healthzPort: 10248
port: 10250
`)

	// Optional: normalize YAML (not required, but makes diffs stable)
	kubeletJSON, err := yaml.YAMLToJSON(kubeletYAML)
	if err == nil {
		if normalized, err2 := yaml.JSONToYAML(kubeletJSON); err2 == nil {
			kubeletYAML = normalized
		}
	}

	// Ensure/create MC and fill spec
	mc := &mcpv1alpha1.MachineConfigruation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultMCName,
			Namespace: mcpObj.Namespace,
		},
	}

	err = utils.EnsureCreatedAndOwned(ctx, r.Client, r.Scheme, mcpObj, mc, r.Log, func(obj client.Object) error {
		cur := obj.(*mcpv1alpha1.MachineConfigruation)
		if cur.Spec.CACertSecretRef == nil {
			cur.Spec.CACertSecretRef = &corev1.LocalObjectReference{}
		}
		cur.Spec.CACertSecretRef.Name = caSecretName

		// Your RemoteSecretReference currently only has SecretName
		if cur.Spec.JoinTokenSecretRef == nil {
			cur.Spec.JoinTokenSecretRef = &mcpv1alpha1.RemoteSecretReference{}
		}
		cur.Spec.JoinTokenSecretRef.SecretName = defaultJoinTokenName

		// Set the blobs only if changed (avoid endless updates)
		if !bytes.Equal(cur.Spec.InitKubeconfig, initKubeconfigBytes) {
			cur.Spec.InitKubeconfig = initKubeconfigBytes
		}
		if !bytes.Equal(cur.Spec.KubeletConfiguration, kubeletYAML) {
			cur.Spec.KubeletConfiguration = kubeletYAML
		}

		return nil
	})
	if err != nil {
		r.Log.Error(err, "failed to ensure default MachineConfiguration", "name", defaultMCName)
		return ctrl.Result{RequeueAfter: RequeueAfterFailure}, err
	}

	return ctrl.Result{}, nil
}
