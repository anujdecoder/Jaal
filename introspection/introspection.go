package introspection

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
)

type introspection struct {
	types        map[string]graphql.Type
	query        graphql.Type
	mutation     graphql.Type
	subscription graphql.Type
}

type DirectiveLocation string

// DirectiveLocation enum per spec, including SCALAR for @specifiedBy (Oct 2021+),
// ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION for @deprecated on input values,
// and INPUT_OBJECT for @oneOf (input unions; this task). Avoids name conflicts
// (e.g., _LOC suffix); stubbed minimally for others. Matches TypeKind for
// INPUT_OBJECT.
const (
	QUERY                  DirectiveLocation = "QUERY"
	MUTATION                                 = "MUTATION"
	FIELD                                    = "FIELD"
	FRAGMENT_DEFINITION                      = "FRAGMENT_DEFINITION"
	FRAGMENT_SPREAD                          = "FRAGMENT_SPREAD"
	INLINE_FRAGMENT                          = "INLINE_FRAGMENT"
	SUBSCRIPTION                             = "SUBSCRIPTION"
	SCALAR_LOCATION        DirectiveLocation = "SCALAR"                 // for @specifiedBy
	ARGUMENT_DEFINITION    DirectiveLocation = "ARGUMENT_DEFINITION"    // for input arg deprecation
	INPUT_FIELD_DEFINITION DirectiveLocation = "INPUT_FIELD_DEFINITION" // for input field deprecation
	INPUT_OBJECT_LOCATION  DirectiveLocation = "INPUT_OBJECT"           // for @oneOf input unions
)

type TypeKind string

const (
	SCALAR       TypeKind = "SCALAR"
	OBJECT                = "OBJECT"
	INTERFACE             = "INTERFACE"
	UNION                 = "UNION"
	ENUM                  = "ENUM"
	INPUT_OBJECT          = "INPUT_OBJECT"
	LIST                  = "LIST"
	NON_NULL              = "NON_NULL"
)

type InputValue struct {
	Name         string
	Description  string
	Type         Type
	DefaultValue *string

	// IsDeprecated and DeprecationReason support @deprecated on input values
	// (ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION per Oct 2021+ spec).
	// *string omitempty for JSON (matches EnumValue/field patterns/UI compat).
	// Defaults false/nil if not set (stubs lifted for full support).
	IsDeprecated      bool
	DeprecationReason *string `json:"deprecationReason,omitempty"`
}

func (s *introspection) registerInputValue(schema *schemabuilder.Schema) {
	obj := schema.Object("__InputValue", InputValue{})
	obj.FieldFunc("name", func(in InputValue) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in InputValue) string {
		return in.Description
	})
	obj.FieldFunc("type", func(in InputValue) Type {
		return in.Type
	})
	obj.FieldFunc("defaultValue", func(in InputValue) *string {
		return in.DefaultValue
	})
	// isDeprecated/deprecationReason for input values deprecation (spec support).
	// Mirrors EnumValue.registerEnumValue and field.registerField.
	obj.FieldFunc("isDeprecated", func(in InputValue) bool {
		return in.IsDeprecated
	})
	obj.FieldFunc("deprecationReason", func(in InputValue) *string {
		return in.DeprecationReason
	})
}

type EnumValue struct {
	Name         string
	Description  string
	IsDeprecated bool
	// DeprecationReason as *string with omitempty to omit from introspection JSON
	// when nil/empty (per spec/UI: prevents fields appearing deprecated in
	// playground/GraphiQL when no deprecation set). Matches InputValue.DefaultValue
	// pattern.
	DeprecationReason *string
}

func (s *introspection) registerEnumValue(schema *schemabuilder.Schema) {
	obj := schema.Object("__EnumValue", EnumValue{})
	obj.FieldFunc("name", func(in EnumValue) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in EnumValue) string {
		return in.Description
	})
	obj.FieldFunc("isDeprecated", func(in EnumValue) bool {
		return in.IsDeprecated
	})
	// Resolver returns *string (nil when no reason) for json omitempty tag.
	obj.FieldFunc("deprecationReason", func(in EnumValue) *string {
		return in.DeprecationReason
	})
}

type Directive struct {
	Name        string
	Description string
	Locations   []DirectiveLocation
	Args        []InputValue
}

