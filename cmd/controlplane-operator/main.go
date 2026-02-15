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
	"log"
	"os"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/controller"
	"github.com/patrostkowski/controlplane-operator/pkg/logger"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	appLogger logger.Logger
)

func main() {
	var metricsAddr string
	var healthProbeAddr string
	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&logLevel, "log-level", "info", "Log level for the controller")
	flag.Parse()

	// parse logLevel string to abstraction-compatible log level
	parsedLogLevel, err := logger.ParseLogLevel(logLevel)
	if err != nil {
		log.Panicln("errors setting the log level:", err)
	}

	// create zap logger based on logging abstraction
	appLogger = logger.NewZapLogger(parsedLogLevel)

	// pass compatible logr.Logger instance for controller runtime components
	ctrl.SetLogger(appLogger.GetBaseLogrInstance())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: nil,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: healthProbeAddr,
	})
	if err != nil {
		panic(err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		panic(err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		panic(err)
	}

	if err := mcpv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}
	if err := certmanagerv1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}
	if err := clientgoscheme.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	if err := controller.SetupManagedControlPlaneController(mgr); err != nil {
		panic(err)
	}
	if err := controller.SetupManagedAddonController(mgr); err != nil {
		panic(err)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}
