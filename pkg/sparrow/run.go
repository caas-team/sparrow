package sparrow

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/go-chi/chi/v5"
)

type Sparrow struct {
	checks      map[string]checks.Check
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
		checks:      make(map[string]checks.Check),
		cResult:     make(chan checks.ResultDTO),
		resultFanIn: make(map[string]chan checks.Result),
		cfg:         cfg,
		cCfgChecks:  make(chan map[string]any),
		db:          db.NewInMemory(),
	}

	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	// TODO Setup before checks run
	// setup http server

	// Start the runtime configuration loader
	go s.loader.Run(ctx)
	go s.api(ctx)
	// register routes dynamically https://github.com/gofiber/fiber/issues/735#issuecomment-678586434

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return err
			}
			return nil
		case result := <-s.cResult:
			s.db.Save(result)
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
		if existingCheck, ok := s.checks[name]; ok {
			// Check already registered, reset config
			err := existingCheck.SetConfig(ctx, checkCfg)
			if err != nil {
				log.Printf("Failed to reset config for check, check will run with last applies config - %s: %s", name, err.Error())
			}
			continue
		}
		// Check is a new Check and needs to be registered
		getRegisteredCheck := checks.RegisteredChecks[name]
		if getRegisteredCheck == nil {
			log.Printf("Check %s is not registered", name)
			continue
		}
		check := getRegisteredCheck()
		s.checks[name] = check

		// Create a fan in channel for the check
		checkChan := make(chan checks.Result)
		s.resultFanIn[name] = checkChan

		err := check.SetConfig(ctx, checkCfg)
		if err != nil {
			log.Printf("Failed to set config for check %s: %s", name, err.Error())
		}
		go fanInResults(checkChan, s.cResult, name)
		err = check.Startup(ctx, checkChan)
		if err != nil {
			log.Printf("Failed to startup check %s: %s", name, err.Error())
			close(checkChan)
			// TODO discuss whether this should return an error instead?
			continue

		}
		go check.Run(ctx)
	}

	for existingCheckName, existingCheck := range s.checks {
		// Check has been removed from config; shutdown and remove
		if _, ok := s.cfg.Checks[existingCheckName]; !ok {
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

func fanInResults(checkChan chan checks.Result, cResult chan checks.ResultDTO, name string) {
	// this is a fan in for the checks
	// it allows augmenting the results with the check name which is needed by the db
	// without putting the responsibility of providing the name on every iteration on the check
	for i := range checkChan {
		cResult <- checks.ResultDTO{
			Name:   name,
			Result: &i,
		}
	}
}
