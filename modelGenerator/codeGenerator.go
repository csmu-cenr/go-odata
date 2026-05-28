package modelGenerator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	FALSE = `false`
	TRUE  = `true`
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
	return snakeCaseToTitleCase(property)
}

func generateModelDefinition(set edmxEntitySet) string {
	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	return fmt.Sprintf(`//goland:noinspection GoUnusedExportedFunction
func New%sCollection(wrapper odataClient.Wrapper) odataClient.ODataModelCollection[%s] {
	return modelDefinition[%s]{client: wrapper.ODataClient(), name: "%s", url: "%s"}
}`, publicName, publicName, publicName, publicName, set.Name)
}

func (g *Generator) generateCodeFromSchema(packageName string, dataService edmxDataServices) map[string]string {

	code := map[string]string{}
	var singleFileMap map[string]string

	nilModel := `
	type NilModel struct {
		Model  string
		Filter string
	}
		
	func (e NilModel) Error() string {
		return fmt.Sprintf(" No matching %s found for %s.", e.Model, e.Filter)
	}
	`

	recordIsOutOfDate := `
	type RecordIsOutOfDate struct {
		Model  string      
		Uuid   string      
		Id     float64     
		Record interface{} 
	}

	func (r RecordIsOutOfDate) Error() string {
		data, err := json.Marshal(r)
		if err != nil {
			return err.Error()
		}
		return string(data)
	}
	`

	customErrors := fmt.Sprintf(`
		%s
		%s
		`,
		nilModel,
		recordIsOutOfDate,
	)

	var deleteCode strings.Builder
	deleteCode.WriteString(fmt.Sprintf(`package %s
	
	import (
		"github.com/Uffe-Code/go-odata/odataClient"
	)
	`, packageName))

	var updateCode strings.Builder
	updateCode.WriteString(fmt.Sprintf(`package %s
	
import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
)
`, packageName))

	var insertCode strings.Builder
	insertCode.WriteString(fmt.Sprintf(`package %s
	
import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
)
`, packageName))

	baseModelCode := fmt.Sprintf(`package %s

import (
	"encoding/json"
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

`, packageName, customErrors)

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
	saveCode.WriteString(fmt.Sprintf(`package %s

	import (
		"encoding/json"
		"fmt"
		"net/http"
		"net/url"
		"reflect"
			
		"github.com/Uffe-Code/go-nullable/nullable"
	)

`, packageName))

	var datasets strings.Builder
	datasets.WriteString(fmt.Sprintf(`
package %s
	
import (
	"github.com/Uffe-Code/go-odata/odataClient"
)

	`, packageName))

	var selectCode strings.Builder
	selectCode.WriteString(fmt.Sprintf(`
package %s

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
)

`, packageName))

	var mapCode strings.Builder
	mapCode.WriteString(fmt.Sprintf(`package %s

import (
	"fmt"
	"net/url"
	"strings"
	
	"github.com/Uffe-Code/go-odata/odataClient"
)

`, packageName))

	for _, schema := range dataService.Schemas {
		for _, enum := range schema.EnumTypes {
			baseModelCode += "\n" + generateEnumStruct(enum) + "\n"
		}

		for _, complexType := range schema.ComplexTypes {
			baseModelCode += "\n" + g.generateModelStruct(complexType) + "\n"
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
	"encoding/json"
	"fmt"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
	"github.com/Uffe-Code/go-odata/date"
	
)

%s

`, packageName, customErrors)

			base := fmt.Sprintf(`package %s
			
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/Uffe-Code/go-nullable/nullable"
	"github.com/Uffe-Code/go-odata/odataClient"
	"github.com/Uffe-Code/go-odata/date"
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

				// Reset builders for each entity in single mode to avoid accumulation
				mapCode.Reset()
				selectCode.Reset()
				datasets.Reset()
				updateCode.Reset()
				insertCode.Reset()
				saveCode.Reset()
				deleteCode.Reset()
			}
			if g.OutputMode.IsOmnibus() {
				baseModelCode += "\n" + g.generateModelStruct(set.getEntityType()) + "\n"
				baseModelCode += "\n" + generateModelDefinition(set) + "\n"
			}
			mapCode.WriteString("\n" + generateMapFunctionCode(set) + "\n")
			selectByTableNameCode += "\n" + generateSelectByTableName(set, "client", selectByTableNameOptions) + "\n"
			selectCode.WriteString("\n" + generateSelectCode(set, "client", "odataClient") + "\n")
			datasets.WriteString("\n" + generateDataSet(set, "client", "odataClient") + "\n")
			updateCode.WriteString("\n" + generateUpdateCode(set, "client", "odataClient") + "\n")
			insertCode.WriteString("\n" + generateInsertCode(set, "client", "odataClient") + "\n")
			saveCode.WriteString("\n" + generateSaveCode(set) + "\n")
			deleteCode.WriteString("\n" + generateDeleteCode(set, "client", "odataClient") + "\n")
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
					// Remove package and import lines from sections before appending
					lines := strings.Split(section, "\n")
					var filteredLines []string
					for _, line := range lines {
						trimmed := strings.TrimSpace(line)
						if strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "import (") || strings.HasPrefix(trimmed, ")") || trimmed == "" {
							// Basic heuristic to skip imports in single mode sections
							// This is a bit fragile but matches the current structure
							if strings.Contains(trimmed, "\"") && (strings.Contains(trimmed, "github.com") || strings.Contains(trimmed, "net/") || strings.Contains(trimmed, "fmt") || strings.Contains(trimmed, "reflect")) {
								continue
							}
							if trimmed == ")" || trimmed == "import (" || strings.HasPrefix(trimmed, "package ") {
								continue
							}
						}
						filteredLines = append(filteredLines, line)
					}
					modelCode += "\n\n" + strings.Join(filteredLines, "\n")
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
	if g.Package.SelectByTableName != "" {
		code[g.Package.SelectByTableName] = selectByTableNameCode
	}
	if g.OutputMode.IsOmnibus() {
		if g.Package.Models != "" {
			code[g.Package.Models] = baseModelCode
		}
		if g.Package.Select != "" {
			code[g.Package.Select] = selectCode.String()
		}
		if g.Package.Datasets != "" {
			code[g.Package.Datasets] = datasets.String()
		}
		if g.Package.Maps != "" {
			code[g.Package.Maps] = mapCode.String()
		}
		if g.Package.Update != "" {
			code[g.Package.Update] = updateCode.String()
		}
		if g.Package.Insert != "" {
			code[g.Package.Insert] = insertCode.String()
		}
		if g.Package.Save != "" {
			code[g.Package.Save] = saveCode.String()
		}
		if g.Package.Delete != "" {
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
				jsonSupport = fmt.Sprintf("`json:\"%s\"", jsonSupport)
			}
			tags := ""
			if prop.Annotations != nil {
				annotations := map[string]map[string]string{}
				for _, annotation := range *prop.Annotations {
					if annotation.EnumMember == nil {
						head, tail := HeadAndTailText(annotation.Term, ".", 1)
						mapped := map[string]string{}
						switch {
						case annotation.Bool != "":
							mapped[tail] = annotation.Bool
						case annotation.String != "":
							mapped[tail] = annotation.String
						case annotation.Int != "":
							mapped[tail] = annotation.Int
						default:
							mapped[tail] = ""
						}
						annotations[head] = mapped
					} else {
						mapped := map[string]string{}
						for _, enum := range *annotation.EnumMember {
							mapped[enum] = ""
						}
						annotations[annotation.Term] = mapped
					}
				}
				for name, term := range annotations {
					tag := fmt.Sprintf(` %s:"`, name)
					values := []string{}
					for k, v := range term {
						if v == "" {
							values = append(values, k)
						} else {
							values = append(values, fmt.Sprintf(`%s=%s`, k, v))
						}
					}
					tag = fmt.Sprintf(`%s%s"`, tag, strings.Join(values, ","))
					tags = fmt.Sprintf(`%s%s`, tags, tag)
				}
			}
			if jsonSupport != "" {
				jsonSupport = fmt.Sprintf("%s%s`", jsonSupport, tags)
			}
			pointer := ""
			if g.Fields.Pointers {
				pointer = "*"
			}
			goType := prop.goType(
				g.Package.IgnoreNullableCheck,
				g.Package.IgnoreCollections,
				g.Package.WrapCollections)
			if value, ok := g.Fields.Swap[goType]; ok {
				goType = value
			}
			structString += fmt.Sprintf("\n\t%s %s%s\t%s", name, pointer, goType, jsonSupport)
		}
	}
	structString += "\n}\n\n"
	structString += fmt.Sprintf("type\t%sAlias\t[]%s\n", publicName, publicName)
	structString += fmt.Sprintf("type\t%sAsc\t[]%s\n", publicName, publicName)
	structString += fmt.Sprintf("type\t%sDesc\t[]%s\n", publicName, publicName)
	structString += fmt.Sprintf(
		"\n\ntype Meta%s struct {\n"+
			"\tAction     string                   `json:\"action,omitempty\"`\n"+
			"\tFieldData  %s                       `json:\"fieldData\"`\n"+
			"\tModel      string                   `json:\"model,omitempty\"`\n"+
			"\tModId      string                   `json:\"modId,omitempty\"`\n"+
			"\tPortalData map[string][]interface{} `json:\"portalData,omitempty\"`\n"+
			"\tRecordId   string                   `json:\"recordId,omitempty\"`\n"+
			"\tStaffId    int                      `json:\"staffId\"`\n"+
			"}\n\n",
		publicName, publicName,
	)
	return structString
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

