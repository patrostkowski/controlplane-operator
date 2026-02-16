//go:build e2e
// +build e2e

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

package e2e

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-logr/logr"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	clusterName = "e2e"
	imageName   = "patrostkowski/controlplane-operator:latest"
	timeout     = 30 * time.Minute
)

var (
	log logr.Logger
)

var (
	noCleanup = flag.Bool(
		"no-cleanup",
		false,
		"do not clean up after tests",
	)

	stage = flag.String(
		"stage",
		"all",
		"which stage to run: setup, run, cleanup, all",
	)

	noDump = flag.Bool(
		"no-dump",
		false,
		"do not dump info on failure",
	)
)

type TestContext struct {
	log logr.Logger
}

func NewTestContext(l logr.Logger) TestContext {
	return TestContext{log: l}
}

func TestMain(m *testing.M) {
	flag.Parse()

	log = ctrlzap.New(ctrlzap.UseDevMode(true))

	tc := NewTestContext(log)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-sigCtx.Done()
		tc.log.Info("received interrupt signal, cleaning up")

		if *stage != "run" && !*noCleanup {
			_ = DumpDebug(ctx)
			_ = tc.CleanUp(ctx)
		}

		os.Exit(1)
	}()

	tc.log.Info(
		"e2e starting",
		"cluster", clusterName,
		"image", imageName,
		"no_cleanup", *noCleanup,
		"stage", "setup",
	)

	if *stage == "setup" || *stage == "all" {
		if err := tc.Setup(ctx); err != nil {
			tc.log.Error(err, "setup failed")
			if !*noCleanup {
				_ = tc.CleanUp(ctx)
			}
			os.Exit(1)
		}

		if *stage == "setup" {
			tc.log.Info("setup stage completed, exiting")
			os.Exit(0)
		}
	}

	code := 0
	if *stage == "run" || *stage == "all" {
		code = m.Run()
	}

	if code != 0 && !*noDump {
		tc.log.Info("tests failed, dumping debug", "exit_code", code)
		_ = DumpDebug(ctx)
	}

	if !*noCleanup && (*stage == "cleanup" || *stage == "all") {
		_ = tc.CleanUp(ctx)
	}

	os.Exit(code)
}
