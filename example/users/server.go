package users

import (
	"net/http"

	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// GetGraphqlServer builds and returns jaal.HTTPHandler (w/ schema + introspection)
// for /graphql route. Per task: create sb, build, register; returns handler + err
// for caller (main.go). Encapsulates original main build logic; enables testing
// handler isolation (e.g., httptest).
func GetGraphqlServer() (http.Handler, error) {
	// New schema + server (from types.go).
	sb := schemabuilder.NewSchema()
	server := NewServer()

	// Register all via aggregator (scalars to subs; see register_schema.go).
	RegisterSchema(sb, server)

	// Get directive definitions and visitors before building
	directiveDefs := sb.GetDirectiveDefinitions()
	directiveVisitors := sb.GetDirectiveVisitors()

	// Build schema (reflect-based; err on dup/invalid types).
	schema, err := sb.Build()
	if err != nil {
		return nil, err
	}

	// Add introspection with custom directive definitions.
	introspection.AddIntrospectionToSchemaWithDirectives(schema, directiveDefs)

	// Return handler with directive visitors (serves POST queries + GET UI; no CDN).
	return jaal.HTTPHandler(schema, jaal.WithDirectiveVisitors(directiveVisitors)), nil
}
