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

package main

import (
	"flag"
	"os"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: nil, // we’ll populate below
	})
	if err != nil {
		panic(err)
	}

	// register your API
	if err := mcpv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}
	if err := certmanagerv1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	// register controllers
	if err := controller.SetupManagedControlPlaneController(mgr); err != nil {
		panic(err)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}
