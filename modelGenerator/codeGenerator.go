package modelGenerator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	NOTHING = ``
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
	var structString strings.Builder
	structString.WriteString(fmt.Sprintf("type %s struct {", publicName))
	propertyKeys := sortedCaseInsensitiveStringKeys(entityType.Properties)

	jsonSupport := ""
	name := ""

	for _, extra := range g.Fields.Extras {
		fmt.Fprintf(&structString, "\n\t%s", extra)
	}
	structString.WriteString("\n")

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
			fmt.Fprintf(&structString, "\n\t%s %s%s\t%s", name, pointer, goType, jsonSupport)
		}
	}

	return structString.String() + "\n}"
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

	var deleteCode strings.Builder
	fmt.Fprintf(&deleteCode, `package %s
	
	import (
		"github.com/Uffe-Code/go-odata/odataClient"
	)
	`, packageName)

	var updateCode strings.Builder
	fmt.Fprintf(&updateCode, `package %s
	
import (
	"github.com/Uffe-Code/go-odata/odataClient"
)
`, packageName)

	var insertCode strings.Builder
	fmt.Fprintf(&insertCode, `package %s
	
import (
	"github.com/Uffe-Code/go-odata/odataClient"
)
`, packageName)

	var baseModelCode strings.Builder
	baseModelCode.WriteString(fmt.Sprintf(`package %s

import (
	"fmt"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
	"github.com/Uffe-Code/go-odata/date"
	
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

`, packageName, nilModel))

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
		{{options}} := client.ODataQueryOptions()
		options = {{options}}.ApplyArguments(defaultFilter, values)

		switch tableName {

	`, packageName)

	selectByTableNameOptions := `options`
	selectByTableNameCode = strings.ReplaceAll(selectByTableNameCode, "{{options}}", selectByTableNameOptions)

	var saveCode strings.Builder
	fmt.Fprintf(&saveCode, `package %s


`, packageName)

	var datasets strings.Builder
	fmt.Fprintf(&datasets, `
package %s
	
import (
	"github.com/Uffe-Code/go-odata/odataClient"
)

	`, packageName)

	var selectCode strings.Builder
	fmt.Fprintf(&selectCode, `
package %s

import (
	"fmt"
	"net/url"

	"github.com/Uffe-Code/go-odata/odataClient"
)

`, packageName)

	var mapCode strings.Builder
	fmt.Fprintf(&mapCode, `package %s

import (
	"fmt"
	"net/url"
	"strings"
	
	"github.com/Uffe-Code/go-odata/odataClient"
)

`, packageName)

	var singleFileMap map[string]string

	for _, schema := range dataService.Schemas {
		for _, enum := range schema.EnumTypes {
			baseModelCode.WriteString("\n" + generateEnumStruct(enum) + "\n")
		}

		for _, complexType := range schema.ComplexTypes {
			baseModelCode.WriteString("\n" + g.generateModelStruct(complexType) + "\n")
		}

		var names []string
		for name := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})

		if g.OutputMode.IsSingle() {
			modelBase := fmt.Sprintf(`package %s
			
import (
	"fmt"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
	"github.com/Uffe-Code/go-odata/date"
	
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

			base := fmt.Sprintf(`package %s
			
import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
)

		`, packageName)
			singleFileMap = map[string]string{}
			singleFileMap[`modelDefinition`] = modelBase
			for _, name := range names {
				singleFileMap[name] = base
			}
		}

		for _, name := range names {
			set := schema.EntitySets[name]
			modelCode := ``
			if g.OutputMode.IsSingle() {
				modelCode, _ = singleFileMap[name]
				modelCode += g.generateModelStruct(set.getEntityType()) + "\n"
				modelCode += "\n" + generateModelDefinition(set) + "\n"
				mapCode.Reset()
				selectCode.Reset()
				datasets.Reset()
				updateCode.Reset()
				insertCode.Reset()
				saveCode.Reset()
				deleteCode.Reset()
			}
			if g.OutputMode.IsOmnibus() {
				baseModelCode.WriteString("\n" + g.generateModelStruct(set.getEntityType()) + "\n")
				baseModelCode.WriteString("\n" + generateModelDefinition(set) + "\n")
			}
			mapCode.WriteString("\n" + generateMapFunctionCode(set) + "\n")
			selectByTableNameCode += "\n" + generateSelectByTableName(set, "client", selectByTableNameOptions) + "\n"
			selectCode.WriteString("\n" + generateSelectCode(set, "client", "odataClient") + "\n")
			datasets.WriteString("\n" + generateDataSet(set, "client", "odataClient") + "\n")
			updateCode.WriteString("\n" + generateUpdateCode(set, "client", "odataClient") + "\n")
			insertCode.WriteString("\n" + generateInsertCode(set, "client", "odataClient") + "\n")
			saveCode.WriteString("\n" + generateSaveCode(set) + "\n")
			deleteCode.WriteString("\n" + generatDeleteCode(set, "client", "odataClient") + "\n")
			if g.OutputMode.IsSingle() {
				sections := []string{
					datasets.String(),
					deleteCode.String(),
					insertCode.String(),
					mapCode.String(),
					saveCode.String(),
					selectCode.String(),
					updateCode.String(),
				}
				for _, section := range sections {
					if strings.TrimSpace(section) == "" {
						continue
					}
					modelCode += "\n\n" + strings.TrimSpace(section)
				}
				singleFileMap[name] = modelCode
			}
		}
	}

	selectByTableNameCode += `
	default:
		return nil, nil
	}
}
	`

	if g.Package.SelectByTableName != NOTHING {
		code[g.Package.SelectByTableName] = selectByTableNameCode
	}
	if g.OutputMode.IsOmnibus() {
		if g.Package.Models != NOTHING {
			code[g.Package.Models] = baseModelCode.String()
		}
		if g.Package.Select != NOTHING {
			code[g.Package.Select] = selectCode.String()
		}
		if g.Package.Datasets != NOTHING {
			code[g.Package.Datasets] = datasets.String()
		}
		if g.Package.Maps != NOTHING {
			code[g.Package.Maps] = mapCode.String()
		}
		if g.Package.Update != NOTHING {
			code[g.Package.Update] = updateCode.String()
		}
		if g.Package.Insert != NOTHING {
			code[g.Package.Insert] = insertCode.String()
		}
		if g.Package.Save != NOTHING {
			code[g.Package.Save] = saveCode.String()
		}
		if g.Package.Delete != NOTHING {
			code[g.Package.Delete] = deleteCode.String()
		}
	}
	if g.OutputMode.IsSingle() {
		for name, contents := range singleFileMap {
			code[fmt.Sprintf(`%s.go`, name)] = contents
		}
	}

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

