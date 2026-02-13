package main_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	testex "go.appointy.com/jaal/test-example"
)

// TestFullFeatures follows test_plan.md: detailed subtests for ALL Jaal features in server.go (UUID @specifiedBy, Role enum, Node interface, User/DeletedUser, UserResult union, UserStatus, UserIdentifierInput @oneOf, CreateUserInput, Query/Mutation).
// Each covers the plan items (introspection, queries, mutations, oneOf cases, errors, etc).
func TestFullFeatures(t *testing.T) {
	handler := testex.HTTPHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	client := ts.Client()

	// Helper to post GraphQL.
	postQuery := func(query string) map[string]interface{} {
		body := map[string]string{"query": query}
		b, _ := json.Marshal(body)
		resp, err := client.Post(ts.URL, "application/json", bytes.NewReader(b))
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return result
	}

	// 2. Introspection Tests (full schema, new specs @specifiedBy/@oneOf).
	t.Run("fullIntrospection", func(t *testing.T) {
		// Full introspection query (from graphql.org + Jaal).
		q := `query IntrospectionQuery {
			__schema {
				queryType { name }
				mutationType { name }
				directives {
					name
					description
					locations
					args {
						name
						description
						type { name kind }
					}
				}
				types {
					name
					kind
					specifiedByURL
					isOneOf
					fields { name }
					interfaces { name }
					possibleTypes { name }
					enumValues { name }
					inputFields { name }
				}
			}
		}`
		res := postQuery(q)
		data := res["data"].(map[string]interface{})
		schema := data["__schema"].(map[string]interface{})
		require.NotNil(t, schema["queryType"])
		require.NotNil(t, schema["mutationType"])

		// Assert @specifiedBy and @oneOf directives.
		directives := schema["directives"].([]interface{})
		hasSpecifiedBy := false
		hasOneOf := false
		for _, d := range directives {
			dm := d.(map[string]interface{})
			if dm["name"] == "specifiedBy" {
				hasSpecifiedBy = true
				require.Contains(t, dm["locations"], "SCALAR")
			}
			if dm["name"] == "oneOf" {
				hasOneOf = true
				require.Contains(t, dm["locations"], "INPUT_OBJECT")
			}
		}
		require.True(t, hasSpecifiedBy)
		require.True(t, hasOneOf)

		// Assert types/fields (UUID scalar, Role enum, Node interface, UserResult union, UserIdentifierInput oneOf, etc).
		types := schema["types"].([]interface{})
		for _, typ := range types {
			tm := typ.(map[string]interface{})
			name := tm["name"].(string)
			kind := tm["kind"].(string)
			if name == "Node" {
				require.Equal(t, "INTERFACE", kind)
			}
			if name == "User" || name == "DeletedUser" {
				require.Equal(t, "OBJECT", kind)
			}
			if name == "Role" {
				require.Equal(t, "ENUM", kind)
			}
			if name == "UserIdentifierInput" {
				require.Equal(t, "INPUT_OBJECT", kind)
				require.True(t, tm["isOneOf"].(bool))
			}
			if name == "UUID" {
				require.Equal(t, "SCALAR", kind)
				require.NotNil(t, tm["specifiedByURL"])
			}
		}
	})

	t.Run("specifiedByOnUUID", func(t *testing.T) {
		q := `{
			__type(name: "UUID") {
				name
				kind
				specifiedByURL
			}
		}`
		res := postQuery(q)
		uuidType := res["data"].(map[string]interface{})["__type"].(map[string]interface{})
		require.Equal(t, "UUID", uuidType["name"])
		require.Equal(t, "SCALAR", uuidType["kind"])
		require.Equal(t, "https://tools.ietf.org/html/rfc4122", uuidType["specifiedByURL"])
	})

	// 3. Query Tests (all queries/fields from schema).
	t.Run("queries", func(t *testing.T) {
		// me with all fields.
		q := `{
			me {
				id
				uuid
				username
				email
				role
				status {
					isActive
					lastLogin
				}
			}
		}`
		res := postQuery(q)
		require.Nil(t, res["errors"])
		data := res["data"].(map[string]interface{})
		require.NotNil(t, data["me"])

		// user with oneOf (id).
		q = `{
			user(by: {id: "u1"}) {
				... on User {
					username
				}
				... on DeletedUser {
					deletedAt
				}
			}
		}`
		res = postQuery(q)
		require.Nil(t, res["errors"])

		// allUsers.
		q = `{
			allUsers {
				id
				username
			}
		}`
		res = postQuery(q)
		require.Nil(t, res["errors"])
	})

	// 4. Mutation Tests (fire all, including oneOf).
	t.Run("mutations", func(t *testing.T) {
		// createUser.
		q := `mutation {
			createUser(input: {username: "new", email: "new@example.com", role: MEMBER}) {
				id
				username
			}
		}`
		res := postQuery(q)
		require.Nil(t, res["errors"])
		require.NotNil(t, res["data"].(map[string]interface{})["createUser"])

		// updateUserRole.
		q = `mutation {
			updateUserRole(id: "u1", newRole: ADMIN) {
				role
			}
		}`
		res = postQuery(q)
		require.Nil(t, res["errors"])
	})

	// 5. @oneOf Validation Tests.
	t.Run("oneOfValidation", func(t *testing.T) {
		// Case 1: Success (exactly one field: id).
		q := `{
			user(by: {id: "u1"}) {
				... on User {
					username
				}
			}
		}`
		res := postQuery(q)
		require.Nil(t, res["errors"])

		// Case 2: Failure (two fields).
		q = `{
			user(by: {id: "u1", email: "test@example.com"}) {
				... on User {
					username
				}
			}
		}`
		res = postQuery(q)
		require.NotNil(t, res["errors"]) // oneOf violation

		// Case 3: Failure (zero fields).
		q = `{
			user(by: {}) {
				... on User {
					username
				}
			}
		}`
		res = postQuery(q)
		require.NotNil(t, res["errors"]) // oneOf violation
	})

	// 6. @specifiedBy Verification (already in introspection).

	// 7. Edge/Feature Tests (interfaces/unions/enums/error cases, non-existent field, etc).
	t.Run("edges", func(t *testing.T) {
		// Interface fragment, union.
		q := `{
			user(by: {id: "u1"}) {
				... on Node {
					id
				}
			}
		}`
		res := postQuery(q)
		require.Nil(t, res["errors"])

		// Non-existent field error.
		q = `{
			me {
				nonExistent
			}
		}`
		res = postQuery(q)
		require.NotNil(t, res["errors"])

		// Invalid enum.
		q = `mutation {
			updateUserRole(id: "u1", newRole: INVALID)
		}`
		res = postQuery(q)
		require.NotNil(t, res["errors"])
	})
}
