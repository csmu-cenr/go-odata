package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Uffe-Code/go-odata/modelGenerator"
)

func main() {

	executable := os.Args[0]
	if strings.Contains(executable, "_debug") {
		os.Args = append(os.Args, "config.json")
	}
	if len(os.Args) != 2 {
		switch executable {
		case "./go-odata":
			fmt.Println("Usage: ./go-odata <configfile>")
		case "go-odata":
			fmt.Println("Usage: go-odata <configfile>")
		default:
			fmt.Println("Usage: go run main.go <configfile>")
		}
		os.Exit(1)
	}
	config := os.Args[1]

	generator, err := modelGenerator.New(config)
	if err != nil {
		panic(err.Error())
	}

	err = generator.GenerateCode()
	if err != nil {
		fmt.Printf("Error while generating code: %s\n", err.Error())
		return
	}

	fmt.Printf("\nCode generated successfully\n")
}
