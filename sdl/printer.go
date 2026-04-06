package sdl

import (
	"sort"
	"strings"
	"text/template"
)

// Printer converts introspection data to SDL format.
type Printer struct {
	schema Schema
	indent string
}

// NewPrinter creates a new SDL printer for the given schema.
func NewPrinter(schema Schema) *Printer {
	return &Printer{
		schema: schema,
		indent: "  ",
	}
}

// Print generates the SDL string for the schema.
func (p *Printer) Print() string {
	var sb strings.Builder

	// Print schema definition if we have root types
	p.printSchemaDefinition(&sb)

	// Group types by kind for organized output
	enums := []FullType{}
	scalars := []FullType{}
	objects := []FullType{}
	interfaces := []FullType{}
	unions := []FullType{}
	inputObjects := []FullType{}

	for _, t := range p.schema.Types {
		// Skip built-in types and introspection types
		if p.isBuiltInType(t.Name) {
			continue
		}
		if strings.HasPrefix(t.Name, "__") {
			continue
		}

		switch t.Kind {
		case ScalarKind:
			scalars = append(scalars, t)
		case EnumKind:
			enums = append(enums, t)
		case ObjectKind:
			objects = append(objects, t)
		case InterfaceKind:
			interfaces = append(interfaces, t)
		case UnionKind:
			unions = append(unions, t)
		case InputObjectKind:
			inputObjects = append(inputObjects, t)
		}
	}

	// Print in order: scalars, enums, interfaces, unions, input objects, objects
	for _, t := range scalars {
		sb.WriteString(p.printScalar(t))
	}
	for _, t := range enums {
		sb.WriteString(p.printEnum(t))
	}
	for _, t := range interfaces {
		sb.WriteString(p.printInterface(t))
	}
	for _, t := range unions {
		sb.WriteString(p.printUnion(t))
	}
	for _, t := range inputObjects {
		sb.WriteString(p.printInputObject(t))
	}
	for _, t := range objects {
		sb.WriteString(p.printObject(t))
	}

	return sb.String()
}

// isBuiltInType checks if a type is a GraphQL built-in type.
func (p *Printer) isBuiltInType(name string) bool {
	builtins := map[string]bool{
		"String":              true,
		"Int":                 true,
		"Float":               true,
		"Boolean":             true,
		"ID":                  true,
		"__Schema":            true,
		"__Type":              true,
		"__Field":             true,
		"__InputValue":        true,
		"__EnumValue":         true,
		"__Directive":         true,
		"__TypeKind":          true,
		"__DirectiveLocation": true,
	}
	return builtins[name]
}

// formatDescription formats a description for SDL output.
func (p *Printer) formatDescription(description string, indent string) string {
	if description == "" {
		return ""
	}

	// Check if description contains newlines or quotes
	if strings.Contains(description, "\n") || strings.Contains(description, "\"") {
		// Use block string
		var sb strings.Builder
		sb.WriteString(indent)
		sb.WriteString("\"\"\"\n")
		lines := strings.Split(description, "\n")
		for _, line := range lines {
			sb.WriteString(indent)
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(indent)
		sb.WriteString("\"\"\"\n")
		return sb.String()
	}

	// Use single line string
	return indent + "\"" + description + "\"\n"
}

// printTypeRef converts a TypeRef to its SDL string representation.
func (p *Printer) printTypeRef(t TypeRef) string {
	switch t.Kind {
	case NonNullKind:
		if t.OfType != nil {
			return p.printTypeRef(*t.OfType) + "!"
		}
	case ListKind:
		if t.OfType != nil {
			return "[" + p.printTypeRef(*t.OfType) + "]"
		}
	default:
		if t.Name != "" {
			return t.Name
		}
		if t.OfType != nil {
			return p.printTypeRef(*t.OfType)
		}
	}
	return ""
}

// printSchemaDefinition prints the schema block if needed.
func (p *Printer) printSchemaDefinition(sb *strings.Builder) {
	var rootTypes []string

	if p.schema.QueryType != nil && p.schema.QueryType.Name != "" {
		rootTypes = append(rootTypes, p.indent+"query: "+p.schema.QueryType.Name)
	}

	if p.schema.MutationType != nil && p.schema.MutationType.Name != "" {
		rootTypes = append(rootTypes, p.indent+"mutation: "+p.schema.MutationType.Name)
	}

	if p.schema.SubscriptionType != nil && p.schema.SubscriptionType.Name != "" {
		rootTypes = append(rootTypes, p.indent+"subscription: "+p.schema.SubscriptionType.Name)
	}

	if len(rootTypes) > 0 {
		sb.WriteString("schema {\n")
		for _, rt := range rootTypes {
			sb.WriteString(rt)
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}
}

// printScalar prints a scalar type definition.
func (p *Printer) printScalar(t FullType) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(t.Description, ""))

	if t.SpecifiedByURL != nil && *t.SpecifiedByURL != "" {
		sb.WriteString("scalar " + t.Name + " @specifiedBy(url: \"" + *t.SpecifiedByURL + "\")\n\n")
	} else {
		sb.WriteString("scalar " + t.Name + "\n\n")
	}

	return sb.String()
}