// validPropertyName removes fields based on the ignore section of the config.json file.
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

func generateDeleteCode(set edmxEntitySet, client string, packageName string) string {

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

func generateInsertCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	instance := snakeCaseToCamelCase(publicName)

	result := `func ({{type}} *{{publicName}}) Insert(headers map[string]string, link string) ({{publicName}}, error) {

	{{client}} := {{packageName}}.New(link)
	for key, value := range headers {
		{{client}}.AddHeader(key, value)
	}
	
	collection := New{{publicName}}Collection({{client}})
	dataset := collection.DataSet()
	modifiedFields := nullable.GetModifiedTags({{type}})
	selectedFields := nullable.GetSelectedTags({{type}},false)

	result, err := dataset.Insert(*{{type}}, modifiedFields)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "{{instance}}.Insert",
			Details:    fmt.Sprintf("%+v", err),
			ErrorNo:    http.StatusInternalServerError,
			InnerError: err,
			Message:    "unexpected error",
		}
		return result, m
	}
	err = nullable.SetSelectedBooleanFields(reflect.ValueOf(&result), selectedFields, true, true)
	if err != nil {
		m := ErrorMessage{
			Attempted:  SET_NULLABLE_BOOLEAN_FIELDS,
			Details:    fmt.Sprintf("%+v", err),
			ErrorNo:    http.StatusInternalServerError,
			InnerError: err,
			Message:    "unexpected error",
		}
		return result, m
	}
	
	return result, err
}`

	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{instance}}", instance)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

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

func generateSaveCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `// {{publicName}}.Save determines whether to save or creates a record at the link provided the authentication provided in headers is valid.
	func ({{type}} *{{publicName}}) Save(headers map[string]string, link string, values url.Values) ({{publicName}}, error) {

	values.Del(FILTER)

	// Check if anything has changed. Bounce out.
	if !{{type}}.Modified(){
		return *{{type}}, nil
	} 

	// Check if {{publicName}} has an edit link. If not try to created.
	if {{type}}.ODataEditLink == "" {
		return {{type}}.Insert(headers, link)
	}

	// Save {{publicName}}
	return {{type}}.Update(headers, link, values)
}

func ({{type}} *{{publicName}}) Modified() bool {
	return nullable.Modified({{type}})
}

func (alias {{publicName}}Alias) SaveAll(headers map[string]string, link string) ([]{{publicName}}, error) {
	result := []{{publicName}}{}
	
	{{publicName}}Slice := []{{publicName}}(alias)
	for _, {{type}} := range {{publicName}}Slice {
		{{type}}.Save(headers, link, DefaultURLValues({{type}}.RowModId))
		result = append(result, {{type}})
	}
	
	return result, nil
}

func (alias {{publicName}}Alias) Marshal(fields []string) ([]byte, error) {
	result := []byte{}
	
	data, err := StructListToMapList(alias, fields)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "PersistentAlias.StructListToMapList",
			Details:    fmt.Sprintf("%+v", err),
			InnerError: err,
			Message:    "unexpected error",
		}
		return result, m
	}
	result, err = json.Marshal(data)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "PersistentAlias.Marshal",
			Details:    fmt.Sprintf("%+v", err),
			InnerError: err,
			Message:    "unexpected error",
		}
		return result, m
	}
	
	return result, nil
}

func ({{type}} *{{publicName}}) SetModifiedIfSelected() error {

	selectedFields := nullable.GetSelectedTags({{type}}, false)
	err := nullable.SetModifiedBooleanFields(reflect.ValueOf({{type}}), selectedFields, true, true)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "SetModifiedIfSelected",
			Details:    fmt.Sprintf("%+v", err),
			ErrorNo:    http.StatusInternalServerError,
			InnerError: err,
			Message:    "unexpected error",
		}
		return m
	}
	return nil

}

func ({{type}} *{{publicName}}) SetModifiedIfDifferent(base *{{publicName}}) error {

	function := "dataModel.{{publicName}}.SetModifiedIfDifferent"

	err := nullable.SetModifiedIfDifferent(reflect.ValueOf({{type}}), reflect.ValueOf(base))

	if err != nil {
		e,_ := err.(nullable.ErrorMessage) 
		m := ErrorMessage{
			Attempted:  "SetModifiedIfDifferent",
			Details:    e.Details,
			ErrorNo:    e.ErrorNo,
			InnerError: err,
			Function:   function,
			Message:    "unexpected error",
		}
		return m
	}
	return nil

}

func ({{type}} *{{publicName}}) GetModifiedTags() []string {
	return nullable.GetModifiedTags({{type}})
}


func ({{type}} *{{publicName}}) Mapped() (map[string]interface{}, error) {
	tags := nullable.GetSelectedTags({{type}},false)
	return StructToMap({{type}},tags)
}


`

	result = strings.ReplaceAll(result, "{{publicName}}", publicName)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	return result
}

func generateUpdateCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	instance := snakeCaseToCamelCase(publicName)

	result := `// {{publicName}}.Update saves the record at the link provided the authentication provided in headers is valid.
	func ({{type}} *{{publicName}}) Update(headers map[string]string, link string, values url.Values) ({{publicName}}, error) {

		{{client}} := {{packageName}}.New(link)
		for key, value := range headers {
			{{client}}.AddHeader(key, value)
		}
	
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()

		modifiedFields := nullable.GetModifiedTags({{type}})
		values.Set(SELECT, strings.Join(modifiedFields, COMMA))
		selectedFields := nullable.GetSelectedTags({{type}},false)

		result, err := dataset.Update({{type}}.ODataEditLink, *{{type}}, values)
		if err != nil {
			m := ErrorMessage{
				Attempted:  "{{instance}}.Update",
				Details:    fmt.Sprintf("%+v", err),
				ErrorNo:    http.StatusInternalServerError,
				InnerError: err,
				Message:    "unexpected error",
			}
			return result, m
		}
		err = nullable.SetSelectedBooleanFields(reflect.ValueOf(&result), selectedFields, true, true)
		if err != nil {
			m := ErrorMessage{
				Attempted:  SET_NULLABLE_BOOLEAN_FIELDS,
				Details:    fmt.Sprintf("%+v", err),
				ErrorNo:    http.StatusInternalServerError,
				InnerError: err,
				Message:    "unexpected error",
			}
			return result, m
		}
		return result, err
	}`

	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{instance}}", instance)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	return result
}

