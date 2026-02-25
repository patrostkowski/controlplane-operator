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

package capi

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	capiv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.cluster.x-k8s.io/v1alpha1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	capiControllerName = "capi-mcp-controller"
	clusterNameLabel   = "cluster.x-k8s.io/cluster-name"
	capiSecretType = "cluster.x-k8s.io/secret"
)

type ManagedControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ManagedControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx).WithValues("mcp", req.NamespacedName)

	capiManagedControlPlane := &capiv1alpha1.ManagedControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, capiManagedControlPlane); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !capiManagedControlPlane.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Ensure backing ManagedControlPlane exists.
	backing := &mcpv1alpha1.ManagedControlPlane{}
	key := types.NamespacedName{Namespace: capiManagedControlPlane.Namespace, Name: capiManagedControlPlane.Name}
	create := false
	if err := r.Get(ctx, key, backing); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		create = true
		backing = &mcpv1alpha1.ManagedControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: capiManagedControlPlane.Namespace,
				Name:      capiManagedControlPlane.Name,
			},
		}
	}

	// ---------------------------------------------------------------------
	// 1) BASE SPEC from ProviderSpec (if present)
	// ---------------------------------------------------------------------
	// If ProviderSpec is set, treat it as "desired backing spec".
	// This prevents creating an incomplete backing ManagedControlPlane that later panics.
	tplSpec, err := r.providerSpecFromTemplateRef(ctx, capiManagedControlPlane)
	if err != nil {
		return ctrl.Result{}, err
	}

	switch {
	case tplSpec != nil:
		// TemplateRef wins over ManagedControlPlane.Spec.ProviderSpec.
		backing.Spec = *tplSpec

	case capiManagedControlPlane.Spec.ProviderSpec != nil:
		backing.Spec = *capiManagedControlPlane.Spec.ProviderSpec
	}

	// ---------------------------------------------------------------------
	// 2) ENFORCE contract-owned fields (version + replicas)
	// ---------------------------------------------------------------------
	desiredVersion := normalizeK8sVersion(capiManagedControlPlane.Spec.Version)
	backing.Spec.Kubernetes.Version = desiredVersion

	if capiManagedControlPlane.Spec.Replicas != nil {
		rep := ptr.To(*capiManagedControlPlane.Spec.Replicas)

		if backing.Spec.APIServer == nil {
			backing.Spec.APIServer = &mcpv1alpha1.APIServerSpec{}
		}
		if backing.Spec.ControllerManager == nil {
			backing.Spec.ControllerManager = &mcpv1alpha1.ControllerManagerSpec{}
		}
		if backing.Spec.Scheduler == nil {
			backing.Spec.Scheduler = &mcpv1alpha1.SchedulerSpec{}
		}
		if backing.Spec.ETCD == nil {
			backing.Spec.ETCD = &mcpv1alpha1.ETCDSpec{}
		}

		backing.Spec.APIServer.Replicas = rep
		backing.Spec.ControllerManager.Replicas = rep
		backing.Spec.Scheduler.Replicas = rep
		backing.Spec.ETCD.Replicas = rep
	}

	// ---------------------------------------------------------------------
	// 3) ENSURE required fields exist (networking)
	// ---------------------------------------------------------------------
	// You made networking required in the ManagedControlPlane controller; guarantee it here.
	// Adjust defaults to whatever you want, or return an error if you prefer strictness.
	if backing.Spec.Kubernetes.Networking == nil {
		backing.Spec.Kubernetes.Networking = &mcpv1alpha1.NetworkingSpec{}
	}
	if strings.TrimSpace(backing.Spec.Kubernetes.Networking.ServiceCIDR) == "" {
		backing.Spec.Kubernetes.Networking.ServiceCIDR = "10.96.0.0/12"
	}
	if strings.TrimSpace(backing.Spec.Kubernetes.Networking.PodCIDR) == "" {
		backing.Spec.Kubernetes.Networking.PodCIDR = "10.244.0.0/16"
	}

	// ---------------------------------------------------------------------
	// 4) Copy labels/annotations
	// ---------------------------------------------------------------------
	if backing.Labels == nil {
		backing.Labels = map[string]string{}
	}
	for k, v := range capiManagedControlPlane.Labels {
		backing.Labels[k] = v
	}
	if backing.Annotations == nil {
		backing.Annotations = map[string]string{}
	}
	for k, v := range capiManagedControlPlane.Annotations {
		backing.Annotations[k] = v
	}

	// Set owner reference (CAPI ManagedControlPlane owns backing ManagedControlPlane).
	if err := controllerutil.SetControllerReference(capiManagedControlPlane, backing, r.Scheme); err != nil {
		lg.Info("SetControllerReference failed; falling back to explicit OwnerReference", "err", err)
		backing.OwnerReferences = appendOrReplaceOwnerRef(backing.OwnerReferences, metav1.OwnerReference{
			APIVersion: capiv1alpha1.SchemeGroupVersion.String(),
			Kind:       capiv1alpha1.KindManagedControlPlane,
			Name:       capiManagedControlPlane.Name,
			UID:        capiManagedControlPlane.UID,
			Controller: ptr.To(true),
		})
	}

	// Persist backing ManagedControlPlane.
	if create {
		if err := r.Create(ctx, backing); err != nil {
			return ctrl.Result{}, err
		}
		lg.Info("created backing ManagedControlPlane", "name", backing.Name)
	} else {
		if err := r.Update(ctx, backing); err != nil {
			return ctrl.Result{}, err
		}
	}

	// ---------------------------------------------------------------------
	// 5) Mirror status back to CAPI ManagedControlPlane
	// ---------------------------------------------------------------------
	initialized := backing.Status.Ready != nil && *backing.Status.Ready
	capiManagedControlPlane.Status.ObservedGeneration = capiManagedControlPlane.Generation
	capiManagedControlPlane.Status.Version = desiredVersion
	capiManagedControlPlane.Status.Initialization.ControlPlaneInitialized = ptr.To(initialized)

	if capiManagedControlPlane.Spec.Replicas != nil {
		rep := *capiManagedControlPlane.Spec.Replicas
		capiManagedControlPlane.Status.Replicas = &rep
		if initialized {
			capiManagedControlPlane.Status.ReadyReplicas = &rep
			capiManagedControlPlane.Status.AvailableReplicas = &rep
			capiManagedControlPlane.Status.UpToDateReplicas = &rep
		}
	}

	now := metav1.Now()
	availableStatus := metav1.ConditionFalse
	reason := "NotReady"
	message := backing.Status.Message
	if initialized {

		cluster, err := r.getOwningClusterForManagedControlPlane(ctx, capiManagedControlPlane)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Ensure endpoint matches backing control plane address
		if err := r.ensureClusterControlPlaneEndpoint(ctx, cluster, backing); err != nil {
			return ctrl.Result{}, err
		}

		// 2) Derive suffix from backing.Name ("<cluster>-<suffix>")
		suffix, err := bootstrapSuffixFromBackingName(backing.Name)
		if err != nil {
			return ctrl.Result{}, err
		}

		// 3) Create the canonical CAPI cert secrets: <cluster>-ca/etcd/proxy/sa
		if err := r.reconcileCanonicalCertSecrets(ctx, cluster, suffix); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.ensureCAPIKubeconfigSecret(ctx, capiManagedControlPlane, backing); err != nil {
			return ctrl.Result{}, err
		}

		availableStatus = metav1.ConditionTrue
		reason = "Ready"
		if message == "" {
			message = "Control plane initialized"
		}
	}
	setCondition(&capiManagedControlPlane.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             availableStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: capiManagedControlPlane.Generation,
	})

	if err := r.Status().Update(ctx, capiManagedControlPlane); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

