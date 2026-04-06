package main

import (
	"log"
	"net/http"

	"go.appointy.com/jaal/example/users"
)

// Basic example demonstrating a full-featured GraphQL server with users API.
// This example showcases:
//   - Object registration with descriptions
//   - Input objects with @oneOf directive
//   - Enum registration with value deprecation
//   - Query, Mutation, and field deprecation
//   - GraphQL Playground integration
func main() {
	// GetGraphqlServer: NewSchema, RegisterSchema(all incl. oneOf ContactByInput),
	// Build, AddIntrospection, jaal.HTTPHandler. Err on fail.
	h, err := users.GetGraphqlServer()
	if err != nil {
		log.Fatalf("Failed to get GraphQL server: %v", err)
	}

	// /graphql for queries/muts/subs + Playground UI.
	http.Handle("/graphql", h)

	log.Println("Server running on :9000")
	log.Println("Playground: http://localhost:9000/graphql")

	if err := http.ListenAndServe(":9000", nil); err != nil {
		log.Fatal(err)
	}
}
