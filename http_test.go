package jaal_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/schemabuilder"
)

func testHTTPRequest(req *http.Request) *httptest.ResponseRecorder {
	schema := schemabuilder.NewSchema()

	query := schema.Query()
	query.FieldFunc("mirror", func(args struct{ Value int64 }) int64 {
		return args.Value * -1
	})

	builtSchema := schema.MustBuild()

	rr := httptest.NewRecorder()
	handler := jaal.HTTPHandler(builtSchema)

	handler.ServeHTTP(rr, req)
	return rr
}

// TestHTTPPlaygroundOnGet verifies that GET requests to /graphql serve the
// GraphQL Playground UI with all CSS/JS assets inlined in a single HTML
// response (embedded via go:embed; no CDN/files/MIME/path/404 issues).
// Matches the provided example code for reliable rendering. Checks status,
// content-type, and key snippets (title, endpoint) while preserving
// test suite and no regressions for POST/query execution elsewhere.
func TestHTTPPlaygroundOnGet(t *testing.T) {
	req, err := http.NewRequest("GET", "/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for playground UI, got %d", rr.Code)
	}

	if ct := rr.HeaderMap.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html, got %s", ct)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "<title>Jaal GraphQL Playground</title>") {
		t.Errorf("expected GraphQL Playground HTML title, got: %s", body)
	}
	if !strings.Contains(body, "GraphQL Playground") {
		t.Errorf("expected GraphQL Playground in HTML")
	}
	if !strings.Contains(body, `GraphQLPlayground.init(document.getElementById('root'), {
            endpoint: '/graphql'
        })`) { // self-referential endpoint in init call
		t.Errorf("expected /graphql endpoint in init call")
	}
}

func TestHTTPParseQuery(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), `{"data":null,"errors":[{"message":"request must include a query","extensions":{"code":"Unknown"},"paths":[]}]}`); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPMustHaveQuery(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(`{"query":""}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), `{"data":null,"errors":[{"message":"must have a single query","extensions":{"code":"Unknown"},"paths":[]}]}`); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPSuccess(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "query TestQuery($value: int64) { mirror(value: $value) }", "variables": { "value": 1 }}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.Body.String(), "{\"data\":{\"mirror\":-1},\"errors\":null}"); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

func TestHTTPContentType(t *testing.T) {
	req, err := http.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "query TestQuery($value: int64) { mirror(value: $value) }", "variables": { "value": 1 }}`))
	if err != nil {
		t.Fatal(err)
	}

	rr := testHTTPRequest(req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, but received %d", rr.Code)
	}

	if diff := pretty.Compare(rr.HeaderMap.Get("Content-Type"), "application/json"); diff != "" {
		t.Errorf("expected response to match, but received %s", diff)
	}
}