// printEnum prints an enum type definition.
func (p *Printer) printEnum(t FullType) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(t.Description, ""))
	sb.WriteString("enum " + t.Name + " {\n")

	for _, v := range t.EnumValues {
		sb.WriteString(p.formatDescription(v.Description, p.indent))
		if v.IsDeprecated {
			reason := ""
			if v.DeprecationReason != nil {
				reason = *v.DeprecationReason
			}
			sb.WriteString(p.indent + v.Name + " @deprecated(reason: \"" + reason + "\")\n")
		} else {
			sb.WriteString(p.indent + v.Name + "\n")
		}
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

// printInterface prints an interface type definition.
func (p *Printer) printInterface(t FullType) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(t.Description, ""))
	sb.WriteString("interface " + t.Name)

	// Print implemented interfaces
	if len(t.Interfaces) > 0 {
		implements := []string{}
		for _, iface := range t.Interfaces {
			if iface.Name != "" {
				implements = append(implements, iface.Name)
			}
		}
		if len(implements) > 0 {
			sb.WriteString(" implements " + strings.Join(implements, " & "))
		}
	}

	sb.WriteString(" {\n")

	for _, f := range t.Fields {
		sb.WriteString(p.printField(f, p.indent))
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

// printUnion prints a union type definition.
func (p *Printer) printUnion(t FullType) string {
	var sb strings.Builder

	types := []string{}
	for _, pt := range t.PossibleTypes {
		if pt.Name != "" {
			types = append(types, pt.Name)
		}
	}

	if len(types) > 0 {
		sort.Strings(types)
		sb.WriteString(p.formatDescription(t.Description, ""))
		sb.WriteString("union " + t.Name + " = " + strings.Join(types, " | ") + "\n\n")
	}

	return sb.String()
}

// printInputObject prints an input object type definition.
func (p *Printer) printInputObject(t FullType) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(t.Description, ""))

	// Check for @oneOf directive
	hasOneOf := false
	for _, d := range t.Directives {
		if d.Name == "oneOf" {
			hasOneOf = true
			break
		}
	}

	if hasOneOf {
		sb.WriteString("input " + t.Name + " @oneOf {\n")
	} else {
		sb.WriteString("input " + t.Name + " {\n")
	}

	for _, f := range t.InputFields {
		sb.WriteString(p.printInputValue(f, p.indent))
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

// printObject prints an object type definition.
func (p *Printer) printObject(t FullType) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(t.Description, ""))
	sb.WriteString("type " + t.Name)

	// Print implemented interfaces
	if len(t.Interfaces) > 0 {
		implements := []string{}
		for _, iface := range t.Interfaces {
			if iface.Name != "" {
				implements = append(implements, iface.Name)
			}
		}
		if len(implements) > 0 {
			sb.WriteString(" implements " + strings.Join(implements, " & "))
		}
	}

	sb.WriteString(" {\n")

	for _, f := range t.Fields {
		sb.WriteString(p.printField(f, p.indent))
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

// printField prints a field definition.
func (p *Printer) printField(f Field, indent string) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(f.Description, indent))
	sb.WriteString(indent + f.Name)

	// Print arguments if any
	if len(f.Args) > 0 {
		sb.WriteString("(")
		for i, arg := range f.Args {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(p.printArgument(arg))
		}
		sb.WriteString(")")
	}

	sb.WriteString(": " + p.printTypeRef(f.Type))

	// Print deprecation directive if applicable
	if f.IsDeprecated {
		reason := ""
		if f.DeprecationReason != nil {
			reason = *f.DeprecationReason
		}
		sb.WriteString(" @deprecated(reason: \"" + reason + "\")")
	}

	sb.WriteString("\n")
	return sb.String()
}

