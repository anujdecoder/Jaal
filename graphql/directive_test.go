package graphql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.appointy.com/jaal/graphql"
)

func TestDirectiveLocationString(t *testing.T) {
	tests := []struct {
		location graphql.DirectiveLocation
		expected string
	}{
		{graphql.LocationField, "FIELD"},
		{graphql.LocationFieldDefinition, "FIELD_DEFINITION"},
		{graphql.LocationQuery, "QUERY"},
		{graphql.LocationMutation, "MUTATION"},
		{graphql.LocationSubscription, "SUBSCRIPTION"},
		{graphql.LocationObject, "OBJECT"},
		{graphql.LocationInterface, "INTERFACE"},
		{graphql.LocationUnion, "UNION"},
		{graphql.LocationEnum, "ENUM"},
		{graphql.LocationInputObject, "INPUT_OBJECT"},
		{graphql.LocationArgumentDefinition, "ARGUMENT_DEFINITION"},
	}

	for _, tt := range tests {
		t.Run(string(tt.location), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.location))
		})
	}
}

func TestDirectiveDefinitionHasLocation(t *testing.T) {
	def := &graphql.DirectiveDefinition{
		Name:       "test",
		Locations:  []graphql.DirectiveLocation{graphql.LocationField, graphql.LocationFieldDefinition},
	}

	assert.True(t, def.HasLocation(graphql.LocationField))
	assert.True(t, def.HasLocation(graphql.LocationFieldDefinition))
	assert.False(t, def.HasLocation(graphql.LocationQuery))
}

func TestDirectiveDefinitionIsRepeatable(t *testing.T) {
	repeatable := &graphql.DirectiveDefinition{
		Name:         "repeatable",
		IsRepeatable: true,
	}

	nonRepeatable := &graphql.DirectiveDefinition{
		Name:         "nonRepeatable",
		IsRepeatable: false,
	}

	assert.True(t, repeatable.IsRepeatable)
	assert.False(t, nonRepeatable.IsRepeatable)
}

func TestDirectiveVisitorFunc(t *testing.T) {
	called := false
	visitor := graphql.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
		called = true
		return "result", nil
	})

	result, err := visitor.VisitField(context.Background(), nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "result", result)
	assert.True(t, called)
}

func TestDirectiveRegistry(t *testing.T) {
	registry := graphql.NewDirectiveRegistry()

	def := &graphql.DirectiveDefinition{
		Name:        "test",
		Description: "Test directive",
		Locations:   []graphql.DirectiveLocation{graphql.LocationFieldDefinition},
	}

	visitor := graphql.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
		return nil, nil
	})

	registry.Register(def, visitor)

	// Test GetDefinition
	retrievedDef := registry.GetDefinition("test")
	assert.NotNil(t, retrievedDef)
	assert.Equal(t, "test", retrievedDef.Name)

	// Test GetVisitor
	retrievedVisitor := registry.GetVisitor("test")
	assert.NotNil(t, retrievedVisitor)

	// Test non-existent
	assert.Nil(t, registry.GetDefinition("nonexistent"))
	assert.Nil(t, registry.GetVisitor("nonexistent"))
}

func TestDirectiveRegistryGetAllDefinitions(t *testing.T) {
	registry := graphql.NewDirectiveRegistry()

	def1 := &graphql.DirectiveDefinition{Name: "directive1"}
	def2 := &graphql.DirectiveDefinition{Name: "directive2"}

	registry.Register(def1, nil)
	registry.Register(def2, nil)

	defs := registry.GetAllDefinitions()
	assert.Len(t, defs, 2)
}

func TestDirectiveRegistryGetAllVisitors(t *testing.T) {
	registry := graphql.NewDirectiveRegistry()

	visitor := graphql.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
		return nil, nil
	})

	registry.Register(&graphql.DirectiveDefinition{Name: "withVisitor"}, visitor)
	registry.Register(&graphql.DirectiveDefinition{Name: "withoutVisitor"}, nil)

	visitors := registry.GetAllVisitors()
	assert.Len(t, visitors, 1)
	assert.NotNil(t, visitors["withVisitor"])
}

func TestDirectiveInstance(t *testing.T) {
	def := &graphql.DirectiveDefinition{
		Name:       "auth",
		Locations:  []graphql.DirectiveLocation{graphql.LocationFieldDefinition},
	}

	instance := &graphql.DirectiveInstance{
		Name:       "auth",
		Args:       map[string]interface{}{"role": "admin"},
		Definition: def,
	}

	assert.Equal(t, "auth", instance.Name)
	assert.Equal(t, "admin", instance.Args["role"])
	assert.NotNil(t, instance.Definition)
}
