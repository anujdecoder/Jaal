# Design Document: Argument Deprecation & Enum Value Deprecation

**Feature:** Phase 1.1 & 1.2 - Complete Deprecation Support  
**Status:** Design  
**Target Version:** Jaal v1.x  
**Spec Compliance:** GraphQL October 2021+  
**Author:** AI Assistant  
**Date:** 2026-04-05

---

## 1. Executive Summary

This document details the implementation plan for completing the `@deprecated` directive support for:
1. **Field Arguments** (ARGUMENT_DEFINITION location)
2. **Input Object Fields** (INPUT_FIELD_DEFINITION location)  
3. **Enum Values** (ENUM_VALUE location)

Currently, these features are stubbed in introspection (always returning `isDeprecated: false` and `deprecationReason: nil`). This design enables full deprecation support across the entire schema.

---

## 2. Current State Analysis

### 2.1 What's Already Implemented
- ✅ `@deprecated` on FIELD_DEFINITION (output fields)
- ✅ `@deprecated` on INPUT_FIELD_DEFINITION (struct fields via tags)
- ✅ Introspection schema has `isDeprecated` and `deprecationReason` fields
- ✅ SDL printer supports `@deprecated` on fields

### 2.2 What's Stubbed/Missing
- ❌ Argument deprecation in introspection (returns hardcoded false/nil)
- ❌ Enum value deprecation (not tracked in EnumMapping)
- ❌ API to mark arguments as deprecated
- ❌ SDL output for deprecated arguments and enum values

### 2.3 Stub Locations
```go
// introspection/introspection.go:386-393
args = append(args, InputValue{
    Name:              name,
    Type:              Type{Inner: a},
    IsDeprecated:      false,  // <-- STUB
    DeprecationReason: nil,    // <-- STUB
})

// introspection/introspection.go:470-471
enumVals = append(enumVals,
    EnumValue{Name: v, Description: val, IsDeprecated: false, DeprecationReason: nil}) // <-- STUB
```

---

## 3. Design Goals

1. **Spec Compliance:** Full support for ARGUMENT_DEFINITION, INPUT_FIELD_DEFINITION, and ENUM_VALUE directive locations
2. **Backward Compatibility:** Zero breaking changes to existing APIs
3. **Consistent API:** Deprecation API should match existing field deprecation pattern
4. **Complete Integration:** Introspection, SDL, validation, and examples all updated
5. **Test Coverage:** Unit, integration, and end-to-end tests

---

## 4. Detailed Design

### 4.1 Argument Deprecation Design

#### 4.1.1 Core Data Model Changes

**File: `graphql/types.go`**

Extend the `Field` struct to track argument deprecation:

```go
// Field knows how to compute field values of an Object
type Field struct {
    Resolve        Resolver
    Type           Type
    Args           map[string]Type
    ParseArguments func(json interface{}) (interface{}, error)
    External       bool
    Expensive      bool
    LazyExecution  bool
    LazyResolver   func(ctx context.Context, fun interface{}) (interface{}, error)
    IsDeprecated      bool
    DeprecationReason *string `json:"deprecationReason,omitempty"`
    Description       string  `json:"description,omitempty"`
    
    // NEW: Argument deprecation metadata
    // ArgIsDeprecated maps argument name to deprecation status
    ArgIsDeprecated map[string]bool `json:"-"`
    // ArgDeprecationReason maps argument name to deprecation reason
    ArgDeprecationReason map[string]string `json:"-"`
}
```

**Alternative Design (Preferred for consistency):**

Create an `Argument` type to replace `map[string]Type`:

```go
// Argument represents a GraphQL field argument
type Argument struct {
    Type              Type
    Description       string
    IsDeprecated      bool
    DeprecationReason *string
    DefaultValue      interface{} // For future use
}

// Field Args becomes:
Args map[string]*Argument
```

**Decision:** Use Alternative Design ( cleaner, extensible, matches spec structure).

#### 4.1.2 Schema Builder API

**File: `schemabuilder/types.go`**

Add deprecation options for field registration:

