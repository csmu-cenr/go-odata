package modelGenerator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ErrorMessage struct {
	Attempted string `json:"attempted" xml:"attempted"`
	Details   any    `json:"details" xml:"details"`
	Function  string `json:"function" xml:"function"`
}

func (e ErrorMessage) Error() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

type Fields struct {
	Extras      []string          `json:"extras"`
	Ignore      Ignore            `json:"ignore"`
	Json        JsonTags          `json:"json"`
	Mandatory   []Mandatory       `json:"mandatory"`
	Public      bool              `json:"public"` // Change a_field__name__ to AFieldName
	Pointers    bool              `json:"pointers"`
	ReadOnlyTag string            `json:"readOnlyTag"`
	Swap        map[string]string `json:"swap"`
}

type Ignore struct {
	StartsWith []string `json:"startsWith"`
	Contains   []string `json:"contains"`
	EndsWith   []string `json:"endsWith"`
	Equals     []string `json:"equals"`
}

type JsonTags struct {
	Tags      bool `json:"tags"`
	OmitEmpty bool `json:"omitempty"`
}

type Generator struct {
	ApiUrl          string              `json:"apiUrl"`
	Fields          Fields              `json:"fields"`
	Meta            bool                `json:"Meta"`
	Package         Package             `json:"package"`
	ReadOnly        bool                `json:"readOnly"`
	EnforceReadOnly map[string][]string `json:"enforceReadOnly"`
	IgnoreReadOnly  map[string][]string `json:"ignoreReadOnly"`
}

type Mandatory struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
	Valid    bool   `json:"valid"`
}

type Package struct {
	CreateDirectoryIfMissing bool   `json:"createDirectoryIfMissing"`
	Datasets                 string `json:"datasets"`
	Delete                   string `json:"delete"`
	DeleteWhere              string `json:"deleteWhere"`
	Directory                string `json:"directory"`
	Extras                   string `json:"extras"`
	FieldsConstants          string `json:"fieldsConstants"`
	FieldsPackageName        string `json:"fieldsPackageName"`
	Insert                   string `json:"insert"`
	IgnoreCollections        bool   `json:"ignoreCollections"`
	IgnoreNullableCheck      bool   `json:"ignoreNullableCheck"`
	Maps                     string `json:"maps"`
	Models                   string `json:"models"`
	Name                     string `json:"-"`
	OdataAlias               string `json:"odataAlias"`
	Save                     string `json:"save"`
	SaveByTableName          string `json:"saveByTableName"`
	Select                   string `json:"select"`
	SelectByTableName        string `json:"selectByTableName"`
	TablesConstants          string `json:"tablesConstants"`
	TablesPackageName        string
	Update                   string `json:"update"`
	UpdateWhere              string `json:"updateWhere"`
	WrapCollections          bool   `json:"wrapCollections"`
}

func New(path string) (Generator, error) {

	function := "New Generator"
	var generator Generator

	data, err := os.ReadFile(path) // just pass the file name
	if err != nil {
		e := ErrorMessage{
			Attempted: fmt.Sprintf(`os.ReadFile("%s")`, path),
			Function:  function,
			Details:   fmt.Sprintf(`Error: %+v`, err),
		}
		return generator, e
	}

	err = json.Unmarshal(data, &generator)
	if err != nil {
		e := ErrorMessage{
			Attempted: `json.Unmarshal(data, &generator)`,
			Function:  function,
			Details:   fmt.Sprintf(`Error: %+v`, err),
		}
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

	link := g.metadataUrl()
	schema, edmx, err := fetchEdmx(link)
	if err != nil {
		return err
	}

	packageName := filepath.Base(dirPath)
	g.Package.Name = packageName

	if g.Meta {
		xmlPath := filepath.Join(dirPath, fmt.Sprintf(`%s.xml`, packageName))
		fmt.Printf(`MetaXmlPath: %s`, xmlPath)
		g.SaveXMLSchema(xmlPath, schema)
	}

	code := g.CodeFromSchema(edmx)
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
	contents = g.FieldConstants(edmx)
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
