// Package directives demonstrates Jaal's custom directive registration feature.
// It shows how to register, configure, and apply custom directives that enforce
// access control (roles, rights), feature flags, auditing, rate limiting, and
// metadata-only cache hints — covering all combinations of ExecutionOrder
// (PreResolver/PostResolver) and OnFailBehavior (ErrorOnFail/SkipOnFail).
package directives

import (
	"time"

	"go.appointy.com/jaal/schemabuilder"
)

// =============================================================================
// Domain Models
// =============================================================================

// Article is a simple domain type returned by queries.
type Article struct {
	ID        schemabuilder.ID `graphql:"id"`
	Title     string           `graphql:"title"`
	Body      string           `graphql:"body"`
	Author    string           `graphql:"author"`
	CreatedAt time.Time        `graphql:"createdAt"`
}

// AdminStats is a protected type; every field is guarded by a type-level
// @auth directive that requires an authenticated context.
type AdminStats struct{}

// AuditLog records an audited event.
type AuditLog struct {
	Action string `graphql:"action"`
	At     string `graphql:"at"`
}

// AuthorProfile is returned by a batch-resolved field on Article, demonstrating
// the DataLoader / automatic batching feature.
type AuthorProfile struct {
	Name string
	Bio  string
}

// Server holds in-memory data for the example.
type Server struct {
	articles  []*Article
	auditLogs []AuditLog
}

// NewServer creates a Server with seed data.
func NewServer() *Server {
	return &Server{
		articles: []*Article{
			{
				ID:        schemabuilder.ID{Value: "a1"},
				Title:     "Getting Started with Jaal",
				Body:      "Jaal is a Go framework for building spec-compliant GraphQL servers...",
				Author:    "Alice",
				CreatedAt: time.Now(),
			},
			{
				ID:        schemabuilder.ID{Value: "a2"},
				Title:     "Custom Directives in GraphQL",
				Body:      "Directives allow you to attach metadata and runtime logic to your schema...",
				Author:    "Bob",
				CreatedAt: time.Now(),
			},
		},
	}
}
