package users

import (
	"time"

	"go.appointy.com/jaal/schemabuilder"
)

// =============================================================================
// Domain Models + Server (extracted for modularity/testing per refactor task)
// =============================================================================

// User demonstrates standard scalars + ID + Time (output payload).
// Tags for field names; CreatedAt uses DateTime scalar.
type User struct {
	ID              schemabuilder.ID `graphql:"id"`
	Name            string           `graphql:"name"`
	Email           string           `graphql:"email"`
	Age             int32            `graphql:"age"`
	ReputationScore float64          `graphql:"reputation"`
	IsActive        bool             `graphql:"isActive"`
	Role            Role             `graphql:"role"`
	CreatedAt       time.Time        `graphql:"createdAt"`
}

// Role enum type (string underlying).
type Role string

// Enum consts for Role (ADMIN etc).
const (
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
	RoleGuest  Role = "GUEST"
)

// CreateUserInput for createUser mutation (w/ deprecation tag).
// graphql/json tags for INPUT_FIELD_DEFINITION spec.
type CreateUserInput struct {
	Name            string
	Email           string
	// Age deprecated example.
	Age             int32 `json:"age" graphql:",deprecated=Use birthdate instead"`
	ReputationScore float64
	IsActive        bool
	Role            Role
}

// ContactByInput for contactBy mutation (oneOf input union demo; embed
// OneOfInput marker for @oneOf spec/exclusive fields).
type ContactByInput struct {
	// schemabuilder.OneOfInput for INPUT_OBJECT @oneOf (exclusive email/phone).
	schemabuilder.OneOfInput
	// Ptr fields optional; exactly one non-null enforced.
	Email *string
	Phone *string
}

// Server mock store for users (in-memory; used by resolvers).
type Server struct {
	users []*User
}

// NewServer creates Server w/ seed data (pattern from original main.go).
func NewServer() *Server {
	return &Server{
		users: []*User{
			{
				ID:              schemabuilder.ID{Value: "u1"},
				Name:            "John Doe",
				Email:           "jdoe@example.com",
				Age:             30,
				ReputationScore: 9.5,
				IsActive:        true,
				Role:            RoleAdmin,
				CreatedAt:       time.Now(),
			},
		},
	}
}