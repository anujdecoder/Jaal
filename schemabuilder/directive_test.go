package schemabuilder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/graphql"
)

func TestDirectiveRegistration(t *testing.T) {
	schema := NewSchema()

	schema.Directive("custom",
		DirectiveDescription("A custom directive"),
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveArgString("param"),
	)

	defs := schema.GetDirectiveDefinitions()
	require.Len(t, defs, 1)
	assert.Equal(t, "custom", defs[0].Name)
	assert.Equal(t, "A custom directive", defs[0].Description)
	assert.True(t, defs[0].HasLocation(graphql.LocationFieldDefinition))
}

func TestDirectiveRegistrationWithVisitor(t *testing.T) {
	schema := NewSchema()

	called := false
	schema.Directive("track",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			called = true
			return nil, nil
		}),
	)

	visitor := schema.GetDirectiveVisitor("track")
	require.NotNil(t, visitor)

	// Test visitor is callable
	_, err := visitor.VisitField(context.Background(), nil, nil, nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestFieldDirectiveApplication(t *testing.T) {
	schema := NewSchema()

	// Register directive first
	schema.Directive("log",
		DirectiveLocations(graphql.LocationFieldDefinition),
	)

	// Apply to field
	query := schema.Query()
	query.FieldFunc("test", func() string { return "test" },
		FieldDirective("log", map[string]interface{}{"level": "info"}),
	)

	builtSchema := schema.MustBuild()
	queryObj := builtSchema.Query.(*graphql.Object)
	field := queryObj.Fields["test"]

	require.Len(t, field.Directives, 1)
	assert.Equal(t, "log", field.Directives[0].Name)
	assert.Equal(t, "info", field.Directives[0].Args["level"])
}

func TestRepeatableDirective(t *testing.T) {
	schema := NewSchema()

	schema.Directive("tag",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveRepeatable(),
	)

	defs := schema.GetDirectiveDefinitions()
	require.Len(t, defs, 1)
	assert.True(t, defs[0].IsRepeatable)
}

func TestDirectiveArgTypes(t *testing.T) {
	schema := NewSchema()

	schema.Directive("typed",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveArgString("stringArg"),
		DirectiveArgInt("intArg"),
		DirectiveArgBool("boolArg"),
		DirectiveArgFloat("floatArg"),
		DirectiveArgNonNull("requiredArg", &graphql.Scalar{Type: "String"}),
	)

	defs := schema.GetDirectiveDefinitions()
	require.Len(t, defs, 1)

	args := defs[0].Args
	require.Len(t, args, 5)
	assert.NotNil(t, args["stringArg"])
	assert.NotNil(t, args["intArg"])
	assert.NotNil(t, args["boolArg"])
	assert.NotNil(t, args["floatArg"])
	assert.NotNil(t, args["requiredArg"])
}

func TestDirectiveExecution(t *testing.T) {
	schema := NewSchema()

	var executed bool
	schema.Directive("track",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			executed = true
			return nil, nil // Continue with normal resolution
		}),
	)

	query := schema.Query()
	query.FieldFunc("data", func() string { return "result" },
		FieldDirective("track", nil),
	)

	builtSchema := schema.MustBuild()

	q, err := graphql.Parse(`{ data }`, nil)
	require.NoError(t, err)

	visitors := schema.GetDirectiveVisitors()
	e := graphql.NewExecutor(visitors)

	_, err = e.Execute(context.Background(), builtSchema.Query, nil, q)
	require.NoError(t, err)
	assert.True(t, executed)
}

func TestDirectiveShortCircuit(t *testing.T) {
	schema := NewSchema()

	schema.Directive("override",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			return "overridden", nil
		}),
	)

	query := schema.Query()
	query.FieldFunc("original", func() string { return "original" },
		FieldDirective("override", nil),
	)

	builtSchema := schema.MustBuild()

	q, err := graphql.Parse(`{ original }`, nil)
	require.NoError(t, err)

	e := graphql.NewExecutor(schema.GetDirectiveVisitors())

	val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
	require.NoError(t, err)

	result := val.(map[string]interface{})
	assert.Equal(t, "overridden", result["original"])
}

func TestDirectiveError(t *testing.T) {
	schema := NewSchema()

	schema.Directive("fail",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			return nil, assert.AnError
		}),
	)

	query := schema.Query()
	query.FieldFunc("data", func() string { return "result" },
		FieldDirective("fail", nil),
	)

	builtSchema := schema.MustBuild()

	q, err := graphql.Parse(`{ data }`, nil)
	require.NoError(t, err)

	e := graphql.NewExecutor(schema.GetDirectiveVisitors())

	_, err = e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Error(t, err)
}

func TestMultipleDirectives(t *testing.T) {
	schema := NewSchema()

	var order []string
	schema.Directive("first",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			order = append(order, "first")
			return nil, nil
		}),
	)

	schema.Directive("second",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			order = append(order, "second")
			return nil, nil
		}),
	)

	query := schema.Query()
	query.FieldFunc("data", func() string { return "result" },
		FieldDirective("first", nil),
		FieldDirective("second", nil),
	)

	builtSchema := schema.MustBuild()

	q, err := graphql.Parse(`{ data }`, nil)
	require.NoError(t, err)

	e := graphql.NewExecutor(schema.GetDirectiveVisitors())

	_, err = e.Execute(context.Background(), builtSchema.Query, nil, q)
	require.NoError(t, err)

	assert.Equal(t, []string{"first", "second"}, order)
}

func TestGetDirectiveRegistry(t *testing.T) {
	schema := NewSchema()

	schema.Directive("auth",
		DirectiveLocations(graphql.LocationFieldDefinition),
		DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
			return nil, nil
		}),
	)

	registry := schema.GetDirectiveRegistry()
	require.NotNil(t, registry)

	def := registry.GetDefinition("auth")
	require.NotNil(t, def)
	assert.Equal(t, "auth", def.Name)

	visitor := registry.GetVisitor("auth")
	require.NotNil(t, visitor)
}
