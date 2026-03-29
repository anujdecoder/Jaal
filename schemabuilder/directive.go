package schemabuilder

import "context"

// DirectiveLocation enumerates all GraphQL spec locations where a directive can
// be applied.  Covers both executable (query-time) and type-system
// (schema-definition-time) locations per the June 2018+ spec.
type DirectiveLocation string

const (
	// --- Executable locations (query-time) ---
	LocationQuery              DirectiveLocation = "QUERY"
	LocationMutation           DirectiveLocation = "MUTATION"
	LocationSubscription       DirectiveLocation = "SUBSCRIPTION"
	LocationField              DirectiveLocation = "FIELD"
	LocationFragmentDefinition DirectiveLocation = "FRAGMENT_DEFINITION"
	LocationFragmentSpread     DirectiveLocation = "FRAGMENT_SPREAD"
	LocationInlineFragment     DirectiveLocation = "INLINE_FRAGMENT"

	// --- Type-system locations (schema-definition-time) ---
	LocationSchema             DirectiveLocation = "SCHEMA"
	LocationScalar             DirectiveLocation = "SCALAR"
	LocationObject             DirectiveLocation = "OBJECT"
	LocationFieldDefinition    DirectiveLocation = "FIELD_DEFINITION"
	LocationArgumentDefinition DirectiveLocation = "ARGUMENT_DEFINITION"
	LocationInterface          DirectiveLocation = "INTERFACE"
	LocationUnion              DirectiveLocation = "UNION"
	LocationEnum               DirectiveLocation = "ENUM"
	LocationEnumValue          DirectiveLocation = "ENUM_VALUE"
	LocationInputObject        DirectiveLocation = "INPUT_OBJECT"
	LocationInputFieldDef      DirectiveLocation = "INPUT_FIELD_DEFINITION"
)

// ExecutionOrder determines when the directive handler runs relative to the
// field resolver.
type ExecutionOrder int

const (
	// PreResolver runs the handler before the field resolver (default).
	PreResolver ExecutionOrder = iota
	// PostResolver runs the handler after the field resolver.
	PostResolver
)

// OnFailBehavior determines what happens when a directive handler returns an
// error.
type OnFailBehavior int

const (
	// ErrorOnFail returns the error to the client (default).
	ErrorOnFail OnFailBehavior = iota
	// SkipOnFail silently returns nil (null) for the field.
	SkipOnFail
)

// DirectiveHandlerFunc is the callback invoked during directive execution.
// It receives the request context and the static args that were supplied when
// the directive was attached to a field or type.
// Return nil to allow the operation to proceed, or a non-nil error to block it.
// For PreResolver directives the handler runs before the field resolver.
// For PostResolver directives the handler runs after the field resolver.
type DirectiveHandlerFunc func(ctx context.Context, args map[string]interface{}) error

// DirectiveArgDef describes one argument accepted by a directive definition.
type DirectiveArgDef struct {
	Name         string      // e.g. "role"
	TypeName     string      // GraphQL type name, e.g. "String", "Int"
	Description  string      // human-readable description
	DefaultValue interface{} // optional default value
}

// DirectiveDefinition is the full definition of a custom directive that can be
// registered on a Schema.
//
// Example:
//
//	sb.Directive("hasRole", &schemabuilder.DirectiveDefinition{
//	    Description:    "Restricts access based on role",
//	    Locations:      []DirectiveLocation{LocationFieldDefinition},
//	    Args:           []DirectiveArgDef{{Name: "role", TypeName: "String"}},
//	    ExecutionOrder: PreResolver,   // default
//	    OnFail:         ErrorOnFail,   // default
//	    Handler: func(ctx context.Context, args map[string]interface{}) error {
//	        if ctx.Value("role") != args["role"] { return errors.New("denied") }
//	        return nil
//	    },
//	})
type DirectiveDefinition struct {
	Name           string
	Description    string
	Locations      []DirectiveLocation
	Args           []DirectiveArgDef
	IsRepeatable   bool
	ExecutionOrder ExecutionOrder     // PreResolver (default) or PostResolver
	OnFail         OnFailBehavior     // ErrorOnFail (default) or SkipOnFail
	Handler        DirectiveHandlerFunc // nil → metadata-only (introspection only)
}

// DirectiveInstance represents one concrete application of a directive to a
// field or type, together with the argument values supplied at registration
// time.
type DirectiveInstance struct {
	Name string
	Args map[string]interface{}
}
