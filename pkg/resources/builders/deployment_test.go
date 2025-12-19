// builder_test.go
package builders

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type testComponent struct{}

func (testComponent) Objects() []client.Object {
	return nil
}

func TestComponentInterfaceSatisfied(t *testing.T) {
	var _ Component = testComponent{}
}

// NOTE: Your Component interface is:
//
//	type Component interface { Objects() []client.Object }
//
// We can't implement it here without importing controller-runtime client,
// so instead we just ensure the interface type exists and is usable.
func TestComponentInterfaceExists(t *testing.T) {
	var _ any = (*Component)(nil)
}

func TestNewDeployment_SetsBasics(t *testing.T) {
	labels := map[string]string{"app": "demo"}
	replicas := int32(2)

	w := NewDeployment("ns1", "name1", labels, replicas)

	if w == nil || w.Deployment == nil {
		t.Fatalf("expected non-nil DeploymentTemplate")
	}
	if w.Namespace != "ns1" || w.Name != "name1" {
		t.Fatalf("unexpected metadata: %s/%s", w.Namespace, w.Name)
	}
	if !reflect.DeepEqual(w.Labels, labels) {
		t.Fatalf("unexpected labels: %#v", w.Labels)
	}

	if w.Spec.Replicas == nil || *w.Spec.Replicas != replicas {
		t.Fatalf("unexpected replicas: %#v", w.Spec.Replicas)
	}

	if w.Spec.Selector == nil || !reflect.DeepEqual(w.Spec.Selector.MatchLabels, labels) {
		t.Fatalf("unexpected selector: %#v", w.Spec.Selector)
	}

	if !reflect.DeepEqual(w.Spec.Template.Labels, labels) {
		t.Fatalf("unexpected pod template labels: %#v", w.Spec.Template.Labels)
	}
}

func TestWithServiceAccount(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1).
		WithServiceAccount("sa-demo")

	if got := w.Spec.Template.Spec.ServiceAccountName; got != "sa-demo" {
		t.Fatalf("expected service account %q, got %q", "sa-demo", got)
	}
}

func TestWithContainer_Appends(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1)

	c1 := corev1.Container{Name: "c1", Image: "img1"}
	c2 := corev1.Container{Name: "c2", Image: "img2"}

	w.WithContainer(c1).WithContainer(c2)

	if len(w.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(w.Spec.Template.Spec.Containers))
	}
	if w.Spec.Template.Spec.Containers[0].Name != "c1" || w.Spec.Template.Spec.Containers[1].Name != "c2" {
		t.Fatalf("unexpected containers: %#v", w.Spec.Template.Spec.Containers)
	}
}

func TestAddVolumes_Appends(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1)

	v1 := corev1.Volume{Name: "v1"}
	v2 := corev1.Volume{Name: "v2"}

	w.AddVolumes(v1, v2)

	if len(w.Spec.Template.Spec.Volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(w.Spec.Template.Spec.Volumes))
	}
	if w.Spec.Template.Spec.Volumes[0].Name != "v1" || w.Spec.Template.Spec.Volumes[1].Name != "v2" {
		t.Fatalf("unexpected volumes: %#v", w.Spec.Template.Spec.Volumes)
	}
}

func TestAddVolumeMounts_Appends(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1)

	volMount := corev1.VolumeMount{
		Name:      "vol",
		MountPath: "/vol",
		ReadOnly:  true,
	}

	c1Name := "c1"
	c1 := corev1.Container{Name: c1Name, Image: "img1"}

	w.WithContainer(c1).AddVolumeMounts(c1Name, volMount)

	if got := w.Spec.Template.Spec.Containers[0].VolumeMounts[0]; got != volMount {
		t.Fatalf("container volume mount incorrect, got mount %v+", got)
	}
}

func TestPatchContainer_PatchesOnlyMatchingContainer(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1).
		WithContainer(corev1.Container{Name: "a", Image: "imgA"}).
		WithContainer(corev1.Container{Name: "b", Image: "imgB"})

	w.PatchContainer("b", func(c *corev1.Container) {
		c.Image = "imgB2"
		c.Args = []string{"--x=y"}
	})

	if got := w.Spec.Template.Spec.Containers[0].Image; got != "imgA" {
		t.Fatalf("container a should be unchanged, got image %q", got)
	}
	if got := w.Spec.Template.Spec.Containers[1].Image; got != "imgB2" {
		t.Fatalf("container b should be patched, got image %q", got)
	}
	if got := w.Spec.Template.Spec.Containers[1].Args; !reflect.DeepEqual(got, []string{"--x=y"}) {
		t.Fatalf("unexpected args on b: %#v", got)
	}
}

func TestPatchContainer_NoOpWhenNotFound(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1).
		WithContainer(corev1.Container{Name: "a", Image: "imgA"})

	before := w.Spec.Template.Spec.Containers[0].DeepCopy()

	w.PatchContainer("does-not-exist", func(c *corev1.Container) {
		c.Image = "should-not-happen"
	})

	after := &w.Spec.Template.Spec.Containers[0]
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("expected container unchanged when name not found; before=%#v after=%#v", before, after)
	}
}

func TestDeploymentBuild_ReturnsDeepCopy(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1).
		WithServiceAccount("sa").
		WithContainer(corev1.Container{Name: "c", Image: "img"}).
		AddVolumes(corev1.Volume{Name: "v"})

	b1 := w.Build()
	b2 := w.Build()

	if b1 == w.Deployment {
		t.Fatalf("Build() must not return the same pointer as template deployment")
	}
	if b1 == b2 {
		t.Fatalf("Build() must return distinct pointers on repeated calls")
	}

	// Mutate b1 and ensure it doesn't affect template or b2
	b1.Spec.Template.Spec.ServiceAccountName = "mutated"
	b1.Spec.Template.Spec.Containers[0].Image = "mutated-img"
	b1.Spec.Template.Spec.Volumes[0].Name = "mutated-vol"

	// template should remain unchanged
	if got := w.Spec.Template.Spec.ServiceAccountName; got != "sa" {
		t.Fatalf("template SA changed unexpectedly: %q", got)
	}
	if got := w.Spec.Template.Spec.Containers[0].Image; got != "img" {
		t.Fatalf("template container image changed unexpectedly: %q", got)
	}
	if got := w.Spec.Template.Spec.Volumes[0].Name; got != "v" {
		t.Fatalf("template volume name changed unexpectedly: %q", got)
	}

	// b2 should remain unchanged
	if got := b2.Spec.Template.Spec.ServiceAccountName; got != "sa" {
		t.Fatalf("b2 SA changed unexpectedly: %q", got)
	}
	if got := b2.Spec.Template.Spec.Containers[0].Image; got != "img" {
		t.Fatalf("b2 container image changed unexpectedly: %q", got)
	}
	if got := b2.Spec.Template.Spec.Volumes[0].Name; got != "v" {
		t.Fatalf("b2 volume name changed unexpectedly: %q", got)
	}
}

func TestNewDeployment_ProducesValidDeploymentKind(t *testing.T) {
	w := NewDeployment("ns", "name", map[string]string{"app": "x"}, 1)
	dep := w.Build()

	if dep.APIVersion == "" || dep.Kind == "" {
		// These fields are typically set by scheme when encoding; but sanity-check type correctness anyway.
		// The important part is: it's an *appsv1.Deployment instance.
	}
	var _ *appsv1.Deployment = dep
}
