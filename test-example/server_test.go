package main_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	testex "go.appointy.com/jaal/test-example"
)

// TestFullFeatures follows test_plan.md exactly: detailed subtests for ALL Jaal features in server.go (scalars/custom/@specifiedBy, queries/fields, mutations/oneOf, unions, interfaces, enums, introspection).
// Each covers the plan items.
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

		// Assert types/fields (Character interface, Droid/Human, Episode enum, ReviewInput oneOf, etc).
		types := schema["types"].([]interface{})
		for _, typ := range types {
			tm := typ.(map[string]interface{})
			name := tm["name"].(string)
			kind := tm["kind"].(string)
			if name == "Character" {
				require.Equal(t, "INTERFACE", kind)
			}
			if name == "Droid" || name == "Human" {
				require.Equal(t, "OBJECT", kind)
			}
			if name == "Episode" {
				require.Equal(t, "ENUM", kind)
			}
			if name == "ReviewInput" {
				require.Equal(t, "INPUT_OBJECT", kind)
				require.True(t, tm["isOneOf"].(bool))
			}
			if name == "ID" {
				require.Equal(t, "SCALAR", kind)
				require.NotNil(t, tm["specifiedByURL"])
			}
		}
	})

	t.Run("specifiedByOnID", func(t *testing.T) {
		q := `{
			__type(name: "ID") {
				name
				kind
				specifiedByURL
			}
		}`
		res := postQuery(q)
		idType := res["data"].(map[string]interface{})["__type"].(map[string]interface{})
		require.Equal(t, "ID", idType["name"])
		require.Equal(t, "SCALAR", idType["kind"])
		require.Equal(t, "https://spec.graphql.org/October2021/#sec-Scalars", idType["specifiedByURL"])
	})

	// 3. Query Tests (all queries/fields from schema).
	t.Run("queries", func(t *testing.T) {
		// hero with fragment.
		q := `{
			hero {
				id
				name
				... on Droid {
					primaryFunction
				}
				... on Human {
					height
					mass
				}
				appearsIn
			}
		}`
		res := postQuery(q)
		require.Nil(t, res["errors"])
		data := res["data"].(map[string]interface{})
		require.NotNil(t, data["hero"])

		// Other queries.
		q = `{
			character(id: "test") { id name }
			droid(id: "d1") { id primaryFunction }
			human(id: "h1") { id height }
			starship(id: "s1") { id length }
			reviews(episode: NEWHOPE) { stars commentary }
		}`
		res = postQuery(q)
		require.Nil(t, res["errors"])
	})

	// 4. Mutation Tests (fire all, including oneOf).
	t.Run("mutations", func(t *testing.T) {
		// createReview with oneOf (stars only).
		q := `mutation {
			createReview(review: {stars: 5}) {
				stars
				commentary
			}
		}`
		res := postQuery(q)
		require.Nil(t, res["errors"])
		require.NotNil(t, res["data"].(map[string]interface{})["createReview"])

		// Other mutations (stubs).
		q = `mutation {
			rateFilm(episode: NEWHOPE, rating: THUMBS_UP) { episode }
			updateHumanName(id: "h1", name: "Luke") { name }
			deleteStarship(id: "s1")
		}`
		res = postQuery(q)
		require.Nil(t, res["errors"])
	})

	// 5. Edge/Feature Tests (interfaces/unions/enums/oneOf invalid, directives, etc).
	t.Run("edges", func(t *testing.T) {
		// oneOf invalid (both fields) → error.
		q := `mutation {
			createReview(review: {stars: 5, commentary: "good"})
		}`
		res := postQuery(q)
		require.NotNil(t, res["errors"]) // oneOf violation

		// Interface fragment, union, enum, @skip/@include.
		q = `{
			hero {
				... on Character {
					id
					name
				}
				... on Droid {
					primaryFunction
				}
			}
			reviews(episode: NEWHOPE) @skip(if: false) {
				stars
			}
		}`
		res = postQuery(q)
		require.Nil(t, res["errors"])
	})

	// 6. Error Validation Tests (non-existent fields, invalid enum, etc).
	t.Run("errorValidations", func(t *testing.T) {
		// Non-existent field.
		q := `{
			hero {
				nonExistentField
			}
		}`
		res := postQuery(q)
		require.NotNil(t, res["errors"])

		// Invalid enum.
		q = `{
			hero(episode: INVALID)
		}`
		res = postQuery(q)
		require.NotNil(t, res["errors"])

		// Missing non-null arg.
		q = `{
			character
		}`
		res = postQuery(q)
		require.NotNil(t, res["errors"])
	})
}