func generateSelectCode(set edmxEntitySet, client string, packageName string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	// SetSelected sets each {{publicName}} Nullable tag name to the value specified. 
	// fields is the json tags for each Nullable field in {{publicName}}
	// value is the desired value for {{publicName}}Selected.
	// invert if set to true sets {{publicName}}Selected to the !value if the tag is not found in fields.
	func({{type}} *{{publicName}})SetSelected(fields []string, value bool, not bool) error {
		err := nullable.SetSelectedBooleanFields(reflect.ValueOf({{type}}), fields, value, not)
		if err != nil {
			m := ErrorMessage{
				Attempted: "nullable.SetSelectedBooleanFields",
				Details: fmt.Sprintf("%+v",err),
				ErrorNo: http.StatusInternalServerError,
				Message: "unexpected error",
			}
			return m
		}
		return nil
	}

	func {{publicName}}SelectSingle(defaultFilter string, values url.Values, headers map[string]string, link string) ({{publicName}}, error) {
		models, err := {{publicName}}SelectList(defaultFilter, values, headers, link)
		if err != nil {
			return {{publicName}}{}, err
		}
		if len(models) == 0 {
			filter := defaultFilter
			if values.Get(FILTER) != "" {
				filter = fmt.Sprintf("( %s ) and ( %s )", defaultFilter, values.Get(FILTER))
			}
			return {{publicName}}{}, NilModel{Model: "{{publicName}}", Filter: filter}
		}
		return models[0], nil
	}
	
	func {{publicName}}SelectList(defaultFilter string, values url.Values, headers map[string]string, link string) ([]{{publicName}}, error) {

		{{client}} := {{packageName}}.New(link)
		for key, value := range headers {
			{{client}}.AddHeader(key, value)
		}
		options := {{client}}.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, values)
	
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
		metaChan, modelsChan, errsChan := dataset.List(options)
	
		models := []{{publicName}}{}
		
		for {
			select {
			case err, ok := <-errsChan:
				if ok && err != nil {
					return nil, err
				}
				if !ok {
					errsChan = nil
				}
			case _, ok := <-metaChan:
				if !ok {
					metaChan = nil
				}
			case model, ok := <-modelsChan:
				if ok {
					models = append(models, model)
				} else {
					modelsChan = nil
				}
			}
			if errsChan == nil && metaChan == nil && modelsChan == nil {
				break
			}
		}
	
		return models, nil
	}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{client}}", client)
	result = strings.ReplaceAll(result, "{{packageName}}", packageName)
	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))
	return result
}

func generateSelectByTableName(set edmxEntitySet, client string, options string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	case "{{databaseName}}":
		collection := New{{publicName}}Collection({{client}})
		dataset := collection.DataSet()
		metaChan, modelsChan, errsChan := dataset.List(options)
		
		result := make([]map[string]interface{}, 0)
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
		fields = RemoveEnclosingQuotes(fields)

		for {
			select {
			case err, ok := <-errsChan:
				if ok && err != nil {
					return nil, err
				}
				if !ok {
					errsChan = nil
				}
			case _, ok := <-metaChan:
				if !ok {
					metaChan = nil
				}
			case model, ok := <-modelsChan:
				if ok {
					data, err := odataClient.StructToMap(model, fields)
					if err != nil {
						return result, err
					}
					result = append(result, data)
				} else {
					modelsChan = nil
				}
			}
			if errsChan == nil && metaChan == nil && modelsChan == nil {
				break
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

// HeadAndTailText processes text by extracting part of it based on a delimiter and index.
func HeadAndTailText(text, delimiter string, extract int) (string, string) {
	// Check if the inputs make sense
	if delimiter == "" || extract < 0 {
		return text, ""
	}

	// Split the text into parts
	parts := strings.Split(text, delimiter)

	// Check if the extract index is within bounds
	if extract >= len(parts) {
		return text, ""
	}

	// Reconstruct head and tail
	head := strings.Join(parts[:len(parts)-extract], delimiter)
	tail := strings.Join(parts[len(parts)-extract:], delimiter)

	return head, tail
}

// snakeCaseToCamelCase converts a snake_case string to camelCase
func snakeCaseToCamelCase(snake string) string {
	var camelCase string
	upperNext := false

	for i, char := range snake {
		if char == '_' {
			upperNext = true
		} else {
			if upperNext || i == 0 {
				camelCase += string(unicode.ToUpper(char))
				upperNext = false
			} else {
				camelCase += string(char)
			}
		}
	}

	// Convert the first character to lowercase to ensure camelCase format
	if len(camelCase) > 0 {
		camelCase = strings.ToLower(camelCase[:1]) + camelCase[1:]
	}

	return camelCase
}

// snakeCaseToCamelCase converts a snake_case string to camelCase
func snakeCaseToTitleCase(snake string) string {
	var TitleCase string
	upperNext := false

	for i, char := range snake {
		if char == '_' {
			upperNext = true
		} else {
			if upperNext || i == 0 {
				TitleCase += string(unicode.ToUpper(char))
				upperNext = false
			} else {
				TitleCase += string(char)
			}
		}
	}

	return TitleCase
}
