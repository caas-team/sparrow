package main

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
)

func main() {
	cfg := config.NewConfig()
	sparrow := sparrow.New(cfg)

	log.Println("running sparrow")
	if err := sparrow.Run(context.Background()); err != nil {
		panic(err)
	}
}
