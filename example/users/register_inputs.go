package users

import "go.appointy.com/jaal/schemabuilder"

// RegisterCreateUserInput registers the CreateUserInput (w/ deprecation tag example
// for INPUT_FIELD_DEFINITION/ARGUMENT_DEFINITION spec support). Specific reg func
// per task; FieldFunc setup mirrors schemabuilder/input_object.go + main.go original.
// Description via variadic (descriptions feature; to graphql.InputObject/__Type.description).
func RegisterCreateUserInput(sb *schemabuilder.Schema) {
	input := sb.InputObject("CreateUserInput", CreateUserInput{}, "Input for creating a new user (supports name, email, role etc; age field deprecated for legacy).")

	// FieldFuncs to populate target struct from input (name/email etc).
	// Deprecation on Age via tag (reflect.go parse).
	input.FieldFunc("name", func(target *CreateUserInput, source string) { target.Name = source })
	input.FieldFunc("email", func(target *CreateUserInput, source string) { target.Email = source })
	input.FieldFunc("age", func(target *CreateUserInput, source int32) { target.Age = source })
	input.FieldFunc("reputation", func(target *CreateUserInput, source float64) { target.ReputationScore = source })
	input.FieldFunc("isActive", func(target *CreateUserInput, source bool) { target.IsActive = source })
	input.FieldFunc("role", func(target *CreateUserInput, source Role) { target.Role = source })
}

// RegisterContactByInput registers the oneOf input (ContactByInput w/ OneOfInput
// embed for @oneOf spec input union; exactly one field). Specific func per task;
// mirrors RegisterCreateUserInput + README/oneOf demo in mutation.
// Uses setter for desc (alt to variadic; per plan).
func RegisterContactByInput(sb *schemabuilder.Schema) {
	// oneOf input reg (INPUT_OBJECT w/ marker; sets OneOf=true in graphql.InputObject
	// via input_object.go detect; FieldFunc for email/phone).
	// Description via variadic param (descriptions feature; spec for INPUT_OBJECT).
	oneOfInput := sb.InputObject("ContactByInput", ContactByInput{}, "OneOf input for contacting user by exactly one of email or phone (spec input union; exclusive fields).")
	oneOfInput.FieldFunc("email", func(target *ContactByInput, source *string) { target.Email = source })
	oneOfInput.FieldFunc("phone", func(target *ContactByInput, source *string) { target.Phone = source })
}

// RegisterInputs aggregator calls specific input reg funcs (per task for modularity).
// Allows testing individual (e.g., RegisterContactByInput alone) + full like original.
func RegisterInputs(sb *schemabuilder.Schema) {
	RegisterCreateUserInput(sb)
	RegisterContactByInput(sb)
}