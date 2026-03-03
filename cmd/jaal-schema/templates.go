package main

const schemaTemplate = `schema {
{{- if .QueryType }}
  query: {{ .QueryType.Name }}
{{- end }}
{{- if .MutationType }}
  mutation: {{ .MutationType.Name }}
{{- end }}
{{- if .SubscriptionType }}
  subscription: {{ .SubscriptionType.Name }}
{{- end }}
}

`

const scalarTemplate = `{{ formatDescription .Description "" -}}
scalar {{ .Name }}{{ if .SpecifiedBy }} @specifiedBy(url: "{{ .SpecifiedBy }}"){{ end }}

`

const enumTemplate = `{{ formatDescription .Description "" -}}
enum {{ .Name }} {
{{- range .EnumValues }}
{{ formatDescription .Description "  " -}}
  {{ .Name }}{{ if .IsDeprecated }} @deprecated(reason: "{{ .DeprecationReason }}"){{ end }}
{{- end }}
}

`

const objectTemplate = `{{ formatDescription .Description "" -}}
type {{ .Name }}{{ if .Interfaces }} implements{{ range $i, $v := .Interfaces }}{{ if $i }} &{{ end }} {{ $v.String }}{{ end }}{{ end }} {
{{- range .Fields }}
{{ formatDescription .Description "  " -}}
  {{ .Name }}{{ if .Args }}({{ range $i, $a := .Args }}{{ if $i }}, {{ end }}{{ $a.Name }}: {{ $a.Type.String }}{{ if $a.DefaultValue }} = {{ $a.DefaultValue }}{{ end }}{{ end }}){{ end }}: {{ .Type.String }}{{ if .IsDeprecated }} @deprecated(reason: "{{ .DeprecationReason }}"){{ end }}
{{- end }}
}

`

const interfaceTemplate = `{{ formatDescription .Description "" -}}
interface {{ .Name }} {
{{- range .Fields }}
{{ formatDescription .Description "  " -}}
  {{ .Name }}{{ if .Args }}({{ range $i, $a := .Args }}{{ if $i }}, {{ end }}{{ $a.Name }}: {{ $a.Type.String }}{{ if $a.DefaultValue }} = {{ $a.DefaultValue }}{{ end }}{{ end }}){{ end }}: {{ .Type.String }}{{ if .IsDeprecated }} @deprecated(reason: "{{ .DeprecationReason }}"){{ end }}
{{- end }}
}

`

const unionTemplate = `{{ formatDescription .Description "" -}}
union {{ .Name }} = {{ range $i, $v := .PossibleTypes }}{{ if $i }} | {{ end }}{{ $v.String }}{{ end }}

`

const inputObjectTemplate = `{{ formatDescription .Description "" -}}
input {{ .Name }}{{ range .Directives }}{{ if eq .Name "oneOf" }} @oneOf{{ end }}{{ end }} {
{{- range .InputFields }}
{{ formatDescription .Description "  " -}}
  {{ .Name }}: {{ .Type.String }}{{ if .DefaultValue }} = {{ .DefaultValue }}{{ end }}{{ if .IsDeprecated }} @deprecated(reason: "{{ .DeprecationReason }}"){{ end }}
{{- end }}
}

`
