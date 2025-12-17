package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
