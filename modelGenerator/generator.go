package modelGenerator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ModelGeneratorError struct {
	Function  string
	Attempted string
	Detail    interface{}
}

func (e ModelGeneratorError) Error() string {
	bytes, _ := json.Marshal(e)
	return string(bytes)
}

type Generator struct {
	ApiUrl  string `json:"apiUrl"`
	Package struct {
		CreateDirectoryIfMissing bool   `json:"createDirectoryIfMissing"`
		Directory                string `json:"directory"`
		FieldsConstants          string `json:"fieldsConstants"`
		FieldsPackageName        string
		TablesConstants          string `json:"tablesConstants"`
		TablesPackageName        string
		Extras                   string `json:"extras"`
		Models                   string `json:"models"`
		SelectByTableName        string `json:"selectByTableName"`
		Select                   string `json:"select"`
		Datasets                 string `json:"datasets"`
		Maps                     string `json:"maps"`
		Update                   string `json:"update"`
	} `json:"package"`
	Fields struct {
		Public   bool              `json:"public"` // Change a_field__name__ to AFieldName
		Pointers bool              `json:"pointers"`
		Swap     map[string]string `json:"swap"`
		Json     struct {
			Tags      bool `json:"tags"`
			OmitEmpty bool `json:"omitempty"`
		} `json:"json"`
		Extras []string `json:"extras"`
		Ignore struct {
			StartsWith []string `json:"startsWith"`
			Contains   []string `json:"contains"`
			EndsWith   []string `json:"endsWith"`
			Equals     []string `json:"equals"`
		} `json:"ignore"`
	} `json:"fields"`
}

func New(path string) (Generator, error) {

	function := "New Generator"
	var generator Generator

	data, err := os.ReadFile(path) // just pass the file name
	if err != nil {
		e := ModelGeneratorError{
			Attempted: fmt.Sprintf("Reading file: %s", path),
			Function:  function,
			Detail:    err}
		return generator, e
	}

	err = json.Unmarshal(data, &generator)
	if err != nil {
		e := ModelGeneratorError{
			Attempted: fmt.Sprintf("Unmarshalling: %s", string(data)),
			Function:  function,
			Detail:    err}
		return generator, e
	}

	return generator, nil
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
			err := MkdirP(dirPath, 0755)
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
		if err != nil {
			fmt.Printf("error: %s", err.Error())
			continue
		}
	}

	tablesPath, err := filepath.Abs(g.Package.TablesConstants)
	if err != nil {
		return err
	}
	tablesPath = filepath.Dir(tablesPath)
	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		fmt.Printf("%s.\n", err.Error())
		if g.Package.CreateDirectoryIfMissing {
			fmt.Printf("Creating %s.\n", dirPath)
			err := MkdirP(tablesPath, 0755)
			if err != nil {
				fmt.Printf("%s does not exist. %s.", tablesPath, err.Error())
				return err
			}
		} else {
			return err
		}
	}
	g.Package.TablesPackageName = filepath.Base(tablesPath)
	contents := g.generateTableConstants(edmx)
	file, err := os.Create(g.Package.TablesConstants)
	if err != nil {
		return err
	}
	_, err = file.WriteString(contents)
	if err != nil {
		return err
	}

	fieldsPath, err := filepath.Abs(g.Package.FieldsConstants)
	if err != nil {
		return err
	}
	fieldsPath = filepath.Dir(fieldsPath)
	_, err = os.Stat(fieldsPath)
	if os.IsNotExist(err) {
		fmt.Printf("%s.\n", err.Error())
		if g.Package.CreateDirectoryIfMissing {
			fmt.Printf("Creating %s.\n", fieldsPath)
			err := MkdirP(fieldsPath, 0755)
			if err != nil {
				fmt.Printf("%s does not exist. %s.", fieldsPath, err.Error())
				return err
			}
		} else {
			return err
		}
	}
	g.Package.FieldsPackageName = filepath.Base(fieldsPath)
	contents = g.generateFieldConstants(edmx)
	file, err = os.Create(g.Package.FieldsConstants)
	if err != nil {
		return err
	}
	_, err = file.WriteString(contents)
	if err != nil {
		return err
	}

	return err
}
