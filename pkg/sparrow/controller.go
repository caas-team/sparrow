// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
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
	"errors"
	"fmt"
	"net/http"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"github.com/caas-team/sparrow/pkg/db"
	"github.com/getkin/kin-openapi/openapi3"
)

// ChecksController is responsible for managing checks.
type ChecksController struct {
	db      db.DB
	metrics Metrics
	checks  runtime.Checks
	cErr    chan error
	done    chan struct{}
}

// NewChecksController creates a new ChecksController.
func NewChecksController(dbase db.DB, metrics Metrics) *ChecksController {
	return &ChecksController{
		db:      dbase,
		metrics: metrics,
		checks:  runtime.Checks{},
		cErr:    make(chan error, 1),
		done:    make(chan struct{}, 1),
	}
}

// ListenErrors listens for errors and unregisters failed checks.
func (cc *ChecksController) ListenErrors(ctx context.Context) {
	log := logger.FromContext(ctx)

	for {
		select {
		case err := <-cc.cErr:
			var runErr *ErrRunningCheck
			if errors.As(err, &runErr) {
				uErr := cc.UnregisterCheck(ctx, runErr.Check)
				if uErr != nil {
					log.ErrorContext(ctx, "Failed to unregister check", "check", runErr.Check.Name(), "error", uErr)
				}
			}
			log.ErrorContext(ctx, "Error while running check", "error", err)
		case <-ctx.Done():
			return
		case <-cc.done:
			cc.cErr <- nil
			return
		}
	}
}

// Shutdown shuts down the ChecksController.
func (cc *ChecksController) Shutdown(ctx context.Context) (err error) {
	log := logger.FromContext(ctx)

	for _, c := range cc.checks.Iter() {
		cErr := cc.UnregisterCheck(ctx, c)
		if cErr != nil {
			log.Error("Failed to unregister check while shutting down", "error", cErr)
			err = errors.Join(err, cErr)
		}
	}
	cc.done <- struct{}{}
	close(cc.done)
	return err
}

// RegisterCheck registers a new check.
func (cc *ChecksController) RegisterCheck(ctx context.Context, check checks.Check) error {
	log := logger.FromContext(ctx).With("check", check.Name())

	// Add prometheus collectors of check to registry
	for _, collector := range check.GetMetricCollectors() {
		if err := cc.metrics.GetRegistry().Register(collector); err != nil {
			log.ErrorContext(ctx, "Could not add metrics collector to registry")
		}
	}

	go func() {
		err := cc.runCheck(ctx, check)
		if err != nil {
			log.ErrorContext(ctx, "Failed to start check", "error", err)
			cc.cErr <- err
		}
	}()

	cc.checks.Add(check)
	return nil
}

// UnregisterCheck unregisters a check.
func (cc *ChecksController) UnregisterCheck(ctx context.Context, check checks.Check) error {
	log := logger.FromContext(ctx).With("check", check.Name())

	// Remove prometheus collectors of check from registry
	for _, metricsCollector := range check.GetMetricCollectors() {
		if !cc.metrics.GetRegistry().Unregister(metricsCollector) {
			log.ErrorContext(ctx, "Could not remove metrics collector from registry")
		}
	}

	err := check.Shutdown(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to shutdown check", "error", err)
		return err
	}

	for i, existingCheck := range cc.checks.Iter() {
		if existingCheck.Name() == check.Name() {
			cc.checks.Delete(i)
			break
		}
	}

	return nil
}

func (cc *ChecksController) runCheck(ctx context.Context, check checks.Check) error {
	log := logger.FromContext(ctx).With("check", check.Name())

	go func() {
		err := check.Run(ctx)
		if err != nil {
			log.ErrorContext(ctx, "Failed to run check", "error", err)
			cc.cErr <- &ErrRunningCheck{
				Check: check,
				Err:   err,
			}
		}
	}()

	for {
		select {
		case result := <-check.ResultChan():
			cc.db.Save(checks.ResultDTO{
				Name:   check.Name(),
				Result: &result,
			})
		case <-ctx.Done():
			return ctx.Err()
		case <-cc.done:
			return nil
		}
	}
}

var oapiBoilerplate = openapi3.T{
	// this object should probably be user defined
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
	Extensions: make(map[string]any),
	Components: &openapi3.Components{
		Schemas: make(openapi3.Schemas),
	},
	Servers: openapi3.Servers{},
}

// GenerateCheckSpecs generates the OpenAPI specifications for the given checks
// Returns the complete OpenAPI specification for all checks
func (cc *ChecksController) GenerateCheckSpecs(ctx context.Context) (openapi3.T, error) {
	log := logger.FromContext(ctx)
	doc := oapiBoilerplate
	for _, c := range cc.checks.Iter() {
		name := c.Name()
		ref, err := c.Schema()
		if err != nil {
			log.Error("Failed to get schema for check", "name", name, "error", err)
			return openapi3.T{}, &ErrCreateOpenapiSchema{name: name, err: err}
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", name)
		bodyDesc := fmt.Sprintf("Metrics for check %s", name)
		doc.Paths["/v1/metrics/"+name] = &openapi3.PathItem{
			Description: name,
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", name},
				Responses: openapi3.Responses{
					fmt.Sprint(http.StatusOK): &openapi3.ResponseRef{
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
