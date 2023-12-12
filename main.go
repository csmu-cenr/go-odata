package main

import (
	"flag"
	"fmt"

	"github.com/Uffe-Code/go-odata/modelGenerator"
)

func main() {

	var config string
	flag.StringVar(&config, "config", "config.json", "The configuration for the model and results generations.")
	// Parse the command-line arguments
	flag.Parse()

	generator, err := modelGenerator.New(config)
	if err != nil {
		panic(err.Error())
	}

	err = generator.GenerateCode()
	if err != nil {
		fmt.Printf("Error while generating code: %s\n", err.Error())
		return
	}

	fmt.Printf("Code generated successfully\n")
}
