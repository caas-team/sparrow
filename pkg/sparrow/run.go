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
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/caas-team/sparrow/pkg/checks"

	"github.com/caas-team/sparrow/pkg/checks/factory"
	"github.com/caas-team/sparrow/pkg/checks/runtime"

	"github.com/caas-team/sparrow/pkg/sparrow/targets"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = time.Second * 90

type Sparrow struct {
	db db.DB
	// the existing checks
	checks map[string]checks.Check
	server *http.Server

	metrics Metrics

	resultFanIn map[string]chan checks.Result

	cfg *config.Config

	// cCfgChecks is the channel where the loader sends the checkCfg configuration of the checks
	cCfgChecks chan runtime.Config
	// cResult is the channel where the checks send their results to
	cResult chan checks.ResultDTO
	// cErr is used to handle non-recoverable errors of the sparrow components
	cErr chan error
	// cDone is used to signal that the sparrow was shut down because of an error
	cDone chan struct{}
	// shutOnce is used to ensure that the shutdown function is only called once
	shutOnce sync.Once

	loader config.Loader
	tarMan targets.TargetManager

	routingTree *api.RoutingTree
	router      chi.Router
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	sparrow := &Sparrow{
		db:          db.NewInMemory(),
		checks:      make(map[string]checks.Check),
		metrics:     NewMetrics(),
		resultFanIn: make(map[string]chan checks.Result),
		cResult:     make(chan checks.ResultDTO, 1),
		cfg:         cfg,
		cCfgChecks:  make(chan runtime.Config, 1),
		routingTree: api.NewRoutingTree(),
		router:      chi.NewRouter(),
		cErr:        make(chan error, 1),
		cDone:       make(chan struct{}, 1),
	}

	sparrow.server = &http.Server{Addr: cfg.Api.ListeningAddress, Handler: sparrow.router, ReadHeaderTimeout: readHeaderTimeout}

	if cfg.HasTargetManager() {
		gm := targets.NewGitlabManager(cfg.SparrowName, cfg.TargetManager)
		sparrow.tarMan = gm
	}

	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	sparrow.db = db.NewInMemory()
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
		s.cErr <- s.api(ctx)
	}()

	for {
		select {
		case result := <-s.cResult:
			go s.db.Save(result)
		case cfg := <-s.cCfgChecks:
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

// ReconcileChecks registers new Checks, unregisters removed Checks,
// resets the Configs of Checks and starts running the checks
func (s *Sparrow) ReconcileChecks(ctx context.Context, cfg runtime.Config) {
	// generate checks from configuration
	s.enrichTargets(cfg)
	ck, err := factory.NewChecksFromConfig(cfg)
	if err != nil {
		logger.FromContext(ctx).ErrorContext(ctx, "Failed to create checks from config", "error", err)
		return
	}

	// if checks are empty, register all checks
	if len(s.checks) == 0 {
		for _, c := range ck {
			err = s.registerCheck(ctx, c)
			if err != nil {
				logger.FromContext(ctx).ErrorContext(ctx, "Failed to register check", "error", err)
			}
		}
		return
	}

	// unregister checks that are not in the new config
	for name, check := range s.checks {
		if !cfg.Checks.HasCheck(name) {
			s.unregisterCheck(ctx, check)
		}
	}

	// register / update checks that are in the new config
	for _, c := range ck {
		if _, ok := s.checks[c.Name()]; !ok {
			err = s.registerCheck(ctx, c)
			if err != nil {
				logger.FromContext(ctx).ErrorContext(ctx, "Failed to register check", "error", err)
			}
			continue
		}

		// existing config
		err = s.checks[c.Name()].SetConfig(c.GetConfig())
		if err != nil {
			logger.FromContext(ctx).ErrorContext(ctx, "Failed to set config for check", "error", err)
		}
	}
}

// enrichTargets updates the targets of the sparrow's checks with the
// global targets. Per default, the two target lists are merged.
func (s *Sparrow) enrichTargets(cfg runtime.Config) runtime.Config {
	if cfg.Empty() || s.tarMan == nil {
		return cfg
	}

	gt := s.tarMan.GetTargets()

	// merge global targets with health check targets
	for _, gt := range gt {
		if gt.Url == fmt.Sprintf("https://%s", s.cfg.SparrowName) {
			continue
		}
		if cfg.Checks.HasHealthCheck() && !slices.Contains(cfg.Checks.Health.Targets, gt.Url) {
			cfg.Checks.Health.Targets = append(cfg.Checks.Health.Targets, gt.Url)
		}
		if cfg.Checks.HasLatencyCheck() && !slices.Contains(cfg.Checks.Latency.Targets, gt.Url) {
			cfg.Checks.Latency.Targets = append(cfg.Checks.Latency.Targets, gt.Url)
		}
	}

	return cfg
}

// registerCheck registers and executes a new check
func (s *Sparrow) registerCheck(ctx context.Context, check checks.Check) error {
	log := logger.FromContext(ctx).With("name", check.Name())

	s.checks[check.Name()] = check

	// Create a fan in a channel for the check
	checkChan := make(chan checks.Result, 1)
	s.resultFanIn[check.Name()] = checkChan

	go fanInResults(checkChan, s.cResult, check.Name())
	err := check.Startup(ctx, checkChan)
	if err != nil {
		log.ErrorContext(ctx, "Failed to startup check", "error", err)
		close(checkChan)
		return err
	}
	check.RegisterHandler(ctx, s.routingTree)

	// Add prometheus collectors of check to registry
	for _, collector := range check.GetMetricCollectors() {
		if err := s.metrics.GetRegistry().Register(collector); err != nil {
			log.ErrorContext(ctx, "Could not add metrics collector to registry")
		}
	}

	go func() {
		err := check.Run(ctx)
		if err != nil {
			log.ErrorContext(ctx, "Failed to run check", "error", err)
		}
	}()
	return nil
}

// UnregisterCheck removes the check from sparrow and performs a soft shutdown for the check
func (s *Sparrow) unregisterCheck(ctx context.Context, check checks.Check) {
	log := logger.FromContext(ctx).With("name", check.Name())
	// Check has been removed from config; shutdown and remove
	check.DeregisterHandler(ctx, s.routingTree)

	// Remove prometheus collectors of check from registry
	for _, metricsCollector := range check.GetMetricCollectors() {
		if !s.metrics.GetRegistry().Unregister(metricsCollector) {
			log.ErrorContext(ctx, "Could not remove metrics collector from registry")
		}
	}

	err := check.Shutdown(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to shutdown check", "error", err)
	}
	if c, ok := s.resultFanIn[check.Name()]; ok {
		// close fan in the channel if it exists
		close(c)
		delete(s.resultFanIn, check.Name())
	}

	delete(s.checks, check.Name())
}

// This is a fan in for the checks.
//
// It allows augmenting the results with the check name which is needed by the db
// without putting the responsibility of providing the name on every iteration on the check
func fanInResults(checkChan chan checks.Result, cResult chan checks.ResultDTO, name string) {
	for i := range checkChan {
		cResult <- checks.ResultDTO{
			Name:   name,
			Result: &i,
		}
	}
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
		var errS error
		if s.tarMan != nil {
			errS = s.tarMan.Shutdown(ctx)
		}
		errA := s.shutdownAPI(ctx)
		s.loader.Shutdown(ctx)
		if errS != nil || errA != nil {
			log.Error("Failed to shutdown gracefully", "contextError", errC, "apiError", errA, "targetError", errS)
		}

		// Signal that shutdown is complete
		s.cDone <- struct{}{}
	})
}
