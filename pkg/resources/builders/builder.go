package builders

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	Objects() []client.Object
}

type HasPodTemplate interface {
	GetPodTemplate() *corev1.PodTemplateSpec
}

type PodTemplateMutator struct {
	h HasPodTemplate
}

func (m PodTemplateMutator) WithServiceAccount(sa string) {
	m.h.GetPodTemplate().Spec.ServiceAccountName = sa
}

func (m PodTemplateMutator) WithContainer(c corev1.Container) {
	pt := m.h.GetPodTemplate()
	pt.Spec.Containers = append(pt.Spec.Containers, c)
}

func (m PodTemplateMutator) AddVolumes(vols ...corev1.Volume) {
	pt := m.h.GetPodTemplate()
	pt.Spec.Volumes = append(pt.Spec.Volumes, vols...)
}

func (m PodTemplateMutator) PatchContainer(name string, fn func(*corev1.Container)) bool {
	cs := m.h.GetPodTemplate().Spec.Containers
	for i := range cs {
		if cs[i].Name == name {
			fn(&cs[i])
			return true
		}
	}
	return false
}

func (m PodTemplateMutator) AddVolumeMounts(containerName string, mounts ...corev1.VolumeMount) bool {
	return m.PatchContainer(containerName, func(c *corev1.Container) {
		c.VolumeMounts = append(c.VolumeMounts, mounts...)
	})
}
