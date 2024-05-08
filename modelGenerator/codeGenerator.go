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

func (g *Generator) validPropertyName(property string) bool {
	for _, ignore := range g.Fields.Ignore.StartsWith {
		if strings.HasPrefix(property, ignore) {
			return false
		}
	}
	for _, ignore := range g.Fields.Ignore.Contains {
		if strings.Contains(property, ignore) {
			return false
		}
	}
	for _, ignore := range g.Fields.Ignore.EndsWith {
		if strings.HasSuffix(property, ignore) {
			return false
		}
	}
	for _, ignore := range g.Fields.Ignore.Equals {
		if property == ignore {
			return false
		}
	}
	return true
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
		include := g.validPropertyName(propertyKey)
		if include {
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
			goType := prop.goType()
			if value, ok := g.Fields.Swap[goType]; ok {
				goType = value
			}
			structString += fmt.Sprintf("\n\t%s %s%s\t%s", name, pointer, goType, jsonSupport)
		}
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

	code := map[string]string{}

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

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
	"github.com/Uffe-Code/go-odata/date"
	"time"
	
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

	selectByTableNameCode := fmt.Sprintf(`package %s

	import (
		"net/url"
		"strings"
	
		"github.com/Uffe-Code/go-odata/odataClient"
	)
	

	func SelectByTableName(tableName string, defaultFilter string, values url.Values, headers map[string]string, link string) ([]map[string]interface{}, error) {

		client := odataClient.New(link)
		for key, value := range headers {
			client.AddHeader(key, value)
		}
		options := client.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, values)

		switch tableName {

	`, packageName)

	datasets := fmt.Sprintf(`
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

	mapCode := fmt.Sprintf(`package %s

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
		for name := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})

		for _, name := range names {
			set := schema.EntitySets[name]
			modelCode += "\n" + g.generateModelStruct(set.getEntityType()) + "\n"
			modelCode += "\n" + generateModelDefinition(set) + "\n"
			mapCode += "\n" + generateMapFunctionCode(set) + "\n"
			selectByTableNameCode += "\n" + generateSelectByTableName(set, "client", "options") + "\n"
			selectCode += "\n" + generateSelectCode(set, "client", "odataClient") + "\n"
			datasets += "\n" + generateDataSet(set, "client", "odataClient") + "\n"
		}
	}

	selectByTableNameCode += `
	default:
		return nil, nil
	}
	return nil, nil
}
	`
	code[g.Package.Models] = modelCode
	code[g.Package.SelectByTableName] = selectByTableNameCode
	code[g.Package.Select] = selectCode
	code[g.Package.Datasets] = datasets
	code[g.Package.Maps] = mapCode

	packageLine := fmt.Sprintf("package %s", packageName)

	files, err := os.ReadDir(g.Package.Extras)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", g.Package.Extras, err)
		return code
	}

	for _, extra := range files {
		path := fmt.Sprintf("%s%s%s", g.Package.Extras, string(os.PathSeparator), extra.Name())
		extraCode, err := addPackageNameToExtra(packageLine, path)
		if err != nil {
			fmt.Printf("Issue with %s. Details: %s\n", extra, err)
		}
		code[filepath.Base(extra.Name())] = extraCode
	}

	return code
}

func (g *Generator) generateFieldConstants(dataService edmxDataServices) string {
	result := fmt.Sprintf("package %s\r", g.Package.FieldsPackageName)

	for _, schema := range dataService.Schemas {
		result = fmt.Sprintf("%s\rconst (", result)
		var names []string
		for name := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})
		for _, name := range names {

			set := schema.EntitySets[name]
			entityType := set.getEntityType()

			result = fmt.Sprintf("%s\r\r\t// %s", result, strings.Trim(strings.ToUpper(entityType.Name), "_"))

			propertyKeys := sortedCaseInsensitiveStringKeys(entityType.Properties)

			for _, property := range propertyKeys {
				if g.validPropertyName(property) {
					result = fmt.Sprintf("%s\r\t%s__%s\t=\t`%s`", result, strings.ToUpper(entityType.Name), strings.ToUpper(property), property)
				}
			}
		}
		result = fmt.Sprintf("%s\r\r)\r", result)
	}
	return result

}

func (g *Generator) generateTableConstants(dataService edmxDataServices) string {
	result := fmt.Sprintf("package %s\r", g.Package.TablesPackageName)

	for _, schema := range dataService.Schemas {
		result = fmt.Sprintf("%s\rconst (\r", result)
		var names []string
		for name := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})
		for _, name := range names {
			result = fmt.Sprintf("%s\r\t%s\t=\t`%s`", result, strings.ToUpper(name), name)
		}
		result = fmt.Sprintf("%s\r\r)\r", result)
	}
	return result

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

func generateMapFunctionCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	func {{publicName}}MapFunction(defaultFilter string, urlValues url.Values, root string, headers map[string]string, link string) (map[string]interface{}, error) {
		values := url.Values{}
		selectArgument := fmt.Sprintf("%sselect", root)
		values.Add("$select", urlValues.Get(selectArgument))
		filterArgument := fmt.Sprintf("%sfilter", root)
		values.Add("$filter", urlValues.Get(filterArgument))
		orderbyArgument := fmt.Sprintf("%sorderby", root)
		values.Add("$orderby", urlValues.Get(orderbyArgument))
		topArgument := fmt.Sprintf("%stop", root)
		values.Add("$top", urlValues.Get(topArgument))
		skipArgument := fmt.Sprintf("%sskip", root)
		values.Add("$skip", urlValues.Get(skipArgument))
		odataeditlinkArgument := fmt.Sprintf("%sodataeditlink", root)
		values.Add("$odataeditlink", urlValues.Get(odataeditlinkArgument))
		fields := strings.Split(urlValues.Get("$select"), ",")
		if urlValues.Get("$odataid") == "true" {
			fields = append(fields, "@odata.id")
		}
		if urlValues.Get("$odataeditlink") == "true" {
			fields = append(fields, "@odata.editLink")
		}
		model, err := {{publicName}}SelectSingle(defaultFilter, values, headers, link)
		if err != nil {
			return make(map[string]interface{}), err
		}
		data, err := odataClient.StructToMap(model, fields)
		if err != nil {
			return make(map[string]interface{}), err
		}
		return data, nil
	}

	func {{publicName}}ListMap(defaultFilter string, urlValues url.Values, root string, headers map[string]string, link string) ([]map[string]interface{}, error) {
		values := url.Values{}
		selectArgument := fmt.Sprintf("%sselect", root)
		values.Add("$select", urlValues.Get(selectArgument))
		filterArgument := fmt.Sprintf("%sfilter", root)
		values.Add("$filter", urlValues.Get(filterArgument))
		orderbyArgument := fmt.Sprintf("%sorderby", root)
		values.Add("$orderby", urlValues.Get(orderbyArgument))
		topArgument := fmt.Sprintf("%stop", root)
		values.Add("$top", urlValues.Get(topArgument))
		skipArgument := fmt.Sprintf("%sskip", root)
		values.Add("$skip", urlValues.Get(skipArgument))
		odataeditlinkArgument := fmt.Sprintf("%sodataeditlink", root)
		values.Add("$odataeditlink", urlValues.Get(odataeditlinkArgument))
		fields := strings.Split(urlValues.Get("$select"), ",")
		if urlValues.Get("$odataid") == "true" {
			fields = append(fields, "@odata.id")
		}
		if urlValues.Get("$odataeditlink") == "true" {
			fields = append(fields, "@odata.editLink")
		}
		models, err := {{publicName}}SelectList(defaultFilter, values, headers, link)
		if err != nil {
			return make([]map[string]interface{}, 0), err
		}
		data, err := odataClient.StructListToMapList(models, fields)
		if err != nil {
			return make([]map[string]interface{}, 0), err
		}
		return data, nil
	}
	`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	return result
}

func generateSelectCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	func {{publicName}}SelectSingle(defaultFilter string, urlValues url.Values, headers map[string]string, link string) ({{publicName}}, error) {
		models, err := {{publicName}}SelectList(defaultFilter, urlValues, headers, link)
		if err != nil {
			return {{publicName}}{}, err
		}
		if len(models) == 0 {
			filter := defaultFilter
			if urlValues.Get("$filter") != "" {
				filter = fmt.Sprintf("( %s ) and ( %s )", defaultFilter, urlValues.Get("$filter"))
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
		for range meta {
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

func generateSelectByTableName(set edmxEntitySet, client string, options string) string {

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
		if options.ODataId == "true" {
			fields = append(fields, "@odata.id")
		}
		if options.ODataEditLink == "true" {
			fields = append(fields, "@odata.editLink")
		}
		if options.ODataEtag == "true" {
			fields = append(fields, "@odata.etag")
		}
		if options.ODataReadLink == "true" {
			fields = append(fields, "@odata.readLink")
		}
		result := make([]map[string]interface{}, 0)
		for range meta {
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

	return result

}
