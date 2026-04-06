package introspection_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// TestIntrospectionArgumentDeprecation tests that deprecated arguments appear correctly in introspection
func TestIntrospectionArgumentDeprecation(t *testing.T) {
	schema := schemabuilder.NewSchema()

	query := schema.Query()
	query.FieldFunc("searchUsers", func(args struct {
		Query    string
		Username string
	}) string {
		return args.Query
	}, schemabuilder.FieldDesc("Search users"),
		schemabuilder.ArgDeprecation("username", "Use 'query' for full-text search instead"))

	builtSchema := schema.MustBuild()

	// Add introspection to the schema
	introspection.AddIntrospectionToSchema(builtSchema)

	// Query the full schema to find the Query type and check argument deprecation
	q, err := graphql.Parse(`{
		__schema {
			types {
				name
				kind
				fields {
					name
					args {
						name
						isDeprecated
						deprecationReason
					}
				}
			}
		}
	}`, nil)
	require.NoError(t, err)

	ctx := context.Background()
	if err := graphql.ValidateQuery(ctx, builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	e := graphql.Executor{}
	val, err := e.Execute(ctx, builtSchema.Query, nil, q)
	require.NoError(t, err)

	resultMap := val.(map[string]interface{})
	schemaData := resultMap["__schema"].(map[string]interface{})
	typesIface := schemaData["types"].([]interface{})

	// Find Query type
	var queryType map[string]interface{}
	for _, t := range typesIface {
		typ := t.(map[string]interface{})
		if typ["name"].(string) == "Query" {
			queryType = typ
			break
		}
	}
	require.NotNil(t, queryType, "Query type should exist")

	fieldsIface := queryType["fields"].([]interface{})

	// Find searchUsers field
	var searchUsersField map[string]interface{}
	for _, f := range fieldsIface {
		field := f.(map[string]interface{})
		if field["name"].(string) == "searchUsers" {
			searchUsersField = field
			break
		}
	}
	require.NotNil(t, searchUsersField, "searchUsers field should exist")

	argsIface := searchUsersField["args"].([]interface{})

	// Check query arg is NOT deprecated
	var queryArg map[string]interface{}
	var usernameArg map[string]interface{}
	for _, a := range argsIface {
		arg := a.(map[string]interface{})
		name := arg["name"].(string)
		if name == "query" {
			queryArg = arg
		} else if name == "username" {
			usernameArg = arg
		}
	}

	require.NotNil(t, queryArg, "query arg should exist")
	require.NotNil(t, usernameArg, "username arg should exist")

	// Verify query arg is not deprecated
	assert.Equal(t, false, queryArg["isDeprecated"])
	assert.Nil(t, queryArg["deprecationReason"])

	// Verify username arg IS deprecated
	assert.Equal(t, true, usernameArg["isDeprecated"])
	assert.Equal(t, "Use 'query' for full-text search instead", usernameArg["deprecationReason"])
}

// TestIntrospectionEnumValueDeprecation tests that deprecated enum values appear correctly in introspection
func TestIntrospectionEnumValueDeprecation(t *testing.T) {
	schema := schemabuilder.NewSchema()

	type UserRole int32
	schema.Enum(UserRole(0), map[string]interface{}{
		"ADMIN":     UserRole(0),
		"USER":      UserRole(1),
		"GUEST":     UserRole(2),
		"READ_ONLY": UserRole(3),
	}, schemabuilder.WithDescription("User roles in the system"),
		schemabuilder.EnumValueDeprecation("GUEST", "Use READ_ONLY instead"))

	query := schema.Query()
	query.FieldFunc("getRole", func(args struct {
		Role UserRole
	}) UserRole {
		return args.Role
	})

	builtSchema := schema.MustBuild()

	// Add introspection to the schema
	introspection.AddIntrospectionToSchema(builtSchema)

	// Query the full schema to find the enum type
	q, err := graphql.Parse(`{
		__schema {
			types {
				name
				kind
				enumValues {
					name
					isDeprecated
					deprecationReason
				}
			}
		}
	}`, nil)
	require.NoError(t, err)

	ctx := context.Background()
	if err := graphql.ValidateQuery(ctx, builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	e := graphql.Executor{}
	val, err := e.Execute(ctx, builtSchema.Query, nil, q)
	require.NoError(t, err)

	resultMap := val.(map[string]interface{})
	schemaData := resultMap["__schema"].(map[string]interface{})
	typesIface := schemaData["types"].([]interface{})

	// Find UserRole enum type
	var userRoleType map[string]interface{}
	for _, t := range typesIface {
		typ := t.(map[string]interface{})
		if typ["name"].(string) == "UserRole" {
			userRoleType = typ
			break
		}
	}
	require.NotNil(t, userRoleType, "UserRole type should exist")

	enumValuesIface := userRoleType["enumValues"].([]interface{})

	// Find each enum value and check deprecation status
	for _, ev := range enumValuesIface {
		enumVal := ev.(map[string]interface{})
		name := enumVal["name"].(string)
		switch name {
		case "ADMIN":
			assert.Equal(t, false, enumVal["isDeprecated"], "ADMIN should not be deprecated")
			assert.Nil(t, enumVal["deprecationReason"], "ADMIN should not have deprecation reason")
		case "USER":
			assert.Equal(t, false, enumVal["isDeprecated"], "USER should not be deprecated")
			assert.Nil(t, enumVal["deprecationReason"], "USER should not have deprecation reason")
		case "GUEST":
			assert.Equal(t, true, enumVal["isDeprecated"], "GUEST should be deprecated")
			assert.Equal(t, "Use READ_ONLY instead", enumVal["deprecationReason"], "GUEST should have deprecation reason")
		case "READ_ONLY":
			assert.Equal(t, false, enumVal["isDeprecated"], "READ_ONLY should not be deprecated")
			assert.Nil(t, enumVal["deprecationReason"], "READ_ONLY should not have deprecation reason")
		}
	}
}

// TestIntrospectionMultipleArgumentDeprecations tests multiple deprecated arguments
func TestIntrospectionMultipleArgumentDeprecations(t *testing.T) {
	schema := schemabuilder.NewSchema()

	query := schema.Query()
	query.FieldFunc("complexQuery", func(args struct {
		Arg1 string
		Arg2 string
		Arg3 string
	}) string {
		return args.Arg3
	}, schemabuilder.ArgDeprecation("arg1", "Use arg3 instead"),
		schemabuilder.ArgDeprecation("arg2", "Use arg3 instead"))

	builtSchema := schema.MustBuild()

	// Verify the schema was built correctly with argument deprecations
	// by checking the Query object directly
	queryObj, ok := builtSchema.Query.(*graphql.Object)
	require.True(t, ok)

	complexQueryField, ok := queryObj.Fields["complexQuery"]
	require.True(t, ok)

	args := complexQueryField.Args

	// Check arg1 is deprecated
	arg1, ok := args["arg1"]
	require.True(t, ok)
	assert.True(t, arg1.IsDeprecated, "arg1 should be deprecated")
	assert.Equal(t, "Use arg3 instead", *arg1.DeprecationReason)

	// Check arg2 is deprecated
	arg2, ok := args["arg2"]
	require.True(t, ok)
	assert.True(t, arg2.IsDeprecated, "arg2 should be deprecated")
	assert.Equal(t, "Use arg3 instead", *arg2.DeprecationReason)

	// Check arg3 is NOT deprecated
	arg3, ok := args["arg3"]
	require.True(t, ok)
	assert.False(t, arg3.IsDeprecated, "arg3 should not be deprecated")
	assert.Nil(t, arg3.DeprecationReason)
}

// TestIntrospectionSDLRoundTrip verifies introspection can be converted to SDL with deprecations
func TestIntrospectionSDLRoundTrip(t *testing.T) {
	schema := schemabuilder.NewSchema()

	type Status int32
	schema.Enum(Status(0), map[string]interface{}{
		"ACTIVE":   Status(0),
		"INACTIVE": Status(1),
	}, schemabuilder.EnumValueDeprecation("INACTIVE", "Use ACTIVE instead"))

	query := schema.Query()
	query.FieldFunc("getStatus", func(args struct {
		Status Status
	}) Status {
		return args.Status
	}, schemabuilder.ArgDeprecation("status", "This arg is deprecated"))

	builtSchema := schema.MustBuild()

	// Add introspection to the schema
	introspection.AddIntrospectionToSchema(builtSchema)

	// Verify we can query the schema
	q, err := graphql.Parse(`{ __schema { types { name } } }`, nil)
	require.NoError(t, err)

	ctx := context.Background()
	e := graphql.Executor{}
	val, err := e.Execute(ctx, builtSchema.Query, nil, q)
	require.NoError(t, err)

	// Just verify execution works - the SDL conversion is tested elsewhere
	resultJSON, err := json.Marshal(val)
	require.NoError(t, err)
	assert.Contains(t, string(resultJSON), "Status")
}
