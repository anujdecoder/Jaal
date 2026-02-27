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
	user := sb.Object("User", User{}, schemabuilder.WithDescription("User payload representing a person in the system."))

	// FieldFunc w/ desc for FIELD_DEFINITION (extension; options to graphql.Field
	// /__Field.description/Playground; e.g., for id/name).
	// Mirrors query/mut field descs.
	user.FieldFunc("id", func(u *User) schemabuilder.ID { return u.ID }, schemabuilder.FieldDesc("Unique identifier for the user."))
	user.FieldFunc("name", func(u *User) string { return u.Name }, schemabuilder.FieldDesc("Full name of the user."))
	user.FieldFunc("email", func(u *User) string { return u.Email }, schemabuilder.FieldDesc("Email address."))
	user.FieldFunc("age", func(u *User) int32 { return u.Age }, schemabuilder.FieldDesc("Age in years."))
	user.FieldFunc("reputation", func(u *User) float64 { return u.ReputationScore }, schemabuilder.FieldDesc("Reputation score (0-10)."))
	user.FieldFunc("isActive", func(u *User) bool { return u.IsActive }, schemabuilder.FieldDesc("Whether the user is active."))
	user.FieldFunc("role", func(u *User) Role { return u.Role }, schemabuilder.FieldDesc("User role (ADMIN/MEMBER/GUEST)."))
	user.FieldFunc("createdAt", func(u *User) time.Time { return u.CreatedAt }, schemabuilder.FieldDesc("Account creation timestamp."))
}
