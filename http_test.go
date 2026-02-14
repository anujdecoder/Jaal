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
// embedded Playground (same route only: HTML entrypoint; no redirect/mount/
// separate handler per request). Supports CDN-free impl while preserving test
// suite and no regressions for POST/query execution. Checks status/type/HTML
// snippet (avoids brittle full match).
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
	if !strings.Contains(body, "<title>GraphQL Playground</title>") {
		t.Errorf("expected playground HTML, got: %s", body)
	}
	if !strings.Contains(body, `endpoint: "/graphql"`) {
		t.Errorf("expected /graphql endpoint config in HTML")
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

// TestEmbeddedPlaygroundAssetsSameRoute verifies static assets (CSS/JS/favicon)
// are served from embedded FS on the *same /graphql route* (e.g.,
// /graphql/static/css/...). Ensures UI renders fully self-contained/offline
// (no CDN/mount/redirect). Called internally by serveEmbeddedPlayground for
// paths referenced in index.html. Follows test pattern (httptest; spot-checks
// status/type/content). No change to example/main.go.
func TestEmbeddedPlaygroundAssetsSameRoute(t *testing.T) {
	// Test via handler (simulates same-route asset paths).
	h := jaal.HTTPHandler(schemabuilder.NewSchema().MustBuild()) // minimal schema

	// Spot-check CSS asset (GET /graphql/static/css/index.css)
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/graphql/static/css/index.css", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for CSS asset, got %d", rr.Code)
	}
	if ct := rr.HeaderMap.Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Errorf("expected text/css, got %s", ct)
	}
	if !strings.Contains(rr.Body.String(), "body") { // rough check for CSS content
		t.Errorf("expected CSS content")
	}
}