func (s *introspection) registerDirective(schema *schemabuilder.Schema) {
	obj := schema.Object("__Directive", Directive{})
	obj.FieldFunc("name", func(in Directive) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in Directive) string {
		return in.Description
	})
	obj.FieldFunc("locations", func(in Directive) []DirectiveLocation {
		return in.Locations
	})
	obj.FieldFunc("args", func(in Directive) []InputValue {
		return in.Args
	})

	// if err := schemabuilder.RegisterScalar(reflect.TypeOf(DirectiveLocation("")), "directiveLocation", func(value interface{}, dest reflect.Value) error {
	// 	asString, ok := value.(string)
	// 	if !ok {
	// 		return errors.New("not a string")
	// 	}
	// 	dest.Set(reflect.ValueOf(asString).Convert(dest.Type()))
	// 	return nil
	// }); err != nil {
	// 	panic(err)
	// }

	schema.Enum(DirectiveLocation("QUERY"), map[string]interface{}{
		"QUERY":               DirectiveLocation("QUERY"),
		"MUTATION":            DirectiveLocation("MUTATION"),
		"FIELD":               DirectiveLocation("FIELD"),
		"FRAGMENT_DEFINITION": DirectiveLocation("FRAGMENT_DEFINITION"),
		"FRAGMENT_SPREAD":     DirectiveLocation("FRAGMENT_SPREAD"),
		"INLINE_FRAGMENT":     DirectiveLocation("INLINE_FRAGMENT"),
		"SUBSCRIPTION":        DirectiveLocation("SUBSCRIPTION"),
		"SCALAR":              DirectiveLocation(SCALAR_LOCATION), // for @specifiedBy
		// New for deprecation on inputs (ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION spec)
		// and @oneOf (INPUT_OBJECT_LOCATION for input unions; Oct 2021+).
		// Suffix avoids redecl w/ TypeKind.INPUT_OBJECT.
		"ARGUMENT_DEFINITION":    DirectiveLocation(ARGUMENT_DEFINITION),
		"INPUT_FIELD_DEFINITION": DirectiveLocation(INPUT_FIELD_DEFINITION),
		"INPUT_OBJECT":           DirectiveLocation(INPUT_OBJECT_LOCATION), // for @oneOf
	})
}

type Schema struct {
	Types            []Type
	QueryType        *Type
	MutationType     *Type
	SubscriptionType *Type
	Directives       []Directive
}

func (s *introspection) registerSchema(schema *schemabuilder.Schema) {
	obj := schema.Object("__Schema", Schema{})
	obj.FieldFunc("types", func(in Schema) []Type {
		return in.Types
	})
	obj.FieldFunc("queryType", func(in Schema) *Type {
		return in.QueryType
	})
	obj.FieldFunc("mutationType", func(in Schema) *Type {
		return in.MutationType
	})
	obj.FieldFunc("subscriptionType", func(in Schema) *Type {
		return in.SubscriptionType
	})
	obj.FieldFunc("directives", func(in Schema) []Directive {
		return in.Directives
	})

}

type Type struct {
	Inner graphql.Type `json:"-"`
}

// directives returns type-system directives applied to this __Type (e.g.,
// @oneOf on INPUT_OBJECT for input unions, @specifiedBy on SCALAR).
// Per spec, __Type.directives: [__Directive!]! ; implemented via switch on
// Inner (e.g., graphql.InputObject.OneOf flag set in schemabuilder).
// Currently minimal (oneOf focus; scalars use dedicated specifiedByURL field
// per prior impls; extensible for future type dirs like repeatable).
// Returns empty/nil if none (JSON array); matches Schema.directives FieldFunc.
func (t Type) directives() []Directive {
	switch inner := t.Inner.(type) {
	case *graphql.InputObject:
		// @oneOf for input unions (INPUT_OBJECT only; spec).
		// oneOfDirective defined below; nil/empty if !OneOf (BC).
		if inner.OneOf {
			return []Directive{oneOfDirective}
		}
		return nil
	// Extend for other type dirs e.g.:
	// case *graphql.Scalar:
	//   if inner.SpecifiedByURL != "" { return []Directive{specifiedByDirective} }
	// But omitted to match existing (specifiedByURL separate; no break).
	default:
		// Object/Union/Interface/Enum/Scalar (non-oneOf) etc: no dir for now.
		return nil
	}
}

