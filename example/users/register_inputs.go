package users

import "go.appointy.com/jaal/schemabuilder"

// RegisterCreateUserInput registers the CreateUserInput (w/ deprecation option example
// for INPUT_FIELD_DEFINITION/ARGUMENT_DEFINITION spec support). Specific reg func
// per task; FieldFunc setup mirrors schemabuilder/input_object.go + main.go original.
// Description via options (descriptions feature; to graphql.InputObject/__Type.description).
func RegisterCreateUserInput(sb *schemabuilder.Schema) {
	input := sb.InputObject("CreateUserInput", CreateUserInput{}, schemabuilder.WithDescription("Input for creating a new user (supports name, email, role etc; age field deprecated for legacy)."))

	// FieldFuncs to populate target struct from input (name/email etc).
	// Deprecation on Age via options (schemabuilder.Deprecated).
	input.FieldFunc("name", func(target *CreateUserInput, source string) { target.Name = source }, schemabuilder.FieldDesc("Name of the user."))
	input.FieldFunc("email", func(target *CreateUserInput, source string) { target.Email = source }, schemabuilder.FieldDesc("Email address."))
	input.FieldFunc("age", func(target *CreateUserInput, source int32) { target.Age = source }, schemabuilder.FieldDesc("Age in years (deprecated)."), schemabuilder.Deprecated("Use birthdate instead"))
	input.FieldFunc("reputation", func(target *CreateUserInput, source float64) { target.ReputationScore = source }, schemabuilder.FieldDesc("Reputation score."))
	input.FieldFunc("isActive", func(target *CreateUserInput, source bool) { target.IsActive = source }, schemabuilder.FieldDesc("Whether the user is active."))
	input.FieldFunc("role", func(target *CreateUserInput, source Role) { target.Role = source }, schemabuilder.FieldDesc("User role."))
}

// RegisterIdentifierInput registers the oneOf input for identifier (id or email;
// exactly one field). Uses MarkOneOf() method to mark as @oneOf input.
func RegisterIdentifierInput(sb *schemabuilder.Schema) {
	// INPUT_OBJECT w/ MarkOneOf() method for @oneOf; FieldFunc for id/email.
	// Description for Playground/__Type.
	identifierInput := sb.InputObject("IdentifierInput", IdentifierInput{}, schemabuilder.WithDescription("OneOf identifier: exactly one of ID or email (spec input union)."))
	identifierInput.MarkOneOf()
	identifierInput.FieldFunc("id", func(target *IdentifierInput, source *schemabuilder.ID) { target.ID = source }, schemabuilder.FieldDesc("User ID to identify an existing user."))
	identifierInput.FieldFunc("email", func(target *IdentifierInput, source *string) { target.Email = source }, schemabuilder.FieldDesc("Email address to identify an existing user."))
}

// RegisterUserInput registers input for user fields (copy fields; no deprecation).
// Specific func.
func RegisterUserInput(sb *schemabuilder.Schema) {
	// FieldFuncs populate; desc for INPUT_OBJECT.
	userInput := sb.InputObject("UserInput", UserInput{}, schemabuilder.WithDescription("User fields for creation (name, email etc)."))
	userInput.FieldFunc("name", func(target *UserInput, source string) { target.Name = source }, schemabuilder.FieldDesc("Name of the user."))
	userInput.FieldFunc("email", func(target *UserInput, source string) { target.Email = source }, schemabuilder.FieldDesc("Email address."))
	userInput.FieldFunc("age", func(target *UserInput, source int32) { target.Age = source }, schemabuilder.FieldDesc("Age in years."))
	userInput.FieldFunc("reputation", func(target *UserInput, source float64) { target.ReputationScore = source }, schemabuilder.FieldDesc("Reputation score."))
	userInput.FieldFunc("isActive", func(target *UserInput, source bool) { target.IsActive = source }, schemabuilder.FieldDesc("Whether the user is active."))
	userInput.FieldFunc("role", func(target *UserInput, source Role) { target.Role = source }, schemabuilder.FieldDesc("User role."))
}

// RegisterCreateUserByContactInput registers composite input for createUserByContact
// (identifier oneOf + userInput; per task improvement).
func RegisterCreateUserByContactInput(sb *schemabuilder.Schema) {
	// Composite input; desc.
	contactInput := sb.InputObject("CreateUserByContactInput", CreateUserByContactInput{}, schemabuilder.WithDescription("Create user by identifier (oneOf id/email) and user fields."))
	// FieldFunc for sub-objects (target populate; mirrors input FieldFunc).
	// Note: sub-inputs registered separately; parser handles nested.
	contactInput.FieldFunc("identifier", func(target *CreateUserByContactInput, source IdentifierInput) { target.Identifier = source }, schemabuilder.FieldDesc("Identifier input payload."))
	contactInput.FieldFunc("userInput", func(target *CreateUserByContactInput, source UserInput) { target.UserInput = source }, schemabuilder.FieldDesc("User input payload."))
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