```go
// ArgDeprecation marks a specific argument as deprecated
func ArgDeprecation(argName string, reason string) FieldOption {
    return func(cfg *fieldConfig) {
        if cfg.argDeprecations == nil {
            cfg.argDeprecations = make(map[string]string)
        }
        cfg.argDeprecations[argName] = reason
    }
}

// Update fieldConfig struct
type fieldConfig struct {
    description       string
    deprecated        string
    nonNull           bool
    argDeprecations   map[string]string // NEW: arg name -> reason
}
```

**File: `schemabuilder/function.go`**

Update `buildFunctionAndFuncCtx` to apply argument deprecation:

```go
func (sb *schemaBuilder) buildFunction(typ reflect.Type, m *method) (*graphql.Field, error) {
    // ... existing code ...
    
    args, err := funcCtx.argsTypeMap(argType)
    if err != nil {
        return nil, err
    }
    
    // NEW: Apply argument deprecations from method config
    for argName, reason := range m.ArgDeprecations {
        if arg, ok := args[argName]; ok {
            arg.IsDeprecated = true
            arg.DeprecationReason = &reason
        }
    }
    
    return &graphql.Field{
        // ... existing fields ...
        Args: args,
    }, nil
}
```

#### 4.1.3 Input Object Field Deprecation

**File: `schemabuilder/input.go`**

Already partially supported via struct tags. Need to ensure it flows through to introspection.

Current support in `generateArgParser`:
```go
if obj.FieldDeprecations != nil {
    if depReason, ok := obj.FieldDeprecations[name]; ok {
        argType.FieldDeprecations[name] = depReason
    }
}
```

Need to verify this flows to `graphql.InputObject.FieldDeprecations` and then to introspection.

### 4.2 Enum Value Deprecation Design

#### 4.2.1 Core Data Model Changes

**File: `schemabuilder/types.go`**

Extend `EnumMapping` to support per-value deprecation:

```go
// EnumMapping is a representation of an enum that includes both the mapping and reverse mapping
type EnumMapping struct {
    Map         map[string]interface{}
    ReverseMap  map[interface{}]string
    Description string
    // NEW: Per-value deprecation information
    ValueDeprecations map[string]string // enum value name -> deprecation reason
}
```

#### 4.2.2 Schema Builder API

Add new enum registration option:

```go
// EnumValueDeprecation marks specific enum values as deprecated
func EnumValueDeprecation(valueName string, reason string) TypeOption {
    return func(cfg *typeConfig) {
        if cfg.enumDeprecations == nil {
            cfg.enumDeprecations = make(map[string]string)
        }
        cfg.enumDeprecations[valueName] = reason
    }
}

// Update typeConfig struct
type typeConfig struct {
    description       string
    deprecated        string
    directives        []string
    enumDeprecations  map[string]string // NEW: for enum value deprecations
}
```

Update `Schema.Enum()` method signature to accept options:

```go
// Enum registers an enumType in the schema with optional configuration
func (s *Schema) Enum(val interface{}, enumMap interface{}, opts ...TypeOption) {
    typ := reflect.TypeOf(val)
    if s.enumTypes == nil {
        s.enumTypes = make(map[reflect.Type]*EnumMapping)
    }

    eMap, rMap := getEnumMap(enumMap, typ)
    cfg := applyTypeOptions(opts)
    
    s.enumTypes[typ] = &EnumMapping{
        Map:               eMap,
        ReverseMap:        rMap,
        Description:       cfg.description,
        ValueDeprecations: cfg.enumDeprecations, // NEW
    }
}
```

### 4.3 Introspection Updates

**File: `introspection/introspection.go`**

Update `registerType` to use actual deprecation data:

```go
// In registerType, update the fields resolver for Object/Interface:
object.FieldFunc("fields", func(t Type, args struct {
    IncludeDeprecated *bool
}) []field {
    // ... existing code ...
    for name, a := range f.Args {
        // Use actual deprecation data from field
        isDep := false
        var depReason *string
        if a.IsDeprecated {
            isDep = true
            depReason = a.DeprecationReason
        }
        
        args = append(args, InputValue{
            Name:              name,
            Type:              Type{Inner: a.Type},
            IsDeprecated:      isDep,
            DeprecationReason: depReason,
        })
    }
})

// Update enumValues resolver:
object.FieldFunc("enumValues", func(t Type, args struct {
    IncludeDeprecated *bool
}) []EnumValue {
    switch t := t.Inner.(type) {
    case *graphql.Enum:
        var enumVals []EnumValue
        for k, v := range t.ReverseMap {
            val := fmt.Sprintf("%v", k)
            
            // NEW: Check for enum value deprecation
            isDep := false
            var depReason *string
            if reason, ok := t.ValueDeprecations[v]; ok && reason != "" {
                isDep = true
                depReason = &reason
            }
            
            enumVals = append(enumVals, EnumValue{
                Name:              v,
                Description:       val,
                IsDeprecated:      isDep,
                DeprecationReason: depReason,
            })
        }
        sort.Slice(enumVals, func(i, j int) bool { return enumVals[i].Name < enumVals[j].Name })
        return enumVals
    }
    return nil
})
```

