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
	"time"

	"github.com/caas-team/sparrow/pkg/sparrow/gitlab"
	targets "github.com/caas-team/sparrow/pkg/sparrow/targets"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/go-chi/chi/v5"
)

type Sparrow struct {
	db db.DB
	// the existing checks
	checks map[string]checks.Check
	client *http.Client

	resultFanIn map[string]chan checks.Result
	cResult     chan checks.ResultDTO

	cfg        *config.Config
	loader     config.Loader
	cCfgChecks chan map[string]any
	targets    targets.TargetManager

	routingTree *api.RoutingTree
	router      chi.Router
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	sparrow := &Sparrow{
		db:          db.NewInMemory(),
		checks:      make(map[string]checks.Check),
		client:      &http.Client{},
		resultFanIn: make(map[string]chan checks.Result),
		cResult:     make(chan checks.ResultDTO, 1),
		cfg:         cfg,
		cCfgChecks:  make(chan map[string]any, 1),
		routingTree: api.NewRoutingTree(),
		router:      chi.NewRouter(),
		targets:     targets.NewGitlabManager(gitlab.New("targetsRepo", "gitlabToken", 1), 5*time.Minute, 15*time.Minute),
	}

	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	sparrow.db = db.NewInMemory()
	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "sparrow")
	defer cancel()

	go s.loader.Run(ctx)
	go s.targets.Reconcile(ctx)
	// Start the api
	go func() {
		err := s.api(ctx)
		if err != nil {
			panic(fmt.Sprintf("Failed to start api: %v", err))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return err
			}
			return nil
		case result := <-s.cResult:
			go s.db.Save(result)
		case configChecks := <-s.cCfgChecks:
			// Config got updated
			// Set checks
			s.cfg.Checks = configChecks
			s.ReconcileChecks(ctx)
		}
	}
}

// ReconcileChecks registers new Checks, unregisters removed Checks,
// resets the Configs of Checks and starts running the checks
func (s *Sparrow) ReconcileChecks(ctx context.Context) {
	for name, checkCfg := range s.cfg.Checks {
		name := name
		log := logger.FromContext(ctx).With("name", name)
		if existingCheck, ok := s.checks[name]; ok {
			// Check already registered, reset config
			err := existingCheck.SetConfig(ctx, checkCfg)
			if err != nil {
				log.ErrorContext(ctx, "Failed to reset config for check, check will run with last applies config", "error", err)
			}
			continue
		}
		// Check is a new Check and needs to be registered
		getRegisteredCheck := checks.RegisteredChecks[name]
		if getRegisteredCheck == nil {
			log.WarnContext(ctx, "Check is not registered")
			continue
		}
		check := getRegisteredCheck()
		s.checks[name] = check

		// Create a fan in a channel for the check
		checkChan := make(chan checks.Result, 1)
		s.resultFanIn[name] = checkChan

		check.SetClient(s.client)
		err := check.SetConfig(ctx, checkCfg)
		if err != nil {
			log.ErrorContext(ctx, "Failed to set config for check", "name", name, "error", err)
		}
		go fanInResults(checkChan, s.cResult, name)
		err = check.Startup(ctx, checkChan)
		if err != nil {
			log.ErrorContext(ctx, "Failed to startup check", "name", name, "error", err)
			close(checkChan)
			continue
		}
		check.RegisterHandler(ctx, s.routingTree)

		go func() {
			err := check.Run(ctx)
			if err != nil {
				log.ErrorContext(ctx, "Failed to run check", "name", name, "error", err)
			}
		}()
	}

	for existingCheckName, existingCheck := range s.checks {
		log := logger.FromContext(ctx).With("checkName", existingCheckName)
		if _, ok := s.cfg.Checks[existingCheckName]; ok {
			continue
		}

		// Check has been removed from config; shutdown and remove
		existingCheck.DeregisterHandler(ctx, s.routingTree)
		err := existingCheck.Shutdown(ctx)
		if err != nil {
			log.ErrorContext(ctx, "Failed to shutdown check", "error", err)
		}
		if c, ok := s.resultFanIn[existingCheckName]; ok {
			// close fan in the channel if it exists
			close(c)
			delete(s.resultFanIn, existingCheckName)
		}

		delete(s.checks, existingCheckName)
	}
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

// GlobalTarget represents a GlobalTarget that can be checked
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}
