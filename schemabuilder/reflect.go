package schemabuilder

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"unicode"

	"go.appointy.com/jaal/graphql"
)

// graphQLFieldInfo contains basic struct field information related to GraphQL.
// Per descriptions feature extension, supports Description for FIELD_DEFINITION
// (parsed from tag; "" default; to graphql.Field for __Field.description/Playground).
type graphQLFieldInfo struct {
	// Skipped indicates that this field should not be included in GraphQL.
	Skipped bool

	// Name is the GraphQL field name that should be exposed for this field.
	Name string

	// KeyField indicates that this field should be treated as a Object Key field.
	KeyField bool

	// OptionalInputField indicates that this field should be treated as an optional
	// field on graphQL input args.
	OptionalInputField bool

	// Description for FIELD_DEFINITION (spec; parsed from tag e.g., `graphql:"name,description=..."`;
	// "" default for BC; exposed in introspection).
	Description string
}

// parseGraphQLFieldInfo parses a struct field and returns the parsed GraphQL metadata.
// Supports description per spec via tag, e.g., `json:"name" graphql:"description=Fetch by ID"`.
// Tags only influence name, optionality, and description.
func parseGraphQLFieldInfo(field reflect.StructField) (*graphQLFieldInfo, error) {
	if field.PkgPath != "" { //If the field of struct is not exported, then it is not exposed
		return &graphQLFieldInfo{Skipped: true}, nil
	}

	// Primary tag from json (existing pattern); fallback/graphql tag for options.
	tag := field.Tag.Get("graphql")
	if tag == "" {
		tag = field.Tag.Get("json")
	}
	tags := strings.Split(tag, ",")
	var name string
	if len(tags) > 0 {
		name = strings.TrimSpace(tags[0])
	}
	if name == "-" {
		return &graphQLFieldInfo{Skipped: true}, nil
	}

	if name == "" {
		name = makeGraphql(field.Name)
	}

	var key bool
	var optional bool
	var description string
	for _, opt := range tags[1:] {
		opt = strings.TrimSpace(opt)
		if strings.HasPrefix(opt, "description=") {
			// Description for FIELD_DEFINITION (extension; e.g., `graphql:"name,description=Fetch by ID"`).
			description = strings.TrimPrefix(opt, "description=")
		} else if opt == "optional" {
			optional = true
		}
		// key/others extensible; omitted for minimal.
	}

	return &graphQLFieldInfo{Name: name, KeyField: key, OptionalInputField: optional, Description: description}, nil
}

// makeGraphql converts a field name "MyField" into a graphQL field name "myField".
func makeGraphql(s string) string {
	var b bytes.Buffer
	for i, c := range s {
		if i == 0 {
			b.WriteRune(unicode.ToLower(c))
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// Common Types that we will need to perform type assertions against.
var errType = reflect.TypeOf((*error)(nil)).Elem()
var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var selectionSetType = reflect.TypeOf(&graphql.SelectionSet{})
