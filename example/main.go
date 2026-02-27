package main

import (
	"log"
	"net/http"

	"go.appointy.com/jaal/example/users"
)

// GetGraphqlServer encapsulates schema build (from users/server.go).
// main simplified: get handler, register route, serve (per refactor task).
func main() {
	// GetGraphqlServer: NewSchema, RegisterSchema(all incl. oneOf ContactByInput),
	// Build, AddIntrospection, jaal.HTTPHandler. Err on fail.
	h, err := users.GetGraphqlServer()
	if err != nil {
		log.Fatalf("Failed to get GraphQL server: %v", err)
	}

	// /graphql for queries/muts/subs + Playground UI.
	http.Handle("/graphql", h)

	log.Println("Server running on :8080")
	log.Println("Playground: http://localhost:8080/graphql")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
