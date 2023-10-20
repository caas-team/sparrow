package sparrow

import (
	"context"
	"fmt"

	"github.com/caas-team/sparrow/pkg/checks"
)

type Sparrow struct {
	checks []checks.Check
	config *Config
	c      chan checks.Result
}

// New creates a new sparrow from a given configfile
func New(config *Config) *Sparrow {
	// TODO read this from config file
	return &Sparrow{
		config: config,
		c:      make(chan checks.Result),
	}
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	// TODO Setup before checks run
	// setup database
	// setup http server

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-s.c:
				// TODO write result to database
				fmt.Println(result)
			}
		}
	}()

	for checkName, checkConfig := range s.config.Checks {
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

	return nil
}
