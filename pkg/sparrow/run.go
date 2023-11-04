package sparrow

import (
	"context"
	"fmt"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
)

type Sparrow struct {
	checks  map[string]checks.Check
	cResult chan checks.Result

	loader     config.Loader
	cfg        *config.Config
	cCfgChecks chan map[string]any
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	// TODO read this from config file
	sparrow := &Sparrow{
		checks:     make(map[string]checks.Check),
		cResult:    make(chan checks.Result),
		cfg:        cfg,
		cCfgChecks: make(chan map[string]any),
	}
	sparrow.loader = config.NewLoader(cfg, sparrow.cCfgChecks)
	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	// TODO Setup before checks run
	// setup database
	// setup http server
	for {
		select {
		case <-ctx.Done():
			return nil
		case result := <-s.cResult:
			// TODO write result to database
			fmt.Println(result)
		case configChecks := <-s.cCfgChecks:
			// Config got updated
			// Set checks
			s.cfg.Checks = configChecks
			s.ReconceilChecks(ctx)
		}
	}
}

// Register new Checks, unregister removed Checks & reset Configs of Checks
func (s *Sparrow) ReconceilChecks(ctx context.Context) {
	for name, check := range s.cfg.Checks {
		if existingCheck, ok := s.checks[name]; ok {
			// Check already registered, reset config
			err := existingCheck.SetConfig(ctx, check)
			if err != nil {
				// log.Errorf("failed to reset config for check, check will run with last applies config - %s: %w", existingCheck.Name(), err)
			}
			continue
		}
		// Check is a new Check and needs to be registered
		check := checks.RegisteredChecks[name]()
		s.checks[name] = check

		err := check.SetConfig(ctx, check)
		if err != nil {
			// log.Errorf("failed to set config for check %s: %w", check.Name(), err)
		}
		err = check.Startup(ctx, s.cResult)
		if err != nil {
			// log.Errorf("failed to startup check %s: %w", check.Name(), err)
		}
		go check.Run(ctx)
	}

	for existingCheckName, existingCheck := range s.checks {
		// Check has been removed from config; shutdown and remove
		if _, ok := s.cfg.Checks[existingCheckName]; !ok {
			existingCheck.Shutdown(ctx)
			delete(s.checks, existingCheckName)
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
	for checkName, c := range s.checks {
		ref, err := c.Schema()
		if err != nil {
			return openapi3.T{}, fmt.Errorf("failed to get schema for check %s: %w", checkName, err)
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", checkName)
		bodyDesc := fmt.Sprintf("Metrics for check %s", checkName)
		doc.Paths["/v1/metrics/"+checkName] = &openapi3.PathItem{
			Description: checkName,
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", checkName},
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
