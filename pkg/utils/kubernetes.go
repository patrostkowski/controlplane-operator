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

package utils

import (
	"strconv"

	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
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
	ca common.SecretMount,
	client common.SecretMount,
) *api.Config {
	serverURL :=
		"https://" +
			apiServerService +
			"." + namespace +
			".svc:" +
			strconv.Itoa(int(apiServerPort))

	cfg := api.NewConfig()

	cfg.Clusters[DefaultContextName] = &api.Cluster{
		Server:               serverURL,
		CertificateAuthority: ca.CertPath(),
	}

	cfg.AuthInfos[user] = &api.AuthInfo{
		ClientCertificate: client.CertPath(),
		ClientKey:         client.KeyPath(),
	}

	cfg.Contexts[DefaultContextName] = &api.Context{
		Cluster:  DefaultContextName,
		AuthInfo: user,
	}
	cfg.CurrentContext = DefaultContextName

	return cfg
}

const DefaultContextName = "local"

// BuildKubeconfigWithCertData builds kubeconfig that embeds cert material inlined (Data fields).
func BuildKubeconfigWithCertData(serverURL, user string, ca, crt, key []byte) *api.Config {
	cfg := api.NewConfig()

	cfg.Clusters[DefaultContextName] = &api.Cluster{
		Server:                   serverURL,
		CertificateAuthorityData: ca,
	}

	cfg.AuthInfos[user] = &api.AuthInfo{
		ClientCertificateData: crt,
		ClientKeyData:         key,
	}

	cfg.Contexts[DefaultContextName] = &api.Context{
		Cluster:  DefaultContextName,
		AuthInfo: user,
	}
	cfg.CurrentContext = DefaultContextName

	return cfg
}
