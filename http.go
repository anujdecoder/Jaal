package jaal

import (
	"context"
	_ "embed" // for GraphiQL assets inlined in HTML (no runtime FS/CDN)
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

// Embedded GraphiQL assets (downloaded from CDN once during development; inlined
// in HTML at runtime for no external loads/CDN/MIME/path issues). This matches
// the provided example code for reliable browser rendering (single HTML response).
//go:embed playground/react.production.min.js
var reactJS []byte

//go:embed playground/react-dom.production.min.js
var reactDOMJS []byte

//go:embed playground/graphiql.min.js
var graphiqlJS []byte

//go:embed playground/graphiql.min.css
var graphiqlCSS []byte

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve the GraphiQL playground UI on GET (and HEAD for completeness, e.g.
	// curl -I) requests. This makes visiting the /graphql URL in a browser show
	// the interactive playground with all assets inlined in a single HTML
	// response (embedded; no CDN/files/MIME/path issues). Uses r.URL.Path as
	// the GraphQL endpoint (self-referential). POST requests are handled as
	// GraphQL queries below. Matches the provided example code.
	// HEAD support follows common HTTP practices for UI endpoints.
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		// Title "Jaal GraphiQL Playground"; endpoint = current path (e.g., /graphql).
		servePlayground(w, "Jaal GraphiQL Playground", r.URL.Path)
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

// playgroundHTMLTemplate is the GraphiQL playground HTML shell (the %s placeholders
// are for the title, CSS, 3 JS files, and endpoint; assets are inlined from embeds
// at runtime to avoid any CDN/external loads or separate files/MIME/path issues).
// This matches the provided example code exactly for reliable browser rendering
// (single HTML response on GET to /graphql).
const playgroundHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8" />
    <title>%s</title>
    <style>
        body {
            height: 100%%;
            margin: 0;
            overflow: hidden;
        }
        #graphiql {
            height: 100vh;
        }
    </style>
    <style>
%s
    </style>
    <script>
%s
    </script>
    <script>
%s
    </script>
    <script>
%s
    </script>
</head>
<body>
    <div id="graphiql">Loading...</div>
    <script>
      // The GraphQL fetcher posts to the graphqlEndpoint (self-referential to
      // the current /graphql route for queries/mutations).
      function graphQLFetcher(graphQLParams) {
        return fetch(
          '%s',
          {
            method: 'post',
            headers: {
              Accept: 'application/json',
              'Content-Type': 'application/json',
            },
            body: JSON.stringify(graphQLParams),
            credentials: 'omit',
          },
        ).then(function (response) {
          return response.json().catch(function () {
            return response.text();
          });
        });
      }

      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: graphQLFetcher,
        }),
        document.getElementById('graphiql'),
      );
    </script>
</body>
</html>`

// servePlayground writes the GraphiQL HTML with all CSS/JS assets inlined from
// embeds (no CDN, no separate files). Used by the automatic GET/HEAD handling
// in HTTPHandler (self-contained on /graphql route). Casts []byte embeds to
// string for fmt.Sprintf. This ensures no MIME/404/render errors.
func servePlayground(w http.ResponseWriter, title, graphqlEndpoint string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Inline embedded assets directly into <style>/<script> tags (per example).
	html := fmt.Sprintf(playgroundHTMLTemplate, title, string(graphiqlCSS), string(reactJS), string(reactDOMJS), string(graphiqlJS), graphqlEndpoint)
	_, _ = w.Write([]byte(html))
}
