// Package main provides a comprehensive GraphQL server example matching the provided schema,
// including all features, directives (@specifiedBy, @oneOf), and a mutation with oneOf input. Uses separate Register* funcs for readability.
// Run with `go run server.go` to start server + playground at http://localhost:8080/graphql.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// Server holds example data.
type Server struct {
	Users []*User
}

// UUID scalar with @specifiedBy.
type UUID string

// Role enum.
type Role string

const (
	Admin  Role = "ADMIN"
	Member Role = "MEMBER"
	GUEST  Role = "GUEST"
)

// Node interface.
type Node interface {
	ID() schemabuilder.ID
}

// User implements Node.
type User struct {
	IDVal     schemabuilder.ID
	UUIDVal   UUID
	Username  string
	Email     string
	RoleVal   Role
	StatusVal UserStatus
}

func (u *User) ID() schemabuilder.ID { return u.IDVal }

// DeletedUser implements Node.
type DeletedUser struct {
	IDVal     schemabuilder.ID
	DeletedAt string
}

func (d *DeletedUser) ID() schemabuilder.ID { return d.IDVal }

// UserResult union.
type UserResult struct {
	schemabuilder.Union
	*User
	*DeletedUser
}

// UserStatus.
type UserStatus struct {
	IsActive  bool
	LastLogin string
}

// UserIdentifierInput with @oneOf.
type UserIdentifierInput struct {
	ID    *schemabuilder.ID `json:"id,omitempty"`
	Email *string           `json:"email,omitempty"`
}

// CreateUserInput.
type CreateUserInput struct {
	Username string
	Email    string
	Role     Role
}

// RegisterSchema orchestrates all (separate funcs per request for readability).
func RegisterSchema(sb *schemabuilder.Schema, s *Server) {
	RegisterScalars(sb)
	RegisterEnums(sb)
	RegisterInterfaces(sb)
	RegisterObjects(sb)
	RegisterInputs(sb)
	RegisterQueries(sb, s)
	RegisterMutations(sb, s)
}

// RegisterScalars registers UUID with @specifiedBy.
func RegisterScalars(sb *schemabuilder.Schema) {
	typ := reflect.TypeOf(UUID(""))
	_ = schemabuilder.RegisterScalar(typ, "UUID", func(value interface{}, dest reflect.Value) error {
		if s, ok := value.(string); ok {
			dest.Set(reflect.ValueOf(UUID(s)))
			return nil
		}
		return errors.New("invalid UUID")
	}, "https://tools.ietf.org/html/rfc4122")
}

// RegisterEnums registers Role.
func RegisterEnums(sb *schemabuilder.Schema) {
	sb.Enum(Admin, map[string]interface{}{
		"ADMIN":  Admin,
		"MEMBER": Member,
		"GUEST":  GUEST,
	})
}

// RegisterInterfaces registers Node interface.
func RegisterInterfaces(sb *schemabuilder.Schema) {
	// Jaal uses struct marker for interface (per README).
	sb.Object("Node", struct{ schemabuilder.Interface }{})
}

// RegisterObjects registers User, DeletedUser, UserStatus, etc (full fields).
func RegisterObjects(sb *schemabuilder.Schema) {
	// User.
	user := sb.Object("User", User{})
	user.FieldFunc("id", func(ctx context.Context, in *User) schemabuilder.ID { return schemabuilder.ID{Value: in.IDVal.Value} })
	user.FieldFunc("uuid", func(ctx context.Context, in *User) UUID { return in.UUIDVal })
	user.FieldFunc("username", func(ctx context.Context, in *User) string { return in.Username })
	user.FieldFunc("email", func(ctx context.Context, in *User) string { return in.Email })
	user.FieldFunc("role", func(ctx context.Context, in *User) Role { return in.RoleVal })
	user.FieldFunc("status", func(ctx context.Context, in *User) UserStatus { return in.StatusVal })

	// DeletedUser.
	deleted := sb.Object("DeletedUser", DeletedUser{})
	deleted.FieldFunc("id", func(ctx context.Context, in *DeletedUser) schemabuilder.ID {
		return schemabuilder.ID{Value: in.IDVal.Value}
	})
	deleted.FieldFunc("deletedAt", func(ctx context.Context, in *DeletedUser) string { return in.DeletedAt })

	// UserStatus.
	sb.Object("UserStatus", UserStatus{})

	// UserResult union.
	sb.Object("UserResult", UserResult{})
}

