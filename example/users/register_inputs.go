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

// RegisterIdentifierInput registers the oneOf input for identifier (id or email;
// embed OneOfInput for @oneOf spec input union; exactly one field). Specific func
// per task; mirrors RegisterCreateUserInput + README/oneOf.
func RegisterIdentifierInput(sb *schemabuilder.Schema) {
	// INPUT_OBJECT w/ marker; OneOf=true via detect; FieldFunc for id/email.
	// Description for Playground/__Type.
	identifierInput := sb.InputObject("IdentifierInput", IdentifierInput{}, "OneOf identifier: exactly one of ID or email (spec input union).")
	identifierInput.FieldFunc("id", func(target *IdentifierInput, source *schemabuilder.ID) { target.ID = source })
	identifierInput.FieldFunc("email", func(target *IdentifierInput, source *string) { target.Email = source })
}

// RegisterUserInput registers input for user fields (copy fields; no deprecation).
// Specific func.
func RegisterUserInput(sb *schemabuilder.Schema) {
	// FieldFuncs populate; desc for INPUT_OBJECT.
	userInput := sb.InputObject("UserInput", UserInput{}, "User fields for creation (name, email etc).")
	userInput.FieldFunc("name", func(target *UserInput, source string) { target.Name = source })
	userInput.FieldFunc("email", func(target *UserInput, source string) { target.Email = source })
	userInput.FieldFunc("age", func(target *UserInput, source int32) { target.Age = source })
	userInput.FieldFunc("reputation", func(target *UserInput, source float64) { target.ReputationScore = source })
	userInput.FieldFunc("isActive", func(target *UserInput, source bool) { target.IsActive = source })
	userInput.FieldFunc("role", func(target *UserInput, source Role) { target.Role = source })
}

// RegisterCreateUserByContactInput registers composite input for createUserByContact
// (identifier oneOf + userInput; per task improvement).
func RegisterCreateUserByContactInput(sb *schemabuilder.Schema) {
	// Composite input; desc.
	contactInput := sb.InputObject("CreateUserByContactInput", CreateUserByContactInput{}, "Create user by identifier (oneOf id/email) and user fields.")
	// FieldFunc for sub-objects (target populate; mirrors input FieldFunc).
	// Note: sub-inputs registered separately; parser handles nested.
	contactInput.FieldFunc("identifier", func(target *CreateUserByContactInput, source IdentifierInput) { target.Identifier = source })
	contactInput.FieldFunc("userInput", func(target *CreateUserByContactInput, source UserInput) { target.UserInput = source })
}

// RegisterInputs aggregator calls specific input reg funcs (per task for modularity;
// includes new for createUserByContact).
// Allows testing individual; full like original.
func RegisterInputs(sb *schemabuilder.Schema) {
	RegisterCreateUserInput(sb)
	RegisterIdentifierInput(sb)
	RegisterUserInput(sb)
	RegisterCreateUserByContactInput(sb)
}