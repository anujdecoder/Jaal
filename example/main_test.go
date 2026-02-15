package main_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect" // for TypeOf in scalar register
	"testing"
	"time" // for DateTime scalar

	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// Blackbox test (package main_test) to avoid undefined internals from main.go.
// Runs example/main.go-equivalent schema via HTTP, uses full IntrospectionQuery.
// Tests 1: specifiedByURL only on DateTime.
// Tests 2: deprecation only on age in CreateUserInput.
// (Full registers replicated minimally; public only.)
func TestIntrospectionFromMain(t *testing.T) {
	// Minimal public schema replicate (scalars, input with dep tag, enum, ops for collection).
	sb := schemabuilder.NewSchema()

	// DateTime with specifiedBy (test 1).
	typ := reflect.TypeOf(time.Time{}) // reflect OK public.
	// Note: full scalar func from main init.
	_ = schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
		// Stub full from main.
		return nil
	}, "https://tools.ietf.org/html/rfc3339")

	// Input with dep tag on age (test 2).
	type TestInput struct { // Public test struct to avoid private CreateUserInput.
		Age int32 `json:"age" graphql:",deprecated=Use birthdate instead"`
		// Other fields non-dep.
		Name string
	}
	input := sb.InputObject("TestInput", TestInput{})
	input.FieldFunc("age", func(target *TestInput, source int32) { target.Age = source })
	input.FieldFunc("name", func(target *TestInput, source string) { target.Name = source })

	// Minimal to build + collect types.
	_ = sb.Object("TestObj", struct{}{})
	sb.Query().FieldFunc("test", func() bool { return true })

	schema, err := sb.Build()
	if err != nil {
		t.Fatal(err)
	}
	introspection.AddIntrospectionToSchema(schema)

	// HTTP server + full IntrospectionQuery fetch (from introspection/introspection_query.go).
	h := jaal.HTTPHandler(schema)
	server := httptest.NewServer(h)
	defer server.Close()
	reqBody, _ := json.Marshal(map[string]string{"query": introspection.IntrospectionQuery})
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result["errors"] != nil {
		t.Fatal(result["errors"])
	}
	schemaData := result["data"].(map[string]interface{})["__schema"].(map[string]interface{})

	// Test 1: specifiedByURL ONLY for DateTime (not others).
	types := schemaData["types"].([]interface{})
	dateTimeURL := ""
	for _, tIface := range types {
		typ := tIface.(map[string]interface{})
		if typ["name"] == "DateTime" {
			dateTimeURL = typ["specifiedByURL"].(string)
		} else if typ["specifiedByURL"] != nil && typ["specifiedByURL"] != "" {
			t.Errorf("non-DateTime scalar %v has specifiedByURL", typ["name"])
		}
	}
	if dateTimeURL != "https://tools.ietf.org/html/rfc3339" {
		t.Errorf("DateTime specifiedByURL missing: %v", dateTimeURL)
	}

	// Test 2: isDeprecated + reason ONLY for "age" in input (CreateUserInput equiv TestInput).
	for _, tIface := range types {
		typ := tIface.(map[string]interface{})
		if typ["name"] == "TestInput" {
			fields := typ["inputFields"].([]interface{})
			for _, fIface := range fields {
				f := fIface.(map[string]interface{})
				if f["name"] == "age" {
					if !f["isDeprecated"].(bool) || f["deprecationReason"] != "Use birthdate instead" {
						t.Errorf("age dep mismatch: %v %v", f["isDeprecated"], f["deprecationReason"])
					}
				} else if f["isDeprecated"].(bool) {
					t.Errorf("non-age field %v incorrectly deprecated", f["name"])
				}
			}
			break
		}
	}
}

