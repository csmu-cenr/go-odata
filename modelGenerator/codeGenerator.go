package modelGenerator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func addPackageNameToExtra(packageLine string, extraPath string) (string, error) {

	var readPath string
	firstChar := extraPath[0:1]
	switch firstChar {
	case "/":
		// absolute
		readPath = extraPath
	default:
		// relative
		appDirPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return "", err
		}
		readPath = filepath.Join(appDirPath, string(os.PathSeparator), extraPath)
	}

	bytes, err := os.ReadFile(readPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(bytes), "\n")
	lines[0] = packageLine
	return strings.Join(lines, "\n"), nil
}

func publicAttribute(property string) string {

	// Split the string by underscores
	segments := strings.Split(property, "_")

	// Capitalize the first letter of each segment
	for i := range segments {
		segments[i] = strings.Title(segments[i])
	}

	// Join the segments back together
	result := strings.Join(segments, "")

	return result
}

func (g *Generator) generateModelStruct(entityType edmxEntityType) string {

	publicName := publicAttribute(entityType.Name)
	structString := fmt.Sprintf("type %s struct {", publicName)
	propertyKeys := sortedCaseInsensitiveStringKeys(entityType.Properties)

	jsonSupport := ""
	name := ""

	for _, extra := range g.Fields.Extras {
		structString += fmt.Sprintf("\n\t%s", extra)
	}
	structString += "\n"

	for _, propertyKey := range propertyKeys {
		prop := entityType.Properties[propertyKey]
		name = prop.Name
		if g.Fields.Public {
			name = publicAttribute(name)
		}
		if g.Fields.Json.Tags {
			jsonSupport = prop.Name
			if g.Fields.Json.OmitEmpty {
				jsonSupport += ",omitempty"
			}
			jsonSupport = fmt.Sprintf("`json:\"%s\"`", jsonSupport)
		}
		pointer := ""
		if g.Fields.Pointers {
			pointer = "*"
		}
		structString += fmt.Sprintf("\n\t%s %s%s\t%s", name, pointer, prop.goType(), jsonSupport)
	}

	return structString + "\n}"
}

func generateModelDefinition(set edmxEntitySet) string {
	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	return fmt.Sprintf(`//goland:noinspection GoUnusedExportedFunction
func New%sCollection(wrapper odataClient.Wrapper) odataClient.ODataModelCollection[%s] {
	return modelDefinition[%s]{client: wrapper.ODataClient(), name: "%s", url: "%s"}
}`, publicName, publicName, publicName, publicName, set.Name)
}

func generateEnumStruct(enum edmxEnumType) string {
	stringValues := map[string]string{}
	intValues := map[int64]string{}
	isIntValues := true

	for _, member := range enum.Members {
		stringValues[member.Name] = member.Value
		i, err := strconv.ParseInt(member.Value, 10, 64)
		if err != nil {
			isIntValues = false
		} else {
			intValues[i] = member.Name
		}
	}

	goType := "string"
	if isIntValues {
		goType = "int64"
	}
	goString := fmt.Sprintf(`type %s %s

const (`, enum.Name, goType)

	if isIntValues {
		intKeys := sortedKeys(intValues)
		for _, i := range intKeys {
			key := intValues[i]
			goString += fmt.Sprintf("\n\t%s %s = %d", key, enum.Name, i)
		}
	} else {
		stringKeys := sortedCaseInsensitiveStringKeys(stringValues)
		for _, key := range stringKeys {
			str := stringValues[key]
			goString += fmt.Sprintf("\n\t%s %s = \"%s\"", key, enum.Name, str)
		}
	}

	return goString + "\n)"
}

