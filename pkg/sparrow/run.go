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
	"time"

	"github.com/caas-team/sparrow/pkg/sparrow/targets"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = time.Second * 90

type Sparrow struct {
	db db.DB
	// the existing checks
	checks map[string]checks.Check
	client *http.Client
	server *http.Server

	metrics Metrics

	resultFanIn map[string]chan checks.Result
	cfg         *config.Config

	// cCfgChecks is the channel where the loader sends the runtime configuration of the checks
	cCfgChecks chan map[string]any
	// cResult is the channel where the checks send their results to
	cResult chan checks.ResultDTO
	// cErr is used to handle non-recoverable errors of the sparrow components
	cErr chan error
	// cDone is used to signal that the sparrow was shut down because of an error
	cDone chan struct{}

	loader  config.Loader
	targets targets.TargetManager

	routingTree *api.RoutingTree
	router      chi.Router
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	sparrow := &Sparrow{
		db:          db.NewInMemory(),
		checks:      make(map[string]checks.Check),
		client:      &http.Client{},
		metrics:     NewMetrics(),
		resultFanIn: make(map[string]chan checks.Result),
		cResult:     make(chan checks.ResultDTO, 1),
		cfg:         cfg,
		cCfgChecks:  make(chan map[string]any, 1),
		routingTree: api.NewRoutingTree(),
		router:      chi.NewRouter(),
		cErr:        make(chan error, 1),
		cDone:       make(chan struct{}, 1),
	}

	sparrow.server = &http.Server{Addr: cfg.Api.ListeningAddress, Handler: sparrow.router, ReadHeaderTimeout: readHeaderTimeout}

	if cfg.HasTargetManager() {
		gm := targets.NewGitlabManager(cfg.SparrowName, cfg.TargetManager)
		sparrow.targets = gm
	}

	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	sparrow.db = db.NewInMemory()
	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()

	go func() {
		s.cErr <- s.loader.Run(ctx)
	}()
	go func() {
		if s.targets != nil {
			s.cErr <- s.targets.Reconcile(ctx)
		}
	}()
	go func() {
		s.cErr <- s.api(ctx)
	}()

	go s.handleErrors(ctx)

	for {
		select {
		case result := <-s.cResult:
			go s.db.Save(result)
		case configChecks := <-s.cCfgChecks:
			s.cfg.Checks = configChecks
			s.ReconcileChecks(ctx)
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

	if s.targets == nil {
		return checkCfg
	}
	gt := s.targets.GetTargets()

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

	getRegisteredCheck := checks.RegisteredChecks[name]
	if getRegisteredCheck == nil {
		log.WarnContext(ctx, "Check is not registered")
		return
	}
	check := getRegisteredCheck()
	s.checks[name] = check

	// Create a fan in a channel for the check
	checkChan := make(chan checks.Result, 1)
	s.resultFanIn[name] = checkChan

	check.SetClient(s.client)
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
}

// UnregisterCheck removes the check from sparrow and performs a soft shutdown for the check
func (s *Sparrow) unregisterCheck(ctx context.Context, name string, check checks.Check) {
	log := logger.FromContext(ctx).With("name", name)
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

	var errS error
	if s.cfg.HasTargetManager() {
		errS = s.targets.Shutdown(ctx)
	}
	errA := s.shutdownAPI(ctx)
	if errS != nil || errA != nil {
		log.Error("Failed to shutdown gracefully", "contextError", errC, "apiError", errA, "targetError", errS)
	}

	s.cDone <- struct{}{}
}

// handleErrors handles errors from the sparrow components
// If a non-recoverable error happens, the sparrow will shut down
// the various components and return the error.
func (s *Sparrow) handleErrors(ctx context.Context) {
	log := logger.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Info("Context done, shutting down sparrow")
			s.shutdown(ctx)

		case err := <-s.cErr:
			if err != nil {
				log.Error("Error in sparrow component, shutting down", "error", err)
				s.shutdown(ctx)
			}
		}
	}
}
