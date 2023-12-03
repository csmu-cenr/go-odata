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

	structString := fmt.Sprintf("type %s struct {", entityType.Name)
	propertyKeys := sortedCaseInsensitiveStringKeys(entityType.Properties)

	pointer := ""
	if g.Fields.Pointers {
		pointer = "*"
	}

	jsonSupport := ""
	name := ""

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
		structString += fmt.Sprintf("\n\t%s %s%s\t%s", name, pointer, prop.goType(), jsonSupport)
	}

	return structString + "\n}"
}

func generateModelDefinition(set edmxEntitySet) string {
	entityType := set.getEntityType()

	return fmt.Sprintf(`//goland:noinspection GoUnusedExportedFunction
func New%sCollection(wrapper odataClient.Wrapper) odataClient.ODataModelCollection[%s] {
	return modelDefinition[%s]{client: wrapper.ODataClient(), name: "%s", url: "%s"}
}`, entityType.Name, entityType.Name, entityType.Name, entityType.Name, set.Name)
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
	resultCode := fmt.Sprintf(`package %s

	import (
	%s
	)

	func Results(tableName string, defaultFilter string, urlValues url.Values, headers map[string]string, url string) ([]byte, error) {

		odataClient := odataClient.New(url)
		for key, value := range headers {
			odataClient.AddHeader(key, value)
		}
		queryOptions := odataClient.ODataQueryOptions()
		queryOptions = queryOptions.ApplyArguments(defaultFilter, urlValues)
	
		switch tableName {

	`, packageName, resultPackages)

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
			return strings.ToLower(names[i]) < strings.ToLower(names[j])
		})

		for _, name := range names {
			set := schema.EntitySets[name]
			modelCode += "\n" + g.generateModelStruct(set.getEntityType()) + "\n"
			modelCode += "\n" + generateModelDefinition(set) + "\n"
			resultCode += "\n" + generateResult(set) + "\n"
		}
	}

	resultCode += `
	default:
		return nil, nil
	}
	return nil, nil
}
	`
	code[g.Package.Models] = modelCode
	code[g.Package.Results] = resultCode
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

func generateResult(set edmxEntitySet) string {

	entityType := set.getEntityType()

	return fmt.Sprintf(`

	case "%s":
		collection := New%sCollection(odataClient)
		dataset := collection.DataSet()
		meta, data, errs := dataset.List(queryOptions)
		for err := range errs {
			return nil, err
		}
		for result := range meta {
			for model := range data {
				result.Value = append(result.Value, model)
			}
			if result.Value == nil {
				result.Value = []any{}
			}
			body, _ := json.Marshal(result)
			return body, nil
		}

	`, set.Name, entityType.Name)

}