func (g *Generator) generateCodeFromSchema(packageName string, dataService edmxDataServices) map[string]string {

	var code map[string]string
	code = make(map[string]string)

	nilModel := `
	
	type NilModel struct {
		Model  string
		Filter string
	}
	
	func (e NilModel) Error() string {
		return fmt.Sprintf(" No matching %s found for %s.", e.Model, e.Filter)
	}
	`

	modelCode := fmt.Sprintf(`package %s

import (
	"fmt"
	"time"

	"csmu.balance-infosystems.com/depot-maestro/date"
	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
)

type modelDefinition[T any] struct { client odataClient.ODataClient; name string; url string }

func (md modelDefinition[T]) Name() string {
	return md.name
}

func (md modelDefinition[T]) Url() string {
	return md.url
}

func (md modelDefinition[T]) DataSet() odataClient.ODataDataSet[T, odataClient.ODataModelDefinition[T]] {
	return odataClient.NewDataSet[T](md.client, md)
}

%s

`, packageName, nilModel)

	mappedByTableNameCode := fmt.Sprintf(`package %s

	import (
		"net/url"
		"strings"
	
		"github.com/Uffe-Code/go-odata/odataClient"
	)
	

	func MappedInterfaceListByTableName(tableName string, defaultFilter string, values url.Values, headers map[string]string, link string) ([]map[string]interface{}, error) {

		client := odataClient.New(link)
		for key, value := range headers {
			client.AddHeader(key, value)
		}
		options := client.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, values)

		switch tableName {

	`, packageName)

	structDataSets := fmt.Sprintf(`
package %s
	
import (
	"github.com/Uffe-Code/go-odata/odataClient"
)

	`, packageName)

	selectCode := fmt.Sprintf(`
package %s

import (
	"fmt"
	"net/url"

	"github.com/Uffe-Code/go-odata/odataClient"
)

`, packageName)

	maps := fmt.Sprintf(`%s

import (
	"fmt"
	"net/url"
	"strings"
	
	"github.com/Uffe-Code/go-odata/odataClient"
)

`, packageName)

	for _, schema := range dataService.Schemas {
		for _, enum := range schema.EnumTypes {
			modelCode += "\n" + generateEnumStruct(enum) + "\n"
		}

		for _, complexType := range schema.ComplexTypes {
			modelCode += "\n" + g.generateModelStruct(complexType) + "\n"
		}

		var names []string
		for name, _ := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})

		for _, name := range names {
			set := schema.EntitySets[name]
			modelCode += "\n" + g.generateModelStruct(set.getEntityType()) + "\n"
			modelCode += "\n" + generateModelDefinition(set) + "\n"
			maps += "\n" + ""
			mappedByTableNameCode += "\n" + generateMappedInterfacesByTableName(set, "client", "options", "odataClient") + "\n"
			selectCode += "\n" + generateSelectCode(set, "client", "odataClient") + "\n"
			structDataSets += "\n" + generateDataSet(set, "client", "odataClient") + "\n"
		}
	}

	mappedByTableNameCode += `
	default:
		return nil, nil
	}
	return nil, nil
}
	`
	code[g.Package.Models] = modelCode
	code[g.Package.MappedInterfaceListByTableName] = mappedByTableNameCode
	code[g.Package.Select] = selectCode
	code[g.Package.StructDataSets] = structDataSets
	packageLine := fmt.Sprintf("package %s", packageName)
	for _, extra := range g.Package.Extras {
		extraCode, err := addPackageNameToExtra(packageLine, extra)
		if err != nil {
			fmt.Printf("Issue with %s. Details: %s\n", extra, err)
		}
		code[filepath.Base(extra)] = extraCode
	}
	return code
}

func generateDataSet(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `func {{publicName}}DataSet(headers map[string]string, url string) odataClient.ODataDataSet[{{publicName}}, odataClient.ODataModelDefinition[{{publicName}}]] {
		{{client}} := {{packageName}}.New(url)
		for key, value := range headers {
			{{client}}.AddHeader(key, value)
		}
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
		return dataset
	}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	return result
}

func generateSelectCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	func {{publicName}}SelectSingle(defaultFilter string, values url.Values, headers map[string]string, link string) ({{publicName}}, error) {
		models, err := {{publicName}}SelectList(defaultFilter, values, headers, link)
		if err != nil {
			return {{publicName}}{}, err
		}
		if len(models) == 0 {
			filter := defaultFilter
			if values.Get("$filter") != "" {
				filter = fmt.Sprintf("( %s ) and ( %s )", defaultFilter, values.Get("$filter"))
			}
			return {{publicName}}{}, NilModel{Model: "{{publicName}}", Filter: filter}
		}
		return models[0], nil
	}
	
	func {{publicName}}SelectList(defaultFilter string, urlValues url.Values, headers map[string]string, url string) ([]{{publicName}}, error) {

		{{client}} := {{packageName}}.New(url)
		for key, value := range headers {
			{{client}}.AddHeader(key, value)
		}
		options := {{client}}.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, urlValues)
	
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
		meta, data, errs := dataset.List(options)
	
		models := []{{publicName}}{}
		for err := range errs {
			return nil, err
		}
		for _ = range meta {
			for model := range data {
				models = append(models, model)
			}
		}
	
		return models, nil
	}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	return result
}

func generateMappedInterfacesByTableName(set edmxEntitySet, client string, options string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	case "{{databaseName}}":
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
		meta, data, errs := dataset.List(options)
		for err := range errs {
			return nil, err
		}
		fields := strings.Split(options.Select, ",")
		result := make([]map[string]interface{}, 0)
		for _ = range meta {
			for model := range data {
				data, err := odataClient.StructToMap(model, fields)
				if err != nil {
					return result, err
				}
				result = append(result, data)
			}
		}
		return result, nil

`

	result = strings.ReplaceAll(result, "{{databaseName}}", set.Name)
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{options}}", options)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)

	return result

}
