package jaal

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/jerrors"
)

type HandlerOption func(*handlerOptions)

type handlerOptions struct {
	Middlewares []MiddlewareFunc
}

// HTTPHandler implements the handler required for executing the graphql queries and mutations.
// For GET requests to /graphql, it serves the embedded GraphQL Playground UI
// (index.html + assets like CSS/JS/favicon from FS; see serveEmbeddedPlayground).
// This allows interactive exploration (no separate handler/mount/redirect/CDN;
// same route only, no example changes). For POST requests from clients, it
// executes the query as before (preserving compatibility with examples,
// README.md, and existing clients). Other methods return an error.
// The playground assets are embedded via go:embed (stdlib only), following the
// minimalistic pattern of the codebase.
func HTTPHandler(schema *graphql.Schema, opts ...HandlerOption) http.Handler {
	h := &httpHandler{
		handler: handler{
			schema:   schema,
			executor: &graphql.Executor{},
		},
	}

	o := handlerOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	prev := h.execute
	for i := range o.Middlewares {
		prev = o.Middlewares[len(o.Middlewares)-1-i](prev)
	}
	h.exec = prev

	return h
}

type handler struct {
	schema   *graphql.Schema
	executor *graphql.Executor
}

type httpHandler struct {
	handler

	exec HandlerFunc
}

type httpPostBody struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type httpResponse struct {
	Data   interface{}      `json:"data"`
	Errors []*jerrors.Error `json:"errors"`
}

//go:embed playground
// playgroundFiles embeds the GraphQL Playground static assets (index.html, CSS,
// JS, favicon) directly into the Go binary at build time. This allows the server
// to serve the complete, self-contained UI without any CDN or network dependencies,
// ensuring offline/isolated operation.
//
// The embedded structure:
// - playground/index.html (customized; see below for config)
// - playground/static/css/index.css, /static/js/middleware.js, /favicon.png
//
// Assets sourced from graphql-playground-react (no Go deps added). See
// PlaygroundHandler and serveGraphQLPlayground for serving logic.
// Follows stdlib embedding (Go 1.16+; our go.mod supports) and codebase's
// minimal pattern (cf. introspection files).
var playgroundFiles embed.FS

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve embedded Playground on same /graphql route for GET requests (UI +
	// assets; see serveEmbeddedPlayground). Fulfills: /graphql URL opened in
	// browser spins up playground, no separate handler/redirect/mount/CDN,
	// no example changes. Simplified GET check (no browser detection) per
	// request. Mirrors ws.go routing (method/path before query execution).
	if r.Method == http.MethodGet {
		serveEmbeddedPlayground(w, r)
		return
	}

	// writeResponse is a closure that formats execution results or errors as JSON,
	// following the error handling style used throughout the codebase (e.g.,
	// jerrors.ConvertError, httpResponse struct). Non-POST requests (now only
	// non-GET methods like PUT/DELETE hit this) return an error.
	writeResponse := func(value interface{}, err error) {
		response := httpResponse{}
		if err != nil {
			response.Errors = []*jerrors.Error{jerrors.ConvertError(err)}
		} else {
			response.Data = value
		}

		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		_, _ = w.Write(responseJSON)
	}

	// Original non-POST error for compatibility with clients sending queries
	// via POST (as in README.md, examples, client.go, and tests).
	if r.Method != "POST" {
		writeResponse(nil, errors.New("request must be a POST"))
		return
	}

	if r.Body == nil {
		writeResponse(nil, errors.New("request must include a query"))
		return
	}

	var params httpPostBody
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeResponse(nil, err)
		return
	}

	query, err := graphql.Parse(params.Query, params.Variables)
	if err != nil {
		writeResponse(nil, err)
		return
	}

	root := h.schema.Query
	if query.Kind == "mutation" {
		root = h.schema.Mutation
	}

	if err := graphql.ValidateQuery(r.Context(), root, query.SelectionSet); err != nil {
		writeResponse(nil, err)
		return
	}

	ctx := addVariables(r.Context(), params.Variables)

	output, err := h.exec(ctx, root, query)
	writeResponse(output, err)
}

func (h *httpHandler) execute(ctx context.Context, root graphql.Type, query *graphql.Query) (interface{}, error) {
	return h.executor.Execute(ctx, root, nil, query)
}

type graphqlVariableKeyType int

const graphqlVariableKey graphqlVariableKeyType = 0

// ExtractVariables is used to returns the variables received as part of the graphql request.
// This is intended to be used from within the interceptors.
func ExtractVariables(ctx context.Context) map[string]interface{} {
	if v := ctx.Value(graphqlVariableKey); v != nil {
		return v.(map[string]interface{})
	}

	return nil
}

func addVariables(ctx context.Context, v map[string]interface{}) context.Context {
	return context.WithValue(ctx, graphqlVariableKey, v)
}

// getPlaygroundFS returns a sub-FS rooted at the embedded "playground/" dir for
// serving assets. Internal helper (no export) to keep single-handler API.
func getPlaygroundFS() (http.FileSystem, error) {
	fsys, err := fs.Sub(playgroundFiles, "playground")
	if err != nil {
		return nil, fmt.Errorf("jaal: failed to embed playground assets: %w", err)
	}
	return http.FS(fsys), nil
}

// serveEmbeddedPlayground serves the GraphQL Playground on the *same /graphql
// route* (no separate handler, no redirect, no example changes). 
// - If GET /graphql (or /graphql/): serves index.html (embedded UI entrypoint).
// - If GET /graphql/static/... or /graphql/favicon.png: serves asset from
//   embedded FS (stripped internally; index.html references these paths).
// - Enables full UI render offline/self-contained (no CDN/network).
// - For POST: query execution (unchanged; see ServeHTTP).
// Pattern follows conditional logic in ws.go/http.go (method/path checks),
// using stdlib FileServer/FS for assets. HTML paths updated in index.html.
func serveEmbeddedPlayground(w http.ResponseWriter, r *http.Request) {
	// Root UI page.
	if r.URL.Path == "/graphql" || r.URL.Path == "/graphql/" {
		// Serve index.html directly from embed (sets type; no FS for simplicity).
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexBytes, err := fs.ReadFile(playgroundFiles, "playground/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(indexBytes)
		return
	}

	// Assets (CSS/JS/favicon) under /graphql/... for same-route serving.
	// Matches paths in index.html (e.g., /graphql/static/css/index.css).
	if strings.HasPrefix(r.URL.Path, "/graphql/static/") || r.URL.Path == "/graphql/favicon.png" {
		fsys, err := getPlaygroundFS()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Strip /graphql prefix internally to map to FS root (e.g., /graphql/static/css/...
		// -> static/css/... in embed).
		http.StripPrefix("/graphql", http.FileServer(fsys)).ServeHTTP(w, r)
		return
	}

	// Non-UI GET: not found (keeps GraphQL endpoint clean).
	http.NotFound(w, r)
}
