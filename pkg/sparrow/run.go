// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sparrow

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/caas-team/sparrow/pkg/factory"
	"github.com/caas-team/sparrow/pkg/sparrow/targets"
)

const shutdownTimeout = time.Second * 90

type Sparrow struct {
	config  *config.Config
	db      db.DB
	api     api.API
	loader  config.Loader
	tarMan  targets.TargetManager
	metrics Metrics
	errorHandler
	checkCoordinator
}

// checkCoordinator is used to coordinate the checks and the reconciler
type checkCoordinator struct {
	// controller is used to manage the checks
	controller *ChecksController
	// cRuntime is used to signal that the runtime configuration has changed
	cRuntime chan runtime.Config
}

// errorHandler is used to handle non-recoverable errors of the sparrow components
type errorHandler struct {
	// cErr is used to handle non-recoverable errors of the sparrow components
	cErr chan error
	// cDone is used to signal that the sparrow was shut down because of an error
	cDone chan struct{}
	// shutOnce is used to ensure that the shutdown function is only called once
	shutOnce sync.Once
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	metrics := NewMetrics()
	dbase := db.NewInMemory()

	sparrow := &Sparrow{
		config:  cfg,
		db:      dbase,
		api:     api.New(cfg.Api),
		metrics: metrics,
		errorHandler: errorHandler{
			cErr:     make(chan error, 1),
			cDone:    make(chan struct{}, 1),
			shutOnce: sync.Once{},
		},
		checkCoordinator: checkCoordinator{
			controller: NewChecksController(dbase, metrics),
			cRuntime:   make(chan runtime.Config, 1),
		},
	}

	if cfg.HasTargetManager() {
		gm := targets.NewGitlabManager(cfg.SparrowName, cfg.TargetManager)
		sparrow.tarMan = gm
	}
	sparrow.loader = config.NewLoader(cfg, sparrow.cRuntime)

	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	log := logger.FromContext(ctx)
	defer cancel()

	go func() {
		s.cErr <- s.loader.Run(ctx)
	}()
	go func() {
		if s.tarMan != nil {
			s.cErr <- s.tarMan.Reconcile(ctx)
		}
	}()

	go func() {
		s.cErr <- s.startupAPI(ctx)
	}()
	go func() {
		s.controller.ListenErrors(ctx)
	}()

	for {
		select {
		case cfg := <-s.cRuntime:
			s.ReconcileChecks(ctx, cfg)
		case <-ctx.Done():
			s.shutdown(ctx)
		case err := <-s.cErr:
			if err != nil {
				log.Error("Non-recoverable error in sparrow component", "error", err)
				s.shutdown(ctx)
			}
		case <-s.cDone:
			return fmt.Errorf("sparrow was shut down")
		}
	}
}

// ReconcileChecks reconciles the checks.
// It registers new checks, updates existing checks and unregisters checks not in the new config.
func (s *Sparrow) ReconcileChecks(ctx context.Context, cfg runtime.Config) {
	log := logger.FromContext(ctx)

	cfg = s.enrichTargets(cfg)
	newChecks, err := factory.NewChecksFromConfig(cfg)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create checks from config", "error", err)
		return
	}

	// Update existing checks and create a list of checks to unregister
	var unregList []checks.Check
	for _, c := range s.controller.checks.Iter() {
		conf := cfg.For(c.Name())
		if conf != nil {
			err = c.SetConfig(conf)
			if err != nil {
				log.ErrorContext(ctx, "Failed to set config for check", "check", c.Name(), "error", err)
			}
			delete(newChecks, c.Name())
		} else {
			unregList = append(unregList, c)
		}
	}

	// Unregister checks not in the new config
	for _, c := range unregList {
		err = s.controller.UnregisterCheck(ctx, c)
		if err != nil {
			log.ErrorContext(ctx, "Failed to unregister check", "check", c.Name(), "error", err)
		}
	}

	// Register new checks
	for _, c := range newChecks {
		err = s.controller.RegisterCheck(ctx, c)
		if err != nil {
			log.ErrorContext(ctx, "Failed to register check", "check", c.Name(), "error", err)
		}
	}
}

// enrichTargets updates the targets of the sparrow's checks with the
// global targets. Per default, the two target lists are merged.
func (s *Sparrow) enrichTargets(cfg runtime.Config) runtime.Config {
	if cfg.Empty() || s.tarMan == nil {
		return cfg
	}

	for _, gt := range s.tarMan.GetTargets() {
		if gt.Url == fmt.Sprintf("https://%s", s.config.SparrowName) {
			continue
		}
		if cfg.HasHealthCheck() && !slices.Contains(cfg.Health.Targets, gt.Url) {
			cfg.Health.Targets = append(cfg.Health.Targets, gt.Url)
		}
		if cfg.HasLatencyCheck() && !slices.Contains(cfg.Latency.Targets, gt.Url) {
			cfg.Latency.Targets = append(cfg.Latency.Targets, gt.Url)
		}
		if cfg.HasDNSCheck() && !slices.Contains(cfg.Dns.Targets, gt.Url) {
			t, _ := strings.CutPrefix(gt.Url, "https://")
			cfg.Dns.Targets = append(cfg.Dns.Targets, t)
		}
	}

	return cfg
}

type ErrShutdown struct {
	errAPI       error
	errTarMan    error
	errCheckCont error
}

// shutdown shuts down the sparrow and all managed components gracefully.
// It returns an error if one is present in the context or if any of the
// components fail to shut down.
func (s *Sparrow) shutdown(ctx context.Context) {
	errC := ctx.Err()
	log := logger.FromContext(ctx)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	s.shutOnce.Do(func() {
		log.Info("Shutting down sparrow gracefully")
		var sErrs ErrShutdown
		if s.tarMan != nil {
			sErrs.errTarMan = s.tarMan.Shutdown(ctx)
		}
		sErrs.errAPI = s.api.Shutdown(ctx)
		s.loader.Shutdown(ctx)
		sErrs.errCheckCont = s.controller.Shutdown(ctx)

		if sErrs.errTarMan != nil || sErrs.errAPI != nil || sErrs.errCheckCont != nil {
			log.Error("Failed to shutdown gracefully", "contextError", errC, "errors", sErrs)
		}

		// Signal that shutdown is complete
		s.cDone <- struct{}{}
	})
}
