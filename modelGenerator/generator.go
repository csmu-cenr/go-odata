package modelGenerator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Generator struct {
	ApiUrl  string `json:"apiUrl"`
	Package struct {
		CreateDirectoryIfMissing bool     `json:"createDirectoryIfMissing"`
		Directory                string   `json:"directory"`
		Extras                   []string `json:"extras"`
		Models                   string   `json:"models"`
		ByteResults              string   `json:"byteResults`
		StructResults            string   `json:"structResults`
	} `json:"package"`
	Imports struct {
		Models  []string `json:"models"`
		Results []string `json:"results"`
	} `json:"imports"`
	Fields struct {
		Public bool `json:"public"` // Change a_field__name__ to AFieldName
		Json   struct {
			Tags      bool `json:"tags"`
			OmitEmpty bool `json:"omitempty"`
		} `json:"json"`
		Extras []string `json:"extras"`
	} `json:"fields"`
}

func New(path string) (Generator, error) {

	var generator Generator

	bytes, err := os.ReadFile(path) // just pass the file name
	if err != nil {
		return generator, err
	}

	err = json.Unmarshal(bytes, &generator)
	return generator, err
}

func (g Generator) metadataUrl() string {
	return strings.TrimRight(g.ApiUrl, "/") + "/$metadata"
}

func (g Generator) GenerateCode() error {

	dirPath, err := filepath.Abs(g.Package.Directory)
	if err != nil {
		return err
	}

	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		fmt.Printf("%s.\n", err.Error())
		if g.Package.CreateDirectoryIfMissing {
			fmt.Printf("Creating %s.\n", dirPath)
			err := os.Mkdir(dirPath, 0751)
			if err != nil {
				fmt.Printf("%s does not exist. %s.", dirPath, err.Error())
				return err
			}
		} else {
			return err
		}
	}

	edmx, err := fetchEdmx(g.metadataUrl())
	if err != nil {
		return err
	}

	packageName := filepath.Base(dirPath)
	code := g.generateCodeFromSchema(packageName, edmx)
	for fileName, contents := range code {
		filePath := fmt.Sprintf("%s%s%s", dirPath, string(filepath.Separator), fileName)
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		_, err = file.WriteString(contents)
	}

	return err
}