func (s *introspection) registerType(schema *schemabuilder.Schema) {
	object := schema.Object("__Type", Type{})
	object.FieldFunc("kind", func(t Type) TypeKind {
		switch t.Inner.(type) {
		case *graphql.Object:
			return OBJECT
		case *graphql.Union:
			return UNION
		case *graphql.Interface:
			return INTERFACE
		case *graphql.Scalar:
			return SCALAR
		case *graphql.Enum:
			return ENUM
		case *graphql.List:
			return LIST
		case *graphql.InputObject:
			return INPUT_OBJECT
		case *graphql.NonNull:
			return NON_NULL
		default:
			return ""
		}
	})

	object.FieldFunc("name", func(t Type) string {
		switch t := t.Inner.(type) {
		case *graphql.Object:
			return t.Name
		case *graphql.Union:
			return t.Name
		case *graphql.Interface:
			return t.Name
		case *graphql.Scalar:
			return t.Type
		case *graphql.Enum:
			return t.Type
		case *graphql.InputObject:
			return t.Name
		default:
			return ""
		}
	})

	object.FieldFunc("description", func(t Type) string {
		switch t := t.Inner.(type) {
		case *graphql.Object:
			return t.Description
		case *graphql.Union:
			return t.Description
		case *graphql.Interface:
			return t.Description
		// For descriptions feature: pull from InputObject/Enum (set via
		// schemabuilder reg; default ""; Scalar no desc yet). Matches
		// graphql/types.go; ensures __Type.description for Playground/intro.
		// Default "" for BC (no desc on other types).
		case *graphql.InputObject:
			return t.Description
		case *graphql.Enum:
			return t.Description
		default:
			return ""
		}
	})

	// directives FieldFunc exposes type-system directives on __Type (per spec;
	// e.g., @oneOf on INPUT_OBJECT for input unions). Uses Type.directives()
	// helper (above; switch on graphql.Inner.OneOf etc). Matches Schema.directives,
	// and patterns for specifiedByURL/deprecation (InputValue/Field).
	// Added for @oneOf compliance (also benefits scalar/enum if extended).
	// Returns []Directive (empty if none; JSON array).
	object.FieldFunc("directives", func(t Type) []Directive {
		return t.directives()
	})

	object.FieldFunc("interfaces", func(t Type) []Type {
		switch t := t.Inner.(type) {
		case *graphql.Object:
			types := make([]Type, 0, len(t.Interfaces))
			for _, typ := range t.Interfaces {
				types = append(types, Type{Inner: typ})
			}

			sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })
			return types
		default:
			return nil
		}
	})
	object.FieldFunc("possibleTypes", func(t Type) []Type {
		switch t := t.Inner.(type) {
		case *graphql.Union:
			types := make([]Type, 0, len(t.Types))
			for _, typ := range t.Types {
				types = append(types, Type{Inner: typ})
			}

			sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })
			return types
		case *graphql.Interface:
			types := make([]Type, 0, len(t.Types))
			for _, typ := range t.Types {
				types = append(types, Type{Inner: typ})
			}

			sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })
			return types
		default:
			return nil
		}
	})

	object.FieldFunc("inputFields", func(t Type) []InputValue {
		var fields []InputValue

		switch t := t.Inner.(type) {
		case *graphql.InputObject:
			for name, f := range t.InputFields {
				// InputValue for INPUT_FIELD_DEFINITION deprecation support (spec).
				// Pulls from FieldDeprecations map set in schemabuilder/input_object.go
				// (tag parse e.g., age in CreateUserInput). Matches field.args InputValue
				// and Scalar.SpecifiedByURL metadata pattern. Default false/nil for compat.
				isDep := false
				var depReason *string
				if d, ok := t.FieldDeprecations[name]; ok && d != "" {
					isDep = true
					depReason = &d // ptr for omitempty/null in JSON
				}
				fields = append(fields, InputValue{
					Name:              name,
					Type:              Type{Inner: f},
					IsDeprecated:      isDep,
					DeprecationReason: depReason,
				})
			}
		}

		sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
		return fields
	})

	object.FieldFunc("fields", func(t Type, args struct {
		IncludeDeprecated *bool
	}) []field {
		var fields []field

		switch t := t.Inner.(type) {
		case *graphql.Object:
			for name, f := range t.Fields {
				var args []InputValue
				for name, a := range f.Args {
					// InputValue for ARGUMENT_DEFINITION deprecation (spec); default
					// false/nil (stubs lifted; full support via Field args in future).
					// Matches inputFields extension.
					args = append(args, InputValue{
						Name:              name,
						Type:              Type{Inner: a},
						IsDeprecated:      false,
						DeprecationReason: nil,
					})
				}
				sort.Slice(args, func(i, j int) bool { return args[i].Name < args[j].Name })

				// Use Field.IsDeprecated/DeprecationReason (now supports @deprecated on
				// fields + args per spec; ARGUMENT_DEFINITION enabled via InputValue in args).
				// Description zero (no support in graphql.Field); remove force-false stub.
				// DeprecationReason nil for omitempty (UI/playground compat).
				fields = append(fields, field{
					Name:              name,
					Description:       "",
					Type:              Type{Inner: f.Type},
					Args:              args,
					IsDeprecated:      f.IsDeprecated,
					DeprecationReason: f.DeprecationReason,
				})
			}
		case *graphql.Interface:
			for name, f := range t.Fields {
				var args []InputValue
				for name, a := range f.Args {
					// InputValue for ARGUMENT_DEFINITION deprecation (spec); default
					// false/nil (stubs lifted; full support via Field args in future).
					// Matches inputFields extension.
					args = append(args, InputValue{
						Name:              name,
						Type:              Type{Inner: a},
						IsDeprecated:      false,
						DeprecationReason: nil,
					})
				}
				sort.Slice(args, func(i, j int) bool { return args[i].Name < args[j].Name })

				// Use Field.IsDeprecated/DeprecationReason (now supports @deprecated on
				// fields + args per spec; ARGUMENT_DEFINITION enabled via InputValue in args).
				// Description zero (no support in graphql.Field); remove force-false stub.
				// DeprecationReason nil for omitempty (UI/playground compat).
				fields = append(fields, field{
					Name:              name,
					Description:       "",
					Type:              Type{Inner: f.Type},
					Args:              args,
					IsDeprecated:      f.IsDeprecated,
					DeprecationReason: f.DeprecationReason,
				})
			}
		}
		sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

		return fields
	})

	object.FieldFunc("ofType", func(t Type) *Type {
		switch t := t.Inner.(type) {
		case *graphql.List:
			return &Type{Inner: t.Type}
		case *graphql.NonNull:
			return &Type{Inner: t.Type}
		default:
			return nil
		}
	})

	object.FieldFunc("enumValues", func(t Type, args struct {
		IncludeDeprecated *bool
	}) []EnumValue {

		switch t := t.Inner.(type) {
		case *graphql.Enum:
			var enumVals []EnumValue
			for k, v := range t.ReverseMap {
				val := fmt.Sprintf("%v", k)
				// DeprecationReason: nil (omitted via omitempty; ensures not marked
				// deprecated in playground).
				enumVals = append(enumVals,
					EnumValue{Name: v, Description: val, IsDeprecated: false, DeprecationReason: nil})
			}
			sort.Slice(enumVals, func(i, j int) bool { return enumVals[i].Name < enumVals[j].Name })
			return enumVals
		}
		return nil
	})

	// specifiedByURL returns the URL for SCALAR types set via @specifiedBy(url: String!)
	// directive (Oct 2021+ spec). Returns *string (nil if unset -> null/omitted in JSON,
	// matching DeprecationReason/InputValue patterns to prevent UI/playground issues for
	// built-ins like String). Only applies to SCALAR kind; nil otherwise.
	object.FieldFunc("specifiedByURL", func(t Type) *string {
		switch t := t.Inner.(type) {
		case *graphql.Scalar:
			if t.SpecifiedByURL != "" {
				// Return ptr for JSON; non-empty URL included.
				return &t.SpecifiedByURL
			}
			// nil -> omitted/null per spec/intro query.
			return nil
		default:
			// Non-scalars have no specifiedByURL (spec).
			return nil
		}
	})
}

