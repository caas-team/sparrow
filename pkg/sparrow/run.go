package sparrow

import (
	"context"
	"fmt"
	"log"

	"github.com/caas-team/sparrow/pkg/db"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
)

type Sparrow struct {
	checks      map[string]checks.Check
	resultFanIn map[string]chan checks.Result
	cResult     chan checks.ResultDTO

	loader     config.Loader
	cfg        *config.Config
	cCfgChecks chan map[string]any

	db db.DB
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	// TODO read this from config file
	sparrow := &Sparrow{
		checks:      make(map[string]checks.Check),
		resultFanIn: make(map[string]chan checks.Result),
		cfg:         cfg,
		cCfgChecks:  make(chan map[string]any),
	}
	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	sparrow.db = db.NewInMemory()
	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	// TODO Setup before checks run
	// setup http server

	// Start the runtime configuration loader
	go s.loader.Run(ctx)

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

var oapiBoilerplate = openapi3.T{
	OpenAPI: "3.0.0",
	Info: &openapi3.Info{
		Title:       "Sparrow Metrics API",
		Description: "Serves metrics collected by sparrows checks",
		Contact: &openapi3.Contact{
			URL:   "https://caas.telekom.de",
			Email: "caas-request@telekom.de",
			Name:  "CaaS Team",
		},
	},
	Paths:      make(openapi3.Paths),
	Extensions: make(map[string]interface{}),
	Components: &openapi3.Components{
		Schemas: make(openapi3.Schemas),
	},
	Servers: openapi3.Servers{},
}

func (s *Sparrow) Openapi() (openapi3.T, error) {
	doc := oapiBoilerplate
	for name, c := range s.checks {
		ref, err := c.Schema()
		if err != nil {
			return openapi3.T{}, fmt.Errorf("failed to get schema for check %s: %w", name, err)
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", name)
		bodyDesc := fmt.Sprintf("Metrics for check %s", name)
		doc.Paths["/v1/metrics/"+name] = &openapi3.PathItem{
			Description: name,
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", name},
				Responses: openapi3.Responses{
					"200": &openapi3.ResponseRef{
						Value: &openapi3.Response{
							Description: &bodyDesc,
							Content:     openapi3.NewContentWithSchemaRef(ref, []string{"application/json"}),
						},
					},
				},
			},
		}

	}

	return doc, nil
}
