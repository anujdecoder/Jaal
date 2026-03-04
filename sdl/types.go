// Package sdl provides utilities for converting GraphQL introspection JSON to SDL format.
package sdl

// IntrospectionResponse represents the full introspection query response.
type IntrospectionResponse struct {
	Schema Schema `json:"__schema"`
}

// Schema represents the __Schema type from introspection.
type Schema struct {
	QueryType        *NamedType  `json:"queryType"`
	MutationType     *NamedType  `json:"mutationType"`
	SubscriptionType *NamedType  `json:"subscriptionType"`
	Types            []FullType  `json:"types"`
	Directives       []Directive `json:"directives"`
}

// NamedType represents a named type reference (for root types).
type NamedType struct {
	Name string `json:"name"`
}

// FullType represents a complete type definition from introspection.
type FullType struct {
	Kind          TypeKind     `json:"kind"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Fields        []Field      `json:"fields"`
	InputFields   []InputValue `json:"inputFields"`
	Interfaces    []TypeRef    `json:"interfaces"`
	EnumValues    []EnumValue  `json:"enumValues"`
	PossibleTypes []TypeRef    `json:"possibleTypes"`
	SpecifiedByURL *string     `json:"specifiedByURL"`
	Directives    []Directive  `json:"directives"`
}

// TypeKind represents the kind of a GraphQL type.
type TypeKind string

const (
	ScalarKind      TypeKind = "SCALAR"
	ObjectKind      TypeKind = "OBJECT"
	InterfaceKind   TypeKind = "INTERFACE"
	UnionKind       TypeKind = "UNION"
	EnumKind        TypeKind = "ENUM"
	InputObjectKind TypeKind = "INPUT_OBJECT"
	ListKind        TypeKind = "LIST"
	NonNullKind     TypeKind = "NON_NULL"
)

// Field represents a field definition.
type Field struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Args              []InputValue `json:"args"`
	Type              TypeRef      `json:"type"`
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason *string      `json:"deprecationReason"`
}

// InputValue represents an input value or argument definition.
type InputValue struct {
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Type              TypeRef `json:"type"`
	DefaultValue      *string `json:"defaultValue"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

// TypeRef represents a type reference (can be wrapped in List or NonNull).
type TypeRef struct {
	Kind   TypeKind `json:"kind"`
	Name   string   `json:"name"`
	OfType *TypeRef `json:"ofType"`
}

// EnumValue represents an enum value definition.
type EnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

// Directive represents a directive definition.
type Directive struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Locations   []DirectiveLocation `json:"locations"`
	Args        []InputValue      `json:"args"`
}

// DirectiveLocation represents where a directive can be applied.
type DirectiveLocation string
