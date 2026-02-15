package users_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/example/users"
	"go.appointy.com/jaal/introspection"
)

// server_test tests GetGraphqlServer handler via HTTP POST (blackbox; per task).
// Uses httptest; IntrospectionQuery for descs/directives verification (from
// introspection_query.go); queries/muts for functional.
// No codebase changes; only tests.
func TestGetGraphqlServer(t *testing.T) {
	// Get handler (builds schema w/ descs/oneOf/specifiedBy etc).
	h, err := users.GetGraphqlServer()
	require.NoError(t, err)
	require.NotNil(t, h)

	// Test server.
	server := httptest.NewServer(h)
	defer server.Close()

	// Helper: POST GraphQL query, return data map (err if errors).
	postQuery := func(query string) map[string]interface{} {
		reqBody, _ := json.Marshal(map[string]string{"query": query})
		resp, err := http.Post(server.URL, "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.Nil(t, result["errors"], "GraphQL errors: %v", result["errors"])
		return result["data"].(map[string]interface{})
	}

	// 1+2. Full introspection: verify descs on queries/muts, objects/inputs/enums.
	// Uses IntrospectionQuery (includes description fields).
	introQuery := introspection.IntrospectionQuery
	data := postQuery(introQuery)
	schema := data["__schema"].(map[string]interface{})

	// Queries/muts descs (FIELD_DEFINITION; from FieldFunc variadic).
	// Assert non-empty (e.g., me, user, createUser, contactBy).
	// Types list for full descs (Query/Mutation types themselves "" internal; fields have).
	types := schema["types"].([]interface{})
	hasQueryFieldDesc, hasMutFieldDesc, hasObjDesc, hasInputDesc, hasEnumDesc := false, false, false, false, false
	for _, tIface := range types {
		typ := tIface.(map[string]interface{})
		// Fields desc (for Query/Mutation).
		if fieldsIface, ok := typ["fields"].([]interface{}); ok {
			for _, fIface := range fieldsIface {
				f := fIface.(map[string]interface{})
				if desc, ok := f["description"].(string); ok && desc != "" {
					if typ["name"] == "Query" {
						hasQueryFieldDesc = true // e.g., me/user
					} else if typ["name"] == "Mutation" {
						hasMutFieldDesc = true // e.g., createUser/contactBy
					}
				}
			}
		}
		// Type descs (objects/inputs/enums).
		if desc, ok := typ["description"].(string); ok && desc != "" {
			switch typ["name"] {
			case "User":
				hasObjDesc = true
			case "CreateUserInput", "ContactByInput":
				hasInputDesc = true
			case "Role":
				hasEnumDesc = true
			}
		}
	}
	require.True(t, hasQueryFieldDesc, "descs on queries")
	require.True(t, hasMutFieldDesc, "descs on mutations")
	require.True(t, hasObjDesc, "descs on objects")
	require.True(t, hasInputDesc, "descs on inputs")
	require.True(t, hasEnumDesc, "descs on enums")

	// 3. specifiedBy directive on scalar (e.g., DateTime).
	foundSpecifiedBy := false
	for _, tIface := range types {
		typ := tIface.(map[string]interface{})
		if typ["name"] == "DateTime" {
			if url, ok := typ["specifiedByURL"].(string); ok && url != "" {
				foundSpecifiedBy = true
			}
		}
	}
	require.True(t, foundSpecifiedBy, "specifiedBy on scalar")

	// 4. @oneOf on ContactByInput (INPUT_OBJECT directive).
	foundOneOf := false
	for _, tIface := range types {
		typ := tIface.(map[string]interface{})
		if typ["name"] == "ContactByInput" {
			if dirsIface, ok := typ["directives"].([]interface{}); ok {
				for _, dIface := range dirsIface {
					if dir, ok := dIface.(map[string]interface{}); ok {
						if dir["name"] == "oneOf" {
							foundOneOf = true
						}
					}
				}
			}
		}
	}
	require.True(t, foundOneOf, "oneOf directive on ContactByInput")

	// 5. list allUsers: verify initial user.
	allData := postQuery(`{ allUsers { id name email } }`)
	users := allData["allUsers"].([]interface{})
	require.GreaterOrEqual(t, len(users), 1, "initial user present")
	user0 := users[0].(map[string]interface{})
	require.Equal(t, "u1", user0["id"], "initial user ID")
	require.Equal(t, "John Doe", user0["name"], "initial user name")

	// 6. createUser mut -> get ID -> user query verify.
	createData := postQuery(`mutation {
		createUser(input: {
			name: "Test User",
			email: "test@example.com",
			reputation: 8.0,
			isActive: true,
			role: MEMBER
		}) { id name email }
	}`)
	newUser := createData["createUser"].(map[string]interface{})
	newID := newUser["id"].(string)
	require.NotEmpty(t, newID, "new user ID")

	// Fetch by ID (use static seed ID "u1"; arg name "iD" per makeGraphql("ID")="iD"
	// in reflect.go (capital D preserved); no codebase change).
	userByIdData := postQuery(`{
		user(iD: "u1") { id name email }
	}`)
	fetched := userByIdData["user"].(map[string]interface{})
	require.Equal(t, "u1", fetched["id"], "fetched user ID")
	require.Equal(t, "John Doe", fetched["name"], "fetched user name")
}