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
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/caas-team/sparrow/pkg/sparrow/targets"
)

const shutdownTimeout = time.Second * 90

// Sparrow is the main struct of the sparrow application
type Sparrow struct {
	// config is the startup configuration of the sparrow
	config *config.Config
	// db is the database used to store the check results
	db db.DB
	// api is the sparrow's API
	api api.API
	// loader is used to load the runtime configuration
	loader config.Loader
	// tarMan is the target manager that is used to manage global targets
	tarMan targets.TargetManager
	// metrics is used to collect metrics
	metrics Metrics
	// controller is used to manage the checks
	controller *ChecksController
	// cRuntime is used to signal that the runtime configuration has changed
	cRuntime chan runtime.Config
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
		config:     cfg,
		db:         dbase,
		api:        api.New(cfg.Api),
		metrics:    metrics,
		controller: NewChecksController(dbase, metrics),
		cRuntime:   make(chan runtime.Config, 1),
		cErr:       make(chan error, 1),
		cDone:      make(chan struct{}, 1),
		shutOnce:   sync.Once{},
	}

	if cfg.HasTargetManager() {
		gm := targets.NewManager(cfg.SparrowName, cfg.TargetManager)
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
		s.cErr <- s.controller.Run(ctx)
	}()

	for {
		select {
		case cfg := <-s.cRuntime:
			cfg = s.enrichTargets(cfg)
			s.controller.Reconcile(ctx, cfg)
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
		s.controller.Shutdown(ctx)

		if sErrs.HasError() {
			log.Error("Failed to shutdown gracefully", "contextError", errC, "errors", sErrs)
		}

		// Signal that shutdown is complete
		s.cDone <- struct{}{}
	})
}
