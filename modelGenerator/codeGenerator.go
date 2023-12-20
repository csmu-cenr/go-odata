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

	modelPackages := ""
	for _, modelPackage := range g.Imports.Models {
		modelPackages += fmt.Sprintf("\t\"%s\"\n", modelPackage)
	}
	modelCode := fmt.Sprintf(`package %s

import (
%s
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

`, packageName, modelPackages)

	resultPackages := ""
	for _, resultPackage := range g.Imports.Results {
		resultPackages += fmt.Sprintf("\t\"%s\"\n", resultPackage)
	}
	resultsAsBytes := fmt.Sprintf(`package %s

	import (
	%s
	)

	func ByteResults(tableName string, defaultFilter string, urlValues url.Values, headers map[string]string, url string) ([]byte, error) {

		client := odataClient.New(url)
		for key, value := range headers {
			client.AddHeader(key, value)
		}
		options := client.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, urlValues)
	
		switch tableName {

	`, packageName, resultPackages)

	structDataSets := fmt.Sprintf(`
package %s
	
import (
	"github.com/Uffe-Code/go-odata/odataClient"
)

	`, packageName)

	structResults := fmt.Sprintf(`
package %s

import (
	"net/url"

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
			resultsAsBytes += "\n" + generateByteResults(set, "client", "options", "odataClient") + "\n"
			structResults += "\n" + generateTypeResult(set, "client", "odataClient") + "\n"
			structDataSets += "\n" + generateDataSet(set, "client", "odataClient") + "\n"
		}
	}

	resultsAsBytes += `
	default:
		return nil, nil
	}
	return nil, nil
}
	`
	code[g.Package.Models] = modelCode
	code[g.Package.ByteResults] = resultsAsBytes
	code[g.Package.StructResults] = structResults
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

func generateTypeResult(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `func {{publicName}}Results(defaultFilter string, urlValues url.Values, headers map[string]string, url string) ([]{{publicName}}, error) {

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

func generateByteResults(set edmxEntitySet, client string, options string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	case "{{databaseName}}":
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
		meta, data, errs := dataset.List({{options}})
		for err := range errs {
			return nil, err
		}
		fields := strings.Split({{options}}.Select, ",")
		for result := range meta {
			for model := range data {
				if len(fields) > 0 {
					modelJson, _ := json.Marshal(model)
					modelText, _ := {{packageName}}.SelectFields(string(modelJson), fields)
					var modelData map[string]interface{}
					json.Unmarshal([]byte(modelText), &modelData)
					result.Value = append(result.Value, modelData)
				} else {
					result.Value = append(result.Value, model)
				}
			}
			if result.Value == nil {
				result.Value = []any{}
			}
			body, _ := json.Marshal(result)
			return body, nil
		}

`

	result = strings.ReplaceAll(result, "{{databaseName}}", set.Name)
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{options}}", options)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)

	return result

}
