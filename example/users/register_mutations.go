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
	// Resolver pattern from README/main.go; field desc for FIELD_DEFINITION.
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
	}, schemabuilder.FieldDesc("Creates a new user from input data."))
}

// RegisterCreateUserByContactMutation registers improved createUserByContact mutation
// (uses CreateUserByContactInput: identifier oneOf + userInput; per task).
// Specific func; replaces old contactBy. Mirrors createUser + oneOf/identifier.
func RegisterCreateUserByContactMutation(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation()

	// createUserByContact: input w/ identifier (oneOf id/email validated) + userInput.
	// Verify no existing by id/email (error if); create from UserInput, assign ID if
	// email-only; append. Field desc for FIELD_DEFINITION.
	// Input object composite; parser handles nested oneOf/UserInput.
	m.FieldFunc("createUserByContact", func(ctx context.Context, args struct {
		Input CreateUserByContactInput
	}) (*User, error) {
		if args.Input.Identifier.ID == nil && args.Input.Identifier.Email == nil {
			return nil, errors.New("identifier required (id or email)")
		}
		// Check no existing user (by id or email).
		idStr := ""
		emailStr := ""
		if args.Input.Identifier.ID != nil {
			idStr = args.Input.Identifier.ID.Value
		}
		if args.Input.Identifier.Email != nil {
			emailStr = *args.Input.Identifier.Email
		}
		for _, u := range s.users {
			if (idStr != "" && u.ID.Value == idStr) || (emailStr != "" && u.Email == emailStr) {
				return nil, fmt.Errorf("user already exists by id=%s or email=%s", idStr, emailStr)
			}
		}

		// Create from userInput; assign ID if email-only (task).
		newUser := &User{
			Name:            args.Input.UserInput.Name,
			Email:           args.Input.UserInput.Email,
			Age:             args.Input.UserInput.Age,
			ReputationScore: args.Input.UserInput.ReputationScore,
			IsActive:        args.Input.UserInput.IsActive,
			Role:            args.Input.UserInput.Role,
			CreatedAt:       time.Now(),
		}
		if idStr != "" {
			newUser.ID = schemabuilder.ID{Value: idStr}
		} else {
			// Email-only: add ID.
			newUser.ID = schemabuilder.ID{Value: uuid.New().String()}
		}
		s.users = append(s.users, newUser)
		return newUser, nil
	}, schemabuilder.FieldDesc("Creates user by oneOf identifier (id/email) and user fields; errors if exists."))
}

// RegisterMutation aggregator calls specific mutation reg funcs (per task;
// createUserByContact replaces old contactBy per improvement).
// Shared m := sb.Mutation(); appends fields; enables testing individual.
func RegisterMutation(sb *schemabuilder.Schema, s *Server) {
	RegisterCreateUserMutation(sb, s)
	RegisterCreateUserByContactMutation(sb, s)
}
