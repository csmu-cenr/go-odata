package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/Uffe-Code/go-odata/modelGenerator"
)

func main() {

	var config string
	flag.StringVar(&config, "config", "config.json", "The configuration for the model and results generations.")
	// Parse the command-line arguments
	flag.Parse()

	directoryPath, err := filepath.Abs(filepath.Dir(config))
	if err != nil {
		panic(err.Error())
	}
	basePath := filepath.Base(config)
	configPath := filepath.Join(directoryPath, basePath)
	generator, err := modelGenerator.New(configPath)
	if err != nil {
		panic(err.Error())
	}

	err = generator.GenerateCode()
	if err != nil {
		fmt.Printf("error while generating code: %s", err.Error())
		return
	}

	fmt.Printf("code generated successfully")
}