func SetupManagedControlPlaneController(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(capiControllerName).
		For(&capiv1alpha1.ManagedControlPlane{}).
		Owns(&mcpv1alpha1.ManagedControlPlane{}).
		Complete(&ManagedControlPlaneReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()})
}

func normalizeK8sVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func splitHostPortDefault(addr string, defaultPort int32) (string, int32) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", 0
	}
	if h, p, err := net.SplitHostPort(addr); err == nil {
		if pi, err := strconv.Atoi(p); err == nil {
			return h, int32(pi)
		}
		return h, defaultPort
	}
	if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[") {
		return addr, defaultPort
	}
	return addr, defaultPort
}

func setCondition(conds *[]metav1.Condition, c metav1.Condition) {
	if conds == nil {
		return
	}
	for i := range *conds {
		if (*conds)[i].Type == c.Type {
			(*conds)[i] = c
			return
		}
	}
	*conds = append(*conds, c)
}

func appendOrReplaceOwnerRef(in []metav1.OwnerReference, ref metav1.OwnerReference) []metav1.OwnerReference {
	for i := range in {
		if in[i].UID == ref.UID {
			in[i] = ref
			return in
		}
	}
	return append(in, ref)
}

var _ = fmt.Sprintf("%s", capiv1alpha1.SchemeGroupVersion.String())

