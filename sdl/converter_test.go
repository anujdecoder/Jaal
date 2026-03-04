package sdl

import (
	"strings"
	"testing"
)

// Test IntrospectionJSONToSDL function
func TestIntrospectionJSONToSDL(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expectError bool
		contains    []string
	}{
		{
			name: "valid introspection JSON",
			json: `{
				"__schema": {
					"queryType": {"name": "Query"},
					"mutationType": null,
					"subscriptionType": null,
					"types": [
						{
							"kind": "OBJECT",
							"name": "Query",
							"description": "Root query",
							"fields": [
								{
									"name": "hello",
									"description": "Returns hello",
									"args": [],
									"type": {"kind": "SCALAR", "name": "String", "ofType": null},
									"isDeprecated": false
								}
							]
						},
						{
							"kind": "SCALAR",
							"name": "String",
							"description": ""
						}
					],
					"directives": []
				}
			}`,
			expectError: false,
			contains: []string{
				"schema {",
				"query: Query",
				"type Query",
				"hello: String",
			},
		},
		{
			name:        "invalid JSON",
			json:        `{invalid}`,
			expectError: true,
		},
		{
			name:        "empty JSON object",
			json:        `{}`,
			expectError: false,
			contains:    []string{},
		},
		{
			name: "schema with enum",
			json: `{
				"__schema": {
					"queryType": {"name": "Query"},
					"types": [
						{
							"kind": "ENUM",
							"name": "Role",
							"description": "User role",
							"enumValues": [
								{"name": "ADMIN", "description": "", "isDeprecated": false},
								{"name": "MEMBER", "description": "", "isDeprecated": false}
							]
						}
					],
					"directives": []
				}
			}`,
			expectError: false,
			contains: []string{
				"enum Role",
				"ADMIN",
				"MEMBER",
			},
		},
		{
			name: "schema with input object and oneOf",
			json: `{
				"__schema": {
					"queryType": {"name": "Query"},
					"types": [
						{
							"kind": "INPUT_OBJECT",
							"name": "IdentifierInput",
							"description": "OneOf identifier",
							"inputFields": [
								{"name": "id", "type": {"kind": "SCALAR", "name": "ID"}, "isDeprecated": false},
								{"name": "email", "type": {"kind": "SCALAR", "name": "String"}, "isDeprecated": false}
							],
							"directives": [{"name": "oneOf"}]
						}
					],
					"directives": []
				}
			}`,
			expectError: false,
			contains: []string{
				"input IdentifierInput @oneOf",
				"id: ID",
				"email: String",
			},
		},
		{
			name: "schema with union",
			json: `{
				"__schema": {
					"queryType": {"name": "Query"},
					"types": [
						{
							"kind": "UNION",
							"name": "SearchResult",
							"description": "Search result union",
							"possibleTypes": [
								{"name": "User"},
								{"name": "Post"}
							]
						}
					],
					"directives": []
				}
			}`,
			expectError: false,
			contains: []string{
				"union SearchResult = Post | User",
			},
		},
		{
			name: "schema with deprecated field",
			json: `{
				"__schema": {
					"queryType": {"name": "Query"},
					"types": [
						{
							"kind": "OBJECT",
							"name": "Query",
							"fields": [
								{
									"name": "oldField",
									"args": [],
									"type": {"kind": "SCALAR", "name": "String"},
									"isDeprecated": true,
									"deprecationReason": "Use newField instead"
								}
							]
						}
					],
					"directives": []
				}
			}`,
			expectError: false,
			contains: []string{
				"oldField: String @deprecated(reason: \"Use newField instead\")",
			},
		},
		{
			name: "schema with scalar and specifiedByURL",
			json: `{
				"__schema": {
					"queryType": {"name": "Query"},
					"types": [
						{
							"kind": "SCALAR",
							"name": "DateTime",
							"description": "A datetime scalar",
							"specifiedByURL": "https://tools.ietf.org/html/rfc3339"
						}
					],
					"directives": []
				}
			}`,
			expectError: false,
			contains: []string{
				"scalar DateTime @specifiedBy(url: \"https://tools.ietf.org/html/rfc3339\")",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IntrospectionJSONToSDL([]byte(tt.json))

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
				}
			}
		})
	}
}

