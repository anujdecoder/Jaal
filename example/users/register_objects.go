package users

import (
	"time"

	"go.appointy.com/jaal/schemabuilder"
)

// RegisterObjects registers output objects (e.g., User payload) + FieldFuncs for
// field mapping (ID scalar etc). Specific: RegisterUserObject; aggregator
// RegisterObjects calls them. Pattern from README.md object reg + original
// RegisterObjects/main.go for fields like id/name.
func RegisterObjects(sb *schemabuilder.Schema) {
	// User object registration (output payload w/ description for spec/Playground).
	// FieldFuncs map struct fields (e.g., graphql ID); see User type in users/types.go.
	// Description per feature (Object variadic; in __Type.description).
	user := sb.Object("User", User{}, "User payload representing a person in the system.")

	user.FieldFunc("id", func(u *User) schemabuilder.ID { return u.ID })
	user.FieldFunc("name", func(u *User) string { return u.Name })
	user.FieldFunc("email", func(u *User) string { return u.Email })
	user.FieldFunc("age", func(u *User) int32 { return u.Age })
	user.FieldFunc("reputation", func(u *User) float64 { return u.ReputationScore })
	user.FieldFunc("isActive", func(u *User) bool { return u.IsActive })
	user.FieldFunc("role", func(u *User) Role { return u.Role })
	user.FieldFunc("createdAt", func(u *User) time.Time { return u.CreatedAt })
}