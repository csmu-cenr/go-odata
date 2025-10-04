package main

import (
	"flag"
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
	if len(os.Args) < 2 {
		switch executable {
		case "./go-odata":
			fmt.Println("Usage: ./go-odata <configfile> [--meta|-m]")
		case "go-odata":
			fmt.Println("Usage: go-odata <configfile> [--meta|-m]")
		default:
			fmt.Println("Usage: go run main.go <configfile> [--meta|-m]")
		}
		os.Exit(1)
	}
	config := os.Args[1]

	generator, err := modelGenerator.New(config)
	if err != nil {
		panic(err.Error())
	}

	if len(os.Args) > 2 {

		// primary flag variables
		meta := flag.Bool("meta", false, "Enable meta mode")
		readOnly := flag.Bool("read-only", false, "Honour read-only tags")

		// alias flags (just dummy placeholders)
		metaShort := flag.Bool("m", false, "Alias for --meta")
		readOnlyShort := flag.Bool("r", false, "Alias for --read-only")
		noMeta := flag.Bool("no-meta", false, "Disable meta mode")
		noMetaShort := flag.Bool("n", false, "Alias for --no-meta")
		noReadOnly := flag.Bool("no-read-only", false, "Disable read-only tags")
		noReadOnlyShort := flag.Bool("o", false, "Alias for --no-read-only")

		// Parse command-line arguments
		flag.Parse()

		// Normalize: apply alias logic
		if *metaShort {
			*meta = true
		}
		if *readOnlyShort {
			*readOnly = true
		}
		if *noMeta || *noMetaShort {
			*meta = false
		}
		if *noReadOnly || *noReadOnlyShort {
			*readOnly = false
		}

		generator.ReadOnly = *readOnly
		generator.Meta = *meta

	}

	err = generator.GenerateCode()
	if err != nil {
		fmt.Printf("Error while generating code: %s\n", err.Error())
		return
	}

	fmt.Printf("\nCode generated successfully\n")
}
