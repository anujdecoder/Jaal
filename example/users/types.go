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

// CreateUserInput for createUser mutation.
// Deprecation is now options-based via InputObject.FieldFunc.
type CreateUserInput struct {
	Name            string
	Email           string
	Age             int32
	ReputationScore float64
	IsActive        bool
	Role            Role
}

// ContactByInput for contactBy mutation (oneOf input union demo).
// Exactly one of Email or Phone must be provided (enforced by @oneOf directive).
// Note: This type is deprecated; replaced by CreateUserByContactInput below.
type ContactByInput struct {
	// Ptr fields optional; exactly one non-null enforced by @oneOf.
	Email *string
	Phone *string
}

// IdentifierInput oneOf for id or email (exclusive; used in createUserByContact).
// Exactly one of ID or Email must be provided (enforced by @oneOf directive).
type IdentifierInput struct {
	// Ptr optional; exactly one enforced.
	ID    *schemabuilder.ID
	Email *string
}

// UserInput for user fields in createUserByContact (copy of CreateUserInput fields
// w/o deprecation for simplicity; populate Name etc).
type UserInput struct {
	Name            string
	Email           string
	Age             int32
	ReputationScore float64
	IsActive        bool
	Role            Role
}

// CreateUserByContactInput for improved createUserByContact mutation: two fields -
// identifier (oneOf id/email), userInput (fields). Per task.
type CreateUserByContactInput struct {
	Identifier IdentifierInput
	UserInput  UserInput
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
