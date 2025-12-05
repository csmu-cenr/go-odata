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

	"github.com/google/uuid"
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

// RightUUID returns the last n characters of a new UUID string.
// If stripDashes is true, all '-' characters are removed before slicing.
func rightUUID(n int, stripDashes bool) string {
	u := uuid.New().String()

	if stripDashes {
		u = strings.ReplaceAll(u, "-", "")
	}

	if n <= 0 {
		return ""
	}

	if n >= len(u) {
		return u
	}

	return u[len(u)-n:]
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
		Model 	string 
		Exit  	string
		Filter 	string
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
		"strings"
			
		nullable "github.com/Uffe-Code/go-nullable/nullable"
		{{g.Package.OdataAlias}} "github.com/Uffe-Code/go-odata/odataClient"
	)

`, g.Package.Name)

	saveCode = strings.ReplaceAll(saveCode, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

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
	
	nullable "github.com/Uffe-Code/go-nullable/nullable"
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

		ignoreMap := map[string]bool{}
		for _, s := range g.Sets {
			if s.Ignore {
				ignoreMap[s.Name] = true
			}
		}

		if len(ignoreMap) > 0 {
			fmt.Println()
		}

		for _, name := range names {

			_, ignore := ignoreMap[name]
			debugTable, debug := g.Debug[name]

			if !debug {
				g.DebugFunction = nil
				debugTable = Debug{}
			}

			if g.Verbose || debug {
				if ignore {
					fmt.Println(strings.Repeat("-", 40))
					fmt.Printf("\tIgnored -> Name: %s\n", name)
					fmt.Println(strings.Repeat("-", 40))
				} else {
					if debug && debugTable.Verbose {
						fmt.Println()
						fmt.Printf("Name: %s\n", name)
					}
				}
			} else {
				if ignore {
					fmt.Printf("\tIgnored -> Name: %s\n", name)
				}
			}

			if ignore {
				continue
			}

			fieldsMap := map[string]string{}

			set := schema.EntitySets[name]

			datasets += "\n" + g.DataSet(set) + "\n"
			deleteCode += "\n" + g.DeleteCode(set) + "\n"
			insertCode += "\n" + g.InsertCode(set) + "\n"
			mapCode += "\n" + g.MapFunctionCode(set) + "\n"
			modelStruct, generateModelStruct := debugTable.Functions["generateModelStruct"]
			if generateModelStruct {
				if !modelStruct.Ignore {
					g.DebugFunction = &modelStruct
					result := g.generateModelStruct(set.getEntityType(), map[string]string{})
					fmt.Printf("\n\n%s\n\n", result)
				}
			}
			modelCode += "\n" + g.generateModelStruct(set.getEntityType(), fieldsMap) + "\n"
			modelCode += "\n" + g.ModelDefinition(set) + "\n"
			saveCode += "\n" + g.SaveCode(set, fieldsMap) + "\n"
			selectByTableName += "\n" + g.SelectByTableName(set, selectByTableNameOptions) + "\n"
			saveByTableName += "\n" + g.SaveByTableName(set, fieldsMap) + "\n"
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

func (g *Generator) generateModelStruct(entityType edmxEntityType, fieldsMap map[string]string) string {

	publicName := publicAttribute(entityType.Name)
	structString := fmt.Sprintf("type %s struct {", publicName)
	properties := sortedCaseInsensitiveStringKeys(entityType.Properties)

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

	debugFields := map[string]string{}
	if g.DebugFunction != nil {
		for _, p := range g.DebugFunction.Fields.Named {
			debugFields[p] = p
		}
	}

	for _, p := range properties {

		_, found := debugFields[p]
		if g.DebugFunction != nil {
			if g.DebugFunction.Fields.All {
				found = true
			}
		}
		if found {
			fmt.Printf("Field: %s found\n", p)
		}

		include := g.validPropertyName(p)

		if found && !include {
			fmt.Printf("Field: %s not included\n", p)
		}

		if include {

			ignoreReadOnly = false
			readOnly = false

			if g.ReadOnly {

				_, ignore := ignoreReadOnlyProperties[p]
				if ignore {
					ignoreReadOnly = true
				}

				_, enforce := enforceReadOnlyProperties[p]
				if enforce {
					readOnly = true
				}

			}

			prop := entityType.Properties[p]
			name = prop.Name
			if g.Fields.Public {
				name = publicAttribute(name)
				fieldsMap[p] = p
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
		model, err := {{publicName}}Node(defaultFilter, values, headers, link)
		if err != nil {
			return make(map[string]any), err
		}
		data, err := {{g.Package.OdataAlias}}.StructToMap(model, fields)
		if err != nil {
			return make(map[string]any), err
		}
		return data, nil
	}

	func {{publicName}}SetMap(defaultFilter string, urlValues url.Values, root string, headers map[string]string, link string) ([]map[string]any, error) {
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
		models, err := {{publicName}}Set(defaultFilter, values, headers, link)
		if err != nil {
			return make([]map[string]any, 0), err
		}
		data, err := nullable.StructSetToMapSet(models, fields)
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

func (g Generator) SaveCode(set edmxEntitySet, fieldsMap map[string]string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)

	guardValue := ""
	guardFilter := ""
	guardValid := ""

	_, found := fieldsMap["uuid"]
	if found {
		guardValid = "{{type}}.Uuid.IsValid()"
		guardValue = "guardValue := {{type}}.Uuid.GetData()"
		guardFilter = "guardFilter := fmt.Sprintf(\" uuid eq '%s' \", guardValue)"
	} else {
		_, found := fieldsMap["id"]
		if found {
			guardValid = "{{type}}.Id.IsValid()"
			guardValue = "guardValue := int({{type}}.Id.GetData())"
			guardFilter = "guardFilter := fmt.Sprintf(` \"id\" eq %d`, guardValue)"
		}
	}

	result := `

// {{publicName}}.Save determines whether to save or creates a record at the link provided the authentication provided in headers is valid.
func ({{type}} *{{publicName}}) Save(headers map[string]string, link string, values url.Values) ({{publicName}}, error) {

	function := "{{publicName}}.Save"
	dereferenced := *{{type}}

	{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
	for key, value := range headers {
		{{g.Package.OdataAlias}}.AddHeader(key, value)
	}
	
	ignore := []string{}
	ignoreText := values.Get(IGNORE)
	if len(ignoreText) > 0 {
		ignore = strings.Split(ignoreText, ",")
	}

	creation := []string{}
	creationText := values.Get(CREATION)
	if len(creationText) > 0 {
		creation = strings.Split(creationText, ",")
	}

	modification := []string{}
	modificationText := values.Get(MODIFICATION)
	if len(modificationText) > 0 {
		modification = strings.Split(modificationText, ",")
	}

	icm := []string{}
	icm = append(icm, ignore...)
	icm = append(icm, creation...)
	icm = append(icm, modification...)
	consider := nullable.GetSelectedTagsIgnoring({{type}}, false, icm)

	ccm := []string{}
	ccm = append(ccm, consider...)
	ccm = append(ccm, creation...)
	ccm = append(ccm, modification...)
	ccm = append(ccm, ignore...)
	values.Set(SELECT, strings.Join(ccm, COMMA))

	top := values.Get(TOP)
	if top == "" {
		top = "1"
		values.Set(TOP, top)
	}
	defaultFilter := values.Get(DEFAULT_FILTER)

	var (
		node     {{publicName}}
		existing []{{publicName}}
		err      error
	)

	if {{type}}.ODataEditLink == "" {
		
		if !{{guardValid}}{
			message := BAD_REQUEST
			m := ErrorMessage{
				Attempted:     	"{{guardValid}}",
				Details:       	"field must be valid",
				ErrorNo:       	http.StatusBadRequest,
				Exit:       	"{{exit01}}",
				Function:      	function,
				Message:       	message,
			}
			return dereferenced, m
		}

		{{guardValue}}
		{{guardFilter}}

		filter := values.Get(FILTER)

		if filter == "" {
			filter = guardFilter
		} else {
			if !strings.Contains(filter, guardFilter) {
				filter = fmt.Sprintf("(%s) and %s", filter, guardFilter)
			}
		}
		values.Set(FILTER, filter)
		existing, err = {{publicName}}Set(defaultFilter, values, headers, link)
		if err != nil {
			ee := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := ee.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:  "{{publicName}}Set",
				Code:       ee.Code,
				Details:    ee.Details,
				ErrorNo:    ee.ErrorNo,
				Exit:       "{{exit02}}",
				FileName:   ee.FileName,
				Function:   function,
				InnerError: err,
				LineNumber: ee.LineNumber,
				Message:    message,
				Payload:    nil,
				RequestUrl: ee.RequestUrl,
				User:       nil,
			}
			ee.RequestUrl = ""
			return dereferenced, m
		}
	} else {
		node, err = {{publicName}}Node(defaultFilter, values, headers, link)
		if err != nil {
			ee := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := ee.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:  "{{publicName}}Node",
				Code:       ee.Code,
				Details:    ee.Details,
				ErrorNo:    ee.ErrorNo,
				Exit:       "{{exit03}}",
				FileName:   ee.FileName,
				Function:   function,
				InnerError: err,
				LineNumber: ee.LineNumber,
				Message:    message,
				Payload:    nil,
				RequestUrl: ee.RequestUrl,
				User:       nil,
			}
			ee.RequestUrl = ""
			return dereferenced, m
		}
		existing = append(existing, node)
	}

	if len(existing) > 0 {
		// will only be one
		for _, e := range existing {
			if {{type}}.ODataEditLink == "" {
				{{type}}.ODataEditLink = e.ODataEditLink
			}
			if {{type}}.ODataEditLink != e.ODataEditLink {
				message := UNEXPECTED_ERROR
					m := ErrorMessage{
						Attempted:  "",
						Details:    	"{{type}}.ODataEditLink != e.ODataEditLink",
						ErrorNo:    	http.StatusInternalServerError,
						Exit:       	"{{exit04}}",
						FileName:   	"",
						Function:   	function,
						InnerError: 	err,
						IPAddress:  	"",
						LineNumber: 	0,
						Link:       	"",
						Message:    	message,
						Payload:    	nil,
						RequestUrl: 	"",
						User:       	nil,
						UnixTimestamp: 	time.Now().Unix(),
					}
					return dereferenced, m
				}
			}
			different, err := nullable.LeftIsDifferentFromRightIgnoring(reflect.ValueOf({{type}}), reflect.ValueOf(e), consider, icm)
			if err != nil {
				ee := ExtractError(err)
				message := UNEXPECTED_ERROR
				s, ok := ee.Message.(string)
				if ok {
					message = s
				}
				m := ErrorMessage{
					Attempted:  "nullable.LeftIsDifferentFromRight",
					Details:    ee.Details,
					ErrorNo:    ee.ErrorNo,
					Exit:       "{{exit05}}",
					FileName:   ee.FileName,
					Function:   function,
					InnerError: err,
					LineNumber: ee.LineNumber,
					Message:    message,
					Payload:    nil,
					RequestUrl: "",
					User:       nil,
				}
				return dereferenced, m
			}
			if len(different) > 0 {
				di := []string{}
				di = append(di, different...)
				di = append(di, ignore...)
				cm := []string{}
				cm = append(cm, consider...)
				cm = append(cm, modification...)
				modify, err := nullable.LeftIsDifferentFromRightIgnoring(reflect.ValueOf({{type}}), reflect.ValueOf(e), cm, di)
				values.Set(MODIFIED, strings.Join(modify, COMMA))
				if err != nil {
					ee := ExtractError(err)
					message := UNEXPECTED_ERROR
					s, ok := ee.Message.(string)
					if ok {
						message = s
					}
					m := ErrorMessage{
						Attempted:  "nullable.LeftIsDifferentFromRightIgnoring",
						Code:       ee.Code,
						Details:    ee.Details,
						ErrorNo:    ee.ErrorNo,
						Exit:       "{{exit06}}",
						FileName:   ee.FileName,
						Function:   function,
						InnerError: err,
						LineNumber: ee.LineNumber,
						Message:    message,
						Payload:    nil,
						RequestUrl: ee.RequestUrl,
						User:       nil,
					}
					ee.RequestUrl = ""
					return dereferenced, m
				}
				modify = append(modify, different...)
				err = nullable.SetLeftModified(reflect.ValueOf({{type}}), reflect.ValueOf(e), modify)
				if err != nil {
					ee := ExtractError(err)
					message := UNEXPECTED_ERROR
					s, ok := ee.Message.(string)
					if ok {
						message = s
					}
					m := ErrorMessage{
						Attempted:  "nullable.SetLeftModified",
						Details:    ee.Details,
						ErrorNo:    ee.ErrorNo,
						Exit:       "{{exit07}}",
						FileName:   ee.FileName,
						Function:   function,
						InnerError: err,
						LineNumber: ee.LineNumber,
						Message:    message,
						Payload:    nil,
						RequestUrl: ee.RequestUrl,
						User:       nil,
					}
					ee.RequestUrl = ""
					return dereferenced, m
				}
			} else {
				return dereferenced, nil
			}
		}
	} else {
		nullable.SetModifiedIfSelected(reflect.ValueOf({{type}}))
	}

	values.Del(FILTER)
	values.Del(TOP)
	values.Del(DEFAULT_FILTER)

	// Check if anything has changed. Bounce out.
	if !{{type}}.Modified() {
		return dereferenced, nil
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

	function := "{{publicName}}Alias.SaveAll"
	errs := []error{}

	result := []{{publicName}}{}
	
	{{publicName}}Slice := []{{publicName}}(alias)
	for _, {{type}} := range {{publicName}}Slice {
		_, err := {{type}}.Save(headers, link, DefaultURLValues({{type}}.RowModId))
		if err != nil {
			ee := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := ee.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:    	"{{type}}.Save",
				Code:         	ee.Code,
				Details:       	ee.Details,
				ErrorNo:       	ee.ErrorNo,
				Exit:       	"{{exit08}}",
				FileName:    	ee.FileName,
				Function:      	function,
				InnerError:    	err,
				LineNumber:   	ee.LineNumber,
				Message:       	message,
				Payload: 		nil,
				RequestUrl: 	ee.RequestUrl,
				User: 			nil,
			}
			ee.RequestUrl = ""
			errs = append( errs, m)
		}
		result = append(result, {{type}})
	}
	
	if len( errs ) > 0 {
		message := UNEXPECTED_ERROR
		m := ErrorMessage{
			Attempted:     	"",
			Details:       	"",
			ErrorNo:       	http.StatusInternalServerError,
			Exit:       	"{{exit09}}",
			FileName:    	"",
			Function:      	function,
			InnerError:    	errs,
			IPAddress:     	"",
			LineNumber:   	0,
			Link:       	"",
			Message:       message,
			Payload: 		nil,
			RequestUrl:    "",
			User: 			nil,
		}
		return result, m
	}
	return result, nil
}

func (alias {{publicName}}Alias) Marshal(fields []string) ([]byte, error) {

	function := "{{publicName}}"

	result := []byte{}
	
	data, err := nullable.StructSetToMapSet(alias, fields)
	if err != nil {
		m := ErrorMessage{
			Attempted:  "StructSetToMapSet",
			Details:    fmt.Sprintf("%+v", err),
			ErrorNo:	http.StatusInternalServerError,
			Exit:		"{{exit10}}",
			Function: 	function,
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
			ErrorNo:	http.StatusInternalServerError,
			Exit:		"{{exit11}}",
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
			Exit:		"{{exit12}}",
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
			Exit:		"{{exit13}}",
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

func ({{type}} *{{publicName}}) Mapped() (result map[string]any, err error) {

	function := "{{publicName}}.Mapped"

	tags := nullable.GetSelectedTags({{type}},false)

	result, err = nullable.StructToMap({{type}},tags)
	if err != nil {

		mapped := map[string][]string{}
		mapped["selectedTags"] = tags

		message := UNEXPECTED_ERROR
		m := ErrorMessage{
			Attempted:     	"nullable.StructToMap",
			Details:       	fmt.Sprintf("error: %+s",err),
			ErrorNo:       	http.StatusInternalServerError,
			Exit:       	"{{exit14}}",
			Function:      	function,
			InnerError:    	err,
			Message:       	message,
			Payload: 		mapped,
		}
		return result, m
	} 

	return result, nil
}


`

	result = strings.ReplaceAll(result, "{{exit01}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit02}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit03}}", rightUUID(12, false))

	result = strings.ReplaceAll(result, "{{exit04}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit05}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit06}}", rightUUID(12, false))

	result = strings.ReplaceAll(result, "{{exit07}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit08}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit09}}", rightUUID(12, false))

	result = strings.ReplaceAll(result, "{{exit10}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit11}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit12}}", rightUUID(12, false))

	result = strings.ReplaceAll(result, "{{exit13}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit14}}", rightUUID(12, false))

	result = strings.ReplaceAll(result, "{{guardValid}}", guardValid)
	result = strings.ReplaceAll(result, "{{guardValue}}", guardValue)
	result = strings.ReplaceAll(result, "{{guardFilter}}", guardFilter)

	result = strings.ReplaceAll(result, "{{publicName}}", publicName)

	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))

	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)

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

		modify := values.Get(MODIFIED)
		if modify == "" {
			modifiedFields := nullable.GetModifiedTags(r)
			modify = strings.Join(modifiedFields, COMMA)
		}
		values.Set(SELECT, modify)
		selectedFields := nullable.GetSelectedTags({{type}},false)

		result, err := dataset.Update({{type}}.ODataEditLink, *{{type}}, values)
		if err != nil {
			e := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := e.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:  "dataset.Update({{type}}.ODataEditLink, *{{type}}, values)",
				Code:       e.Code,
				Details:    e.Details,
				ErrorNo:    e.ErrorNo,
				Exit:       "{{exit01}}",
				FileName:   e.FileName,
				Function:   function,
				InnerError: err,
				LineNumber: e.LineNumber,
				Message:    message,
				Payload:    nil,
				RequestUrl: e.RequestUrl,
				User:       nil,
			}
			return result, m
		}
		err = nullable.SetSelectedBooleanFields(reflect.ValueOf(&result), selectedFields, true, true)
		if err != nil {
			m := ErrorMessage{
				Attempted:  SET_NULLABLE_BOOLEAN_FIELDS,
				Details:    fmt.Sprintf("%+v", err),
				ErrorNo:    http.StatusInternalServerError,
				Exit:		"{{exit02}}",
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
	result = strings.ReplaceAll(result, "{{exit01}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit02}}", rightUUID(12, false))

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
				Attempted: 	"json.Unmarshal",
				Details: 	fmt.Sprintf("Error: %+v", err),
				ErrorNo: 	http.StatusBadRequest,
				Exit:		"{{exit01}}",
				Function: 	function,
				Message: 	BAD_REQUEST,
				UnixTimestamp: time.Now().Unix(),
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
				ee := ExtractError(err)
				message := UNEXPECTED_ERROR
				s, ok := ee.Message.(string)
				if ok {
					message = s
				}
				m := ErrorMessage{
					Attempted:  "{{type}}.Save(headers,link,values)",
					Details:    ee.Details,
					ErrorNo:    ee.ErrorNo,
					Exit: 		"{{exit02""}},
					FileName:   ee.FileName,
					Function:   function,
					InnerError: err,
					LineNumber: ee.LineNumber,
					Message:    message,
					Payload: 	nil,
					RequestUrl: ee.RequestUrl,
					UnixTimestamp: time.Now().Unix(),
				}
				ee.RequestUrl = ""
				messages = append(messages, m)
				continue
			}
			if len(selectFields) == 0 {
				selectFields = nullable.GetSelectedTags(saved, false)
				selectFields = append(selectFields, ExtraFields()...)
			}
			output = append(output,saved)
		}

		mapped, err := nullable.StructSetToMapSet(output,selectFields)
		if err != nil {
			ee := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := ee.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:     	"nullable.SetToMapSet(output)",
				Details:       	ee.Details,
				ErrorNo:       	ee.ErrorNo,
				Exit:			"{{exit03}}",
				FileName:    	ee.FileName,
				Function:      	function,
				InnerError:    	err,
				LineNumber:   	ee.LineNumber,
				Message:       	message,
				Payload: 		nil,
				RequestUrl: 	ee.RequestUrl,
				UnixTimestamp: time.Now().Unix(),
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

	result = strings.ReplaceAll(result, "{{exit01}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit02}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit03}}", rightUUID(12, false))

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
	
		function := "{{publicName}}.SetSelected"

		err := nullable.SetSelectedBooleanFields(reflect.ValueOf({{type}}), fields, value, not)
		if err != nil {
			m := ErrorMessage{
				Attempted: 	"nullable.SetSelectedBooleanFields",
				Details: 	fmt.Sprintf("%+v",err),
				ErrorNo: 	http.StatusInternalServerError,
				Exit:		"{{exit01}}",
				Function: 	function,
				Message: 	"unexpected error",
			}
			return m
		}

		return nil
	}

	func {{publicName}}Node(defaultFilter string, values url.Values, headers map[string]string, link string) ({{publicName}}, error) {

		function := "{{publicName}}Node"

		found, err := {{publicName}}Set(defaultFilter, values, headers, link)
		
		if err != nil {
			ee := ExtractError(err)
			message := UNEXPECTED_ERROR
			s, ok := ee.Message.(string)
			if ok {
				message = s
			}
			m := ErrorMessage{
				Attempted:      "{{publicName}}Set",
				Code:         	ee.Code,
				Details:       	ee.Details,
				ErrorNo:       	ee.ErrorNo,
				Exit:       	"{{exit02}}",
				FileName:    	ee.FileName,
				Function:      	function,
				InnerError:    	err,
				LineNumber:   	ee.LineNumber,
				Message:       	message,
				Payload: 		nil,
				RequestUrl: 	ee.RequestUrl,
				User: 			nil,
			}
			ee.RequestUrl = ""
			return {{publicName}}{}, m
		}
		
		if len(found) == 0 {
			filter := defaultFilter
			if values.Get(FILTER) != "" {
				filter = fmt.Sprintf("( %s ) and ( %s )", defaultFilter, values.Get(FILTER))
			}
			return {{publicName}}{}, NilModel{Model: "{{publicName}}", Filter: filter, Exit: "{{exit03}}"}
		}

		return found[0], nil
	}
	
	// {{publicName}}Set
	func {{publicName}}Set(defaultFilter string, values url.Values, headers map[string]string, link string) ([]{{publicName}}, error) {


		{{g.Package.OdataAlias}} := {{g.Package.OdataAlias}}.New(link)
		for key, value := range headers {
			{{g.Package.OdataAlias}}.AddHeader(key, value)
		}
		options := {{g.Package.OdataAlias}}.ODataQueryOptions()
		options = options.ApplyArguments(defaultFilter, values)

		collection := New{{publicName}}Collection({{g.Package.OdataAlias}})
		dataset := collection.DataSet()

		metaCh, dataCh, errCh := dataset.Set(options)

		var (
			models   []{{publicName}}
			firstErr error
		)

		// Consume all channels concurrently via select.
		for metaCh != nil || dataCh != nil || errCh != nil {
			select {
			case err, ok := <-errCh:
				if !ok {
					errCh = nil
					continue
				}
				if err != nil && firstErr == nil {
					firstErr = err
				}

			case _, ok := <-metaCh:
				if !ok {
					metaCh = nil
				}

			case model, ok := <-dataCh:
				if !ok {
					dataCh = nil
					continue
				}
				models = append(models, model)
			}
		}

		if firstErr != nil {
			return nil, firstErr
		}
		return models, nil
	}`
	result = strings.ReplaceAll(result, "{{publicName}}", publicName)
	result = strings.ReplaceAll(result, "{{g.Package.OdataAlias}}", g.Package.OdataAlias)
	result = strings.ReplaceAll(result, "{{g.Package.Name}}", g.Package.Name)
	runes := []rune(publicName)
	firstLower := unicode.ToLower(runes[0])
	result = strings.ReplaceAll(result, "{{type}}", string(firstLower))
	result = strings.ReplaceAll(result, "{{exit01}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit02}}", rightUUID(12, false))
	result = strings.ReplaceAll(result, "{{exit03}}", rightUUID(12, false))
	return result
}

func (g Generator) SelectByTableName(set edmxEntitySet, options string) string {

	entityType := set.getEntityType()
	publicName := publicAttribute(entityType.Name)
	result := `

	case "{{databaseName}}":
		collection := New{{publicName}}Collection(client)
		dataset := collection.DataSet()
		meta, data, errs := dataset.Set(options)
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
