package schemabuilder

import (
	"context"

	"go.appointy.com/jaal/graphql"
)

// DirectiveOption configures a directive during registration.
type DirectiveOption func(*directiveConfig)

type directiveConfig struct {
	description  string
	locations    []graphql.DirectiveLocation
	args         map[string]*graphql.Argument
	isRepeatable bool
	visitor      graphql.DirectiveVisitor
}

// DirectiveDescription sets the description for a directive.
func DirectiveDescription(desc string) DirectiveOption {
	return func(cfg *directiveConfig) {
		cfg.description = desc
	}
}

// DirectiveRepeatable marks a directive as repeatable (can be applied multiple times).
func DirectiveRepeatable() DirectiveOption {
	return func(cfg *directiveConfig) {
		cfg.isRepeatable = true
	}
}

// DirectiveLocations sets the locations where a directive can be applied.
func DirectiveLocations(locations ...graphql.DirectiveLocation) DirectiveOption {
	return func(cfg *directiveConfig) {
		cfg.locations = locations
	}
}

// DirectiveArg adds an argument to a directive with a GraphQL type.
// The typ should be a graphql.Type (e.g., &graphql.Scalar{Type: "String"}).
func DirectiveArg(name string, typ graphql.Type) DirectiveOption {
	return func(cfg *directiveConfig) {
		if cfg.args == nil {
			cfg.args = make(map[string]*graphql.Argument)
		}
		cfg.args[name] = &graphql.Argument{Type: typ}
	}
}

// DirectiveArgString adds a String argument to a directive.
func DirectiveArgString(name string) DirectiveOption {
	return DirectiveArg(name, &graphql.Scalar{Type: "String"})
}

// DirectiveArgInt adds an Int argument to a directive.
func DirectiveArgInt(name string) DirectiveOption {
	return DirectiveArg(name, &graphql.Scalar{Type: "Int"})
}

// DirectiveArgBool adds a Boolean argument to a directive.
func DirectiveArgBool(name string) DirectiveOption {
	return DirectiveArg(name, &graphql.Scalar{Type: "Boolean"})
}

// DirectiveArgFloat adds a Float argument to a directive.
func DirectiveArgFloat(name string) DirectiveOption {
	return DirectiveArg(name, &graphql.Scalar{Type: "Float"})
}

// DirectiveArgNonNull adds a non-null argument to a directive.
func DirectiveArgNonNull(name string, typ graphql.Type) DirectiveOption {
	return DirectiveArg(name, &graphql.NonNull{Type: typ})
}

// DirectiveVisitor sets the visitor for directive execution.
func DirectiveVisitor(visitor graphql.DirectiveVisitor) DirectiveOption {
	return func(cfg *directiveConfig) {
		cfg.visitor = visitor
	}
}

// DirectiveVisitorFunc creates a visitor option from a function.
func DirectiveVisitorFunc(fn func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error)) DirectiveOption {
	return DirectiveVisitor(graphql.DirectiveVisitorFunc(fn))
}

// directiveRegistration stores both definition and visitor for a custom directive.
type directiveRegistration struct {
	definition *graphql.DirectiveDefinition
	visitor    graphql.DirectiveVisitor
}

// FieldDirective applies a directive to a field during schema building.
// Usage: obj.FieldFunc("field", resolver, FieldDirective("auth", map[string]interface{}{"role": "admin"}))
func FieldDirective(name string, args map[string]interface{}) FieldOption {
	return func(cfg *fieldConfig) {
		cfg.directives = append(cfg.directives, &graphql.DirectiveInstance{
			Name: name,
			Args: args,
		})
	}
}

// ApplyDirective applies a directive to a type.
// Usage: schema.Object("User", User{}, ApplyDirective("auth", map[string]interface{}{"role": "user"}))
func ApplyDirective(name string, args map[string]interface{}) TypeOption {
	return func(cfg *typeConfig) {
		cfg.typeDirectives = append(cfg.typeDirectives, &graphql.DirectiveInstance{
			Name: name,
			Args: args,
		})
	}
}

// applyDirectiveOptions applies the directive options and returns the resulting config.
func applyDirectiveOptions(opts []DirectiveOption) directiveConfig {
	cfg := directiveConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}