// RegisterInputs registers UserIdentifierInput with @oneOf, CreateUserInput.
func RegisterInputs(sb *schemabuilder.Schema) {
	// UserIdentifierInput with @oneOf.
	input := sb.InputObject("UserIdentifierInput", UserIdentifierInput{})
	input.OneOf()
	input.FieldFunc("id", func(target *UserIdentifierInput, source *schemabuilder.ID) {
		target.ID = source
	})
	input.FieldFunc("email", func(target *UserIdentifierInput, source *string) {
		target.Email = source
	})

	// CreateUserInput.
	createInput := sb.InputObject("CreateUserInput", CreateUserInput{})
	createInput.FieldFunc("username", func(target *CreateUserInput, source string) {
		target.Username = source
	})
	createInput.FieldFunc("email", func(target *CreateUserInput, source string) {
		target.Email = source
	})
	createInput.FieldFunc("role", func(target *CreateUserInput, source Role) {
		target.Role = source
	})
}

// RegisterQueries registers Query type (full from schema).
func RegisterQueries(sb *schemabuilder.Schema, s *Server) {
	q := sb.Query()
	q.FieldFunc("me", func(ctx context.Context) *User {
		if len(s.Users) > 0 {
			return s.Users[0]
		}
		return &User{IDVal: schemabuilder.ID{Value: "me"}, UUIDVal: UUID(uuid.New().String()), Username: "test", RoleVal: Member}
	})
	q.FieldFunc("user", func(ctx context.Context, args struct {
		By *UserIdentifierInput
	}) interface{} { // UserResult
		// Stub: return User.
		return &User{IDVal: schemabuilder.ID{Value: "u1"}, UUIDVal: UUID(uuid.New().String()), Username: "test", RoleVal: Member}
	})
	q.FieldFunc("allUsers", func(ctx context.Context) []*User {
		return s.Users
	})
}

// RegisterMutations registers Mutation type.
func RegisterMutations(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation()
	m.FieldFunc("createUser", func(ctx context.Context, args struct {
		Input *CreateUserInput
	}) *User {
		u := &User{
			IDVal:    schemabuilder.ID{Value: uuid.New().String()},
			UUIDVal:  UUID(uuid.New().String()),
			Username: args.Input.Username,
			Email:    args.Input.Email,
			RoleVal:  args.Input.Role,
		}
		s.Users = append(s.Users, u)
		return u
	})
	m.FieldFunc("updateUserRole", func(ctx context.Context, args struct {
		ID      schemabuilder.ID
		NewRole Role
	}) *User {
		// Stub.
		return &User{IDVal: schemabuilder.ID{Value: args.ID.Value}, RoleVal: args.NewRole}
	})
}

// HTTPHandler returns the Jaal handler.
func HTTPHandler() http.Handler {
	sb := schemabuilder.NewSchema()
	s := &Server{
		Users: []*User{{IDVal: schemabuilder.ID{Value: "u1"}, UUIDVal: UUID(uuid.New().String()), Username: "test", RoleVal: Member}},
	}
	RegisterSchema(sb, s)
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	introspection.AddIntrospectionToSchema(schema)
	return jaal.HTTPHandler(schema)
}

func main() {
	http.Handle("/graphql", HTTPHandler())
	log.Println("Server running on :8080")
	log.Println("GraphQL Playground: http://localhost:8080/graphql")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
