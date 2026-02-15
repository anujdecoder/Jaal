package main

import (
	"log"
	"net/http"

	"go.appointy.com/jaal/example/users"
)

// =============================================================================
// Refactor Complete: Domain models, Server, and registrations (incl. scalars,
// enums, objects, inputs w/ CreateUserInput + ContactByInput@oneOf, queries,
// mutations w/ contactBy, subs) moved to example/users/ (types.go, register_*.go,
// register_schema.go, server.go) for readability/testing.
// Main simplified: uses users.GetGraphqlServer(); no breaking changes.
// =============================================================================

// (Data store refactored to users/types.go)

// =============================================================================
// 3. Schema Registration (Modular)
// =============================================================================

// init() for scalars refactored to users/register_scalars.go (RegisterScalars).

// RegisterSchema orchestrates the registration of all schema components.
// All Register* funcs (Schema/Enums/Objects/Inputs/Query/Mutation) refactored
// to example/users/ subdir for modularity. See register_schema.go aggregator
// and per-type files. (OneOf ContactByInput/contactBy mutation preserved.)

// RegisterSubscription refactored to example/users/register_subscriptions.go

// =============================================================================
// 4. Main Execution
// =============================================================================

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