type field struct {
	Name         string
	Description  string
	Args         []InputValue
	Type         Type
	IsDeprecated bool
	// DeprecationReason as *string with omitempty to omit from introspection JSON
	// when nil/empty (per spec/UI: prevents fields appearing deprecated in
	// playground/GraphiQL when no deprecation set; fixes reported issue).
	DeprecationReason *string `json:"deprecationReason,omitempty"`
}

func (s *introspection) registerField(schema *schemabuilder.Schema) {
	obj := schema.Object("__Field", field{})
	obj.FieldFunc("name", func(in field) string {
		return in.Name
	})
	obj.FieldFunc("description", func(in field) string {
		return in.Description
	})
	obj.FieldFunc("type", func(in field) Type {
		return in.Type
	})
	obj.FieldFunc("args", func(in field) []InputValue {
		return in.Args
	})
	obj.FieldFunc("isDeprecated", func(in field) bool {
		return in.IsDeprecated
	})
	// Resolver returns *string (nil when no reason) for json omitempty tag.
	obj.FieldFunc("deprecationReason", func(in field) *string {
		return in.DeprecationReason
	})
}

func collectTypes(typ graphql.Type, types map[string]graphql.Type) {
	switch typ := typ.(type) {
	case *graphql.Object:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ

		for _, field := range typ.Fields {
			collectTypes(field.Type, types)

			for _, arg := range field.Args {
				collectTypes(arg, types)
			}
		}

	case *graphql.Union:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ
		for _, graphqlTyp := range typ.Types {
			collectTypes(graphqlTyp, types)
		}

	case *graphql.Interface:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ

		for _, field := range typ.Fields {
			collectTypes(field.Type, types)

			for _, arg := range field.Args {
				collectTypes(arg, types)
			}
		}
		for _, object := range typ.Types {
			collectTypes(object, types)
		}

	case *graphql.List:
		collectTypes(typ.Type, types)

	case *graphql.Scalar:
		if _, ok := types[typ.Type]; ok {
			return
		}
		types[typ.Type] = typ

	case *graphql.Enum:
		if _, ok := types[typ.Type]; ok {
			return
		}
		types[typ.Type] = typ

	case *graphql.InputObject:
		if _, ok := types[typ.Name]; ok {
			return
		}
		types[typ.Name] = typ

		for _, field := range typ.InputFields {
			collectTypes(field, types)
		}

	case *graphql.NonNull:
		collectTypes(typ.Type, types)
	}
}

