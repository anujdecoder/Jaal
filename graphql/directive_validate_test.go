package graphql_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/graphql"
)

func TestValidateDirectiveLocations(t *testing.T) {
	definitions := map[string]*graphql.DirectiveDefinition{
		"fieldOnly": {
			Name:       "fieldOnly",
			Locations:  []graphql.DirectiveLocation{graphql.LocationField},
		},
		"fieldDefOnly": {
			Name:       "fieldDefOnly",
			Locations:  []graphql.DirectiveLocation{graphql.LocationFieldDefinition},
		},
		"multiLocation": {
			Name:       "multiLocation",
			Locations:  []graphql.DirectiveLocation{graphql.LocationField, graphql.LocationFragmentSpread},
		},
	}

	t.Run("valid field location", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "fieldOnly"},
						{Name: "multiLocation"},
					},
				},
			},
		}

		errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
		assert.Empty(t, errors)
	})

	t.Run("invalid field location", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "fieldDefOnly"},
					},
				},
			},
		}

		errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Error(), "cannot be used on FIELD")
	})

	t.Run("valid fragment spread location", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Fragments: []*graphql.FragmentSpread{
				{
					Directives: []*graphql.Directive{
						{Name: "multiLocation"},
					},
					Fragment: &graphql.FragmentDefinition{
						On:           "Test",
						SelectionSet: &graphql.SelectionSet{},
					},
				},
			},
		}

		errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
		assert.Empty(t, errors)
	})

	t.Run("invalid fragment spread location", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Fragments: []*graphql.FragmentSpread{
				{
					Directives: []*graphql.Directive{
						{Name: "fieldDefOnly"},
					},
					Fragment: &graphql.FragmentDefinition{
						On:           "Test",
						SelectionSet: &graphql.SelectionSet{},
					},
				},
			},
		}

		errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Error(), "cannot be used on FRAGMENT_SPREAD")
	})

	t.Run("nested selections", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "parent",
					SelectionSet: &graphql.SelectionSet{
						Selections: []*graphql.Selection{
							{
								Name: "child",
								Directives: []*graphql.Directive{
									{Name: "fieldDefOnly"},
								},
							},
						},
					},
				},
			},
		}

		errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
		require.Len(t, errors, 1)
	})

	t.Run("nil selection set", func(t *testing.T) {
		errors := graphql.ValidateDirectiveLocations(nil, definitions)
		assert.Empty(t, errors)
	})

	t.Run("unknown directive is ignored", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "unknownDirective"},
					},
				},
			},
		}

		errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
		assert.Empty(t, errors)
	})
}

func TestValidateRepeatableDirectives(t *testing.T) {
	definitions := map[string]*graphql.DirectiveDefinition{
		"repeatable": {
			Name:         "repeatable",
			Locations:    []graphql.DirectiveLocation{graphql.LocationField},
			IsRepeatable: true,
		},
		"nonRepeatable": {
			Name:         "nonRepeatable",
			Locations:    []graphql.DirectiveLocation{graphql.LocationField},
			IsRepeatable: false,
		},
	}

	t.Run("repeatable directive used multiple times", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "repeatable"},
						{Name: "repeatable"},
						{Name: "repeatable"},
					},
				},
			},
		}

		errors := graphql.ValidateRepeatableDirectives(selectionSet, definitions)
		assert.Empty(t, errors)
	})

	t.Run("non-repeatable directive used once", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "nonRepeatable"},
					},
				},
			},
		}

		errors := graphql.ValidateRepeatableDirectives(selectionSet, definitions)
		assert.Empty(t, errors)
	})

	t.Run("non-repeatable directive used multiple times", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "nonRepeatable"},
						{Name: "nonRepeatable"},
					},
				},
			},
		}

		errors := graphql.ValidateRepeatableDirectives(selectionSet, definitions)
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Error(), "not repeatable")
	})

	t.Run("mixed directives", func(t *testing.T) {
		selectionSet := &graphql.SelectionSet{
			Selections: []*graphql.Selection{
				{
					Name: "test",
					Directives: []*graphql.Directive{
						{Name: "repeatable"},
						{Name: "nonRepeatable"},
						{Name: "repeatable"},
						{Name: "nonRepeatable"}, // Second occurrence - should error
					},
				},
			},
		}

		errors := graphql.ValidateRepeatableDirectives(selectionSet, definitions)
		require.Len(t, errors, 1)
	})

	t.Run("nil selection set", func(t *testing.T) {
		errors := graphql.ValidateRepeatableDirectives(nil, definitions)
		assert.Empty(t, errors)
	})
}

