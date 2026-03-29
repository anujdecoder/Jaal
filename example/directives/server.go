package directives

import (
	"net/http"

	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// GetGraphqlServer builds and returns an http.Handler that serves the GraphQL
// API with custom directives enabled.  Open http://localhost:<port>/graphql in
// a browser to use the Playground UI.
func GetGraphqlServer() (http.Handler, error) {
	sb := schemabuilder.NewSchema()
	server := NewServer()

	// Register everything: directives, objects, queries, mutations.
	RegisterSchema(sb, server)

	// Build the executable schema.
	schema, err := sb.Build()
	if err != nil {
		return nil, err
	}

	// Add introspection (__schema, __type).
	introspection.AddIntrospectionToSchema(schema)

	return jaal.HTTPHandler(schema), nil
}
