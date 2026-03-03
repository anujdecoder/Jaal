package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

var (
	schemaTmpl       *template.Template
	scalarTmpl       *template.Template
	enumTmpl         *template.Template
	objectTmpl       *template.Template
	interfaceTmpl    *template.Template
	unionTmpl        *template.Template
	inputObjectTmpl  *template.Template
)

func init() {
	funcs := template.FuncMap{
		"formatDescription": formatDescription,
	}

	schemaTmpl = template.Must(template.New("schema").Funcs(funcs).Parse(schemaTemplate))
	scalarTmpl = template.Must(template.New("scalar").Funcs(funcs).Parse(scalarTemplate))
	enumTmpl = template.Must(template.New("enum").Funcs(funcs).Parse(enumTemplate))
	objectTmpl = template.Must(template.New("object").Funcs(funcs).Parse(objectTemplate))
	interfaceTmpl = template.Must(template.New("interface").Funcs(funcs).Parse(interfaceTemplate))
	unionTmpl = template.Must(template.New("union").Funcs(funcs).Parse(unionTemplate))
	inputObjectTmpl = template.Must(template.New("inputObject").Funcs(funcs).Parse(inputObjectTemplate))
}

func formatDescription(desc string, indent string) string {
	if desc == "" {
		return ""
	}
	lines := strings.Split(desc, "\n")
	if len(lines) == 1 {
		return fmt.Sprintf("%s\"\"\"%s\"\"\"\n", indent, lines[0])
	}
	var sb strings.Builder
	sb.WriteString(indent)
	sb.WriteString("\"\"\"\n")
	for _, line := range lines {
		sb.WriteString(indent)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString(indent)
	sb.WriteString("\"\"\"\n")
	return sb.String()
}

func ConvertToSDL(resp IntrospectionResponse) string {
	var sb strings.Builder

	var schemaBuf bytes.Buffer
	if err := schemaTmpl.Execute(&schemaBuf, resp.Schema); err != nil {
		panic(err)
	}
	sb.WriteString(schemaBuf.String())

	for _, t := range resp.Schema.Types {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}

		var buf bytes.Buffer
		var err error
		switch t.Kind {
		case "SCALAR":
			if t.Name == "String" || t.Name == "Int" || t.Name == "Float" || t.Name == "Boolean" || t.Name == "ID" {
				continue
			}
			err = scalarTmpl.Execute(&buf, t)
		case "ENUM":
			err = enumTmpl.Execute(&buf, t)
		case "OBJECT":
			err = objectTmpl.Execute(&buf, t)
		case "INTERFACE":
			err = interfaceTmpl.Execute(&buf, t)
		case "UNION":
			err = unionTmpl.Execute(&buf, t)
		case "INPUT_OBJECT":
			err = inputObjectTmpl.Execute(&buf, t)
		}

		if err != nil {
			panic(err)
		}
		sb.WriteString(buf.String())
	}

	return sb.String()
}
