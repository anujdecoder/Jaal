package schemabuilder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/internal"
)

// TestArgumentDeprecation tests the ArgDeprecation option for marking arguments as deprecated
func TestArgumentDeprecation(t *testing.T) {
	schema := NewSchema()

	query := schema.Query()
	query.FieldFunc("deprecatedArg", func(args struct {
		OldArg string
		NewArg string
	}) string {
		return args.NewArg
	}, ArgDeprecation("oldArg", "Use newArg instead"))

	builtSchema := schema.MustBuild()

	// Verify the argument deprecation is set by inspecting the Query object
	queryObj, ok := builtSchema.Query.(*graphql.Object)
	assert.True(t, ok, "Query should be an Object")

	field, ok := queryObj.Fields["deprecatedArg"]
	assert.True(t, ok, "deprecatedArg field should exist")

	// Check oldArg is deprecated
	oldArg, ok := field.Args["oldArg"]
	assert.True(t, ok, "oldArg should exist")
	assert.True(t, oldArg.IsDeprecated, "oldArg should be deprecated")
	assert.NotNil(t, oldArg.DeprecationReason, "oldArg should have deprecation reason")
	assert.Equal(t, "Use newArg instead", *oldArg.DeprecationReason)

	// Check newArg is not deprecated
	newArg, ok := field.Args["newArg"]
	assert.True(t, ok, "newArg should exist")
	assert.False(t, newArg.IsDeprecated, "newArg should not be deprecated")
	assert.Nil(t, newArg.DeprecationReason, "newArg should not have deprecation reason")
}

// TestMultipleArgumentDeprecations tests deprecating multiple arguments
func TestMultipleArgumentDeprecations(t *testing.T) {
	schema := NewSchema()

	query := schema.Query()
	query.FieldFunc("multiDeprecated", func(args struct {
		Arg1 string
		Arg2 string
		Arg3 string
	}) string {
		return args.Arg3
	}, ArgDeprecation("arg1", "Use arg3 instead"), ArgDeprecation("arg2", "Use arg3 instead"))

	builtSchema := schema.MustBuild()

	queryObj := builtSchema.Query.(*graphql.Object)
	field := queryObj.Fields["multiDeprecated"]

	arg1 := field.Args["arg1"]
	assert.True(t, arg1.IsDeprecated)
	assert.Equal(t, "Use arg3 instead", *arg1.DeprecationReason)

	arg2 := field.Args["arg2"]
	assert.True(t, arg2.IsDeprecated)
	assert.Equal(t, "Use arg3 instead", *arg2.DeprecationReason)

	arg3 := field.Args["arg3"]
	assert.False(t, arg3.IsDeprecated)
}

// TestEnumValueDeprecation tests the EnumValueDeprecation option
func TestEnumValueDeprecation(t *testing.T) {
	schema := NewSchema()

	type Status int32
	schema.Enum(Status(0), map[string]interface{}{
		"ACTIVE":     Status(0),
		"INACTIVE":   Status(1),
		"DEPRECATED": Status(2),
	}, EnumValueDeprecation("DEPRECATED", "Use INACTIVE instead"))

	query := schema.Query()
	query.FieldFunc("getStatus", func(args struct {
		Status Status
	}) Status {
		return args.Status
	})

	builtSchema := schema.MustBuild()

	// Verify enum value deprecation by looking up the type in the Query fields
	queryObj := builtSchema.Query.(*graphql.Object)
	field := queryObj.Fields["getStatus"]

	// The enum type is referenced by the argument (directly *graphql.Enum for pointer types)
	arg := field.Args["status"]
	assert.NotNil(t, arg)

	// Get the enum type from the argument type
	statusEnum, ok := arg.Type.(*graphql.Enum)
	assert.True(t, ok, "Status should be an Enum, got %T", arg.Type)

	assert.Contains(t, statusEnum.ValueDeprecations, "DEPRECATED", "DEPRECATED should have deprecation")
	assert.Equal(t, "Use INACTIVE instead", statusEnum.ValueDeprecations["DEPRECATED"])
	assert.NotContains(t, statusEnum.ValueDeprecations, "ACTIVE", "ACTIVE should not be deprecated")
	assert.NotContains(t, statusEnum.ValueDeprecations, "INACTIVE", "INACTIVE should not be deprecated")
}

