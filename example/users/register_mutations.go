package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.appointy.com/jaal/schemabuilder"
)

// RegisterCreateUserMutation registers createUser (standard input mutation).
// Specific func per task.
func RegisterCreateUserMutation(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation() // Note: shared Mutation obj; funcs append.

	// createUser: takes CreateUserInput, creates/appends User (UUID ID, time).
	// Resolver pattern from README/main.go.
	m.FieldFunc("createUser", func(ctx context.Context, args struct {
		Input CreateUserInput
	}) *User {
		newUser := &User{
			ID:              schemabuilder.ID{Value: uuid.New().String()},
			Name:            args.Input.Name,
			Email:           args.Input.Email,
			Age:             args.Input.Age,
			ReputationScore: args.Input.ReputationScore,
			IsActive:        args.Input.IsActive,
			Role:            args.Input.Role,
			CreatedAt:       time.Now(),
		}
		s.users = append(s.users, newUser)
		return newUser
	})
}

// RegisterContactByMutation registers contactBy mutation (uses ContactByInput
// w/ @oneOf for exclusive email/phone; spec input union demo). Specific func.
func RegisterContactByMutation(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation()

	// contactBy: *ContactByInput arg (oneOf validated in parser); resolver finds
	// by field (simplified match; error otherwise). Mirrors createUser + oneOf
	// example from prior.
	m.FieldFunc("contactBy", func(ctx context.Context, args struct {
		Input *ContactByInput
	}) (*User, error) {
		if args.Input == nil {
			return nil, errors.New("input required")
		}
		// Handle exclusive field (email OR phone; oneOf ensures).
		var matchEmail, matchPhone string
		if args.Input.Email != nil {
			matchEmail = *args.Input.Email
		}
		if args.Input.Phone != nil {
			matchPhone = *args.Input.Phone
		}
		for _, u := range s.users {
			// Match email (phone demo stub; extend User if needed).
			if (matchEmail != "" && u.Email == matchEmail) || (matchPhone != "" && u.Email == matchPhone) {
				return u, nil
			}
		}
		return nil, fmt.Errorf("user not found by email=%s or phone=%s", matchEmail, matchPhone)
	})
}

// RegisterMutation aggregator calls specific mutation reg funcs (per task).
// Shared m := sb.Mutation(); appends fields; enables testing individual.
func RegisterMutation(sb *schemabuilder.Schema, s *Server) {
	RegisterCreateUserMutation(sb, s)
	RegisterContactByMutation(sb, s)
}