func (r *ManagedControlPlaneReconciler) providerSpecFromTemplateRef(
	ctx context.Context,
	capiManagedControlPlane *capiv1alpha1.ManagedControlPlane,
) (*mcpv1alpha1.ManagedControlPlaneSpec, error) {
	ref := capiManagedControlPlane.Spec.TemplateRef
	if ref == nil {
		return nil, nil
	}

	// Only support ManagedControlPlaneTemplate refs in our own API group.
	if ref.Kind != capiv1alpha1.KindManagedControlPlaneTemplate {
		return nil, fmt.Errorf("templateRef.kind must be %q, got %q", capiv1alpha1.KindManagedControlPlaneTemplate, ref.Kind)
	}

	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid templateRef.apiVersion: %w", err)
	}

	if gv.Group != capiv1alpha1.SchemeGroupVersion.Group {
		return nil, fmt.Errorf(
			"templateRef.apiVersion group must be %q, got %q",
			capiv1alpha1.SchemeGroupVersion.Group,
			gv.Group,
		)
	}

	if gv.Version != capiv1alpha1.SchemeGroupVersion.Version {
		return nil, fmt.Errorf(
			"templateRef.apiVersion must be %q, got %q",
			capiv1alpha1.SchemeGroupVersion.String(),
			ref.APIVersion,
		)
	}

	ns := capiManagedControlPlane.Namespace
	if ref.Namespace != "" {
		ns = ref.Namespace
	}

	tpl := &capiv1alpha1.ManagedControlPlaneTemplate{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, tpl); err != nil {
		return nil, err
	}

	// ProviderSpec is optional on the template. If absent, caller can fall back to ManagedControlPlane.Spec.ProviderSpec.
	if tpl.Spec.Template.Spec.ProviderSpec == nil {
		return nil, nil
	}

	return tpl.Spec.Template.Spec.ProviderSpec, nil
}

