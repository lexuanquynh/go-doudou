package strategies

import (
	"github.com/unionj-cloud/go-doudou/kit/stringutils"
	"text/template"
	"unicode"
)

var LowerCaseNamingStrategyTemplate = template.Must(template.New("").Parse(templ))

func init() {
	Registry["lowerCaseNamingStrategy"] = LowerCaseNamingStrategyTemplate
}

func LowerCaseConvert(key string) (result string) {
	if stringutils.IsEmpty(key) {
		return
	}
	result = string(unicode.ToLower(rune(key[0]))) + key[1:]
	return
}

const templ = `// Code generated by go generate; DO NOT EDIT.
// This file was generated by go-doudou at
// {{ .Timestamp }}
package {{ .StructCollector.Package.Name }}

import (
	"encoding/json"
	"github.com/unionj-cloud/go-doudou/kit/namingstrategy/strategies"
)

{{ range $struct := .StructCollector.Structs }}
func (object {{$struct.Name}}) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	{{- range $field := $struct.Fields}}
	objectMap[strategies.LowerCaseConvert("{{$field.Name}}")] = object.{{$field.Name}}
	{{- end }}
	return json.Marshal(objectMap)
}
{{ end }}
`