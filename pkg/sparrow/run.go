package sparrow

import (
	"context"
	"fmt"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
)

type Sparrow struct {
	checks []checks.Check
	cfg    *config.Config
	loader config.Loader
	c      chan checks.Result
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	// TODO read this from config file
	return &Sparrow{
		cfg:    cfg,
		c:      make(chan checks.Result),
		loader: config.NewLoader(cfg),
		// loader: config.NewLoader(UpdateChecksConfig), //CALLBACK
	}
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	// TODO Setup before checks run
	// setup database
	// setup http server

	// TODO CAN BE REMOVED
	for checkName, checkConfig := range s.cfg.Checks {
		check := checks.RegisteredChecks[checkName](checkName)
		s.checks = append(s.checks, check)

		err := check.SetConfig(ctx, checkConfig)
		if err != nil {
			return fmt.Errorf("failed to set config for check %s: %w", check.Name(), err)
		}
		err = check.Startup(ctx, s.c)
		if err != nil {
			return fmt.Errorf("failed to startup check %s: %w", check.Name(), err)
		}
		go check.Run(ctx)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case result := <-s.c:
			// TODO write result to database
			fmt.Println(result)
		case <-s.cfg.Updated:
			// Config got updated
			// Set checks
			s.UpdateChecksConfig(ctx)
		}
	}
}

// Register new Checks, unregister removed Checks & reset Configs of Checks
func (s *Sparrow) UpdateChecksConfig(ctx context.Context) {
	for configCheckName, configChecks := range s.cfg.Checks {
		newCheckCounter := 0
		for i, existingCheck := range s.checks {
			newCheckCounter++

			// Check already registered, reset config
			if configCheckName == existingCheck.Name() {
				err := existingCheck.SetConfig(ctx, configChecks)
				if err != nil {
					// log.Errorf("failed to reset config for check, check will run with last applies config - %s: %w", existingCheck.Name(), err)
				}
				continue
			}

			// TODO: check if to shutdown
			// Check has been removed from config; shutdown and remove
			existingCheck.Shutdown(ctx)
			s.checks = remove(s.checks, i)
		}

		// Check is a new Check and needs to be registered
		if newCheckCounter+1 >= len(s.checks) {
			check := checks.RegisteredChecks[configCheckName](configCheckName)
			s.checks = append(s.checks, check)

			err := check.SetConfig(ctx, configChecks)
			if err != nil {
				// log.Errorf("failed to set config for check %s: %w", check.Name(), err)
			}
			err = check.Startup(ctx, s.c)
			if err != nil {
				// log.Errorf("failed to startup check %s: %w", check.Name(), err)
			}
			go check.Run(ctx)
		}
	}
}

// Remove specific Check from
func remove(s []checks.Check, i int) []checks.Check {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]

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
	for _, c := range s.checks {
		ref, err := c.Schema()
		if err != nil {
			return openapi3.T{}, fmt.Errorf("failed to get schema for check %s: %w", c.Name(), err)
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", c.Name())
		bodyDesc := fmt.Sprintf("Metrics for check %s", c.Name())
		doc.Paths["/v1/metrics/"+c.Name()] = &openapi3.PathItem{
			Description: c.Name(),
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", c.Name()},
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
