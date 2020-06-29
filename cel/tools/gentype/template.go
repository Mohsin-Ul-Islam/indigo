package main

var src = `
// Code generated DO NOT EDIT
// Generated by the Indigo gentype tool
package {{ .PackageName }}

import (
    "reflect"
    "fmt"
    "time"

    "github.com/ezachrisen/indigo/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
    "google.golang.org/protobuf/types/known/timestamppb"
    "github.com/golang/protobuf/ptypes"

	{{ range $key, $value :=  .Imports }}
	{{ $value }} "{{ $key }}"
	{{ end }}
)

func dummyFunc() {
    t := time.Now()
    d := time.Since(t)
    ptypes.DurationProto(d)
}
var dummyTime timestamppb.Timestamp 

{{$packageName:=.PackageName}}

{{ range .Objects }}

    var {{ .Name }}Type = types.NewTypeValue("{{.TypeID}}", traits.IndexerType)

    func (v {{ .Name }}) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	//log.Println("{{ .Name }}ConvertToNative")
	return nil, fmt.Errorf("cannot convert attribute message to native types")
    }
    
    func (v {{ .Name }}) ConvertToType(typeValue ref.Type) ref.Val {
	//log.Println("{{ .Name }}.ConvertToType")
	return types.NewErr("cannot convert attribute message to CEL types")
    }
    
    func (v {{ .Name }}) Equal(other ref.Val) ref.Val {
	//log.Println("{{ .Name }}.Equal")
	return types.NewErr("attribute message does not support equality")
    }
    
    func (v {{ .Name }}) Type() ref.Type {
	//log.Println("{{ .Name }}.Type")
	return {{ .Name }}Type
    }
    
    func (v {{ .Name }}) Value() interface{} {
	//log.Println("{{ .Name }}.Value")
	return v
    }

    func (v {{ .Name }}) Get(index ref.Val) ref.Val {
	//log.Println("{{ .Name }}.Get ", index, index.Type(), index.Value(), index.Type().TypeName())
	field, ok := index.Value().(string)

	if !ok {
		return types.NewErr("Field %v not found in type %s", index.Value(), "{{ .Name }}")
	}

	switch field {

	{{ range .Fields  }}

	/* Code generation debug information
	    Type: {{ .Type }}
	    Name: {{ .Type.TypeName }}
	    Mult: {{ .Type.Multiple }}
	    ID  : {{ .Type.TypeID }}		
	    Obj : {{ .Type.IsObject }}
        Slc : {{ .Type.IsSlice }}
        Sle : {{ .Type.SliceElem }}
        Map : {{ .Type.IsMap }}
        Mk  : {{ .Type.MapKey }}
        Me  : {{ .Type.MapElem }}
	 */

	    case "{{ .Name }}":
         {{ if .Type.IsSlice }}
              return types.NewDynamicList(cel.AttributeProvider{}, v.{{ .Name}})
         {{ else if .Type.IsMap }}
              return types.NewDynamicMap(cel.AttributeProvider{}, v.{{ .Name}})
	     {{ else if eq .Type.TypeName  "string"  }}
	     	 return types.String(v.{{ .Name }})
	     {{ else if eq .Type.TypeName  "int" }}
	     	 return types.Int(v.{{ .Name }})
	     {{ else if eq .Type.TypeName  "float64" }}
	     	 return types.Double(v.{{ .Name }})
	     {{ else if eq .Type.TypeName  "time.Time" }}
              return types.Timestamp{timestamppb.New(v.{{ .Name }})}
	     {{ else if eq .Type.TypeName  "time.Duration" }}
              return types.Duration{ptypes.DurationProto(v.{{ .Name}})}
         {{ else if .Type.IsObject }}
              return v.{{ .Name }}
	     {{ end }}

         {{ end }} 

         default:
           return nil
       }
    }

    func (v {{ .Name }}) ProvideStructDefintion() cel.StructDefinition {
    	 return cel.StructDefinition{
	   Name: "{{.TypeID}}",
	   Fields: map[string]*ref.FieldType{
	      {{ range .Fields }}
	      "{{ .Name }}": 
                 {{ if .Type.IsSlice }} 
                    &ref.FieldType{Type: decls.NewListType(decls.Any)},
                 {{ else if .Type.IsMap }}
                    &ref.FieldType{Type: decls.NewMapType(decls.Any, decls.Any)},
	      	     {{ else if eq .Type.TypeName  "string"  }}
	     	     	 &ref.FieldType{Type: decls.String},
	              {{ else if eq .Type.TypeName  "int" }}
	     	         &ref.FieldType{Type: decls.Int},
	              {{ else if eq .Type.TypeName  "float64" }}
     		         &ref.FieldType{Type: decls.Double}, 
	              {{ else if eq .Type.TypeName  "time.Time" }}
     		         &ref.FieldType{Type: decls.Timestamp}, 
	              {{ else if eq .Type.TypeName  "time.Duration" }}
     		         &ref.FieldType{Type: decls.Duration}, 
                 {{ else if .Type.IsObject }}
                    &ref.FieldType{Type: decls.NewObjectType("{{.Type.TypeID}}")},
 				 {{ end }}
		   {{ end }}			 
		  },		   			   	   
	   }
    }

{{end}}
`