// TestMultipleEnumValueDeprecations tests deprecating multiple enum values
func TestMultipleEnumValueDeprecations(t *testing.T) {
	schema := NewSchema()

	type Role int32
	schema.Enum(Role(0), map[string]interface{}{
		"ADMIN":  Role(0),
		"USER":   Role(1),
		"GUEST":  Role(2),
		"LEGACY": Role(3),
	}, EnumValueDeprecation("GUEST", "Use USER instead"), EnumValueDeprecation("LEGACY", "Remove in v2.0"))

	query := schema.Query()
	query.FieldFunc("getRole", func(args struct {
		Role Role
	}) Role {
		return args.Role
	})

	builtSchema := schema.MustBuild()

	queryObj := builtSchema.Query.(*graphql.Object)
	field := queryObj.Fields["getRole"]
	arg := field.Args["role"]

	roleEnum, ok := arg.Type.(*graphql.Enum)
	assert.True(t, ok, "Role should be an Enum, got %T", arg.Type)

	assert.Equal(t, "Use USER instead", roleEnum.ValueDeprecations["GUEST"])
	assert.Equal(t, "Remove in v2.0", roleEnum.ValueDeprecations["LEGACY"])
	assert.NotContains(t, roleEnum.ValueDeprecations, "ADMIN")
	assert.NotContains(t, roleEnum.ValueDeprecations, "USER")
}

// TestArgumentDeprecationWithFieldDescription tests that argument deprecation works with field descriptions
func TestArgumentDeprecationWithFieldDescription(t *testing.T) {
	schema := NewSchema()

	query := schema.Query()
	query.FieldFunc("complexField", func(args struct {
		OldParam string
		NewParam string
	}) string {
		return args.NewParam
	}, FieldDesc("A field with deprecated arguments"), ArgDeprecation("oldParam", "Use newParam"))

	builtSchema := schema.MustBuild()

	queryObj := builtSchema.Query.(*graphql.Object)
	field := queryObj.Fields["complexField"]

	assert.Equal(t, "A field with deprecated arguments", field.Description)

	oldParam := field.Args["oldParam"]
	assert.True(t, oldParam.IsDeprecated)
	assert.Equal(t, "Use newParam", *oldParam.DeprecationReason)
}

// TestDeprecatedArgumentQueryExecution tests that queries with deprecated arguments still execute correctly
func TestDeprecatedArgumentQueryExecution(t *testing.T) {
	schema := NewSchema()

	query := schema.Query()
	query.FieldFunc("echo", func(args struct {
		Message string
	}) string {
		return args.Message
	}, ArgDeprecation("message", "This argument is deprecated"))

	builtSchema := schema.MustBuild()

	q, err := graphql.Parse(`{ echo(message: "hello") }`, nil)
	assert.Nil(t, err)

	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"echo": "hello"}, internal.AsJSON(val))
}

// TestEnumValueDeprecationQueryExecution tests that queries with deprecated enum values still execute
func TestEnumValueDeprecationQueryExecution(t *testing.T) {
	schema := NewSchema()

	type Priority int32
	schema.Enum(Priority(0), map[string]interface{}{
		"LOW":    Priority(0),
		"MEDIUM": Priority(1),
		"HIGH":   Priority(2),
		"URGENT": Priority(3),
	}, EnumValueDeprecation("URGENT", "Use HIGH instead"))

	query := schema.Query()
	query.FieldFunc("getPriority", func(args struct {
		Priority Priority
	}) string {
		switch args.Priority {
		case 0:
			return "low"
		case 1:
			return "medium"
		case 2, 3:
			return "high"
		default:
			return "unknown"
		}
	})

	builtSchema := schema.MustBuild()

	// Test with deprecated enum value
	q, err := graphql.Parse(`{ getPriority(priority: URGENT) }`, nil)
	assert.Nil(t, err)

	if err := graphql.ValidateQuery(context.Background(), builtSchema.Query, q.SelectionSet); err != nil {
		t.Error(err)
	}

	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{"getPriority": "high"}, internal.AsJSON(val))
}

// TestArgumentDeprecationReasonNilIfNotDeprecated tests that DeprecationReason is nil when not deprecated
func TestArgumentDeprecationReasonNilIfNotDeprecated(t *testing.T) {
	schema := NewSchema()

	query := schema.Query()
	query.FieldFunc("noDeprecation", func(args struct {
		NormalArg string
	}) string {
		return args.NormalArg
	})

	builtSchema := schema.MustBuild()

	queryObj := builtSchema.Query.(*graphql.Object)
	field := queryObj.Fields["noDeprecation"]
	arg := field.Args["normalArg"]

	assert.False(t, arg.IsDeprecated)
	assert.Nil(t, arg.DeprecationReason)
}