// printArgument prints an argument definition.
func (p *Printer) printArgument(arg InputValue) string {
	var sb strings.Builder
	sb.WriteString(arg.Name + ": " + p.printTypeRef(arg.Type))

	if arg.DefaultValue != nil {
		sb.WriteString(" = " + *arg.DefaultValue)
	}

	if arg.IsDeprecated {
		reason := ""
		if arg.DeprecationReason != nil {
			reason = *arg.DeprecationReason
		}
		sb.WriteString(" @deprecated(reason: \"" + reason + "\")")
	}

	return sb.String()
}

// printInputValue prints an input field definition.
func (p *Printer) printInputValue(v InputValue, indent string) string {
	var sb strings.Builder
	sb.WriteString(p.formatDescription(v.Description, indent))
	sb.WriteString(indent + v.Name + ": " + p.printTypeRef(v.Type))

	if v.DefaultValue != nil {
		sb.WriteString(" = " + *v.DefaultValue)
	}

	if v.IsDeprecated {
		reason := ""
		if v.DeprecationReason != nil {
			reason = *v.DeprecationReason
		}
		sb.WriteString(" @deprecated(reason: \"" + reason + "\")")
	}

	sb.WriteString("\n")
	return sb.String()
}

// Template data structures for template-based rendering

// SchemaTemplateData holds data for schema template rendering.
type SchemaTemplateData struct {
	QueryType        *string
	MutationType     *string
	SubscriptionType *string
	Scalars          []TypeTemplateData
	Enums            []TypeTemplateData
	Interfaces       []TypeTemplateData
	Unions           []TypeTemplateData
	InputObjects     []TypeTemplateData
	Objects          []TypeTemplateData
}

// TypeTemplateData holds data for a single type template rendering.
type TypeTemplateData struct {
	Name           string
	Description    string
	SpecifiedByURL string
	IsOneOf        bool
	Implements     []string
	Fields         []FieldTemplateData
	InputFields    []InputFieldTemplateData
	EnumValues     []EnumValueTemplateData
	PossibleTypes  []string
}

// FieldTemplateData holds data for a field template rendering.
type FieldTemplateData struct {
	Name              string
	Description       string
	Args              []ArgTemplateData
	Type              string
	IsDeprecated      bool
	DeprecationReason string
}

// ArgTemplateData holds data for an argument template rendering.
type ArgTemplateData struct {
	Name              string
	Type              string
	DefaultValue      string
	IsDeprecated      bool
	DeprecationReason string
}

// InputFieldTemplateData holds data for an input field template rendering.
type InputFieldTemplateData struct {
	Name              string
	Description       string
	Type              string
	DefaultValue      string
	IsDeprecated      bool
	DeprecationReason string
}

// EnumValueTemplateData holds data for an enum value template rendering.
type EnumValueTemplateData struct {
	Name              string
	Description       string
	IsDeprecated      bool
	DeprecationReason string
}

// Template functions
var templateFuncs = template.FuncMap{
	"formatDescription": formatDescription,
	"formatType":        formatType,
}

// formatDescription formats a description for SDL output using templates.
func formatDescription(description string, indent string) string {
	if description == "" {
		return ""
	}

	// Check if description contains newlines or quotes
	if strings.Contains(description, "\n") || strings.Contains(description, "\"") {
		// Use block string
		var sb strings.Builder
		sb.WriteString(indent)
		sb.WriteString("\"\"\"\n")
		lines := strings.Split(description, "\n")
		for _, line := range lines {
			sb.WriteString(indent)
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(indent)
		sb.WriteString("\"\"\"\n")
		return sb.String()
	}

	// Use single line string
	return indent + "\"" + description + "\"\n"
}

// formatType formats a TypeRef for SDL output.
func formatType(t TypeRef) string {
	switch t.Kind {
	case NonNullKind:
		if t.OfType != nil {
			return formatType(*t.OfType) + "!"
		}
	case ListKind:
		if t.OfType != nil {
			return "[" + formatType(*t.OfType) + "]"
		}
	default:
		if t.Name != "" {
			return t.Name
		}
		if t.OfType != nil {
			return formatType(*t.OfType)
		}
	}
	return ""
}
