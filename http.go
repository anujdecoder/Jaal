package jaal

import (
	"context"
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

// HTTPHandler implements the handler required for executing the graphql queries and mutations
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

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

// playgroundHTML is a simple HTML page that loads GraphiQL from CDN
// to provide an interactive GraphQL playground. It is used by
// PlaygroundHandler.
const playgroundHTML = `<!DOCTYPE html>
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
    <!--
      GraphiQL is served from unpkg CDN for simplicity. You can also
      bundle it locally if you prefer to avoid external dependencies.
      See: https://github.com/graphql/graphiql
    -->
    <link rel="stylesheet" href="https://unpkg.com/graphiql@1.4.0/graphiql.min.css" />
    <script src="https://unpkg.com/react@16.14.0/umd/react.production.min.js"></script>
    <script src="https://unpkg.com/react-dom@16.14.0/umd/react-dom.production.min.js"></script>
    <script src="https://unpkg.com/graphiql@1.4.0/graphiql.min.js"></script>
</head>
<body>
    <div id="graphiql">Loading...</div>
    <script>
      // The GraphQL fetcher posts to the graphqlEndpoint.
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

// PlaygroundHandler returns an HTTP handler that serves an interactive
// GraphiQL playground. This allows browsing the schema, writing and
// executing queries/mutations directly in the browser when the server
// is running.
//
// The graphqlEndpoint is typically "/graphql" (the path where
// HTTPHandler is mounted).
//
// Typical usage in main():
//   http.Handle("/graphql", jaal.HTTPHandler(schema))
//   // Serve playground for all other paths (e.g. root "/")
//   http.Handle("/", jaal.PlaygroundHandler("Jaal Playground", "/graphql"))
//
// Note: This uses external CDN resources; in production consider
// hosting the assets locally for offline use or to reduce latency.
func PlaygroundHandler(title, graphqlEndpoint string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Support GET (and HEAD for completeness) to serve the playground UI.
		// This allows tools like curl -I to work and follows common HTTP practices.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == http.MethodHead {
			// For HEAD, don't write body.
			return
		}
		// Note: fmt.Fprintf is safe here as the template is static and
		// endpoints are controlled by the caller.
		_, _ = fmt.Fprintf(w, playgroundHTML, title, graphqlEndpoint)
	})
}