### 4.4 SDL Printer Updates

**File: `sdl/printer.go`**

#### 4.4.1 Argument Deprecation in SDL

Update `printField` to include argument deprecation:

```go
func (p *Printer) printField(field Field) string {
    var sb strings.Builder
    
    // Print arguments with deprecation
    if len(field.Args) > 0 {
        sb.WriteString("(")
        var args []string
        for name, arg := range field.Args {
            argStr := p.printInputValue(name, arg)
            
            // NEW: Add deprecation directive if present
            if arg.IsDeprecated && arg.DeprecationReason != nil {
                argStr += fmt.Sprintf(" @deprecated(reason: \"%s\")", *arg.DeprecationReason)
            }
            
            args = append(args, argStr)
        }
        sort.Strings(args)
        sb.WriteString(strings.Join(args, ", "))
        sb.WriteString(")")
    }
    
    // ... rest of field printing ...
}

func (p *Printer) printInputValue(name string, arg Argument) string {
    // ... existing logic ...
    return fmt.Sprintf("%s: %s", name, typeStr)
}
```

#### 4.4.2 Enum Value Deprecation in SDL

Update `printEnum` to include value deprecation:

```go
func (p *Printer) printEnum(t FullType) string {
    var sb strings.Builder
    // ... header ...
    
    for _, v := range t.EnumValues {
        sb.WriteString(p.indent + v.Name)
        
        // NEW: Add deprecation directive
        if v.IsDeprecated && v.DeprecationReason != nil {
            sb.WriteString(fmt.Sprintf(" @deprecated(reason: \"%s\")", *v.DeprecationReason))
        }
        
        sb.WriteString("\n")
    }
    
    sb.WriteString("}\n\n")
    return sb.String()
}
```

### 4.5 Validation Updates

**File: `graphql/validate.go`**

Add validation to warn about using deprecated arguments (optional but helpful):

```go
// ValidateDeprecatedArgs checks if deprecated arguments are being used
func ValidateDeprecatedArgs(selectionSet *SelectionSet, typ Type) []error {
    var warnings []error
    // Implementation to check for deprecated arg usage
    return warnings
}
```

Note: This is optional as spec doesn't require validation errors for deprecated usage, just introspection visibility.

---

## 5. Implementation Plan

### 5.1 Phase 1: Core Data Model (Day 1-2)

1. **Update `graphql/types.go`**
   - Create new `Argument` struct
   - Update `Field.Args` from `map[string]Type` to `map[string]*Argument`
   - Add `ValueDeprecations` to `Enum` struct

2. **Update `schemabuilder/types.go`**
   - Add `enumDeprecations` to `typeConfig`
   - Add `argDeprecations` to `fieldConfig`
   - Add `EnumValueDeprecation()` option function
   - Add `ArgDeprecation()` option function

### 5.2 Phase 2: Schema Builder Updates (Day 3-4)

1. **Update `schemabuilder/function.go`**
   - Modify `buildFunctionAndFuncCtx` to apply arg deprecations
   - Update `argsTypeMap` to return `map[string]*Argument`

2. **Update `schemabuilder/schema.go`**
   - Modify `Enum()` to handle deprecation options

3. **Update `schemabuilder/build.go`**
   - Update all places that access `Field.Args` to use new `Argument` type

### 5.3 Phase 3: Introspection Updates (Day 5)

1. **Update `introspection/introspection.go`**
   - Remove stubs for argument deprecation
   - Use actual `Argument.IsDeprecated` and `Argument.DeprecationReason`
   - Update `enumValues` resolver to check `Enum.ValueDeprecations`

