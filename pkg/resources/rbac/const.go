package rbac

const (
	KubeSystemNamespace = "kube-system"

	BootstrapTokenMgmtSecretName = "kubelet-bootstrap-token"
	BootstrapTokenIDKey          = "token-id"
	BootstrapTokenSecretKey      = "token-secret"
	BootstrapTokenDescription    = "Bootstrap token for kubelet workers"
	BootstrapAuthExtraGroups     = "system:bootstrappers:kubelet-bootstrap"

	BootstrapUsageAuth = "true"
	BootstrapUsageSign = "true"

	GroupBootstrappers = "system:bootstrappers:kubelet-bootstrap"
	GroupNodes         = "system:nodes"

	CRBNodeClientAutoApproveName  = "kubelet-bootstrap-auto-approve-node-client-certs"
	CRBSelfNodeClientRotationName = "kubelet-auto-approve-node-client-cert-rotation"
	CRBNodeBootstrapperName       = "kubelet-bootstrap"

	RoleNodeClientCSRApprove = "system:certificates.k8s.io:certificatesigningrequests:nodeclient"
	RoleSelfNodeClientCSR    = "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient"
	RoleNodeBootstrapper     = "system:node-bootstrapper"
)