func TestValidateFieldDirectives(t *testing.T) {
	definitions := map[string]*graphql.DirectiveDefinition{
		"fieldDefDirective": {
			Name:       "fieldDefDirective",
			Locations:  []graphql.DirectiveLocation{graphql.LocationFieldDefinition},
		},
		"fieldDirective": {
			Name:       "fieldDirective",
			Locations:  []graphql.DirectiveLocation{graphql.LocationField},
		},
	}

	t.Run("valid field definition directive", func(t *testing.T) {
		field := &graphql.Field{
			Directives: []*graphql.DirectiveInstance{
				{Name: "fieldDefDirective"},
			},
		}

		errors := graphql.ValidateFieldDirectives(field, definitions)
		assert.Empty(t, errors)
	})

	t.Run("invalid field directive on field definition", func(t *testing.T) {
		field := &graphql.Field{
			Directives: []*graphql.DirectiveInstance{
				{Name: "fieldDirective"},
			},
		}

		errors := graphql.ValidateFieldDirectives(field, definitions)
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Error(), "cannot be used on FIELD_DEFINITION")
	})

	t.Run("field with no directives", func(t *testing.T) {
		field := &graphql.Field{}

		errors := graphql.ValidateFieldDirectives(field, definitions)
		assert.Empty(t, errors)
	})

	t.Run("unknown directive is ignored", func(t *testing.T) {
		field := &graphql.Field{
			Directives: []*graphql.DirectiveInstance{
				{Name: "unknownDirective"},
			},
		}

		errors := graphql.ValidateFieldDirectives(field, definitions)
		assert.Empty(t, errors)
	})
}

func TestValidateSchemaDirectives(t *testing.T) {
	definitions := map[string]*graphql.DirectiveDefinition{
		"validDirective": {
			Name:       "validDirective",
			Locations:  []graphql.DirectiveLocation{graphql.LocationFieldDefinition},
		},
		"invalidDirective": {
			Name:       "invalidDirective",
			Locations:  []graphql.DirectiveLocation{graphql.LocationField},
		},
	}

	t.Run("schema with valid directives", func(t *testing.T) {
		schema := &graphql.Schema{
			Query: &graphql.Object{
				Fields: map[string]*graphql.Field{
					"testField": {
						Directives: []*graphql.DirectiveInstance{
							{Name: "validDirective"},
						},
					},
				},
			},
		}

		errors := graphql.ValidateSchemaDirectives(schema, definitions)
		assert.Empty(t, errors)
	})

	t.Run("schema with invalid directives", func(t *testing.T) {
		schema := &graphql.Schema{
			Query: &graphql.Object{
				Fields: map[string]*graphql.Field{
					"testField": {
						Directives: []*graphql.DirectiveInstance{
							{Name: "invalidDirective"},
						},
					},
				},
			},
		}

		errors := graphql.ValidateSchemaDirectives(schema, definitions)
		require.Len(t, errors, 1)
	})

	t.Run("schema with nil types", func(t *testing.T) {
		schema := &graphql.Schema{
			Query:        nil,
			Mutation:     nil,
			Subscription: nil,
		}

		errors := graphql.ValidateSchemaDirectives(schema, definitions)
		assert.Empty(t, errors)
	})
}
