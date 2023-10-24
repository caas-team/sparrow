package main

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/checks"
	"github.com/caas-team/sparrow/pkg/sparrow"
)

func main() {

	config := sparrow.NewConfig()
	config.Checks["rtt"] = checks.RoundTripConfig{}
	sparrow := sparrow.New(config)

	log.Println("running sparrow")
	if err := sparrow.Run(context.Background()); err != nil {
		panic(err)
	}

}