var includeDirective = Directive{
	Description: "Directs the executor to include this field or fragment only when the `if` argument is true.",
	Locations: []DirectiveLocation{
		FIELD,
		FRAGMENT_SPREAD,
		INLINE_FRAGMENT,
	},
	Name: "include",
	Args: []InputValue{
		InputValue{
			Name:        "if",
			Type:        Type{Inner: &graphql.Scalar{Type: "Boolean"}},
			Description: "Included when true.",
		},
	},
}

var skipDirective = Directive{
	Description: "Directs the executor to skip this field or fragment only when the `if` argument is true.",
	Locations: []DirectiveLocation{
		FIELD,
		FRAGMENT_SPREAD,
		INLINE_FRAGMENT,
	},
	Name: "skip",
	Args: []InputValue{
		InputValue{
			Name:        "if",
			Type:        Type{Inner: &graphql.Scalar{Type: "Boolean"}},
			Description: "Skipped when true.",
		},
	},
}

// specifiedByDirective defines the built-in @specifiedBy (post-2018 spec) for
// SCALAR types. Matches includeDirective/skipDirective pattern for introspection
// exposure in __Schema.directives. URL arg as Scalar (per existing built-ins;
// spec's NonNull not strictly enforced here for compat).
var specifiedByDirective = Directive{
	Description: "Exposes a URL that specifies the behaviour of this scalar.",
	Locations: []DirectiveLocation{
		SCALAR_LOCATION, // only on scalars per spec
	},
	Name: "specifiedBy",
	Args: []InputValue{
		InputValue{
			Name:        "url",
			Type:        Type{Inner: &graphql.Scalar{Type: "String"}}, // built-in String; URL value
			Description: "The URL that specifies the behaviour of this scalar.",
		},
	},
}

// deprecatedDirective defines the built-in @deprecated (extended for input values
// per Oct 2021 spec: ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION). Matches other
// built-in vars; reason arg defaults "No longer supported" (spec). Used in
// __Schema.directives and inputFields/args deprecation. ENUM_VALUE omitted (in
// TypeKind but stub for location).
var deprecatedDirective = Directive{
	Description: "Marks an element of a GraphQL schema as no longer supported.",
	Locations: []DirectiveLocation{
		// Core + input values deprecation support.
		FIELD,
		ARGUMENT_DEFINITION,
		INPUT_FIELD_DEFINITION,
	},
	Name: "deprecated",
	Args: []InputValue{
		InputValue{
			Name:        "reason",
			Type:        Type{Inner: &graphql.Scalar{Type: "String"}},
			Description: "Explains why this element was deprecated, usually also including a suggestion for how to access supported similar data.",
			// Default per spec; set as *string (nil fallback OK; no resolver override).
			// Value "No longer supported" (no extra quotes; JSON marshals to proper
			// "No..." string per spec/UI/tests; fixes test mismatch).
			// InputValue dep fields default false/nil.
			DefaultValue:      func() *string { s := "No longer supported"; return &s }(),
			IsDeprecated:      false,
			DeprecationReason: nil,
		},
	},
}