func (r *ManagedControlPlaneReconciler) ensureCAPIKubeconfigSecret(
	ctx context.Context,
	capiMCP *capiv1alpha1.ManagedControlPlane,
	backing *mcpv1alpha1.ManagedControlPlane,
) error {
	ns := capiMCP.Namespace

	// Resolve cluster name (use the required label)
	clusterName := ""
	if capiMCP.Labels != nil {
		clusterName = strings.TrimSpace(capiMCP.Labels[clusterNameLabel])
	}
	if clusterName == "" {
		return fmt.Errorf("ManagedControlPlane %s/%s missing label %q", ns, capiMCP.Name, clusterNameLabel)
	}

	// Fetch the Cluster object (only to set ownerRef)
	cluster := &clusterv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: clusterName}, cluster); err != nil {
		return err
	}

	// Source secret (provider-owned)
	srcName := fmt.Sprintf("%s-admin-kubeconfig", backing.Name)
	src := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: srcName}, src); err != nil {
		return err
	}

	kubeconfig, ok := src.Data["kubeconfig"]
	if !ok {
		return fmt.Errorf("secret %q missing data.kubeconfig", srcName)
	}

	// Destination secret required by CAPI
	dstName := fmt.Sprintf("%s-kubeconfig", clusterName)
	dst := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: dstName}, dst)

	create := false
	if apierrors.IsNotFound(err) {
		create = true
		dst = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dstName,
				Namespace: ns,
			},
		}
	} else if err != nil {
		return err
	}

	// Ensure required label for cache retrieval
	if dst.Labels == nil {
		dst.Labels = map[string]string{}
	}
	dst.Labels[clusterNameLabel] = clusterName

	// Ensure required type
	dst.Type = corev1.SecretType(capiSecretType)

	// Ensure required key
	if dst.Data == nil {
		dst.Data = map[string][]byte{}
	}
	dst.Data["value"] = kubeconfig

	// OwnerRef to Cluster (optional but good)
	if err := controllerutil.SetControllerReference(cluster, dst, r.Scheme); err != nil {
		return err
	}

	if create {
		return r.Create(ctx, dst)
	}
	return r.Update(ctx, dst)
}

func (r *ManagedControlPlaneReconciler) getOwningClusterForManagedControlPlane(
	ctx context.Context,
	capiMCP *capiv1alpha1.ManagedControlPlane,
) (*clusterv1.Cluster, error) {
	ns := capiMCP.Namespace

	// A) Try ownerReference first (most correct)
	for _, o := range capiMCP.OwnerReferences {
		if o.Kind == "Cluster" && o.APIVersion == clusterv1.GroupVersion.String() {
			c := &clusterv1.Cluster{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: o.Name}, c); err != nil {
				return nil, err
			}
			return c, nil
		}
	}

	// B) Fallback to standard label cluster.x-k8s.io/cluster-name
	if cn, ok := capiMCP.Labels[clusterNameLabel]; ok && strings.TrimSpace(cn) != "" {
		c := &clusterv1.Cluster{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: cn}, c); err != nil {
			return nil, err
		}
		return c, nil
	}

	// C) Last resort: list clusters and find one referencing this MCP as controlPlaneRef
	var list clusterv1.ClusterList
	if err := r.List(ctx, &list, &client.ListOptions{
		Namespace:     ns,
		LabelSelector: labels.Everything(),
	}); err != nil {
		return nil, err
	}
	for i := range list.Items {
		c := &list.Items[i]
		if c.Spec.ControlPlaneRef.Name != "" &&
			c.Spec.ControlPlaneRef.Kind == capiv1alpha1.KindManagedControlPlane &&
			c.Spec.ControlPlaneRef.Name == capiMCP.Name {
			return c, nil
		}
	}

	return nil, fmt.Errorf("unable to determine owning Cluster for ManagedControlPlane %s/%s", ns, capiMCP.Name)
}

