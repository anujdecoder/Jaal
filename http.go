package jaal

import (
	"context"
	_ "embed" // for GraphQL Playground assets inlined in HTML (no runtime FS/CDN)
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/jerrors"
)

type HandlerOption func(*handlerOptions)

type handlerOptions struct {
	Middlewares []MiddlewareFunc
}

// HTTPHandler implements the handler required for executing the graphql queries and mutations.
// For GET requests to /graphql, it serves the embedded GraphQL Playground UI
// (HTML + CSS/JS/favicon inlined; no runtime filesystem or CDN). This allows
// interactive exploration on the same route without extra handlers. For POST
// requests from clients, it executes the query as before (preserving
// compatibility with examples and existing clients). Other methods return an error.
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

// Embedded GraphQL Playground assets (downloaded once during development; inlined
// in HTML at runtime for no external loads/CDN/MIME/path issues).
//go:embed playground/graphql-playground-middleware.js
var playgroundJS []byte

//go:embed playground/graphql-playground-middleware.css
var playgroundCSS []byte

//go:embed playground/graphql-playground-favicon.png
var playgroundFavicon []byte

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve the GraphQL Playground UI on GET (and HEAD for completeness, e.g.
	// curl -I) requests. This makes visiting the /graphql URL in a browser show
	// the interactive playground with all assets inlined in a single HTML
	// response (embedded; no CDN/files/MIME/path issues). Uses r.URL.Path as
	// the GraphQL endpoint (self-referential). POST requests are handled as
	// GraphQL queries below.
	// HEAD support follows common HTTP practices for UI endpoints.
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		// Title "Jaal GraphQL Playground"; endpoint = current path (e.g., /graphql).
		servePlayground(w, "Jaal GraphQL Playground", r.URL.Path)
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

// playgroundHTMLTemplate is the GraphQL Playground HTML shell (the %s placeholders
// are for the title, favicon, CSS, JS, and endpoint). Assets are inlined from
// embeds at runtime to avoid any CDN/external loads or separate files/MIME/path issues.
const playgroundHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="user-scalable=no,initial-scale=1,minimum-scale=1,maximum-scale=1,minimal-ui">
    <title>%s</title>
    <link rel="icon" type="image/png" href="%s" />
    <style>
%s
    </style>
</head>
<body>
    <div id="root">
        <style>html{font-family:"Open Sans",sans-serif;overflow:hidden}body{margin:0;background:#172a3a}.playgroundIn{-webkit-animation:playgroundIn .5s ease-out forwards;animation:playgroundIn .5s ease-out forwards}@-webkit-keyframes playgroundIn{from{opacity:0;-webkit-transform:translateY(10px);-ms-transform:translateY(10px);transform:translateY(10px)}to{opacity:1;-webkit-transform:translateY(0);-ms-transform:translateY(0);transform:translateY(0)}}@keyframes playgroundIn{from{opacity:0;-webkit-transform:translateY(10px);-ms-transform:translateY(10px);transform:translateY(10px)}to{opacity:1;-webkit-transform:translateY(0);-ms-transform:translateY(0);transform:translateY(0)}}</style>
        <style>.fadeOut{-webkit-animation:fadeOut .5s ease-out forwards;animation:fadeOut .5s ease-out forwards}@-webkit-keyframes fadeIn{from{opacity:0;-webkit-transform:translateY(-10px);-ms-transform:translateY(-10px);transform:translateY(-10px)}to{opacity:1;-webkit-transform:translateY(0);-ms-transform:translateY(0);transform:translateY(0)}}@keyframes fadeIn{from{opacity:0;-webkit-transform:translateY(-10px);-ms-transform:translateY(-10px);transform:translateY(-10px)}to{opacity:1;-webkit-transform:translateY(0);-ms-transform:translateY(0);transform:translateY(0)}}@-webkit-keyframes fadeOut{from{opacity:1;-webkit-transform:translateY(0);-ms-transform:translateY(0);transform:translateY(0)}to{opacity:0;-webkit-transform:translateY(-10px);-ms-transform:translateY(-10px);transform:translateY(-10px)}}@keyframes fadeOut{from{opacity:1;-webkit-transform:translateY(0);-ms-transform:translateY(0);transform:translateY(0)}to{opacity:0;-webkit-transform:translateY(-10px);-ms-transform:translateY(-10px);transform:translateY(-10px)}}@-webkit-keyframes appear{from{opacity:0}to{opacity:1}}@keyframes appear{from{opacity:0}to{opacity:1}}</style>
        <div class="playgroundIn" id="loading">Loading GraphQL Playground</div>
    </div>
    <script>
%s
    </script>
    <script>
        GraphQLPlayground.init(document.getElementById('root'), {
            endpoint: '%s'
        });
    </script>
</body>
</html>`

// servePlayground writes the GraphQL Playground HTML with all CSS/JS assets inlined from
// embeds (no CDN, no separate files). Used by the automatic GET/HEAD handling
// in HTTPHandler (self-contained on /graphql route). Casts []byte embeds to
// string for fmt.Sprintf. This ensures no MIME/404/render errors.
func servePlayground(w http.ResponseWriter, title, graphqlEndpoint string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	favicon := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(playgroundFavicon))
	html := fmt.Sprintf(playgroundHTMLTemplate, title, favicon, string(playgroundCSS), string(playgroundJS), graphqlEndpoint)
	_, _ = w.Write([]byte(html))
}