func generateDataSet(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `func {{publicName}}DataSet(headers map[string]string, link string) odataClient.ODataDataSet[{{publicName}}, odataClient.ODataModelDefinition[{{publicName}}]] {
		{{client}} := {{packageName}}.New(link)
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

func generatDeleteCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)

	result := `func (o *{{publicName}}) Delete(headers map[string]string, link string) error {

	{{client}} := {{packageName}}.New(link)
	for key, value := range headers {
		{{client}}.AddHeader(key, value)
	}
	
	collection := New{{publicName}}Collection({{client}})
	dataset := collection.DataSet()
	
	return dataset.Delete(o.ODataEditLink)
}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	result = strings.ReplaceAll(result, "{{client}}", client)

	return result

}

func (g *Generator) generateFieldConstants(dataService edmxDataServices) string {
	result := fmt.Sprintf("package %s\n", g.Package.FieldsPackageName)

	for _, schema := range dataService.Schemas {
		result = fmt.Sprintf("%s\nconst (", result)
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

			result = fmt.Sprintf("%s\n\n\t// %s", result, strings.Trim(strings.ToUpper(entityType.Name), "_"))

			propertyKeys := sortedCaseInsensitiveStringKeys(entityType.Properties)

			for _, property := range propertyKeys {
				if g.validPropertyName(property) {
					result = fmt.Sprintf("%s\n\t%s__%s\t=\t`%s`", result, strings.Trim(strings.ToUpper(entityType.Name), "_"), strings.ToUpper(property), property)
				}
			}
		}
		result = fmt.Sprintf("%s\n\n)\n", result)
	}
	result = strings.Trim(result, "\n")
	return result

}

func generateInsertCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)

	result := `func (o *{{publicName}}) Insert(headers map[string]string, link string, fieldsToUpdate []string) ({{publicName}}, error) {

	{{client}} := {{packageName}}.New(link)
	for key, value := range headers {
		{{client}}.AddHeader(key, value)
	}
	
	collection := New{{publicName}}Collection({{client}})
	dataset := collection.DataSet()
	
	return dataset.Insert(*o, fieldsToUpdate)
}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	result = strings.ReplaceAll(result, "{{client}}", client)

	return result

}

func generateSaveCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `func (o *{{publicName}}) Save(headers map[string]string, link string, fieldsToUpdate []string) ({{publicName}}, error) {

	if o.ODataEditLink == NOTHING {
		return o.Insert(headers, link, fieldsToUpdate)
	}
	return o.Update(headers, link, fieldsToUpdate)
	
}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)

	return result
}

func (g *Generator) generateTableConstants(dataService edmxDataServices) string {
	result := fmt.Sprintf("package %s\r", g.Package.TablesPackageName)

	for _, schema := range dataService.Schemas {
		result = fmt.Sprintf("%s\nconst (\n", result)
		var names []string
		for name := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})
		for _, name := range names {
			result = fmt.Sprintf("%s\n\t%s\t=\t`%s`", result, strings.Trim(strings.ToUpper(name), "_"), name)
		}
		result = fmt.Sprintf("%s\n)", result)
	}
	result = strings.Trim(result, "\n")
	return result

}

func generateUpdateCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)

	result := `func (o *{{publicName}}) Update(headers map[string]string, link string, fieldsToUpdate []string) ({{publicName}}, error) {

		{{client}} := {{packageName}}.New(link)
		for key, value := range headers {
			{{client}}.AddHeader(key, value)
		}
	
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
	
		return dataset.Update(o.ODataEditLink, *o, fieldsToUpdate)
	}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	result = strings.ReplaceAll(result, "{{client}}", client)

	return result

}

func generateMapFunctionCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	func {{publicName}}MapFunction(defaultFilter string, urlValues url.Values, root string, headers map[string]string, link string) (map[string]interface{}, error) {
		values := url.Values{}
		selectArgument := fmt.Sprintf("%sselect", root)
		values.Add(SELECT, urlValues.Get(selectArgument))
		filterArgument := fmt.Sprintf("%sfilter", root)
		values.Add(FILTER, urlValues.Get(filterArgument))
		orderbyArgument := fmt.Sprintf("%sorderby", root)
		values.Add(ORDERBY, urlValues.Get(orderbyArgument))
		topArgument := fmt.Sprintf("%stop", root)
		values.Add(TOP, urlValues.Get(topArgument))
		skipArgument := fmt.Sprintf("%sskip", root)
		values.Add(SKIP, urlValues.Get(skipArgument))
		odataeditlinkArgument := fmt.Sprintf("%sodataeditlink", root)
		values.Add(ODATAEDITLINK, urlValues.Get(odataeditlinkArgument))
		fields := strings.Split(urlValues.Get(SELECT), ",")
		if urlValues.Get("$odataid") == "true" {
			fields = append(fields, "@odata.id")
		}
		if urlValues.Get(ODATAEDITLINK) == "true" {
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
		values.Add(SELECT, urlValues.Get(selectArgument))
		filterArgument := fmt.Sprintf("%sfilter", root)
		values.Add(FILTER, urlValues.Get(filterArgument))
		orderbyArgument := fmt.Sprintf("%sorderby", root)
		values.Add(ORDERBY, urlValues.Get(orderbyArgument))
		topArgument := fmt.Sprintf("%stop", root)
		values.Add(TOP, urlValues.Get(topArgument))
		skipArgument := fmt.Sprintf("%sskip", root)
		values.Add(SKIP, urlValues.Get(skipArgument))
		odataeditlinkArgument := fmt.Sprintf("%sodataeditlink", root)
		values.Add(ODATAEDITLINK, urlValues.Get(odataeditlinkArgument))
		fields := strings.Split(urlValues.Get(SELECT), ",")
		if urlValues.Get("$odataid") == "true" {
			fields = append(fields, "@odata.id")
		}
		if urlValues.Get(ODATAEDITLINK) == "true" {
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
			if urlValues.Get(FILTER) != "" {
				filter = fmt.Sprintf("( %s ) and ( %s )", defaultFilter, urlValues.Get(FILTER))
			}
			return {{publicName}}{}, NilModel{Model: "{{publicName}}", Filter: filter}
		}
		return models[0], nil
	}
	
	func {{publicName}}SelectList(defaultFilter string, urlValues url.Values, headers map[string]string, link string) ([]{{publicName}}, error) {

		{{client}} := {{packageName}}.New(link)
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
