package apiserver

const (
	KubeAPIServerSvcName    = apiServerName
	KubeAPIServerSecurePort = securePort

	apiServerName = "kube-apiserver"
	appLabelKey   = "app"
	appLabelVal   = "kube-apiserver"

	securePort int32 = 6443

	// All certs are mounted under this root, with subdirs per secret/volume.
	mountRoot = "/var/run/k8s"
)
