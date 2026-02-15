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

	// Build schema (reflect-based; err on dup/invalid types).
	schema, err := sb.Build()
	if err != nil {
		return nil, err
	}

	// Add introspection (__schema/__type; enables Playground per jaal.HTTPHandler).
	introspection.AddIntrospectionToSchema(schema)

	// Return handler (serves POST queries + GET UI; no CDN).
	return jaal.HTTPHandler(schema), nil
}