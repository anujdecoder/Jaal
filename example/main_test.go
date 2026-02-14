package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// TestIntrospectionCreateUserInput starts test HTTP server (jaal.HTTPHandler pattern from http_test.go),
// builds schema replicating example/main.go (DateTime scalar, Role enum, CreateUserInput with
// deprecation tag on age, dummy Query field to ensure type collection in introspection).
// HTTP POST introspection query, validates fields + age.isDeprecated=true/"Use birthdate instead"
// (verifies full wire from tag -> FieldDeprecations -> InputValue in introspection.go).
// No breaking changes.
func TestIntrospectionCreateUserInput(t *testing.T) {
	sb := schemabuilder.NewSchema()

	// Replicate scalar register (DateTime).
	typ := reflect.TypeOf(time.Time{})
	_ = schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
		v, ok := value.(string)
		if !ok {
			return nil
		}
		tm, _ := time.Parse(time.RFC3339, v)
		dest.Set(reflect.ValueOf(tm))
		return nil
	}, "https://tools.ietf.org/html/rfc3339")

	// Replicate enum for Role.
	sb.Enum(RoleMember, map[string]interface{}{
		"ADMIN":  RoleAdmin,
		"MEMBER": RoleMember,
		"GUEST":  RoleGuest,
	})

	// Replicate InputObject (dep tag on age).
	input := sb.InputObject("CreateUserInput", CreateUserInput{})
	input.FieldFunc("name", func(target *CreateUserInput, source string) { target.Name = source })
	input.FieldFunc("email", func(target *CreateUserInput, source string) { target.Email = source })
	input.FieldFunc("age", func(target *CreateUserInput, source int32) { target.Age = source })
	input.FieldFunc("reputation", func(target *CreateUserInput, source float64) { target.ReputationScore = source })
	input.FieldFunc("isActive", func(target *CreateUserInput, source bool) { target.IsActive = source })
	input.FieldFunc("role", func(target *CreateUserInput, source Role) { target.Role = source })

	// Minimal User object + dummy Query field using input (ensures CreateUserInput collected in
	// introspection.types via collectTypes; else __type returns null).
	_ = sb.Object("User", User{})
	sb.Query().FieldFunc("testInput", func(args struct{ Input *CreateUserInput }) bool { return true })

	// Build + introspection.
	schema, err := sb.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	introspection.AddIntrospectionToSchema(schema)

	// HTTP test server + call.
	h := jaal.HTTPHandler(schema)
	server := httptest.NewServer(h)
	defer server.Close()

	// Introspection query.
	introQuery := `{ __type(name: "CreateUserInput") { name kind inputFields { name type { name kind } isDeprecated deprecationReason } } }`
	reqBody, _ := json.Marshal(map[string]string{"query": introQuery})
	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	// Decode + validate.
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["errors"] != nil {
		t.Fatalf("errors: %v", result["errors"])
	}
	data := result["data"].(map[string]interface{})
	typData := data["__type"].(map[string]interface{})
	if typData["name"] != "CreateUserInput" {
		t.Fatalf("bad __type: %v", typData)
	}

	// Validate fields + age deprecation.
	inputFields := typData["inputFields"].([]interface{})
	if len(inputFields) != 6 {
		t.Fatalf("fields: %d", len(inputFields))
	}
	for _, fIface := range inputFields {
		field := fIface.(map[string]interface{})
		if field["name"] == "age" {
			if !field["isDeprecated"].(bool) {
				t.Errorf("age not deprecated")
			}
			if field["deprecationReason"] != "Use birthdate instead" {
				t.Errorf("bad reason: %v", field["deprecationReason"])
			}
			return // success
		}
	}
	t.Error("age field missing")
}