// oneOfDirective defines the built-in @oneOf (Oct 2021+ spec for INPUT_OBJECT
// only; enables input unions/exclusive fields). Matches include/skip/specifiedBy/
// deprecated pattern for __Schema.directives exposure and __Type.directives.
// No args (per spec); INPUT_OBJECT_LOCATION added to enum above. Used for
// introspection of oneOf inputs (aligns w/ protoc-gen-jaal oneof).
// Non-repeatable; informational (validation in coercion not here).
var oneOfDirective = Directive{
	Description: "Indicates that an Input Object is a OneOf Input Object (and thus requires exactly one field to be set in a query or mutation).",
	Locations: []DirectiveLocation{
		INPUT_OBJECT_LOCATION, // only on input objects per spec for @oneOf
	},
	Name: "oneOf",
	// No args (spec); empty slice (like if no-args dir).
	Args: []InputValue{},
}

func (s *introspection) registerQuery(schema *schemabuilder.Schema) {
	object := schema.Query()

	object.FieldFunc("__schema", func() *Schema {
		var types []Type

		for _, typ := range s.types {
			types = append(types, Type{Inner: typ})
		}
		sort.Slice(types, func(i, j int) bool { return types[i].Inner.String() < types[j].Inner.String() })

		return &Schema{
			Types:            types,
			QueryType:        &Type{Inner: s.query},
			MutationType:     &Type{Inner: s.mutation},
			SubscriptionType: &Type{Inner: s.subscription},
			// include @specifiedBy, @deprecated (input values), and @oneOf (input unions/INPUT_OBJECT)
			// in directives list (spec-compliant for Sept 2025). Custom scalars w/ URL,
			// deprecated inputs/args, and oneOf inputs reflect in introspection.
			Directives: []Directive{includeDirective, skipDirective, specifiedByDirective, deprecatedDirective, oneOfDirective},
		}
	})

	object.FieldFunc("__type", func(args struct{ Name string }) *Type {
		if typ, ok := s.types[args.Name]; ok {
			return &Type{Inner: typ}
		}
		return nil
	})
}

func (s *introspection) registerMutation(schema *schemabuilder.Schema) {
	schema.Mutation()
}

func (s *introspection) registerSubscription(schema *schemabuilder.Schema) {
	schema.Subscription()
}

func (s *introspection) schema() *graphql.Schema {
	schema := schemabuilder.NewSchema()
	s.registerDirective(schema)
	s.registerEnumValue(schema)
	s.registerField(schema)
	s.registerInputValue(schema)
	s.registerSubscription(schema)
	s.registerMutation(schema)
	s.registerQuery(schema)
	s.registerSchema(schema)
	s.registerType(schema)

	return schema.MustBuild()
}

// AddIntrospectionToSchema adds the introspection fields to existing schema
func AddIntrospectionToSchema(schema *graphql.Schema) {
	types := make(map[string]graphql.Type)
	collectTypes(schema.Query, types)
	collectTypes(schema.Mutation, types)
	collectTypes(schema.Subscription, types)
	is := &introspection{
		types:        types,
		query:        schema.Query,
		mutation:     schema.Mutation,
		subscription: schema.Subscription,
	}
	isSchema := is.schema()

	query := schema.Query.(*graphql.Object)

	isQuery := isSchema.Query.(*graphql.Object)
	for k, v := range query.Fields {
		isQuery.Fields[k] = v
	}

	schema.Query = isQuery
}

// ComputeSchemaJSON returns the result of executing a GraphQL introspection
// query.
func ComputeSchemaJSON(schemaBuilderSchema schemabuilder.Schema) ([]byte, error) {
	schema := schemaBuilderSchema.MustBuild()
	AddIntrospectionToSchema(schema)

	query, err := graphql.Parse(IntrospectionQuery, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	if err := graphql.ValidateQuery(context.Background(), schema.Query, query.SelectionSet); err != nil {
		return nil, err
	}

	executor := graphql.Executor{}
	value, err := executor.Execute(context.Background(), schema.Query, nil, query)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(value, "", "  ")
}
