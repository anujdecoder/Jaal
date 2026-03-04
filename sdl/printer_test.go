package sdl

import (
	"testing"
)

// Helper function to create a string pointer
func strPtr(s string) *string {
	return &s
}

// Test scalar type printing
func TestPrintScalar(t *testing.T) {
	tests := []struct {
		name     string
		scalar   FullType
		expected string
	}{
		{
			name: "simple scalar",
			scalar: FullType{
				Kind: ScalarKind,
				Name: "DateTime",
			},
			expected: "scalar DateTime\n\n",
		},
		{
			name: "scalar with description",
			scalar: FullType{
				Kind:        ScalarKind,
				Name:        "DateTime",
				Description: "A date and time scalar",
			},
			expected: "\"A date and time scalar\"\nscalar DateTime\n\n",
		},
		{
			name: "scalar with specifiedByURL",
			scalar: FullType{
				Kind:           ScalarKind,
				Name:           "DateTime",
				SpecifiedByURL: strPtr("https://tools.ietf.org/html/rfc3339"),
			},
			expected: "scalar DateTime @specifiedBy(url: \"https://tools.ietf.org/html/rfc3339\")\n\n",
		},
		{
			name: "scalar with description and specifiedByURL",
			scalar: FullType{
				Kind:           ScalarKind,
				Name:           "DateTime",
				Description:    "A date and time scalar",
				SpecifiedByURL: strPtr("https://tools.ietf.org/html/rfc3339"),
			},
			expected: "\"A date and time scalar\"\nscalar DateTime @specifiedBy(url: \"https://tools.ietf.org/html/rfc3339\")\n\n",
		},
		{
			name: "scalar with multiline description",
			scalar: FullType{
				Kind:        ScalarKind,
				Name:        "JSON",
				Description: "A JSON scalar\nwith multiple lines",
			},
			expected: "\"\"\"\nA JSON scalar\nwith multiple lines\n\"\"\"\nscalar JSON\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printScalar(tt.scalar)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

// Test enum type printing
func TestPrintEnum(t *testing.T) {
	tests := []struct {
		name     string
		enum     FullType
		expected string
	}{
		{
			name: "simple enum",
			enum: FullType{
				Kind: EnumKind,
				Name: "Role",
				EnumValues: []EnumValue{
					{Name: "ADMIN"},
					{Name: "MEMBER"},
					{Name: "GUEST"},
				},
			},
			expected: "enum Role {\n  ADMIN\n  MEMBER\n  GUEST\n}\n\n",
		},
		{
			name: "enum with description",
			enum: FullType{
				Kind:        EnumKind,
				Name:        "Role",
				Description: "User role enum",
				EnumValues: []EnumValue{
					{Name: "ADMIN"},
					{Name: "MEMBER"},
				},
			},
			expected: "\"User role enum\"\nenum Role {\n  ADMIN\n  MEMBER\n}\n\n",
		},
		{
			name: "enum with deprecated value",
			enum: FullType{
				Kind: EnumKind,
				Name: "Status",
				EnumValues: []EnumValue{
					{Name: "ACTIVE"},
					{Name: "INACTIVE", IsDeprecated: true, DeprecationReason: strPtr("Use ACTIVE instead")},
				},
			},
			expected: "enum Status {\n  ACTIVE\n  INACTIVE @deprecated(reason: \"Use ACTIVE instead\")\n}\n\n",
		},
		{
			name: "enum with value description",
			enum: FullType{
				Kind: EnumKind,
				Name: "Role",
				EnumValues: []EnumValue{
					{Name: "ADMIN", Description: "Administrator role"},
					{Name: "MEMBER", Description: "Member role"},
				},
			},
			expected: "enum Role {\n  \"Administrator role\"\n  ADMIN\n  \"Member role\"\n  MEMBER\n}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printEnum(tt.enum)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

// Test object type printing
func TestPrintObject(t *testing.T) {
	tests := []struct {
		name     string
		object   FullType
		expected string
	}{
		{
			name: "simple object",
			object: FullType{
				Kind: ObjectKind,
				Name: "User",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
					{Name: "name", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			expected: "type User {\n  id: ID\n  name: String\n}\n\n",
		},
		{
			name: "object with description",
			object: FullType{
				Kind:        ObjectKind,
				Name:        "User",
				Description: "A user in the system",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
				},
			},
			expected: "\"A user in the system\"\ntype User {\n  id: ID\n}\n\n",
		},
		{
			name: "object with implements",
			object: FullType{
				Kind: ObjectKind,
				Name: "User",
				Interfaces: []TypeRef{
					{Name: "Node"},
					{Name: "Entity"},
				},
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
				},
			},
			expected: "type User implements Node & Entity {\n  id: ID\n}\n\n",
		},
		{
			name: "object with deprecated field",
			object: FullType{
				Kind: ObjectKind,
				Name: "User",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
					{Name: "oldField", Type: TypeRef{Kind: ScalarKind, Name: "String"}, IsDeprecated: true, DeprecationReason: strPtr("Use newField instead")},
				},
			},
			expected: "type User {\n  id: ID\n  oldField: String @deprecated(reason: \"Use newField instead\")\n}\n\n",
		},
		{
			name: "object with field arguments",
			object: FullType{
				Kind: ObjectKind,
				Name: "Query",
				Fields: []Field{
					{
						Name: "user",
						Type: TypeRef{Kind: ObjectKind, Name: "User"},
						Args: []InputValue{
							{Name: "id", Type: TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: ScalarKind, Name: "ID"}}},
						},
					},
				},
			},
			expected: "type Query {\n  user(id: ID!): User\n}\n\n",
		},
		{
			name: "object with field arguments and default values",
			object: FullType{
				Kind: ObjectKind,
				Name: "Query",
				Fields: []Field{
					{
						Name: "users",
						Type: TypeRef{Kind: ListKind, OfType: &TypeRef{Kind: ObjectKind, Name: "User"}},
						Args: []InputValue{
							{Name: "limit", Type: TypeRef{Kind: ScalarKind, Name: "Int"}, DefaultValue: strPtr("10")},
							{Name: "offset", Type: TypeRef{Kind: ScalarKind, Name: "Int"}, DefaultValue: strPtr("0")},
						},
					},
				},
			},
			expected: "type Query {\n  users(limit: Int = 10, offset: Int = 0): [User]\n}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printObject(tt.object)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

// Test type reference printing
func TestPrintTypeRef(t *testing.T) {
	tests := []struct {
		name     string
		typeRef  TypeRef
		expected string
	}{
		{
			name:     "simple scalar",
			typeRef:  TypeRef{Kind: ScalarKind, Name: "String"},
			expected: "String",
		},
		{
			name:     "simple object",
			typeRef:  TypeRef{Kind: ObjectKind, Name: "User"},
			expected: "User",
		},
		{
			name:     "non-null type",
			typeRef:  TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: ScalarKind, Name: "ID"}},
			expected: "ID!",
		},
		{
			name:     "list type",
			typeRef:  TypeRef{Kind: ListKind, OfType: &TypeRef{Kind: ObjectKind, Name: "User"}},
			expected: "[User]",
		},
		{
			name: "non-null list",
			typeRef: TypeRef{
				Kind: NonNullKind,
				OfType: &TypeRef{
					Kind:   ListKind,
					OfType: &TypeRef{Kind: ObjectKind, Name: "User"},
				},
			},
			expected: "[User]!",
		},
		{
			name: "list of non-null",
			typeRef: TypeRef{
				Kind: ListKind,
				OfType: &TypeRef{
					Kind:   NonNullKind,
					OfType: &TypeRef{Kind: ObjectKind, Name: "User"},
				},
			},
			expected: "[User!]",
		},
		{
			name: "non-null list of non-null",
			typeRef: TypeRef{
				Kind: NonNullKind,
				OfType: &TypeRef{
					Kind: ListKind,
					OfType: &TypeRef{
						Kind:   NonNullKind,
						OfType: &TypeRef{Kind: ObjectKind, Name: "User"},
					},
				},
			},
			expected: "[User!]!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printTypeRef(tt.typeRef)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test description formatting
func TestFormatDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		indent      string
		expected    string
	}{
		{
			name:        "empty description",
			description: "",
			indent:      "",
			expected:    "",
		},
		{
			name:        "simple description",
			description: "A simple description",
			indent:      "",
			expected:    "\"A simple description\"\n",
		},
		{
			name:        "description with indent",
			description: "A description",
			indent:      "  ",
			expected:    "  \"A description\"\n",
		},
		{
			name:        "multiline description",
			description: "Line 1\nLine 2",
			indent:      "",
			expected:    "\"\"\"\nLine 1\nLine 2\n\"\"\"\n",
		},
		{
			name:        "description with quotes",
			description: "A \"quoted\" description",
			indent:      "",
			expected:    "\"\"\"\nA \"quoted\" description\n\"\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDescription(tt.description, tt.indent)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test input object type printing
func TestPrintInputObject(t *testing.T) {
	tests := []struct {
		name     string
		input    FullType
		expected string
	}{
		{
			name: "simple input object",
			input: FullType{
				Kind: InputObjectKind,
				Name: "UserInput",
				InputFields: []InputValue{
					{Name: "name", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
					{Name: "email", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			expected: "input UserInput {\n  name: String\n  email: String\n}\n\n",
		},
		{
			name: "input object with description",
			input: FullType{
				Kind:        InputObjectKind,
				Name:        "UserInput",
				Description: "Input for creating a user",
				InputFields: []InputValue{
					{Name: "name", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			expected: "\"Input for creating a user\"\ninput UserInput {\n  name: String\n}\n\n",
		},
		{
			name: "input object with oneOf directive",
			input: FullType{
				Kind: InputObjectKind,
				Name: "IdentifierInput",
				Directives: []Directive{
					{Name: "oneOf"},
				},
				InputFields: []InputValue{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
					{Name: "email", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			expected: "input IdentifierInput @oneOf {\n  id: ID\n  email: String\n}\n\n",
		},
		{
			name: "input object with default values",
			input: FullType{
				Kind: InputObjectKind,
				Name: "PaginationInput",
				InputFields: []InputValue{
					{Name: "limit", Type: TypeRef{Kind: ScalarKind, Name: "Int"}, DefaultValue: strPtr("10")},
					{Name: "offset", Type: TypeRef{Kind: ScalarKind, Name: "Int"}, DefaultValue: strPtr("0")},
				},
			},
			expected: "input PaginationInput {\n  limit: Int = 10\n  offset: Int = 0\n}\n\n",
		},
		{
			name: "input object with deprecated field",
			input: FullType{
				Kind: InputObjectKind,
				Name: "UserInput",
				InputFields: []InputValue{
					{Name: "name", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
					{Name: "oldField", Type: TypeRef{Kind: ScalarKind, Name: "String"}, IsDeprecated: true, DeprecationReason: strPtr("Use name instead")},
				},
			},
			expected: "input UserInput {\n  name: String\n  oldField: String @deprecated(reason: \"Use name instead\")\n}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printInputObject(tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

// Test interface type printing
func TestPrintInterface(t *testing.T) {
	tests := []struct {
		name     string
		iface    FullType
		expected string
	}{
		{
			name: "simple interface",
			iface: FullType{
				Kind: InterfaceKind,
				Name: "Node",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: ScalarKind, Name: "ID"}}},
				},
			},
			expected: "interface Node {\n  id: ID!\n}\n\n",
		},
		{
			name: "interface with description",
			iface: FullType{
				Kind:        InterfaceKind,
				Name:        "Node",
				Description: "A node with an ID",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
				},
			},
			expected: "\"A node with an ID\"\ninterface Node {\n  id: ID\n}\n\n",
		},
		{
			name: "interface implementing other interfaces",
			iface: FullType{
				Kind: InterfaceKind,
				Name: "Timestamped",
				Interfaces: []TypeRef{
					{Name: "Node"},
				},
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: ScalarKind, Name: "ID"}},
					{Name: "createdAt", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			expected: "interface Timestamped implements Node {\n  id: ID\n  createdAt: String\n}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printInterface(tt.iface)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

// Test union type printing
func TestPrintUnion(t *testing.T) {
	tests := []struct {
		name     string
		union    FullType
		expected string
	}{
		{
			name: "simple union",
			union: FullType{
				Kind: UnionKind,
				Name: "SearchResult",
				PossibleTypes: []TypeRef{
					{Name: "User"},
					{Name: "Post"},
					{Name: "Comment"},
				},
			},
			expected: "union SearchResult = Comment | Post | User\n\n",
		},
		{
			name: "union with description",
			union: FullType{
				Kind:        UnionKind,
				Name:        "SearchResult",
				Description: "A search result union",
				PossibleTypes: []TypeRef{
					{Name: "User"},
					{Name: "Post"},
				},
			},
			expected: "\"A search result union\"\nunion SearchResult = Post | User\n\n",
		},
		{
			name: "empty union",
			union: FullType{
				Kind:          UnionKind,
				Name:          "EmptyUnion",
				PossibleTypes: []TypeRef{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := NewPrinter(Schema{})
			result := printer.printUnion(tt.union)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

// Test full schema printing
func TestPrintFullSchema(t *testing.T) {
	schema := Schema{
		QueryType: &NamedType{Name: "Query"},
		MutationType: &NamedType{Name: "Mutation"},
		Types: []FullType{
			{
				Kind:        ObjectKind,
				Name:        "Query",
				Description: "Root query type",
				Fields: []Field{
					{
						Name: "me",
						Type: TypeRef{Kind: ObjectKind, Name: "User"},
					},
					{
						Name: "user",
						Type: TypeRef{Kind: ObjectKind, Name: "User"},
						Args: []InputValue{
							{Name: "id", Type: TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: ScalarKind, Name: "ID"}}},
						},
					},
				},
			},
			{
				Kind:        ObjectKind,
				Name:        "Mutation",
				Description: "Root mutation type",
				Fields: []Field{
					{
						Name: "createUser",
						Type: TypeRef{Kind: ObjectKind, Name: "User"},
						Args: []InputValue{
							{Name: "input", Type: TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: InputObjectKind, Name: "UserInput"}}},
						},
					},
				},
			},
			{
				Kind:        ObjectKind,
				Name:        "User",
				Description: "A user in the system",
				Fields: []Field{
					{Name: "id", Type: TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: ScalarKind, Name: "ID"}}},
					{Name: "name", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
					{Name: "email", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			{
				Kind: InputObjectKind,
				Name: "UserInput",
				InputFields: []InputValue{
					{Name: "name", Type: TypeRef{Kind: NonNullKind, OfType: &TypeRef{Kind: ScalarKind, Name: "String"}}},
					{Name: "email", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
			{
				Kind: ScalarKind,
				Name: "ID",
			},
			{
				Kind: ScalarKind,
				Name: "String",
			},
		},
	}

	printer := NewPrinter(schema)
	result := printer.Print()

	// Check that the result contains expected elements
	expectedElements := []string{
		"schema {",
		"query: Query",
		"mutation: Mutation",
		"type Query",
		"type Mutation",
		"type User",
		"input UserInput",
		"me: User",
		"user(id: ID!): User",
		"createUser(input: UserInput!): User",
	}

	for _, expected := range expectedElements {
		if !contains(result, expected) {
			t.Errorf("expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
		}
	}
}

// Test built-in type filtering
func TestIsBuiltInType(t *testing.T) {
	printer := NewPrinter(Schema{})

	builtins := []string{"String", "Int", "Float", "Boolean", "ID", "__Schema", "__Type", "__Field", "__InputValue", "__EnumValue", "__Directive", "__TypeKind", "__DirectiveLocation"}
	for _, builtin := range builtins {
		if !printer.isBuiltInType(builtin) {
			t.Errorf("expected %q to be a built-in type", builtin)
		}
	}

	nonBuiltins := []string{"User", "Post", "DateTime", "Role"}
	for _, nonBuiltin := range nonBuiltins {
		if printer.isBuiltInType(nonBuiltin) {
			t.Errorf("expected %q to not be a built-in type", nonBuiltin)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(s)-len(substr)+1] != "" && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}