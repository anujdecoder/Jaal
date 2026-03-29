package directives_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/example/directives"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// postQuery sends a GraphQL query to the test server and returns the decoded
// response body.  Fails the test on transport or decoding errors.
func postQuery(t *testing.T, url, query string) map[string]interface{} {
	t.Helper()
	reqBody, _ := json.Marshal(map[string]string{"query": query})
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// postQueryExpectData is like postQuery but asserts no GraphQL errors and
// returns just the "data" portion.
func postQueryExpectData(t *testing.T, url, query string) map[string]interface{} {
	t.Helper()
	result := postQuery(t, url, query)
	require.Nil(t, result["errors"], "unexpected GraphQL errors: %v", result["errors"])
	return result["data"].(map[string]interface{})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestDirectivesExample_SchemaBuilds(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	require.NotNil(t, h)
}

func TestDirectivesExample_PublicQuery(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	data := postQueryExpectData(t, ts.URL, `{ articles { title author } }`)
	articles := data["articles"].([]interface{})
	require.Len(t, articles, 2)
	a0 := articles[0].(map[string]interface{})
	assert.Equal(t, "Getting Started with Jaal", a0["title"])
}

func TestDirectivesExample_HasRole_Denied(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// secretArticle requires ADMIN — no context role, so should error.
	result := postQuery(t, ts.URL, `{ secretArticle { title } }`)
	require.NotNil(t, result["errors"], "expected access denied error")
}

func TestDirectivesExample_FeatureFlag_Disabled(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// dashboard field is feature-flagged; without the flag context it returns null.
	data := postQueryExpectData(t, ts.URL, `{ dashboard }`)
	assert.Nil(t, data["dashboard"], "dashboard should be null when feature flag is off")
}

func TestDirectivesExample_CacheControl_MetadataOnly(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// @cacheControl has nil handler so query executes normally.
	data := postQueryExpectData(t, ts.URL, `{ cachedArticles { title } }`)
	articles := data["cachedArticles"].([]interface{})
	require.Len(t, articles, 2)
}

func TestDirectivesExample_TypeLevelAuth_Denied(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// adminStats is guarded by type-level @auth; should error without auth.
	result := postQuery(t, ts.URL, `{ adminStats { totalArticles totalUsers } }`)
	require.NotNil(t, result["errors"], "expected unauthenticated error")
}

func TestDirectivesExample_AuditPostResolver(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// auditedArticle triggers @audit PostResolver.
	data := postQueryExpectData(t, ts.URL, `{ auditedArticle { title } }`)
	require.NotNil(t, data["auditedArticle"])

	// Check audit logs populated.
	data2 := postQueryExpectData(t, ts.URL, `{ auditLogs { action } }`)
	logs := data2["auditLogs"].([]interface{})
	require.GreaterOrEqual(t, len(logs), 1)
	assert.Equal(t, "read_article", logs[0].(map[string]interface{})["action"])
}

func TestDirectivesExample_Introspection_CustomDirectives(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	data := postQueryExpectData(t, ts.URL, introspection.IntrospectionQuery)
	schema := data["__schema"].(map[string]interface{})
	dirs := schema["directives"].([]interface{})

	// Collect directive names.
	names := map[string]bool{}
	for _, d := range dirs {
		dm := d.(map[string]interface{})
		names[dm["name"].(string)] = true
	}

	// Custom directives should be present alongside built-ins.
	for _, expected := range []string{"hasRole", "featureFlag", "rateLimit", "audit", "cacheControl", "auth"} {
		assert.True(t, names[expected], "@%s should appear in introspection directives", expected)
	}
	// Built-ins still present.
	for _, builtin := range []string{"include", "skip", "deprecated"} {
		assert.True(t, names[builtin], "@%s (built-in) should still appear", builtin)
	}
}

func TestDirectivesExample_CombinedDirectives_AdminAudit_Denied(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// adminAuditedArticle has @hasRole(ADMIN) + @audit — without role, should
	// fail at the pre-resolver @hasRole, and audit should NOT fire.
	result := postQuery(t, ts.URL, `{ adminAuditedArticle { title } }`)
	require.NotNil(t, result["errors"], "expected access denied error for combined directive field")
}

func TestDirectivesExample_MutationDirective_Denied(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// createArticle requires EDITOR role — should be denied.
	result := postQuery(t, ts.URL, `mutation {
		createArticle(title: "Test", body: "body", author: "anon") { title }
	}`)
	require.NotNil(t, result["errors"], "expected access denied for mutation without EDITOR role")
}

// ---------------------------------------------------------------------------
// Batch / DataLoader integration tests
// ---------------------------------------------------------------------------

func TestDirectivesExample_BatchAuthorProfile(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	data := postQueryExpectData(t, ts.URL, `{
		articles { title authorProfile { name bio } }
	}`)
	articles := data["articles"].([]interface{})
	require.Len(t, articles, 2)

	a0 := articles[0].(map[string]interface{})
	profile0 := a0["authorProfile"].(map[string]interface{})
	assert.Equal(t, "Alice", profile0["name"])
	assert.Contains(t, profile0["bio"].(string), "batch-resolved")

	a1 := articles[1].(map[string]interface{})
	profile1 := a1["authorProfile"].(map[string]interface{})
	assert.Equal(t, "Bob", profile1["name"])
}

func TestDirectivesExample_BatchPreviewWithArgs(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	data := postQueryExpectData(t, ts.URL, `{
		articles { title preview(maxLen: 10) }
	}`)
	articles := data["articles"].([]interface{})
	require.Len(t, articles, 2)

	// "Jaal is a Go framework..." truncated to 10 runes
	p0 := articles[0].(map[string]interface{})["preview"].(string)
	assert.Equal(t, 10, len([]rune(p0)), "preview should be truncated to maxLen")

	// "Directives allow you..." truncated to 10 runes
	p1 := articles[1].(map[string]interface{})["preview"].(string)
	assert.Equal(t, 10, len([]rune(p1)), "preview should be truncated to maxLen")
}

func TestDirectivesExample_BatchMixedWithNormalFields(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// Query both normal fields (title, author) and batch field (authorProfile)
	data := postQueryExpectData(t, ts.URL, `{
		articles { title author authorProfile { name } }
	}`)
	articles := data["articles"].([]interface{})
	require.Len(t, articles, 2)

	a0 := articles[0].(map[string]interface{})
	assert.Equal(t, "Getting Started with Jaal", a0["title"])
	assert.Equal(t, "Alice", a0["author"])
	assert.Equal(t, "Alice", a0["authorProfile"].(map[string]interface{})["name"])
}

func TestDirectivesExample_BatchIntrospection(t *testing.T) {
	h, err := directives.GetGraphqlServer()
	require.NoError(t, err)
	ts := httptest.NewServer(h)
	defer ts.Close()

	data := postQueryExpectData(t, ts.URL, `{
		__type(name: "Article") {
			fields { name description }
		}
	}`)
	fields := data["__type"].(map[string]interface{})["fields"].([]interface{})

	fieldNames := map[string]string{}
	for _, f := range fields {
		fm := f.(map[string]interface{})
		name := fm["name"].(string)
		desc := ""
		if d, ok := fm["description"].(string); ok {
			desc = d
		}
		fieldNames[name] = desc
	}

	assert.Contains(t, fieldNames, "authorProfile", "batch field visible in introspection")
	assert.Contains(t, fieldNames["authorProfile"], "batch", "description mentions batching")
	assert.Contains(t, fieldNames, "preview", "batch-with-args field visible in introspection")
}

// ---------------------------------------------------------------------------
// Direct schema build test (no HTTP)
// ---------------------------------------------------------------------------

func TestDirectivesExample_DirectSchemaBuild(t *testing.T) {
	// Build schema programmatically using the exported registration functions.
	sb := schemabuilder.NewSchema()
	server := directives.NewServer()
	directives.RegisterSchema(sb, server)

	schema, err := sb.Build()
	require.NoError(t, err)
	introspection.AddIntrospectionToSchema(schema)

	require.NotNil(t, schema.Query)
	require.NotNil(t, schema.Mutation)

	// Custom directives present in schema metadata.
	require.GreaterOrEqual(t, len(schema.CustomDirectives), 6,
		"expected at least 6 custom directives (hasRole, featureFlag, rateLimit, audit, cacheControl, auth)")
}
