package autogen

import (
	"fmt"
	"github.com/graph-gophers/graphql-go/internal/common"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func GenFile(inputFile string, outputDir string, packageName string) error  {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	inputStr :=string(data)
	outputStr, err := GenString(inputStr)
	if err != nil {
		return err
	}
	_, fileName := filepath.Split(file.Name())
	ext := filepath.Ext(file.Name())
	fileName = strings.TrimSuffix(fileName, ext)
	fileName += ".go"
	outputFile := filepath.Join(outputDir + fileName)
	wf, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer wf.Close()
	_, err = wf.WriteString("package " + packageName + "\n")
	if err != nil {
		return err
	}
	_, err = wf.WriteString(outputStr)
	return err
}

func GenString(inputSchema string) (string, error)  {
	s, err := ParseSchema(inputSchema)
	if err != nil {
		return "",err
	}
	output := GenSchema(s)
	return output, nil
}

func GenSchema(s *schema.Schema) string  {
	var types[]string
	for _, t := range s.Types {
		obj, ok := t.(*schema.Object)
		if ok && isUserDefinedObject(obj) {
			types = append(types, GenObject(obj))
		}
		interf, ok := t.(*schema.Interface)
		if ok  {
			types = append(types, GenInterface(interf))
		}
	}
	return strings.Join(types, "\n")
}

func isUserDefinedObject(t *schema.Object) bool  {
	if !strings.HasPrefix(t.TypeName(), "__") {
		return true
	}
	return false
}

func GenObject(t *schema.Object) string {
	var output string
	if len(t.Desc) != 0 {
		output = "// " + t.Desc + "\n"
	}
	output += fmt.Sprintf("type %s struct {\n", t.TypeName())
	ident := "    "
	for _, f := range t.Fields {
		output += ident + GetFieldDef(f) + "\n"
	}
	output += fmt.Sprintf("}\n")
	return output
}

func GenInterface(t *schema.Interface) string {
	var output string
	output = fmt.Sprintf("type %s struct {\n", t.TypeName())
	ident := "    "
	for _, f := range t.Fields {
		output += ident + GetFieldDef(f) + "\n"
	}
	output += fmt.Sprintf("}\n")
	return output
}

func GetFieldDef(f *schema.Field) string {
	def := GetFileName(f) + " " + GetFieldTypeName(f)
	def += fmt.Sprintf("`json:\"%s\" form:\"%s\" desc:\"%s\"`", f.Name, f.Name, f.Desc) // tag
	if len(f.Desc) > 0 {
		def +=  " // " + f.Desc
	}
	return def
}

func GetFileName(f *schema.Field) string {
	words := strings.Split(f.Name, "_")
	newName := ""
	for _, word := range words {
		newName += UpperWord(word)
	}
	return newName
}

func UpperWord(w string) string  {
	special := map[string]string{
		"id": "ID",
		"ip": "IP",
	}
	if s, ok := special[w]; ok {
		return s
	}
	upper := strings.ToUpper(w[0:1])
	if len(w) > 1 {
		upper += w[1:]
	}
	return upper
}

func GetFieldTypeName(f *schema.Field) string {
	_, name, _ := GetRealGolangTypeName(f.Type, nil, "")
	return name
}

//func GetGraphqlFieldTypeName(f *schema.Field) string {
//	typeName := ""
//	realType := GetRealType(f.Type)
//	typeName += realType.String()
//	return typeName
//}

func GetRealGolangTypeName(t common.Type, parentType common.Type, golangType string) (common.Type, string, common.Type) {
	kind := t.Kind()
	switch kind {
	case "NON_NULL":
		t, _ := t.(*common.NonNull)
		return GetRealGolangTypeName(t.OfType, t, golangType)
	case "LIST":
		t, _ := t.(*common.List)
		return GetRealGolangTypeName(t.OfType, t, golangType + "[]")
	default:
		pointer := ""
		if parentType != nil {
			if _, ok := parentType.(*common.NonNull); !ok {
				pointer = "*"
			}
		} else {
			pointer = "*"
		}
		return t, golangType + pointer + GetGolangTypeName(t.String()), t
	}
}

//func GetRealType(t common.Type) common.Type {
//	kind := t.Kind()
//	switch kind {
//	case "NON_NULL":
//		t, _ := t.(*common.NonNull)
//		return GetRealType(t.OfType)
//	case "LIST":
//		t, _ := t.(*common.List)
//		return GetRealType(t.OfType)
//	default:
//		return t
//	}
//}

func GetGolangTypeName(graphqlType string) string {
	m := map[string]string{
		"String": "string",
		"Int": "int32",
		"Boolean": "bool",
	}
	if n, ok := m[graphqlType]; ok {
		return n
	}
	return graphqlType
}

func GetNonNullFieldType(f *common.NonNull) string {
	return f.OfType.String()
}

func GetListFieldType(f *common.List) string {
	return f.OfType.String()
}

func ParseSchema(schemaString string) (*schema.Schema,error) {
	s := schema.New()

	if err := s.Parse(schemaString, false); err != nil {
		return nil, err
	}
	return s, nil
}