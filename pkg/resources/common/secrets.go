package common

import (
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

// SecretMount describes a single cert-manager Secret as mounted into a Pod.
type SecretMount struct {
	SecretName string
	MountDir   string
	CertFile   string
	KeyFile    string
	CAFile     string
}

func (s SecretMount) CertPath() string { return filepath.Join(s.MountDir, s.CertFile) }
func (s SecretMount) KeyPath() string  { return filepath.Join(s.MountDir, s.KeyFile) }
func (s SecretMount) CAPath() string   { return filepath.Join(s.MountDir, s.CAFile) }

func (s SecretMount) Volume() corev1.Volume {
	return corev1.Volume{
		Name: s.SecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: s.SecretName},
		},
	}
}

func (s SecretMount) Mount(readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      s.SecretName,
		MountPath: s.MountDir,
		ReadOnly:  readOnly,
	}
}

func SecretVolumes(ms ...SecretMount) []corev1.Volume {
	seen := map[string]struct{}{}
	out := make([]corev1.Volume, 0, len(ms))
	for _, m := range ms {
		if _, ok := seen[m.SecretName]; ok {
			continue
		}
		seen[m.SecretName] = struct{}{}
		out = append(out, m.Volume())
	}
	return out
}

func SecretMounts(readOnly bool, ms ...SecretMount) []corev1.VolumeMount {
	seen := map[string]struct{}{}
	out := make([]corev1.VolumeMount, 0, len(ms))
	for _, m := range ms {
		if _, ok := seen[m.SecretName]; ok {
			continue
		}
		seen[m.SecretName] = struct{}{}
		out = append(out, m.Mount(readOnly))
	}
	return out
}
