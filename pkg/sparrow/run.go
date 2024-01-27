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
	"github.com/caas-team/sparrow/pkg/sparrow/targets"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks/register"
	"github.com/caas-team/sparrow/pkg/checks/types"
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

	resultFanIn map[string]chan types.Result

	cfg *config.Config

	cCfgChecks chan map[string]any
	// cResult is the channel where the checks send their results to
	cResult chan types.ResultDTO
	// cErr is used to handle non-recoverable errors of the sparrow components
	cErr chan error
	// cDone is used to signal that the sparrow was shut down because of an error
	cDone chan struct{}
	// shutOnce is used to ensure that the shutdown function is only called once
	shutOnce sync.Once

	loader config.Loader
	tarMan targets.TargetManager

	router chi.Router
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	sparrow := &Sparrow{
		db:          db.NewInMemory(),
		checks:      make(map[string]checks.Check),
		metrics:     NewMetrics(),
		resultFanIn: make(map[string]chan types.Result),
		cResult:     make(chan types.ResultDTO, 1),
		cfg:         cfg,
		cCfgChecks:  make(chan map[string]any, 1),
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
		case configChecks := <-s.cCfgChecks:
			s.cfg.Checks = configChecks
			s.ReconcileChecks(ctx)
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
func (s *Sparrow) ReconcileChecks(ctx context.Context) {
	for name, checkCfg := range s.cfg.Checks {
		name := name
		log := logger.FromContext(ctx).With("name", name)

		c := s.updateCheckTargets(checkCfg)
		if existingCheck, ok := s.checks[name]; ok {
			// Check already registered, reset config
			err := existingCheck.SetConfig(ctx, c)
			if err != nil {
				log.ErrorContext(ctx, "Failed to reset config for check, check will run with last applies config", "error", err)
			}
			continue
		}

		// Check is a new Check and needs to be registered
		s.registerCheck(ctx, name, c)
	}

	for existingCheckName, existingCheck := range s.checks {
		if _, ok := s.cfg.Checks[existingCheckName]; ok {
			// Check is known check
			continue
		}

		// Check has been removed from config
		s.unregisterCheck(ctx, existingCheckName, existingCheck)
	}
}

// updateCheckTargets updates the targets of a check with the
// global targets. The targets are merged per default, if found in the
// passed config.
func (s *Sparrow) updateCheckTargets(cfg any) any {
	if cfg == nil {
		return nil
	}

	// check if map with targets
	checkCfg, ok := cfg.(map[string]any)
	if !ok {
		return checkCfg
	}
	if _, ok = checkCfg["targets"]; !ok {
		return checkCfg
	}

	// Check if targets is a slice
	actuali, ok := checkCfg["targets"].([]any)
	if !ok {
		return checkCfg
	}
	if len(actuali) == 0 {
		return checkCfg
	}

	// convert to string slice
	var actual []string
	for _, v := range actuali {
		if _, ok := v.(string); !ok {
			return checkCfg
		}
		actual = append(actual, v.(string))
	}
	var urls []string

	if s.tarMan == nil {
		return checkCfg
	}
	gt := s.tarMan.GetTargets()

	// filter out globalTargets that are already in the config and self
	for _, t := range gt {
		if slices.Contains(actual, t.Url) {
			continue
		}
		if t.Url == fmt.Sprintf("https://%s", s.cfg.SparrowName) {
			continue
		}
		urls = append(urls, t.Url)
	}

	checkCfg["targets"] = append(actual, urls...)
	return checkCfg
}

// registerCheck registers and executes a new check
func (s *Sparrow) registerCheck(ctx context.Context, name string, checkCfg any) {
	log := logger.FromContext(ctx).With("name", name)

	getRegisteredCheck := register.RegisteredChecks[name]
	if getRegisteredCheck == nil {
		log.WarnContext(ctx, "Check is not registered")
		return
	}
	check := getRegisteredCheck()
	s.checks[name] = check

	// Create a fan in a channel for the check
	checkChan := make(chan types.Result, 1)
	s.resultFanIn[name] = checkChan

	err := check.SetConfig(ctx, checkCfg)
	if err != nil {
		log.ErrorContext(ctx, "Failed to set config for check", "error", err)
	}
	go fanInResults(checkChan, s.cResult, name)
	err = check.Startup(ctx, checkChan)
	if err != nil {
		log.ErrorContext(ctx, "Failed to startup check", "error", err)
		close(checkChan)
		return
	}

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
}

// UnregisterCheck removes the check from sparrow and performs a soft shutdown for the check
func (s *Sparrow) unregisterCheck(ctx context.Context, name string, check checks.Check) {
	log := logger.FromContext(ctx).With("name", name)

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
	if c, ok := s.resultFanIn[name]; ok {
		// close fan in the channel if it exists
		close(c)
		delete(s.resultFanIn, name)
	}

	delete(s.checks, name)
}

// This is a fan in for the checks.
//
// It allows augmenting the results with the check name which is needed by the db
// without putting the responsibility of providing the name on every iteration on the check
func fanInResults(checkChan chan types.Result, cResult chan types.ResultDTO, name string) {
	for i := range checkChan {
		cResult <- types.ResultDTO{
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
