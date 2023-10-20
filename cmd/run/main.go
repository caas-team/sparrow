package main

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/sparrow"
)

func main() {

	config := sparrow.NewConfig()
	sparrow := sparrow.New(config)

	log.Println("running sparrow")
	if err := sparrow.Run(context.Background()); err != nil {
		panic(err)
	}

}
