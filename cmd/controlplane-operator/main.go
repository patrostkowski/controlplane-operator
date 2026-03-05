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
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	provider "github.com/patrostkowski/controlplane-operator/pkg/controller/multicluster"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// var scheme = runtime.NewScheme()
//
// func init() {
// 	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
// 	utilruntime.Must(mcpv1alpha1.AddToScheme(scheme))
// 	utilruntime.Must(certmanagerv1.AddToScheme(scheme))
// 	utilruntime.Must(apiextv1.AddToScheme(scheme))
// }

func main() {
	var metricsAddr string
	var healthProbeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	ctx := signals.SetupSignalHandler()

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Log.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err = mcpv1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err = certmanagerv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err = clientgoscheme.AddToScheme(scheme); err != nil {
		panic(err)
	}

	// Create the MCP provider.
	provider := provider.New(provider.Options{
		ClusterOptions: []cluster.Option{
			func(o *cluster.Options) {
				o.Scheme = scheme
			},
		},
	})

	mgr, err := mcmanager.New(cfg, provider, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: healthProbeAddr,
		// Consider use Unstructured: true
		// For external managedControlPlaneProviders
		Client: client.Options{
			// TODO: customize cache behavior per object type
			// with label selector
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{&corev1.Secret{}},
			},
		},
	})
	if err != nil {
		log.Log.Error(err, "unable to set up mcp controller manager")
		panic(err)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		panic(err)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		panic(err)
	}

	if err := controller.SetupManagedControlPlaneController(mgr); err != nil {
		panic(err)
	}

	if err := controller.SetupManagedAddonController(mgr); err != nil {
		panic(err)
	}

	if err := provider.SetupProviderController(mgr); err != nil {
		panic(err)
	}

	if err := mgr.Start(ctx); utils.IgnoreCanceled(err) != nil {
		log.Log.Error(err, "unable to start manager")
		return
	}
}
