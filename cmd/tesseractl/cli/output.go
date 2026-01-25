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

package cli

import (
	"fmt"
	"time"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

func printObject(obj runtime.Object, outFmt string) error {
	switch outFmt {
	case "", "table":
		return fmt.Errorf("table output is only supported for get/list, not generic objects")
	case "yaml":
		return printYAML(obj)
	case "json":
		return printJSON(obj)
	default:
		return fmt.Errorf("unsupported output %q (use: yaml|json)", outFmt)
	}
}

func printJSON(obj runtime.Object) error {
	fmt.Print(utils.GetObjJSON(obj))
	return nil
}

func printYAML(obj runtime.Object) error {
	fmt.Print(utils.GetObjYaml(obj))
	return nil
}

func printMCPTable(list *mcpv1alpha1.ManagedControlPlaneList) {
	fmt.Printf("%-30s %-7s %-16s %-10s %-6s\n", "NAME", "READY", "ADDRESS", "VERSION", "AGE")

	now := time.Now()
	for _, m := range list.Items {
		ready := ""
		if m.Status.Ready != nil {
			if *m.Status.Ready {
				ready = "True"
			} else {
				ready = "False"
			}
		}

		age := humanDuration(now.Sub(m.CreationTimestamp.Time))
		fmt.Printf("%-30s %-7s %-16s %-10s %-6s\n",
			m.Name,
			ready,
			m.Status.Address,
			m.Spec.Version,
			age,
		)
	}
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours()) / 24
	return fmt.Sprintf("%dd", days)
}