### 5.4 Phase 4: SDL Printer (Day 6)

1. **Update `sdl/printer.go`**
   - Update `printField` to output deprecated arguments
   - Update `printEnum` to output deprecated enum values
   - Update type definitions to use new `Argument` type

2. **Update `sdl/converter.go`**
   - Ensure deprecation data is properly mapped from introspection

### 5.5 Phase 5: Tests (Day 7-10)

#### 5.5.1 Unit Tests

**New file: `schemabuilder/deprecation_test.go`**

```go
func TestArgumentDeprecation(t *testing.T) {
    schema := NewSchema()
    query := schema.Query()
    
    query.FieldFunc("user", func(args struct {
        ID   *ID
        Name *string
    }) string {
        return "user"
    }, ArgDeprecation("name", "Use ID instead"))
    
    built := schema.MustBuild()
    userField := built.Query.(*graphql.Object).Fields["user"]
    
    assert.True(t, userField.Args["name"].IsDeprecated)
    assert.Equal(t, "Use ID instead", *userField.Args["name"].DeprecationReason)
    assert.False(t, userField.Args["id"].IsDeprecated)
}

func TestEnumValueDeprecation(t *testing.T) {
    schema := NewSchema()
    type Status int32
    
    schema.Enum(Status(0), map[string]interface{}{
        "ACTIVE":   Status(0),
        "INACTIVE": Status(1),
    }, EnumValueDeprecation("INACTIVE", "Use ACTIVE"))
    
    built := schema.MustBuild()
    // Verify enum mapping has deprecation
}
```

**New file: `introspection/deprecation_test.go`**

```go
func TestIntrospectionArgumentDeprecation(t *testing.T) {
    // Test that deprecated args show correctly in introspection
}

func TestIntrospectionEnumValueDeprecation(t *testing.T) {
    // Test that deprecated enum values show correctly
}
```

#### 5.5.2 Integration Tests

**Update: `graphql/end_to_end_test.go`**

Add tests for:
- Query with deprecated argument (should still work)
- Introspection query for deprecated args
- Introspection query for deprecated enum values

#### 5.5.3 SDL Tests

**Update: `sdl/printer_test.go`**

Add test cases:
- Field with deprecated argument
- Enum with deprecated values
- Round-trip (parse SDL -> introspection -> SDL)

### 5.6 Phase 6: Examples (Day 11-12)

#### 5.6.1 Update Example Schema

**File: `example/users/register_queries.go`**

Add example with deprecated argument:

```go
func RegisterQuery(sb *schemabuilder.Schema, s *Server) {
    query := sb.Query()
    
    // Existing queries...
    
    // NEW: Example with deprecated argument
    query.FieldFunc("userByID", func(ctx context.Context, args struct {
        ID   *schemabuilder.ID
        Name *string // Deprecated
    }) *User {
        // Return user by ID, ignore name
        return s.getUserByID(args.ID.Value)
    }, 
        schemabuilder.FieldDesc("Get user by ID (name param deprecated)"),
        schemabuilder.ArgDeprecation("name", "Use ID instead"),
    )
}
```

**File: `example/users/register_enums.go`**

Add enum with deprecated value:

```go
func RegisterEnums(sb *schemabuilder.Schema) {
    // Existing Role enum...
    
    // NEW: Status enum with deprecated value
    type Status int32
    sb.Enum(Status(0), map[string]interface{}{
        "ACTIVE":     Status(0),
        "INACTIVE":   Status(1),
        "SUSPENDED":  Status(2),
    }, 
        schemabuilder.WithDescription("User account status"),
        schemabuilder.EnumValueDeprecation("INACTIVE", "Use SUSPENDED instead"),
    )
}
```

#### 5.6.2 Update Integration Tests

**File: `example/users/server_test.go`**

Add tests verifying:
- Deprecated args appear in introspection
- Deprecated enum values appear in introspection
- SDL output includes deprecation directives

### 5.7 Phase 7: Documentation (Day 13)

1. **Update `README.md`**
   - Add section on argument deprecation
   - Add section on enum value deprecation
   - Update deprecation section to show all supported locations

2. **Add code documentation**
   - Document new exported functions
   - Add examples in GoDoc format

---

## 6. Backward Compatibility