// Test IntrospectionDataToSDL function
func TestIntrospectionDataToSDL(t *testing.T) {
	schema := Schema{
		QueryType: &NamedType{Name: "Query"},
		Types: []FullType{
			{
				Kind: ObjectKind,
				Name: "Query",
				Fields: []Field{
					{Name: "hello", Type: TypeRef{Kind: ScalarKind, Name: "String"}},
				},
			},
		},
	}

	result := IntrospectionDataToSDL(schema)

	expectedElements := []string{
		"schema {",
		"query: Query",
		"type Query",
		"hello: String",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
		}
	}
}

// Test complex nested types
func TestNestedTypes(t *testing.T) {
	json := `{
		"__schema": {
			"queryType": {"name": "Query"},
			"types": [
				{
					"kind": "OBJECT",
					"name": "Query",
					"fields": [
						{
							"name": "users",
							"args": [],
							"type": {
								"kind": "NON_NULL",
								"name": null,
								"ofType": {
									"kind": "LIST",
									"name": null,
									"ofType": {
										"kind": "NON_NULL",
										"name": null,
										"ofType": {
											"kind": "OBJECT",
											"name": "User",
											"ofType": null
										}
									}
								}
							},
							"isDeprecated": false
						}
					]
				},
				{
					"kind": "OBJECT",
					"name": "User",
					"fields": [
						{"name": "id", "args": [], "type": {"kind": "SCALAR", "name": "ID"}, "isDeprecated": false},
						{"name": "name", "args": [], "type": {"kind": "SCALAR", "name": "String"}, "isDeprecated": false}
					]
				}
			],
			"directives": []
		}
	}`

	result, err := IntrospectionJSONToSDL([]byte(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "users: [User!]!"
	if !strings.Contains(result, expected) {
		t.Errorf("expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
	}
}

// Test field with arguments
func TestFieldWithArguments(t *testing.T) {
	json := `{
		"__schema": {
			"queryType": {"name": "Query"},
			"types": [
				{
					"kind": "OBJECT",
					"name": "Query",
					"fields": [
						{
							"name": "user",
							"args": [
								{
									"name": "id",
									"type": {"kind": "NON_NULL", "ofType": {"kind": "SCALAR", "name": "ID"}},
									"defaultValue": null,
									"isDeprecated": false
								},
								{
									"name": "includeDeleted",
									"type": {"kind": "SCALAR", "name": "Boolean"},
									"defaultValue": "false",
									"isDeprecated": false
								}
							],
							"type": {"kind": "OBJECT", "name": "User"},
							"isDeprecated": false
						}
					]
				},
				{
					"kind": "OBJECT",
					"name": "User",
					"fields": [
						{"name": "id", "args": [], "type": {"kind": "SCALAR", "name": "ID"}, "isDeprecated": false}
					]
				}
			],
			"directives": []
		}
	}`

	result, err := IntrospectionJSONToSDL([]byte(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "user(id: ID!, includeDeleted: Boolean = false): User"
	if !strings.Contains(result, expected) {
		t.Errorf("expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
	}
}

// Test schema with subscription
func TestSchemaWithSubscription(t *testing.T) {
	json := `{
		"__schema": {
			"queryType": {"name": "Query"},
			"mutationType": {"name": "Mutation"},
			"subscriptionType": {"name": "Subscription"},
			"types": [
				{
					"kind": "OBJECT",
					"name": "Query",
					"fields": [{"name": "hello", "args": [], "type": {"kind": "SCALAR", "name": "String"}, "isDeprecated": false}]
				},
				{
					"kind": "OBJECT",
					"name": "Mutation",
					"fields": [{"name": "greet", "args": [], "type": {"kind": "SCALAR", "name": "String"}, "isDeprecated": false}]
				},
				{
					"kind": "OBJECT",
					"name": "Subscription",
					"fields": [{"name": "onGreet", "args": [], "type": {"kind": "SCALAR", "name": "String"}, "isDeprecated": false}]
				}
			],
			"directives": []
		}
	}`

	result, err := IntrospectionJSONToSDL([]byte(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedElements := []string{
		"query: Query",
		"mutation: Mutation",
		"subscription: Subscription",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
		}
	}
}