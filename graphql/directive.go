package graphql

import "context"

// DirectiveLocation represents where a directive can be applied.
// Per GraphQL Oct 2021 spec, includes both executable and type system locations.
type DirectiveLocation string

const (
	// Executable directive locations (applied to queries at runtime)
	LocationQuery              DirectiveLocation = "QUERY"
	LocationMutation           DirectiveLocation = "MUTATION"
	LocationSubscription       DirectiveLocation = "SUBSCRIPTION"
	LocationField              DirectiveLocation = "FIELD"
	LocationFragmentDefinition DirectiveLocation = "FRAGMENT_DEFINITION"
	LocationFragmentSpread     DirectiveLocation = "FRAGMENT_SPREAD"
	LocationInlineFragment     DirectiveLocation = "INLINE_FRAGMENT"

	// Type system directive locations (applied to schema definitions)
	LocationSchema               DirectiveLocation = "SCHEMA"
	LocationScalar               DirectiveLocation = "SCALAR"
	LocationObject               DirectiveLocation = "OBJECT"
	LocationFieldDefinition      DirectiveLocation = "FIELD_DEFINITION"
	LocationArgumentDefinition   DirectiveLocation = "ARGUMENT_DEFINITION"
	LocationInterface            DirectiveLocation = "INTERFACE"
	LocationUnion                DirectiveLocation = "UNION"
	LocationEnum                 DirectiveLocation = "ENUM"
	LocationEnumValue            DirectiveLocation = "ENUM_VALUE"
	LocationInputObject          DirectiveLocation = "INPUT_OBJECT"
	LocationInputFieldDefinition DirectiveLocation = "INPUT_FIELD_DEFINITION"
)

// DirectiveDefinition represents a custom directive definition in the schema.
// Per GraphQL Oct 2021 spec, includes isRepeatable flag.
type DirectiveDefinition struct {
	Name         string
	Description  string
	Locations    []DirectiveLocation
	Args         map[string]*Argument
	IsRepeatable bool
}

// HasLocation checks if the directive can be applied at the given location.
func (d *DirectiveDefinition) HasLocation(loc DirectiveLocation) bool {
	for _, l := range d.Locations {
		if l == loc {
			return true
		}
	}
	return false
}

// DirectiveInstance represents an instance of a directive applied to a schema element.
type DirectiveInstance struct {
	Name       string
	Args       map[string]interface{}
	Definition *DirectiveDefinition // Reference to the definition (may be nil during build)
}

// DirectiveVisitor defines the interface for directive execution behavior.
// Implement this interface to add runtime behavior to directives.
type DirectiveVisitor interface {
	// VisitField is called during field resolution for FIELD_DEFINITION location directives.
	// Return (nil, nil) to continue with normal resolution.
	// Return (result, nil) to short-circuit and return the result.
	// Return (nil, error) to return an error.
	VisitField(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error)
}

// DirectiveVisitorFunc is an adapter to allow using functions as DirectiveVisitor.
type DirectiveVisitorFunc func(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error)

// VisitField implements DirectiveVisitor for DirectiveVisitorFunc.
func (f DirectiveVisitorFunc) VisitField(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error) {
	return f(ctx, directive, field, source)
}

// MultiLocationVisitor extends DirectiveVisitor to support multiple locations.
type MultiLocationVisitor interface {
	DirectiveVisitor

	// VisitArgument is called for ARGUMENT_DEFINITION location directives.
	VisitArgument(ctx context.Context, directive *DirectiveInstance, argName string, value interface{}) (interface{}, error)

	// VisitInputField is called for INPUT_FIELD_DEFINITION location directives.
	VisitInputField(ctx context.Context, directive *DirectiveInstance, fieldName string, value interface{}) (interface{}, error)

	// VisitEnumValue is called for ENUM_VALUE location directives.
	VisitEnumValue(ctx context.Context, directive *DirectiveInstance, value string) error
}

// DirectiveRegistry holds all registered directive definitions and their visitors.
type DirectiveRegistry struct {
	definitions map[string]*DirectiveDefinition
	visitors    map[string]DirectiveVisitor
}

// NewDirectiveRegistry creates a new empty directive registry.
func NewDirectiveRegistry() *DirectiveRegistry {
	return &DirectiveRegistry{
		definitions: make(map[string]*DirectiveDefinition),
		visitors:    make(map[string]DirectiveVisitor),
	}
}

// Register adds a directive definition and optional visitor to the registry.
func (r *DirectiveRegistry) Register(def *DirectiveDefinition, visitor DirectiveVisitor) {
	r.definitions[def.Name] = def
	if visitor != nil {
		r.visitors[def.Name] = visitor
	}
}

// GetDefinition returns the directive definition by name, or nil if not found.
func (r *DirectiveRegistry) GetDefinition(name string) *DirectiveDefinition {
	return r.definitions[name]
}

// GetVisitor returns the directive visitor by name, or nil if not found.
func (r *DirectiveRegistry) GetVisitor(name string) DirectiveVisitor {
	return r.visitors[name]
}

// GetAllDefinitions returns all registered directive definitions.
func (r *DirectiveRegistry) GetAllDefinitions() []*DirectiveDefinition {
	defs := make([]*DirectiveDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		defs = append(defs, def)
	}
	return defs
}

// GetAllVisitors returns a map of all registered directive visitors.
func (r *DirectiveRegistry) GetAllVisitors() map[string]DirectiveVisitor {
	visitors := make(map[string]DirectiveVisitor, len(r.visitors))
	for name, visitor := range r.visitors {
		visitors[name] = visitor
	}
	return visitors
}
