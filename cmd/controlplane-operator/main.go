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
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	"golang.org/x/sync/errgroup"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	mcctrl "github.com/patrostkowski/controlplane-operator/pkg/controller/multicluster"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

func main() {
	var metricsAddr string
	var healthProbeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	ctx := signals.SetupSignalHandler()

	localMgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: nil,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: healthProbeAddr,
		// Consider use Unstructured: true
		// For external providers
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{&corev1.Secret{}},
			},
		},
	})
	if err != nil {
		log.Log.Error(err, "unable to set up mcp controller manager")
		panic(err)
	}

	if err = mcpv1alpha1.AddToScheme(localMgr.GetScheme()); err != nil {
		panic(err)
	}
	if err = certmanagerv1.AddToScheme(localMgr.GetScheme()); err != nil {
		panic(err)
	}
	if err = apiextv1.AddToScheme(localMgr.GetScheme()); err != nil {
		panic(err)
	}

	if err = localMgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		panic(err)
	}

	if err = localMgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		panic(err)
	}

	// Create the provider against the local manager.
	provider, err := mcctrl.New(localMgr, mcctrl.Options{})
	if err != nil {
		log.Log.Error(err, "unable to set up provider")
		panic(err)
	}

	// Create a multi-cluster manager attached to the provider.
	mcMgr, err := mcmanager.New(ctrl.GetConfigOrDie(), provider, mcmanager.Options{
		LeaderElection: false,
		Metrics: server.Options{
			BindAddress: "0", // only one can listen
		},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		log.Log.Error(err, "unable to set multicluster manager")
		os.Exit(1)
	}

	if err := controller.SetupManagedControlPlaneController(localMgr); err != nil {
		panic(err)
	}

	if err := controller.SetupManagedAddonController(mcMgr); err != nil {
		panic(err)
	}

	// Starting everything.
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return utils.IgnoreCanceled(localMgr.Start(ctx))
	})
	g.Go(func() error {
		return utils.IgnoreCanceled(mcMgr.Start(ctx))
	})
	if err := g.Wait(); err != nil {
		log.Log.Error(err, "unable to start")
		os.Exit(1)
	}
}
