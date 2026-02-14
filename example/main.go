package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// =============================================================================
// 1. Domain Models
// =============================================================================

// User demonstrates the use of standard GraphQL scalars + ID + Time.
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

type Role string

const (
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
	RoleGuest  Role = "GUEST"
)

// Inputs
// Deprecation on input values demo (spec: INPUT_FIELD_DEFINITION via graphql/json tag).
// e.g., Age field deprecated; reflected in introspection __InputValue.isDeprecated/deprecationReason.
// Parse support in reflect.go/input_object.go.
type CreateUserInput struct {
	Name            string
	Email           string
	// Age deprecated (example for ARGUMENT_DEFINITION too in FieldFunc args).
	Age             int32 `json:"age" graphql:",deprecated=Use birthdate instead"`
	ReputationScore float64
	IsActive        bool
	Role            Role
}

// =============================================================================
// 2. Data Store (Mock)
// =============================================================================

type Server struct {
	users []*User
}

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

// =============================================================================
// 3. Schema Registration (Modular)
// =============================================================================

func init() {
	typ := reflect.TypeOf(time.Time{})
	// Register DateTime scalar with @specifiedBy URL for full spec compliance
	// (Oct 2021+; exposed in introspection as __Type.specifiedByURL).
	// URL links external spec (RFC3339); backward compat for other scalars.
	schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
		v, ok := value.(string)
		if !ok {
			return errors.New("invalid type expected string")
		}

		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}

		dest.Set(reflect.ValueOf(t))

		return nil
	}, "https://tools.ietf.org/html/rfc3339")
}

// RegisterSchema orchestrates the registration of all schema components.
func RegisterSchema(sb *schemabuilder.Schema, s *Server) {
	RegisterEnums(sb)
	RegisterObjects(sb)
	RegisterInputs(sb)

	RegisterQuery(sb, s)
	RegisterMutation(sb, s)
	RegisterSubscription(sb)
}

func RegisterEnums(sb *schemabuilder.Schema) {
	sb.Enum(RoleMember, map[string]interface{}{
		"ADMIN":  RoleAdmin,
		"MEMBER": RoleMember,
		"GUEST":  RoleGuest,
	})
}

func RegisterObjects(sb *schemabuilder.Schema) {
	user := sb.Object("User", User{})

	// FieldFuncs allow you to map struct fields to GraphQL fields explicitly
	user.FieldFunc("id", func(u *User) schemabuilder.ID { return u.ID })
	user.FieldFunc("name", func(u *User) string { return u.Name })
	user.FieldFunc("email", func(u *User) string { return u.Email })
	user.FieldFunc("age", func(u *User) int32 { return u.Age })
	user.FieldFunc("reputation", func(u *User) float64 { return u.ReputationScore })
	user.FieldFunc("isActive", func(u *User) bool { return u.IsActive })
	user.FieldFunc("role", func(u *User) Role { return u.Role })
	user.FieldFunc("createdAt", func(u *User) time.Time { return u.CreatedAt })
}

func RegisterInputs(sb *schemabuilder.Schema) {
	input := sb.InputObject("CreateUserInput", CreateUserInput{})

	input.FieldFunc("name", func(target *CreateUserInput, source string) { target.Name = source })
	input.FieldFunc("email", func(target *CreateUserInput, source string) { target.Email = source })
	input.FieldFunc("age", func(target *CreateUserInput, source int32) { target.Age = source })
	input.FieldFunc("reputation", func(target *CreateUserInput, source float64) { target.ReputationScore = source })
	input.FieldFunc("isActive", func(target *CreateUserInput, source bool) { target.IsActive = source })
	input.FieldFunc("role", func(target *CreateUserInput, source Role) { target.Role = source })
}

func RegisterQuery(sb *schemabuilder.Schema, s *Server) {
	q := sb.Query()

	q.FieldFunc("me", func(ctx context.Context) *User {
		if len(s.users) > 0 {
			return s.users[0]
		}
		return nil
	})

	q.FieldFunc("user", func(ctx context.Context, args struct {
		ID schemabuilder.ID
	}) (*User, error) {
		for _, u := range s.users {
			if u.ID.Value == args.ID.Value {
				return u, nil
			}
		}
		return nil, fmt.Errorf("user not found")
	})

	q.FieldFunc("allUsers", func(ctx context.Context) []*User {
		return s.users
	})
}

func RegisterMutation(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation()

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

// RegisterSubscription adds a functional subscription.
func RegisterSubscription(sb *schemabuilder.Schema) {
	s := sb.Subscription()

	// The resolver must return a function that returns the channel.
	s.FieldFunc("currentTime", func(ctx context.Context) func() time.Time {
		return time.Now
	})
}

// =============================================================================
// 4. Main Execution
// =============================================================================

func main() {
	sb := schemabuilder.NewSchema()
	server := NewServer()

	RegisterSchema(sb, server)

	schema, err := sb.Build()
	if err != nil {
		log.Fatalf("Failed to build schema: %v", err)
	}

	introspection.AddIntrospectionToSchema(schema)

	// Jaal HTTPHandler handles Queries and Mutations.
	// Note: For Subscriptions to work in a real browser env, you usually need
	// a WebSocket handler wrapper, but this is the standard Jaal HTTP entry point.
	http.Handle("/graphql", jaal.HTTPHandler(schema))

	log.Println("Server running on :8080")
	log.Println("Playground: http://localhost:8080/graphql")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
