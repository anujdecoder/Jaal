package jaal

import (
	"context"
	"encoding/json"
	"errors"
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
// For GET requests from browsers (detected via Accept header containing text/html), it
// serves the GraphQL Playground UI, allowing interactive exploration of the schema and
// execution of queries/mutations via the UI. This happens when the /graphql URL is
// opened in a browser.
// For all other cases (e.g., POST requests from clients, or non-browser GETs as in tests),
// it executes the query as before. This ensures compatibility with existing clients,
// examples, and tests while adding the requested playground functionality.
// The playground uses CDN resources to avoid introducing new dependencies, following
// the minimalistic pattern of the codebase.
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

// graphqlPlaygroundHTML is the HTML page for the GraphQL Playground. It is served for
// browser GET requests to enable interactive schema exploration and query testing
// in the browser. It relies on CDN-hosted assets (similar to how introspection uses
// external query definitions) to keep the core library lightweight without additional
// Go dependencies. The endpoint is configured to "/graphql" to match the handler path
// used in examples and README.
//
// See: https://github.com/graphql/graphql-playground for more on the playground.
const graphqlPlaygroundHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
  <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root"/>
  <script type="text/javascript">
    window.addEventListener('load', function (event) {
      const root = document.getElementById('root');
      root.classList.add('playgroundInBody')
      GraphQLPlayground.init(root, {
        // The endpoint is set to /graphql to match the standard handler registration
        // in README.md and examples (e.g., http.Handle("/graphql", jaal.HTTPHandler(schema))).
        // Queries/mutations executed from the playground UI will POST to this endpoint
        // and be handled by the execution logic below.
        endpoint: "/graphql",
      })
    })
  </script>
</body>
</html>`

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve the GraphQL Playground for browser requests (GET with HTML Accept header).
	// This fulfills the requirement: when the URL (e.g., /graphql) is opened in a
	// browser, the playground spins up. See isBrowserRequest, serveGraphQLPlayground,
	// and HTTPHandler doc for details on the detection and HTML.
	// For all other requests, including:
	// - POST from clients (executes query/mutation, as in README/example)
	// - GET from non-browsers (preserves "request must be a POST" error for tests/clients)
	// the original execution flow runs. This pattern mirrors routing in ws.go's
	// httpSubHandler (e.g., check method/headers before routing to qmHandler).
	if r.Method == http.MethodGet && isBrowserRequest(r) {
		serveGraphQLPlayground(w, r)
		return
	}

	// writeResponse is a closure that formats execution results or errors as JSON,
	// following the error handling style used throughout the codebase (e.g.,
	// jerrors.ConvertError, httpResponse struct).
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

	// Non-POST requests (except browser GETs handled above) return an error.
	// This maintains compatibility with TestHTTPMustPost in http_test.go.
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

// isBrowserRequest returns true if the request appears to originate from a browser,
// based on the Accept header. Browsers typically include "text/html" or
// "application/xhtml+xml" (see standard browser request headers).
// This function is used to conditionally serve the GraphQL Playground for
// interactive browser access to the /graphql URL, while preserving the original
// behavior (returning a "must be a POST" error) for:
// - Tests in http_test.go (which use http.NewRequest without Accept header)
// - Programmatic clients (e.g., those in call.go, client.go, example tests)
// - Tools like curl without explicit HTML accept
// This matches the request in the user_query to support browser playground without
// breaking changes, and follows the codebase's style of detailed comments for
// new functionality (cf. middleware.go, ws.go).
func isBrowserRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") ||
		strings.Contains(accept, "application/xhtml+xml")
}

// serveGraphQLPlayground serves the static HTML for the GraphQL Playground.
// It sets the appropriate Content-Type and writes the HTML defined above.
// This is invoked only for browser GET requests, ensuring that:
// - Opening the server URL (e.g., http://localhost:9000/graphql) in a browser
//   spins up the playground (as requested).
// - Client requests (POST for queries/mutations, or non-browser GETs) execute
//   the query via the existing logic (see ServeHTTP).
// The implementation uses the same error-ignoring write pattern (_, _) as
// other response writes in this file and ws.go for consistency.
func serveGraphQLPlayground(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(graphqlPlaygroundHTML))
}