### 6.1 Breaking Changes

**NONE** - All changes are additive.

### 6.2 Migration Path

Existing code continues to work without changes:

```go
// Old code (still works)
schema.Enum(Status(0), map[string]interface{}{
    "ACTIVE": Status(0),
})

query.FieldFunc("user", resolver)

// New code (optional enhancement)
schema.Enum(Status(0), map[string]interface{}{
    "ACTIVE": Status(0),
}, schemabuilder.EnumValueDeprecation("OLD", "Use NEW"))

query.FieldFunc("user", resolver, schemabuilder.ArgDeprecation("oldArg", "Use newArg"))
```

### 6.3 Internal Compatibility

The change from `map[string]Type` to `map[string]*Argument` is internal-only:
- The `graphql` package is internal to the framework
- The `schemabuilder` public API remains unchanged except for new optional functions

---

## 7. API Reference

### 7.1 New Public Functions

```go
// schemabuilder package

// ArgDeprecation marks a field argument as deprecated
func ArgDeprecation(argName string, reason string) FieldOption

// EnumValueDeprecation marks an enum value as deprecated
func EnumValueDeprecation(valueName string, reason string) TypeOption
```

### 7.2 Usage Examples

#### Deprecating a Field Argument

```go
schema := schemabuilder.NewSchema()
query := schema.Query()

query.FieldFunc("searchUsers", func(ctx context.Context, args struct {
    Query    string
    Username string  // Deprecated: use Query for full-text search
}) []*User {
    // Implementation uses Query, ignores Username
    return searchUsers(args.Query)
},
    schemabuilder.FieldDesc("Search users by query string"),
    schemabuilder.ArgDeprecation("username", "Use 'query' for full-text search instead"),
)
```

#### Deprecating an Enum Value

```go
schema := schemabuilder.NewSchema()

type UserRole int32
const (
    RoleAdmin UserRole = iota
    RoleUser
    RoleGuest      // Deprecated
    RoleReadOnly   // Replacement for Guest
)

schema.Enum(UserRole(0), map[string]interface{}{
    "ADMIN":     RoleAdmin,
    "USER":      RoleUser,
    "GUEST":     RoleGuest,     // Will be deprecated
    "READ_ONLY": RoleReadOnly,
},
    schemabuilder.WithDescription("User roles in the system"),
    schemabuilder.EnumValueDeprecation("GUEST", "Use READ_ONLY instead"),
)
```

#### Deprecating Input Object Field

```go
// Already supported via struct tags
type CreateUserInput struct {
    Name     string
    Email    string
    Username string `graphql:"deprecated=Use email instead"`
}

input := schema.InputObject("CreateUserInput", CreateUserInput{})
```

---

## 8. SDL Output Examples

### 8.1 Deprecated Argument

```graphql
type Query {
  searchUsers(
    query: String
    username: String @deprecated(reason: "Use 'query' for full-text search instead")
  ): [User]
}
```

### 8.2 Deprecated Enum Value

```graphql
enum UserRole {
  ADMIN
  USER
  GUEST @deprecated(reason: "Use READ_ONLY instead")
  READ_ONLY
}
```

### 8.3 Deprecated Input Field

```graphql
input CreateUserInput {
  name: String
  email: String
  username: String @deprecated(reason: "Use email instead")
}
```

---

## 9. Introspection Examples

### 9.1 Query for Deprecated Arguments

```graphql
{
  __type(name: "Query") {
    fields {
      name
      args {
        name
        isDeprecated
        deprecationReason
      }
    }
  }
}
```

**Response:**
```json
{
  "data": {
    "__type": {
      "fields": [{
        "name": "searchUsers",
        "args": [
          {"name": "query", "isDeprecated": false, "deprecationReason": null},
          {"name": "username", "isDeprecated": true, "deprecationReason": "Use 'query' for full-text search instead"}
        ]
      }]
    }
  }
}
```

### 9.2 Query for Deprecated Enum Values

```graphql
{
  __type(name: "UserRole") {
    enumValues {
      name
      isDeprecated
      deprecationReason
    }
  }
}
```

