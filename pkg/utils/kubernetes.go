package utils

import (
	"path/filepath"
	"strconv"

	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd/api"
)

func SecretMount(volumeName, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		ReadOnly:  true,
	}
}

func SecretVolume(volumeName, secretName string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	}
}

func HttpsHealthProbe(port int32, path string, initialDelay, period, timeout, failureThreshold int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Scheme: corev1.URISchemeHTTPS,
				Host:   "127.0.0.1",
				Port:   intstr.FromInt(int(port)),
				Path:   path,
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
		TimeoutSeconds:      timeout,
		FailureThreshold:    failureThreshold,
		SuccessThreshold:    1,
	}
}

func TcpProbe(port int32, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(port)),
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

func ConfigMapVolume(mountName, cmName, cmKey, path string) corev1.Volume {
	return corev1.Volume{
		Name: mountName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				Items: []corev1.KeyToPath{
					{Key: cmKey, Path: path},
				},
			},
		},
	}
}

func BuildComponentKubeconfig(
	namespace string,
	apiServerService string,
	apiServerPort int32,
	user string,
	clientSecret string,
) *api.Config {
	serverURL :=
		"https://" +
			apiServerService +
			"." + namespace +
			".svc:" +
			strconv.Itoa(int(apiServerPort))

	cfg := api.NewConfig()

	caDir := filepath.Join(common.PKIMountRoot, pki.SecretManagedCA)
	clientDir := filepath.Join(common.PKIMountRoot, clientSecret)

	// --- Cluster ---
	cfg.Clusters["local"] = &api.Cluster{
		Server:               serverURL,
		CertificateAuthority: filepath.Join(caDir, common.TLSCrtKey),
	}

	// --- User ---
	cfg.AuthInfos[user] = &api.AuthInfo{
		ClientCertificate: filepath.Join(clientDir, common.TLSCrtKey),
		ClientKey:         filepath.Join(clientDir, common.TLSKeyKey),
	}

	// --- Context ---
	cfg.Contexts["local"] = &api.Context{
		Cluster:  "local",
		AuthInfo: user,
	}
	cfg.CurrentContext = "local"

	return cfg
}
