package modelGenerator

import (
	"encoding/xml"
	"fmt"
	"io"
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

func (g Generator) ModelDefinition(set edmxEntitySet) string {
	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := fmt.Sprintf(`//goland:noinspection GoUnusedExportedFunction
	func New{{publicName}}Collection(wrapper {{g.Package.OdataAlias}}.Wrapper) {{g.Package.OdataAlias}}.ODataModelCollection[{{publicName}}] {
		return modelDefinition[{{publicName}}]{client: wrapper.ODataClient(), name: "publicName", url: "%s"}
	}`, set.Name)
	result = strings.ReplaceAll(result, `{{g.Package.OdataAlias}}`, g.Package.OdataAlias)
	result = strings.ReplaceAll(result, `{{publicName}}`, publicName)
	return result
}

func (g *Generator) CodeFromSchema(dataService edmxDataServices) map[string]string {

	code := map[string]string{}

	customErrors := `
	
	type NilModel struct {
		Model  string
		Filter string
	}
	
	type RecordIsOutOfDate struct {
		Model  string      
		Uuid   string      
		Id     float64     
		Record any 
	}

	func (e NilModel) Error() string {
		return fmt.Sprintf(" No matching %s found for %s.", e.Model, e.Filter)
	}

	func (r RecordIsOutOfDate) Error() string {
	data, err := json.Marshal(r)
	if err != nil {
		return err.Error()
	}
	return string(data)
}


	`

	deleteCode := fmt.Sprintf(`package %s
	
	import (
		{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
	)
	`, g.Package.Name)
	deleteCode = strings.ReplaceAll(deleteCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	updateCode := fmt.Sprintf(`package %s
	
import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	nullable "github.com/Uffe-Code/go-nullable/nullable"
	{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
)
`, g.Package.Name)
	updateCode = strings.ReplaceAll(updateCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	insertCode := fmt.Sprintf(`package %s
	
import (
	"fmt"
	"net/http"
	"reflect"

	nullable "github.com/Uffe-Code/go-nullable/nullable"
	{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
)
`, g.Package.Name)
	insertCode = strings.ReplaceAll(insertCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	modelCode := fmt.Sprintf(`package %s

import (
	"encoding/json"
	"fmt"

	nullable "github.com/Uffe-Code/go-nullable/nullable"
	{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
	date "github.com/Uffe-Code/go-odata/date"
	
)

type modelDefinition[T any] struct { client {{g.Package.OdataAlias}}.ODataClient; name string; url string }

func (md modelDefinition[T]) Name() string {
	return md.name
}

func (md modelDefinition[T]) Url() string {
	return md.url
}

func (md modelDefinition[T]) DataSet() {{g.Package.OdataAlias}}.ODataDataSet[T, {{g.Package.OdataAlias}}.ODataModelDefinition[T]] {
	return {{g.Package.OdataAlias}}.NewDataSet[T](md.client, md)
}

%s

`, g.Package.Name, customErrors)
	modelCode = strings.ReplaceAll(modelCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	selectByTableName := fmt.Sprintf(`package %s

	import (
		"net/url"
		"strings"

		{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
	)
	

	func SelectByTableName(tableName string, defaultFilter string, values url.Values, headers map[string]string, link string) ([]map[string]any, error) {

		client := {{g.Package.OdataAlias}}.New(link)
		for key, value := range headers {
			client.AddHeader(key, value)
		}
		{{options}} := client.ODataQueryOptions()
		options = {{options}}.ApplyArguments(defaultFilter, values)

		switch tableName {

	`, g.Package.Name)

	selectByTableNameOptions := `options`
	selectByTableName = strings.ReplaceAll(selectByTableName, "{{options}}", selectByTableNameOptions)
	selectByTableName = strings.ReplaceAll(selectByTableName, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	saveByTableName := fmt.Sprintf(`package %s

	import (
		"encoding/json"
		"fmt"
		"net/http"
		"net/url"

		nullable "github.com/Uffe-Code/go-nullable/nullable"
		{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
	)
	

	func SaveByTableName(tableName string, defaultFilter string, values url.Values, headers map[string]string, link string, data []byte) (results map[string][]map[string]any, messages []error) {

		client := {{g.Package.OdataAlias}}.New(link)
		for key, value := range headers {
			client.AddHeader(key, value)
		}
		{{options}} := client.ODataQueryOptions()
		options = {{options}}.ApplyArguments(defaultFilter, values)


		switch tableName {

	`, g.Package.Name)
	saveByTableNameOptions := `options`
	saveByTableName = strings.ReplaceAll(saveByTableName, "{{options}}", saveByTableNameOptions)
	saveByTableName = strings.ReplaceAll(saveByTableName, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	saveCode := fmt.Sprintf(`package %s

	import (
		"encoding/json"
		"fmt"
		"net/http"
		"net/url"
		"reflect"
			
		nullable "github.com/Uffe-Code/go-nullable/nullable"
	)

`, g.Package.Name)

	datasets := fmt.Sprintf(`
package %s
	
import (
	{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
)

	`, g.Package.Name)
	datasets = strings.ReplaceAll(datasets, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	selectCode := fmt.Sprintf(`
package %s

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	nullable "github.com/Uffe-Code/go-nullable/nullable"
	{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
)

`, g.Package.Name)
	selectCode = strings.ReplaceAll(selectCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	mapCode := fmt.Sprintf(`package %s

import (
	"fmt"
	"net/url"
	"strings"
	
	{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
)

`, g.Package.Name)
	mapCode = strings.ReplaceAll(mapCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	for _, schema := range dataService.Schemas {
		for _, enum := range schema.EnumTypes {
			modelCode += "\n" + g.EnumStruct(enum) + "\n"
		}

		for _, complexType := range schema.ComplexTypes {
			modelCode += "\n" + g.generateModelStruct(complexType, map[string]string{}) + "\n"
		}

		var names []string
		for name := range schema.EntitySets {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			return strings.TrimLeft(strings.ToLower(names[i]), "_") < strings.TrimLeft(strings.ToLower(names[j]), "_")
		})

		for _, name := range names {

			fields := map[string]string{}

			set := schema.EntitySets[name]
			datasets += "\n" + g.DataSet(set) + "\n"
			deleteCode += "\n" + g.DeleteCode(set) + "\n"
			insertCode += "\n" + g.InsertCode(set) + "\n"
			mapCode += "\n" + g.MapFunctionCode(set) + "\n"
			modelCode += "\n" + g.generateModelStruct(set.getEntityType(), fields) + "\n"
			modelCode += "\n" + g.ModelDefinition(set) + "\n"
			saveCode += "\n" + g.SaveCode(set) + "\n"
			selectByTableName += "\n" + g.SelectByTableName(set, selectByTableNameOptions) + "\n"
			saveByTableName += "\n" + g.SaveByTableName(set, fields) + "\n"
			selectCode += "\n" + g.SelectCode(set) + "\n"
			updateCode += "\n" + g.UpdateCode(set) + "\n"
		}
	}

	selectByTableName += `
	default:
		return nil, nil
	}
}
	`

	saveByTableName += `
	default:
		return nil, nil
	}
}
`

	if g.Package.Models != "" {
		code[g.Package.Models] = modelCode
	}
	if g.Package.SelectByTableName != "" {
		code[g.Package.SelectByTableName] = selectByTableName
	}
	if g.Package.SaveByTableName != "" {
		code[g.Package.SaveByTableName] = saveByTableName
	}
	if g.Package.Select != "" {
		code[g.Package.Select] = selectCode
	}
	if g.Package.Datasets != "" {
		code[g.Package.Datasets] = datasets
	}
	if g.Package.Maps != "" {
		code[g.Package.Maps] = mapCode
	}
	if g.Package.Update != "" {
		code[g.Package.Update] = updateCode
	}
	if g.Package.Insert != "" {
		code[g.Package.Insert] = insertCode
	}
	if g.Package.Save != "" {
		code[g.Package.Save] = saveCode
	}
	if g.Package.Delete != "" {
		code[g.Package.Delete] = deleteCode
	}
	packageLine := fmt.Sprintf("package %s", g.Package.Name)

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

func (g *Generator) FieldConstants(dataService edmxDataServices) string {

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

func (g *Generator) generateModelStruct(entityType edmxEntityType, fields map[string]string) string {

	publicName := publicAttribute(entityType.Name)
	structString := fmt.Sprintf("type %s struct {", publicName)
	propertyKeys := sortedCaseInsensitiveStringKeys(entityType.Properties)

	// FileMaker incorrectly reports read only attributes
	enforceReadOnlyProperties := map[string]string{}
	ignoreReadOnlyProperties := map[string]string{}

	r := []rune(entityType.Name)
	tableName := string(r[:len(r)-1])

	ignoreTable, found := g.IgnoreReadOnly[tableName]
	if found {
		for _, r := range ignoreTable {
			ignoreReadOnlyProperties[r] = r
		}
	}

	enforceTable, found := g.EnforceReadOnly[tableName]
	if found {
		for _, r := range enforceTable {
			enforceReadOnlyProperties[r] = r
		}
	}

	readOnlyTag := g.Fields.ReadOnlyTag
	readOnly := false
	ignoreReadOnly := false

	jsonSupport := ""
	name := ""

	for _, extra := range g.Fields.Extras {
		structString += fmt.Sprintf("\n\t%s", extra)
	}
	structString += "\n"

	for _, propertyKey := range propertyKeys {
		include := g.validPropertyName(propertyKey)
		if include {

			ignoreReadOnly = false
			readOnly = false

			if g.ReadOnly {

				_, ignore := ignoreReadOnlyProperties[propertyKey]
				if ignore {
					ignoreReadOnly = true
				}

				_, enforce := enforceReadOnlyProperties[propertyKey]
				if enforce {
					readOnly = true
				}

			}

			prop := entityType.Properties[propertyKey]
			name = prop.Name
			if g.Fields.Public {
				name = publicAttribute(name)
				fields[propertyKey] = propertyKey
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
							if enum == readOnlyTag && !ignoreReadOnly {
								if g.ReadOnly {
									readOnly = true
								}
							}
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
				g.Package.WrapCollections,
				readOnly)
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
			"\tPortalData map[string][]any `json:\"portalData,omitempty\"`\n"+
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

func (g Generator) DataSet(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `func {{publicName}}DataSet(headers map[string]string, link string) {{g.Package.OdataAlias}}.ODataDataSet[{{publicName}}, {{g.Package.OdataAlias}}.ODataModelDefinition[{{publicName}}]] {
		{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
		for key, value := range headers {
			{{g.Package.OdataAlias}}.AddHeader(key, value)
		}
		collection := New{{publicName}}Collection({{g.Package.OdataAlias}})
		dataset := collection.DataSet()
		return dataset
	}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
	result = strings.ReplaceAll(result, "{{g.Package.Name}}", g.Package.Name)
	return result
}

func (g Generator) DeleteCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)

	result := `func (o *{{publicName}}) Delete(headers map[string]string, link string) error {

	{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
	for key, value := range headers {
		{{g.Package.OdataAlias}}.AddHeader(key, value)
	}
	
	collection := New{{publicName}}Collection({{g.Package.OdataAlias}})
	dataset := collection.DataSet()
	
	return dataset.Delete(o.ODataEditLink)
}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{g.Package.Name}}", g.Package.Name)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	return result

}

func (g Generator) EnumStruct(enum edmxEnumType) string {
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

func (g Generator) InsertCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	instance := snakeCaseToCamelCase(publicName)

	result := `func ({{type}} *{{publicName}}) Insert(headers map[string]string, link string) ({{publicName}}, error) {

	{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
	for key, value := range headers {
		{{g.Package.OdataAlias}}.AddHeader(key, value)
	}
	
	collection := New{{publicName}}Collection({{g.Package.OdataAlias}})
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
	result = strings.ReplaceAll(result, "{{g.Package.Name}}", g.Package.Name)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
	result = strings.ReplaceAll(result, "{{instance}}", instance)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	return result

}

func (g Generator) MapFunctionCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	func {{publicName}}MapFunction(defaultFilter string, urlValues url.Values, root string, headers map[string]string, link string) (map[string]any, error) {
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
		model, err := {{publicName}}Singular(defaultFilter, values, headers, link)
		if err != nil {
			return make(map[string]any), err
		}
		data, err := {{g.Package.OdataAlias}}.StructToMap(model, fields)
		if err != nil {
			return make(map[string]any), err
		}
		return data, nil
	}

	func {{publicName}}MultipleMap(defaultFilter string, urlValues url.Values, root string, headers map[string]string, link string) ([]map[string]any, error) {
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
		models, err := {{publicName}}Multiple(defaultFilter, values, headers, link)
		if err != nil {
			return make([]map[string]any, 0), err
		}
		data, err := StructMultipleToMapMultiple(models, fields)
		if err != nil {
			return make([]map[string]any, 0), err
		}
		return data, nil
	}
	`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
	return result
}

func (g Generator) SaveCode(set edmxEntitySet) string {

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

	function := "{{publicName}}"

	result := []byte{}
	
	data, err := StructMultipleToMapMultiple(alias, fields)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "StructMultipleToMapMultiple",
			Details:    fmt.Sprintf("%+v", err),
			Function: function,
			InnerError: err,
			Message:    "unexpected error",
		}
		return result, m
	}
	result, err = json.Marshal(data)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "json.Marshal",
			Details:    fmt.Sprintf("%+v", err),
			Function:	function,
			InnerError: err,
			Message:    "unexpected error",
		}
		return result, m
	}
	
	return result, nil
}

func ({{type}} *{{publicName}}) SetModifiedIfSelected() error {

	function := "{{publicName}}"

	selectedFields := nullable.GetSelectedTags({{type}}, false)
	err := nullable.SetModifiedBooleanFields(reflect.ValueOf({{type}}), selectedFields, true, true)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "SetModifiedIfSelected",
			Details:    fmt.Sprintf("%+v", err),
			ErrorNo:    http.StatusInternalServerError,
			Function: 	function,
			InnerError: err,
			Message:    "unexpected error",
		}
		return m
	}
	return nil

}

func ({{type}} *{{publicName}}) SetModifiedIfDifferent(base *{{publicName}}) error {

	function := "{{publicName}}.SetModifiedIfDifferent"

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


func ({{type}} *{{publicName}}) Mapped() (map[string]any, error) {
	tags := nullable.GetSelectedTags({{type}},false)
	return nullable.StructToMap({{type}},tags)
}


`

	result = strings.ReplaceAll(result, "{{publicName}}", publicName)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	return result
}

// SaveXML writes v to path as UTF-8 XML with indentation.
// It creates parent directories as needed and writes atomically.
func (g Generator) SaveXMLSchema(path string, v edmxXmlData) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Write to a temp file in the same directory for atomic rename.
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// If anything fails before rename, clean up the temp file.
		_ = os.Remove(tmpName)
	}()

	// XML header
	if _, err := io.WriteString(tmp, xml.Header); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write header: %w", err)
	}

	// Encode with indentation.
	enc := xml.NewEncoder(tmp)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode xml: %w", err)
	}
	if err := enc.Flush(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("flush: %w", err)
	}

	// Ensure data is on disk before renaming.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("fsync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	// Atomic replace.
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	// Optionally, set final mode (e.g., 0644)
	_ = os.Chmod(path, 0o644)
	return nil
}

func (g Generator) UpdateCode(set edmxEntitySet) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	instance := snakeCaseToCamelCase(publicName)

	result := `// {{publicName}}.Update saves the record at the link provided the authentication provided in headers is valid.
	func ({{type}} *{{publicName}}) Update(headers map[string]string, link string, values url.Values) ({{publicName}}, error) {

		{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
		for key, value := range headers {
			{{g.Package.OdataAlias}}.AddHeader(key, value)
		}
	
		collection := New{{publicName}}Collection({{g.Package.OdataAlias}})
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
	result = strings.ReplaceAll(result, "{{g.Package.Name}}", g.Package.Name)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
	result = strings.ReplaceAll(result, "{{instance}}", instance)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	return result
}

func (g Generator) SaveByTableName(set edmxEntitySet, fields map[string]string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)

	result := `
	case "{{databaseName}}":

		function := "saveByTableName.{{databaseName}}"

		input := map[string]{{publicName}}{}
		output := []{{publicName}}{}

		err := json.Unmarshal(data, &input)
		if err != nil {
			m := ErrorMessage{
				Attempted: "json.Unmarshal",
				Details: fmt.Sprintf("Error: %+v", err),
				ErrorNo: http.StatusBadRequest,
				Function: function,
				Message: BAD_REQUEST,
			}
			messages = append( messages, m )
			return map[string][]map[string]any{}, messages
		}

		selectFields := []string{}

		// make sure fields are populated
		for _, {{type}} := range input {
			errored := false
			{{checks}}
			if errored {
				continue
			}
			saved, err := {{type}}.Save(headers,link,values)
			if err != nil {
				e := ExtractError(err)
				message := UNEXPECTED_ERROR
				s, ok := e.Message.(string)
				if ok {
					message = s
				}
				m := ErrorMessage{
					Attempted:     "{{type}}.Save(headers,link,values)",
					Details:       e.Details,
					ErrorNo:       e.ErrorNo,
					FileName:    	e.FileName,
					Function:      function,
					InnerError:    err,
					LineNumber:   e.LineNumber,
					Message:       message,
					Payload: nil,
					RequestUrl: e.RequestUrl,
				}
				messages = append(messages, m)
				continue
			}
			if len(selectFields) == 0 {
				selectFields = nullable.GetSelectedTags(saved, false)
				selectFields = append(selectFields, ExtraFields()...)
			}
			output = append(output,saved)
		}

		mapped, err := nullable.StructMultipleToMapMultiple(output,selectFields)
		if err != nil {
			e := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := e.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:     	"nullable.MultipleToMapMultiple(output)",
				Details:       	e.Details,
				ErrorNo:       	e.ErrorNo,
				FileName:    	e.FileName,
				Function:      	function,
				InnerError:    	err,
				LineNumber:   	e.LineNumber,
				Message:       	message,
				Payload: 		nil,
				RequestUrl: 	e.RequestUrl,
			}
			messages = append(messages, m)
		}
		results := map[string][]map[string]any{"{{databaseName}}":mapped}
		return results, messages
`
	checks := ""
	for _, m := range g.Fields.Mandatory {
		_, found := fields[m.Name]
		if found {
			fieldName := snakeCaseToTitleCase(m.Name)
			if m.Valid {
				checks += fmt.Sprintf(`if {{type}}.%s.IsValid() != {{valid}} {
					m := ErrorMessage{
						Details: "{{publicName}}.%s.IsValid() must be {{valid}}",
						ErrorNo: http.StatusBadRequest,
						Function: function,
						Message: BAD_REQUEST,
					}
					messages = append(messages, m)
					errored = true
				}`, fieldName, fieldName) + "\n\n"
				checks = strings.ReplaceAll(checks, "{{valid}}", fmt.Sprintf(`%t`, m.Valid))
			}
			if m.Selected {
				checks += fmt.Sprintf(`if {{type}}.%s.IsSelected() != {{selected}} {
					m := ErrorMessage{
						Details: "{{publicName}}.%s.IsSelected() must be {{selected}}",
						ErrorNo: http.StatusBadRequest,
						Function: function,
						Message: BAD_REQUEST,
					}
					messages = append(messages, m)
					errored = true
				}`, fieldName, fieldName) + "\n\n"
				checks = strings.ReplaceAll(checks, "{{selected}}", fmt.Sprintf(`%t`, m.Selected))
			}
		}
	}

	result = strings.ReplaceAll(result, "{{checks}}", checks)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	result = strings.ReplaceAll(result, "{{databaseName}}", set.Name)
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

	return result

}

func (g Generator) SelectCode(set edmxEntitySet) string {

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

	func {{publicName}}Singular(defaultFilter string, values url.Values, headers map[string]string, link string) ({{publicName}}, error) {
		models, err := {{publicName}}Multiple(defaultFilter, values, headers, link)
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
	
	func {{publicName}}Multiple(defaultFilter string, values url.Values, headers map[string]string, link string) ([]{{publicName}}, error) {

		{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
		for key, value := range headers {
			{{g.Package.OdataAlias}}.AddHeader(key, value)
		}
		options := {{g.Package.OdataAlias}}.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, values)
	
		collection := New{{publicName}}Collection({{g.Package.OdataAlias}})
		dataset := collection.DataSet()
		meta, data, errs := dataset.Multiple(options)
	
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
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
	result = strings.ReplaceAll(result, "{{g.Package.Name}}", g.Package.Name)
	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))
	return result
}

func (g Generator) SelectByTableName(set edmxEntitySet, options string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	case "{{databaseName}}":
		collection := New{{publicName}}Collection(client)
		dataset := collection.DataSet()
		meta, data, errs := dataset.Multiple(options)
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
		result := make([]map[string]any, 0)
		for range meta {
			fields = RemoveEnclosingQuotes(fields)
			for model := range data {
				data, err := {{g.Package.OdataAlias}}.StructToMap(model, fields)
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
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
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