**Response:**
```json
{
  "data": {
    "__type": {
      "enumValues": [
        {"name": "ADMIN", "isDeprecated": false, "deprecationReason": null},
        {"name": "GUEST", "isDeprecated": true, "deprecationReason": "Use READ_ONLY instead"},
        {"name": "READ_ONLY", "isDeprecated": false, "deprecationReason": null},
        {"name": "USER", "isDeprecated": false, "deprecationReason": null}
      ]
    }
  }
}
```

---

## 10. Testing Checklist

### 10.1 Unit Tests

- [ ] `schemabuilder` - ArgDeprecation option applies correctly
- [ ] `schemabuilder` - EnumValueDeprecation option applies correctly
- [ ] `schemabuilder` - Multiple argument deprecations work
- [ ] `graphql` - Argument struct stores deprecation correctly
- [ ] `graphql` - Enum stores value deprecations correctly

### 10.2 Integration Tests

- [ ] `introspection` - Deprecated args appear in __Field.args
- [ ] `introspection` - Deprecated enum values appear in __Type.enumValues
- [ ] `sdl` - Deprecated args appear in SDL output
- [ ] `sdl` - Deprecated enum values appear in SDL output
- [ ] `graphql` - Queries with deprecated args still execute correctly

### 10.3 End-to-End Tests

- [ ] Full schema build with all deprecation types
- [ ] Introspection round-trip (schema -> introspection -> SDL)
- [ ] Playground correctly displays deprecated args/values

### 10.4 Example Tests

- [ ] `example/users` - Server builds with deprecated args/values
- [ ] `example/users` - Integration tests verify introspection
- [ ] `example/users` - SDL output includes deprecations

---

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking internal API changes | Low | High | All changes are additive; internal structures changed but public API unchanged |
| Performance regression | Low | Medium | Benchmark tests before/after; deprecation checks are O(1) map lookups |
| SDL output incompatibility | Medium | Medium | Comprehensive printer tests; validate against spec examples |
| Playground rendering issues | Low | Low | Test with actual GraphQL Playground; fallback to standard introspection |

---

## 12. Future Considerations

### 12.1 Schema Directive Support
When custom directives are implemented, `@deprecated` should be migrated to use the directive framework:

```go
// Future API (Phase 2)
obj.FieldFunc("field", resolver,
    schemabuilder.Directive("deprecated", map[string]interface{}{
        "reason": "Use newField instead",
    }),
)
```

### 12.2 Deprecation by Date
Consider adding optional "deprecated as of" date:

```go
ArgDeprecationWithDate("oldArg", "Use newArg", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
```

### 12.3 Automatic Deprecation Warnings
Consider runtime warnings when deprecated args are used (configurable):

```go
schema := sb.MustBuild(WithDeprecationWarnings(true))
```

---

## 13. Appendix

### 13.1 Files to Modify

| File | Lines of Change | Description |
|------|-----------------|-------------|
| `graphql/types.go` | ~30 | Add Argument struct, update Field.Args, add ValueDeprecations to Enum |
| `schemabuilder/types.go` | ~40 | Add new option functions and config fields |
| `schemabuilder/function.go` | ~20 | Apply arg deprecations when building functions |
| `schemabuilder/schema.go` | ~10 | Update Enum to handle deprecation options |
| `schemabuilder/build.go` | ~15 | Update Args access patterns |
| `introspection/introspection.go` | ~30 | Remove stubs, use actual deprecation data |
| `sdl/printer.go` | ~25 | Print deprecation directives |
| `sdl/types.go` | ~5 | Ensure deprecation fields exist |
| `example/users/register_queries.go` | ~15 | Add example with deprecated arg |
| `example/users/register_enums.go` | ~15 | Add example with deprecated enum value |
| `example/users/server_test.go` | ~40 | Add integration tests |
| `README.md` | ~30 | Document new features |

**Total Estimated Lines:** ~300 lines of production code, ~150 lines of tests

### 13.2 Related Spec Sections

- [GraphQL Spec - @deprecated directive](https://spec.graphql.org/October2021/#sec--deprecated)
- [GraphQL Spec - Introspection - Deprecated](https://spec.graphql.org/October2021/#sec-Deprecation)
- [GraphQL Spec - Enum Value](https://spec.graphql.org/October2021/#sec-Enum-Value)
- [GraphQL Spec - Arguments](https://spec.graphql.org/October2021/#sec-Language.Arguments)

---

*End of Design Document*