func (r *ManagedControlPlaneReconciler) reconcileCanonicalCertSecrets(
	ctx context.Context,
	cluster *clusterv1.Cluster,
	bootstrapSuffix string, // e.g. "rb9lj"
) error {
	log := ctrl.LoggerFrom(ctx).WithValues("cluster", cluster.Name, "ns", cluster.Namespace)

	type mapping struct {
		dstName string
		srcName string
		// dstKey -> srcKey
		keyMap map[string]string
	}

	mappings := []mapping{
		{
			dstName: fmt.Sprintf("%s-ca", cluster.Name),
			srcName: fmt.Sprintf("%s-%s-managed-ca", cluster.Name, bootstrapSuffix),
			keyMap: map[string]string{
				"tls.crt": "tls.crt",
				"tls.key": "tls.key",
			},
		},
		{
			dstName: fmt.Sprintf("%s-etcd", cluster.Name),
			srcName: fmt.Sprintf("%s-%s-etcd-ca", cluster.Name, bootstrapSuffix),
			keyMap: map[string]string{
				"tls.crt": "tls.crt",
				"tls.key": "tls.key",
			},
		},
		{
			dstName: fmt.Sprintf("%s-proxy", cluster.Name),
			srcName: fmt.Sprintf("%s-%s-front-proxy-ca", cluster.Name, bootstrapSuffix),
			keyMap: map[string]string{
				"tls.crt": "tls.crt",
				"tls.key": "tls.key",
			},
		},
		{
			dstName: fmt.Sprintf("%s-sa", cluster.Name),
			srcName: fmt.Sprintf("%s-%s-sa-signer", cluster.Name, bootstrapSuffix),
			keyMap: map[string]string{
				"tls.crt": "tls.crt",
				"tls.key": "tls.key",
			},
		},
	}

	for _, m := range mappings {
		// 1) Load source secret
		src := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: m.srcName, Namespace: cluster.Namespace}, src); err != nil {
			if apierrors.IsNotFound(err) {
				// If you prefer "hard fail", return err here.
				// This makes it "eventually consistent" while your other reconciler creates src secrets.
				log.V(1).Info("source secret not found yet; will retry", "source", m.srcName, "dest", m.dstName)
				continue
			}
			return fmt.Errorf("get source secret %s: %w", m.srcName, err)
		}

		// 2) CreateOrPatch destination
		dst := &corev1.Secret{ObjectMeta: ctrl.ObjectMeta{
			Name:      m.dstName,
			Namespace: cluster.Namespace,
		}}

		_, err := controllerutil.CreateOrPatch(ctx, r.Client, dst, func() error {
			if dst.Labels == nil {
				dst.Labels = map[string]string{}
			}
			dst.Labels[clusterNameLabel] = cluster.Name
			dst.Type = corev1.SecretType(capiSecretType)

			if dst.Data == nil {
				dst.Data = map[string][]byte{}
			}

			// Populate requested keys from src
			for dstKey, srcKey := range m.keyMap {
				v, ok := src.Data[srcKey]
				if !ok || len(v) == 0 {
					return fmt.Errorf("source secret %s missing key %q (needed for %s)", m.srcName, srcKey, m.dstName)
				}
				dst.Data[dstKey] = v
			}

			// Optional: make GC work by setting owner reference to Cluster
			// If you don't want GC, remove this line.
			return controllerutil.SetControllerReference(cluster, dst, r.Scheme)
		})
		if err != nil {
			return fmt.Errorf("create/patch dest secret %s: %w", m.dstName, err)
		}
	}

	return nil
}


func bootstrapSuffixFromBackingName(backingName string) (string, error) {
	parts := strings.Split(backingName, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot derive bootstrap suffix from backing name %q", backingName)
	}
	// backing.Name is "<cluster>-<suffix>"
	return parts[len(parts)-1], nil
}

func (r *ManagedControlPlaneReconciler) ensureClusterControlPlaneEndpoint(
	ctx context.Context,
	cluster *clusterv1.Cluster,
	backing *mcpv1alpha1.ManagedControlPlane,
) error {
	addr := strings.TrimSpace(backing.Status.Address)
	if addr == "" {
		// nothing to set yet
		return nil
	}

	const apiserverPort int32 = 6443

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, cluster, func() error {
		changed := false

		if cluster.Spec.ControlPlaneEndpoint.Host != addr {
			cluster.Spec.ControlPlaneEndpoint.Host = addr
			changed = true
		}
		if cluster.Spec.ControlPlaneEndpoint.Port != apiserverPort {
			cluster.Spec.ControlPlaneEndpoint.Port = apiserverPort
			changed = true
		}

		// No-op if already correct
		if !changed {
			return nil
		}
		return nil
	})
	return err
}

