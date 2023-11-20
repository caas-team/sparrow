package sparrow

import (
	"context"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/api"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/go-chi/chi/v5"
)

type Sparrow struct {
	// TODO refactor this struct to be less convoluted
	// split up responsibilities more clearly
	checks      map[string]checks.Check
	routingTree api.RoutingTree
	resultFanIn map[string]chan checks.Result
	cResult     chan checks.ResultDTO

	loader     config.Loader
	cfg        *config.Config
	cCfgChecks chan map[string]any
	router     chi.Router
	db         db.DB
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	// TODO read this from config file
	sparrow := &Sparrow{
		router:      chi.NewRouter(),
		routingTree: api.NewRoutingTree(),
		checks:      make(map[string]checks.Check),
		cResult:     make(chan checks.ResultDTO),
		resultFanIn: make(map[string]chan checks.Result),
		cfg:         cfg,
		cCfgChecks:  make(chan map[string]any),
		db:          db.NewInMemory(),
	}

	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	sparrow.db = db.NewInMemory()
	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx, "sparrow")
	defer cancel()
	// TODO Setup before checks run
	// setup http server

	// Start the runtime configuration loader
	go s.loader.Run(ctx)
	go s.api(ctx)

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

// Register new Checks, unregister removed Checks & reset Configs of Checks
func (s *Sparrow) ReconcileChecks(ctx context.Context) {
	for name, checkCfg := range s.cfg.Checks {
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

		// Create a fan in channel for the check
		checkChan := make(chan checks.Result)
		s.resultFanIn[name] = checkChan

		err := check.SetConfig(ctx, checkCfg)
		if err != nil {
			log.ErrorContext(ctx, "Failed to set config for check", "name", name, "error", err)
		}
		go fanInResults(checkChan, s.cResult, name)
		err = check.Startup(ctx, checkChan)
		if err != nil {
			log.ErrorContext(ctx, "Failed to startup check", "name", name, "error", err)
			close(checkChan)
			// TODO discuss whether this should return an error instead?
			continue
		}
		check.RegisterHandler(ctx, &s.routingTree)
		go check.Run(ctx)
	}

	for existingCheckName, existingCheck := range s.checks {
		// Check has been removed from config; shutdown and remove
		if _, ok := s.cfg.Checks[existingCheckName]; !ok {
			existingCheck.DeregisterHandler(ctx, &s.routingTree)
			existingCheck.Shutdown(ctx)
			if c, ok := s.resultFanIn[existingCheckName]; ok {
				// close fan in channel if it exists
				close(c)
				delete(s.resultFanIn, existingCheckName)
			}

			delete(s.checks, existingCheckName)
		}
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
