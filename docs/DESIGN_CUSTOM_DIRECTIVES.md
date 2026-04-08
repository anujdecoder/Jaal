# Design Document: Custom Directive Support

**Feature:** Custom Directives Framework  
**Status:** ✅ **IMPLEMENTED**  
**Target Version:** Jaal v1.x  
**Spec Compliance:** GraphQL October 2021+  
**Author:** AI Assistant  
**Date:** 2026-04-07  
**Implementation Date:** 2026-04-07

---

## Implementation Status

| Component | Status | Files |
|-----------|--------|-------|
| Core directive types | ✅ Complete | `graphql/directive.go` |
| Directive registration API | ✅ Complete | `schemabuilder/directive.go` |
| Field directive application | ✅ Complete | `schemabuilder/types.go`, `schemabuilder/function.go` |
| Schema directive storage | ✅ Complete | `schemabuilder/schema.go` |
| Directive execution | ✅ Complete | `graphql/execute.go` |
| Introspection support | ✅ Complete | `introspection/introspection.go` |
| SDL generation | ✅ Complete | `sdl/printer.go`, `sdl/types.go` |
| HTTP handler integration | ✅ Complete | `http.go` |
| Unit tests | ✅ Complete | `graphql/directive_test.go`, `schemabuilder/directive_test.go` |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current State Analysis](#2-current-state-analysis)
3. [Design Goals](#3-design-goals)
4. [Detailed Design](#4-detailed-design)
5. [Core Codebase Updates](#5-core-codebase-updates)
6. [Existing Tests Updates](#6-existing-tests-updates)
7. [New Tests](#7-new-tests)
8. [Example Updates](#8-example-updates)
9. [SDL Generation](#9-sdl-generation)
10. [Introspection Updates](#10-introspection-updates)
11. [Backward Compatibility](#11-backward-compatibility)
12. [Documentation Updates](#12-documentation-updates)
13. [Performance Considerations](#13-performance-considerations)
14. [Security Considerations](#14-security-considerations)
15. [Migration Guide](#15-migration-guide)
16. [Success Metrics](#16-success-metrics)
17. [Implementation Timeline](#17-implementation-timeline)
18. [Appendix](#18-appendix)

---

## 1. Executive Summary

This document details the implementation plan for adding **Custom Directive Support** to Jaal. Custom directives are a core GraphQL feature that allows developers to define their own directives like `@auth`, `@cache`, `@validate`, etc., that can modify query execution behavior.

**Implementation Status (Updated 2026-04-07):**
- ✅ Built-in directives: `@skip`, `@include`, `@deprecated`, `@specifiedBy`, `@oneOf`
- ✅ Custom directive definitions: **IMPLEMENTED**
- ✅ Custom directive execution: **IMPLEMENTED**
- ✅ Directive visitors: **IMPLEMENTED**
- ✅ Repeatable directives: **IMPLEMENTED**

**Features Implemented:**
- ✅ Define custom directives with name, description, locations, arguments
- ✅ Register directive visitors for execution-time behavior
- ✅ Support repeatable directives (Oct 2021+ spec)
- ✅ Include custom directives in introspection
- ✅ Generate SDL with custom directive definitions
- ✅ Validate directive locations

---

## 2. Current State Analysis

### 2.1 Existing Directive Support

**Built-in Directives (introspection.go):**
```go
var includeDirective = Directive{
    Name: "include",
    Locations: []DirectiveLocation{FIELD, FRAGMENT_SPREAD, INLINE_FRAGMENT},
    Args: []InputValue{{Name: "if", Type: Type{Inner: &graphql.Scalar{Type: "Boolean"}}}},
}

var skipDirective = Directive{...}
var deprecatedDirective = Directive{...}
var specifiedByDirective = Directive{...}
var oneOfDirective = Directive{...}
```

**Directive Execution (execute.go):**
```go
func shouldIncludeNode(directives []*Directive) (bool, error) {
    // Only handles @skip and @include
    skipDirective := findDirectiveWithName(directives, "skip")
    includeDirective := findDirectiveWithName(directives, "include")
    // ...
}
```

**Directive Types (graphql/types.go):**
```go
type Directive struct {
    Name string
    Args interface{}
}
```

### 2.2 Gaps Identified

| Component | Current State | Required |
|-----------|--------------|----------|
| Directive Definition | Hardcoded built-ins only | User-definable |
| Directive Locations | Enum exists in introspection | Full spec locations |
| Directive Arguments | Basic support | Full type support |
| Directive Execution | @skip/@include only | Extensible visitors |
| Repeatable Flag | Not supported | Required (Oct 2021+) |
| Schema Registration | Not available | API needed |
| Introspection | Built-ins only | Custom directives |
| SDL Output | Not supported | Full support |

---

## 3. Design Goals

### 3.1 Primary Goals

1. **Spec Compliance:** Full GraphQL October 2021 directive support
2. **Extensibility:** Easy-to-use API for defining custom directives
3. **Type Safety:** Leverage Go's type system for directive definitions
4. **Performance:** Minimal overhead for directive execution
5. **Backward Compatibility:** Zero breaking changes

### 3.2 Non-Goals

1. Schema transformation directives (advanced feature)
2. Client-side directive handling
3. Federation-specific directives (separate feature)

### 3.3 Design Principles

1. **Convention over Configuration:** Sensible defaults, override when needed
2. **Composability:** Multiple directives can be applied to same element
3. **Explicit is Better:** Clear registration and execution semantics
4. **Fail Fast:** Validation errors at schema build time

---

## 4. Detailed Design

### 4.1 Directive Definition Types

**New File: `graphql/directive.go`**

```go
package graphql

import "context"

// DirectiveLocation represents where a directive can be applied.
// Per GraphQL Oct 2021 spec, includes both executable and type system locations.
type DirectiveLocation string

const (
    // Executable directive locations (applied to queries at runtime)
    LocationQuery              DirectiveLocation = "QUERY"
    LocationMutation           DirectiveLocation = "MUTATION"
    LocationSubscription       DirectiveLocation = "SUBSCRIPTION"
    LocationField              DirectiveLocation = "FIELD"
    LocationFragmentDefinition DirectiveLocation = "FRAGMENT_DEFINITION"
    LocationFragmentSpread     DirectiveLocation = "FRAGMENT_SPREAD"
    LocationInlineFragment     DirectiveLocation = "INLINE_FRAGMENT"

    // Type system directive locations (applied to schema definitions)
    LocationSchema               DirectiveLocation = "SCHEMA"
    LocationScalar               DirectiveLocation = "SCALAR"
    LocationObject               DirectiveLocation = "OBJECT"
    LocationFieldDefinition      DirectiveLocation = "FIELD_DEFINITION"
    LocationArgumentDefinition   DirectiveLocation = "ARGUMENT_DEFINITION"
    LocationInterface            DirectiveLocation = "INTERFACE"
    LocationUnion                DirectiveLocation = "UNION"
    LocationEnum                 DirectiveLocation = "ENUM"
    LocationEnumValue            DirectiveLocation = "ENUM_VALUE"
    LocationInputObject          DirectiveLocation = "INPUT_OBJECT"
    LocationInputFieldDefinition DirectiveLocation = "INPUT_FIELD_DEFINITION"
)

// DirectiveDefinition represents a custom directive definition in the schema.
// Per GraphQL Oct 2021 spec, includes isRepeatable flag.
type DirectiveDefinition struct {
    Name          string
    Description   string
    Locations     []DirectiveLocation
    Args          map[string]*Argument
    IsRepeatable  bool
}

// DirectiveInstance represents an instance of a directive applied to a schema element.
type DirectiveInstance struct {
    Name       string
    Args       map[string]interface{}
    Definition *DirectiveDefinition // Reference to the definition
}

// DirectiveVisitor defines the interface for directive execution behavior.
// Implement this interface to add runtime behavior to directives.
type DirectiveVisitor interface {
    // VisitField is called during field resolution for FIELD_DEFINITION location directives.
    // Return (nil, nil) to continue with normal resolution.
    // Return (result, nil) to short-circuit and return the result.
    // Return (nil, error) to return an error.
    VisitField(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error)
}

// DirectiveVisitorFunc is an adapter to allow using functions as DirectiveVisitor.
type DirectiveVisitorFunc func(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error)

func (f DirectiveVisitorFunc) VisitField(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error) {
    return f(ctx, directive, field, source)
}

// MultiLocationVisitor extends DirectiveVisitor to support multiple locations.
type MultiLocationVisitor interface {
    DirectiveVisitor
    
    // VisitArgument is called for ARGUMENT_DEFINITION location directives.
    VisitArgument(ctx context.Context, directive *DirectiveInstance, argName string, value interface{}) (interface{}, error)
    
    // VisitInputField is called for INPUT_FIELD_DEFINITION location directives.
    VisitInputField(ctx context.Context, directive *DirectiveInstance, fieldName string, value interface{}) (interface{}, error)
    
    // VisitEnumValue is called for ENUM_VALUE location directives.
    VisitEnumValue(ctx context.Context, directive *DirectiveInstance, value string) error
}
```

### 4.2 Schema Builder Directive API

**New File: `schemabuilder/directive.go`**

```go
package schemabuilder

import (
    "context"
    "reflect"
    
    "go.appointy.com/jaal/graphql"
)

// DirectiveConfig holds configuration for a custom directive definition.
type DirectiveConfig struct {
    Description  string
    Locations    []graphql.DirectiveLocation
    Args         map[string]reflect.Type  // arg name -> Go type
    IsRepeatable bool
}

// DirectiveOption configures a directive during registration.
type DirectiveOption func(*directiveConfig)

type directiveConfig struct {
    description  string
    locations    []graphql.DirectiveLocation
    args         map[string]*graphql.Argument
    isRepeatable bool
    visitor      graphql.DirectiveVisitor
}

// DirectiveDescription sets the description for a directive.
func DirectiveDescription(desc string) DirectiveOption {
    return func(cfg *directiveConfig) {
        cfg.description = desc
    }
}

// DirectiveRepeatable marks a directive as repeatable (can be applied multiple times).
func DirectiveRepeatable() DirectiveOption {
    return func(cfg *directiveConfig) {
        cfg.isRepeatable = true
    }
}

// DirectiveArg adds an argument to a directive.
func DirectiveArg(name string, typ reflect.Type, opts ...ArgOption) DirectiveOption {
    return func(cfg *directiveConfig) {
        if cfg.args == nil {
            cfg.args = make(map[string]*graphql.Argument)
        }
        arg := &graphql.Argument{Type: convertType(typ)}
        for _, opt := range opts {
            opt(arg)
        }
        cfg.args[name] = arg
    }
}

// DirectiveVisitor sets the visitor for directive execution.
func DirectiveVisitor(visitor graphql.DirectiveVisitor) DirectiveOption {
    return func(cfg *directiveConfig) {
        cfg.visitor = visitor
    }
}

// DirectiveVisitorFunc creates a visitor from a function.
func DirectiveVisitorFunc(fn func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error)) DirectiveOption {
    return DirectiveVisitor(graphql.DirectiveVisitorFunc(fn))
}

// Directive registers a custom directive definition.
// Example:
//
//  schema.Directive("auth", 
//      DirectiveDescription("Requires authentication"),
//      DirectiveLocations(graphql.LocationFieldDefinition),
//      DirectiveArg("role", reflect.TypeOf("")),
//      DirectiveVisitorFunc(authVisitor),
//  )
func (s *Schema) Directive(name string, opts ...DirectiveOption) {
    cfg := &directiveConfig{}
    for _, opt := range opts {
        opt(cfg)
    }
    
    if s.directives == nil {
        s.directives = make(map[string]*directiveRegistration)
    }
    
    s.directives[name] = &directiveRegistration{
        definition: &graphql.DirectiveDefinition{
            Name:         name,
            Description:  cfg.description,
            Locations:    cfg.locations,
            Args:         cfg.args,
            IsRepeatable: cfg.isRepeatable,
        },
        visitor: cfg.visitor,
    }
}

// DirectiveLocations sets the locations where a directive can be applied.
func DirectiveLocations(locations ...graphql.DirectiveLocation) DirectiveOption {
    return func(cfg *directiveConfig) {
        cfg.locations = locations
    }
}

// FieldDirective applies a directive to a field during schema building.
func FieldDirective(name string, args map[string]interface{}) FieldOption {
    return func(cfg *fieldConfig) {
        cfg.directives = append(cfg.directives, &graphql.DirectiveInstance{
            Name: name,
            Args: args,
        })
    }
}

// directiveRegistration stores both definition and visitor.
type directiveRegistration struct {
    definition *graphql.DirectiveDefinition
    visitor    graphql.DirectiveVisitor
}
```

### 4.3 Schema Type Updates

**Update: `schemabuilder/schema.go`**

```go
type Schema struct {
    objects      map[string]*Object
    enumTypes    map[reflect.Type]*EnumMapping
    inputObjects map[string]*InputObject
    directives   map[string]*directiveRegistration  // NEW
}

// GetDirectiveDefinitions returns all registered directive definitions.
func (s *Schema) GetDirectiveDefinitions() []*graphql.DirectiveDefinition {
    var defs []*graphql.DirectiveDefinition
    for _, reg := range s.directives {
        defs = append(defs, reg.definition)
    }
    return defs
}

// GetDirectiveVisitor returns the visitor for a directive, if any.
func (s *Schema) GetDirectiveVisitor(name string) graphql.DirectiveVisitor {
    if reg, ok := s.directives[name]; ok {
        return reg.visitor
    }
    return nil
}
```

### 4.4 Field Configuration Updates

**Update: `schemabuilder/types.go`**

```go
type fieldConfig struct {
    description     string
    deprecated      string
    nonNull         bool
    argDeprecations map[string]string
    directives      []*graphql.DirectiveInstance  // NEW: directives applied to field
}

type typeConfig struct {
    description      string
    deprecated       string
    directives       []string
    enumDeprecations map[string]string
    typeDirectives   []*graphql.DirectiveInstance  // NEW: directives on type
}

// ApplyDirective applies a directive to a type.
func ApplyDirective(name string, args map[string]interface{}) TypeOption {
    return func(cfg *typeConfig) {
        cfg.typeDirectives = append(cfg.typeDirectives, &graphql.DirectiveInstance{
            Name: name,
            Args: args,
        })
    }
}
```

### 4.5 Execution Integration

**Update: `graphql/execute.go`**

```go
type Executor struct {
    iterate           bool
    directiveVisitors map[string]DirectiveVisitor  // NEW
}

// NewExecutor creates an executor with directive visitors.
func NewExecutor(directiveVisitors map[string]DirectiveVisitor) *Executor {
    return &Executor{
        directiveVisitors: directiveVisitors,
    }
}

func (e *Executor) resolveAndExecute(ctx context.Context, field *Field, source interface{}, selection *Selection) (interface{}, error) {
    // Execute type-system directives (FIELD_DEFINITION location)
    for _, directive := range field.Directives {
        if visitor, ok := e.directiveVisitors[directive.Name]; ok {
            result, err := visitor.VisitField(ctx, directive, field, source)
            if err != nil {
                return nil, err
            }
            if result != nil {
                // Directive short-circuited execution
                return e.execute(ctx, field.Type, result, selection.SelectionSet)
            }
        }
    }
    
    // Execute executable directives (FIELD location)
    for _, directive := range selection.Directives {
        if visitor, ok := e.directiveVisitors[directive.Name]; ok {
            result, err := visitor.VisitField(ctx, directive, field, source)
            if err != nil {
                return nil, err
            }
            if result != nil {
                return e.execute(ctx, field.Type, result, selection.SelectionSet)
            }
        }
    }
    
    // Continue with normal resolution
    value, err := safeExecuteResolver(ctx, field, source, selection.Args, selection.SelectionSet)
    if err != nil {
        return nil, err
    }
    
    // ... rest of existing code
}
```

### 4.6 Validation Updates

**Update: `graphql/validate.go`**

```go
// ValidateDirectiveLocations checks that directives are used in valid locations.
func ValidateDirectiveLocations(selectionSet *SelectionSet, typ Type, definitions map[string]*DirectiveDefinition) []error {
    var errors []error
    
    // Validate executable directives
    walkSelection(selectionSet, func(selection *Selection) {
        for _, directive := range selection.Directives {
            def, ok := definitions[directive.Name]
            if !ok {
                errors = append(errors, fmt.Errorf("unknown directive @%s", directive.Name))
                continue
            }
            
            if !hasLocation(def.Locations, LocationField) {
                errors = append(errors, fmt.Errorf(
                    "directive @%s cannot be used on FIELD (allowed: %v)",
                    directive.Name, def.Locations,
                ))
            }
        }
    })
    
    return errors
}

// ValidateRepeatableDirectives checks that non-repeatable directives are not duplicated.
func ValidateRepeatableDirectives(selectionSet *SelectionSet, definitions map[string]*DirectiveDefinition) []error {
    var errors []error
    
    walkSelection(selectionSet, func(selection *Selection) {
        seen := make(map[string]bool)
        for _, directive := range selection.Directives {
            def, ok := definitions[directive.Name]
            if !ok {
                continue
            }
            
            if seen[directive.Name] && !def.IsRepeatable {
                errors = append(errors, fmt.Errorf(
                    "directive @%s is not repeatable but was applied multiple times",
                    directive.Name,
                ))
            }
            seen[directive.Name] = true
        }
    })
    
    return errors
}
```

---

## 5. Core Codebase Updates

### 5.1 New Files to Create

| File | Purpose | Lines (est.) |
|------|---------|--------------|
| `graphql/directive.go` | Directive types and visitor interface | ~150 |
| `schemabuilder/directive.go` | Directive registration API | ~200 |
| `graphql/directive_test.go` | Unit tests for directive types | ~300 |
| `schemabuilder/directive_test.go` | Tests for directive registration | ~400 |

### 5.2 Files to Modify

| File | Changes | Lines (est.) |
|------|---------|--------------|
| `graphql/types.go` | Add directives to Field, Object types | ~30 |
| `graphql/execute.go` | Directive visitor execution | ~50 |
| `graphql/validate.go` | Directive validation rules | ~80 |
| `schemabuilder/schema.go` | Directive storage, Build() updates | ~40 |
| `schemabuilder/types.go` | Directive options, fieldConfig updates | ~30 |
| `schemabuilder/function.go` | Apply directives to fields | ~20 |
| `schemabuilder/build.go` | Include directives in schema | ~30 |
| `introspection/introspection.go` | Custom directives in introspection | ~50 |
| `sdl/printer.go` | SDL generation for directives | ~60 |
| `sdl/types.go` | Directive types for SDL | ~20 |
| `http.go` | Pass directive visitors to executor | ~20 |

### 5.3 Detailed Changes

#### 5.3.1 `graphql/types.go` Updates

```go
// Add to Field struct
type Field struct {
    // ... existing fields ...
    
    // Directives applied to this field (FIELD_DEFINITION location)
    Directives []*DirectiveInstance `json:"-"`
}

// Add to Object struct
type Object struct {
    // ... existing fields ...
    
    // Directives applied to this object (OBJECT location)
    Directives []*DirectiveInstance `json:"-"`
}

// Add to InputObject struct  
type InputObject struct {
    // ... existing fields ...
    
    // Directives applied to this input object (INPUT_OBJECT location)
    Directives []*DirectiveInstance `json:"-"`
}

// Add to Enum struct
type Enum struct {
    // ... existing fields ...
    
    // Directives applied to this enum (ENUM location)
    Directives []*DirectiveInstance `json:"-"`
    
    // Per-value directives (ENUM_VALUE location)
    ValueDirectives map[string][]*DirectiveInstance `json:"-"`
}

// Add to Interface struct
type Interface struct {
    // ... existing fields ...
    
    // Directives applied to this interface (INTERFACE location)
    Directives []*DirectiveInstance `json:"-"`
}
```

#### 5.3.2 `schemabuilder/function.go` Updates

```go
func (sb *schemaBuilder) buildFunctionAndFuncCtx(typ reflect.Type, m *method) (*graphql.Field, *funcContext, error) {
    // ... existing code ...
    
    return &graphql.Field{
        // ... existing fields ...
        
        // Apply directives from method config
        Directives: m.Directives,
    }, funcCtx, nil
}

// Add to method struct
type method struct {
    MarkedNonNullable bool
    Fn                interface{}
    Description       string
    DeprecationReason *string
    ArgDeprecations   map[string]string
    Directives        []*graphql.DirectiveInstance  // NEW
}
```

#### 5.3.3 `schemabuilder/build.go` Updates

```go
func (sb *schemaBuilder) buildStruct(typ reflect.Type) error {
    // ... existing code ...
    
    // Apply type-level directives
    if obj, ok := sb.objects[typ]; ok {
        for _, dir := range obj.typeDirectives {
            graphqlObj.Directives = append(graphqlObj.Directives, dir)
        }
    }
    
    // ... rest of code ...
}

// Add directive definitions to schema
func (sb *schemaBuilder) buildDirectives() map[string]*graphql.DirectiveDefinition {
    defs := make(map[string]*graphql.DirectiveDefinition)
    for name, reg := range sb.directives {
        defs[name] = reg.definition
    }
    return defs
}
```

---

## 6. Existing Tests Updates

### 6.1 Tests Requiring Updates

| Test File | Changes Needed |
|-----------|----------------|
| `graphql/endtoend_test.go` | Add directive execution tests |
| `graphql/parser_test.go` | Add directive parsing tests |
| `graphql/validate_test.go` | Add directive validation tests |
| `schemabuilder/deprecation_test.go` | Test directive + deprecation interaction |
| `introspection/introspection_test.go` | Test custom directives in introspection |
| `sdl/printer_test.go` | Test SDL output with custom directives |

### 6.2 Specific Test Updates

#### `graphql/endtoend_test.go`

Add test case:
```go
func TestCustomDirectiveExecution(t *testing.T) {
    schema := schemabuilder.NewSchema()
    
    // Register custom directive
    schema.Directive("upper",
        schemabuilder.DirectiveDescription("Converts string to uppercase"),
        schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
        schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
            // Get normal result
            result, err := f.Resolve(ctx, src, nil, nil)
            if err != nil {
                return nil, err
            }
            if str, ok := result.(string); ok {
                return strings.ToUpper(str), nil
            }
            return result, nil
        }),
    )
    
    query := schema.Query()
    query.FieldFunc("hello", func() string {
        return "hello world"
    }, schemabuilder.FieldDirective("upper", nil))
    
    builtSchema := schema.MustBuild()
    
    q, _ := graphql.Parse(`{ hello }`, nil)
    e := graphql.NewExecutor(schema.GetDirectiveVisitors())
    val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
    
    assert.NoError(t, err)
    assert.Equal(t, map[string]interface{}{"hello": "HELLO WORLD"}, internal.AsJSON(val))
}
```

#### `introspection/introspection_test.go`

Add test case:
```go
func TestCustomDirectiveIntrospection(t *testing.T) {
    schema := schemabuilder.NewSchema()
    
    schema.Directive("auth",
        schemabuilder.DirectiveDescription("Requires authentication"),
        schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
        schemabuilder.DirectiveArg("role", reflect.TypeOf("")),
    )
    
    // ... build schema ...
    
    // Query introspection
    q, _ := graphql.Parse(`{
        __schema {
            directives {
                name
                description
                locations
                args { name type { name } }
            }
        }
    }`, nil)
    
    // Execute and verify custom directive appears
}
```

---

## 7. New Tests

### 7.1 Unit Tests

**New File: `graphql/directive_test.go`**

```go
package graphql_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "go.appointy.com/jaal/graphql"
)

func TestDirectiveLocationString(t *testing.T) {
    tests := []struct {
        location graphql.DirectiveLocation
        expected string
    }{
        {graphql.LocationField, "FIELD"},
        {graphql.LocationFieldDefinition, "FIELD_DEFINITION"},
        {graphql.LocationQuery, "QUERY"},
    }
    
    for _, tt := range tests {
        assert.Equal(t, tt.expected, string(tt.location))
    }
}

func TestDirectiveDefinitionValidation(t *testing.T) {
    def := &graphql.DirectiveDefinition{
        Name:        "test",
        Locations:   []graphql.DirectiveLocation{graphql.LocationField},
        IsRepeatable: false,
    }
    
    assert.Equal(t, "test", def.Name)
    assert.False(t, def.IsRepeatable)
}

func TestDirectiveVisitorFunc(t *testing.T) {
    called := false
    visitor := graphql.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        called = true
        return "result", nil
    })
    
    result, err := visitor.VisitField(context.Background(), nil, nil, nil)
    assert.NoError(t, err)
    assert.Equal(t, "result", result)
    assert.True(t, called)
}
```

**New File: `schemabuilder/directive_test.go`**

```go
package schemabuilder

import (
    "context"
    "reflect"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "go.appointy.com/jaal/graphql"
)

func TestDirectiveRegistration(t *testing.T) {
    schema := NewSchema()
    
    schema.Directive("custom",
        DirectiveDescription("A custom directive"),
        DirectiveLocations(graphql.LocationFieldDefinition),
        DirectiveArg("param", reflect.TypeOf("")),
    )
    
    defs := schema.GetDirectiveDefinitions()
    assert.Len(t, defs, 1)
    assert.Equal(t, "custom", defs[0].Name)
    assert.Equal(t, "A custom directive", defs[0].Description)
}

func TestFieldDirectiveApplication(t *testing.T) {
    schema := NewSchema()
    
    // Register directive first
    schema.Directive("log",
        DirectiveLocations(graphql.LocationFieldDefinition),
    )
    
    // Apply to field
    query := schema.Query()
    query.FieldFunc("test", func() string { return "test" },
        FieldDirective("log", map[string]interface{}{"level": "info"}),
    )
    
    builtSchema := schema.MustBuild()
    queryObj := builtSchema.Query.(*graphql.Object)
    field := queryObj.Fields["test"]
    
    assert.Len(t, field.Directives, 1)
    assert.Equal(t, "log", field.Directives[0].Name)
}

func TestDirectiveExecution(t *testing.T) {
    schema := NewSchema()
    
    var executed bool
    schema.Directive("track",
        DirectiveLocations(graphql.LocationFieldDefinition),
        DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
            executed = true
            return nil, nil // Continue with normal resolution
        }),
    )
    
    query := schema.Query()
    query.FieldFunc("data", func() string { return "result" },
        FieldDirective("track", nil),
    )
    
    builtSchema := schema.MustBuild()
    
    q, _ := graphql.Parse(`{ data }`, nil)
    visitors := schema.GetDirectiveVisitors()
    e := graphql.NewExecutor(visitors)
    
    _, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
    assert.NoError(t, err)
    assert.True(t, executed)
}

func TestDirectiveShortCircuit(t *testing.T) {
    schema := NewSchema()
    
    schema.Directive("override",
        DirectiveLocations(graphql.LocationFieldDefinition),
        DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
            return "overridden", nil
        }),
    )
    
    query := schema.Query()
    query.FieldFunc("original", func() string { return "original" },
        FieldDirective("override", nil),
    )
    
    builtSchema := schema.MustBuild()
    
    q, _ := graphql.Parse(`{ original }`, nil)
    e := graphql.NewExecutor(schema.GetDirectiveVisitors())
    
    val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
    assert.NoError(t, err)
    
    result := val.(map[string]interface{})
    assert.Equal(t, "overridden", result["original"])
}

func TestRepeatableDirective(t *testing.T) {
    schema := NewSchema()
    
    schema.Directive("tag",
        DirectiveLocations(graphql.LocationFieldDefinition),
        DirectiveRepeatable(),
    )
    
    query := schema.Query()
    query.FieldFunc("item", func() string { return "item" },
        FieldDirective("tag", map[string]interface{}{"name": "a"}),
        FieldDirective("tag", map[string]interface{}{"name": "b"}), // Should be allowed
    )
    
    builtSchema := schema.MustBuild()
    queryObj := builtSchema.Query.(*graphql.Object)
    field := queryObj.Fields["item"]
    
    assert.Len(t, field.Directives, 2)
}

func TestNonRepeatableDirectiveError(t *testing.T) {
    schema := NewSchema()
    
    schema.Directive("single",
        DirectiveLocations(graphql.LocationFieldDefinition),
        // NOT repeatable
    )
    
    query := schema.Query()
    query.FieldFunc("item", func() string { return "item" },
        FieldDirective("single", nil),
        FieldDirective("single", nil), // Should fail validation
    )
    
    _, err := schema.Build()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not repeatable")
}
```

### 7.2 Integration Tests

**New File: `graphql/directive_integration_test.go`**

```go
package graphql_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "go.appointy.com/jaal/graphql"
    "go.appointy.com/jaal/introspection"
    "go.appointy.com/jaal/schemabuilder"
)

func TestDirectiveFullIntegration(t *testing.T) {
    // Setup schema with custom directive
    schema := schemabuilder.NewSchema()
    
    // Auth directive
    schema.Directive("auth",
        schemabuilder.DirectiveDescription("Requires authentication"),
        schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
        schemabuilder.DirectiveArg("role", reflect.TypeOf("")),
        schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
            role := d.Args["role"].(string)
            userRole := ctx.Value("role")
            if userRole == nil || userRole.(string) != role {
                return nil, fmt.Errorf("unauthorized: requires role %s", role)
            }
            return nil, nil // Continue
        }),
    )
    
    // Cache directive
    schema.Directive("cache",
        schemabuilder.DirectiveDescription("Cache field result"),
        schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
        schemabuilder.DirectiveArg("ttl", reflect.TypeOf(int(0))),
        schemabuilder.DirectiveRepeatable(),
    )
    
    // Schema types
    type User struct {
        ID   string
        Name string
    }
    
    query := schema.Query()
    query.FieldFunc("publicData", func() string { return "public" })
    query.FieldFunc("adminData", func() string { return "admin" },
        schemabuilder.FieldDirective("auth", map[string]interface{}{"role": "admin"}),
        schemabuilder.FieldDirective("cache", map[string]interface{}{"ttl": 60}),
    )
    
    userObj := schema.Object("User", User{})
    userObj.FieldFunc("id", func(u User) string { return u.ID })
    userObj.FieldFunc("name", func(u User) string { return u.Name })
    
    builtSchema := schema.MustBuild()
    introspection.AddIntrospectionToSchema(builtSchema)
    
    // Test 1: Introspection includes custom directives
    t.Run("introspection", func(t *testing.T) {
        q, _ := graphql.Parse(`{
            __schema {
                directives {
                    name
                    description
                    locations
                    isRepeatable
                    args { name }
                }
            }
        }`, nil)
        
        e := graphql.Executor{}
        val, err := e.Execute(context.Background(), builtSchema.Query, nil, q)
        assert.NoError(t, err)
        
        data := val.(map[string]interface{})["__schema"].(map[string]interface{})
        directives := data["directives"].([]interface{})
        
        var authDir, cacheDir map[string]interface{}
        for _, d := range directives {
            dir := d.(map[string]interface{})
            switch dir["name"] {
            case "auth":
                authDir = dir
            case "cache":
                cacheDir = dir
            }
        }
        
        assert.NotNil(t, authDir)
        assert.Equal(t, "Requires authentication", authDir["description"])
        
        assert.NotNil(t, cacheDir)
        assert.Equal(t, true, cacheDir["isRepeatable"])
    })
    
    // Test 2: Directive execution - authorized
    t.Run("authorized", func(t *testing.T) {
        ctx := context.WithValue(context.Background(), "role", "admin")
        q, _ := graphql.Parse(`{ adminData }`, nil)
        
        e := graphql.NewExecutor(schema.GetDirectiveVisitors())
        val, err := e.Execute(ctx, builtSchema.Query, nil, q)
        assert.NoError(t, err)
        assert.Equal(t, "admin", val.(map[string]interface{})["adminData"])
    })
    
    // Test 3: Directive execution - unauthorized
    t.Run("unauthorized", func(t *testing.T) {
        ctx := context.WithValue(context.Background(), "role", "user")
        q, _ := graphql.Parse(`{ adminData }`, nil)
        
        e := graphql.NewExecutor(schema.GetDirectiveVisitors())
        _, err := e.Execute(ctx, builtSchema.Query, nil, q)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "unauthorized")
    })
    
    // Test 4: SDL generation
    t.Run("sdl", func(t *testing.T) {
        sdl := generateSDL(builtSchema)
        assert.Contains(t, sdl, "directive @auth")
        assert.Contains(t, sdl, "directive @cache")
        assert.Contains(t, sdl, "@auth(role: \"admin\")")
    })
}
```

---

## 8. Example Updates

### 8.1 New Example File

**New File: `example/users/register_directives.go`**

```go
package users

import (
    "context"
    "fmt"
    "reflect"
    
    "go.appointy.com/jaal/graphql"
    "go.appointy.com/jaal/schemabuilder"
)

// RegisterDirectives registers custom directives for the users example.
func RegisterDirectives(sb *schemabuilder.Schema) {
    // @auth directive - requires authentication with optional role
    sb.Directive("auth",
        schemabuilder.DirectiveDescription("Requires authentication. Optionally checks for specific role."),
        schemabuilder.DirectiveLocations(
            graphql.LocationFieldDefinition,
            graphql.LocationObject,
        ),
        schemabuilder.DirectiveArg("role", reflect.TypeOf(""), schemabuilder.ArgDesc("Required role")),
        schemabuilder.DirectiveVisitorFunc(authDirectiveVisitor),
    )
    
    // @cache directive - caches field results
    sb.Directive("cache",
        schemabuilder.DirectiveDescription("Caches the field result for specified TTL."),
        schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
        schemabuilder.DirectiveArg("ttl", reflect.TypeOf(int(0)), schemabuilder.ArgDesc("Time to live in seconds")),
        schemabuilder.DirectiveRepeatable(),
        schemabuilder.DirectiveVisitorFunc(cacheDirectiveVisitor),
    )
    
    // @deprecated directive extension - already built-in, showing custom behavior
    // This demonstrates that built-in directives can have custom visitors too
}

// authDirectiveVisitor implements authentication checking
func authDirectiveVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
    // Get user from context (set by middleware)
    user, ok := ctx.Value("user").(*User)
    if !ok || user == nil {
        return nil, fmt.Errorf("unauthorized: authentication required")
    }
    
    // Check role if specified
    if role, ok := d.Args["role"].(string); ok && role != "" {
        if user.Role != role {
            return nil, fmt.Errorf("forbidden: requires role %s", role)
        }
    }
    
    // Continue with normal resolution
    return nil, nil
}

// cacheDirectiveVisitor implements caching (simplified example)
var cacheStore = make(map[string]interface{})

func cacheDirectiveVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
    // Generate cache key (simplified)
    key := fmt.Sprintf("%s:%v", f, src)
    
    // Check cache
    if cached, ok := cacheStore[key]; ok {
        return cached, nil
    }
    
    // Execute and cache
    result, err := f.Resolve(ctx, src, nil, nil)
    if err != nil {
        return nil, err
    }
    
    cacheStore[key] = result
    return nil, nil // Continue with normal resolution
}
```

### 8.2 Update Query Registration

**Update: `example/users/register_queries.go`**

```go
func RegisterQuery(sb *schemabuilder.Schema, s *Server) {
    q := sb.Query()
    
    // ... existing fields ...
    
    // Admin-only field with auth directive
    q.FieldFunc("adminStats", func(ctx context.Context) *AdminStats {
        return &AdminStats{
            TotalUsers: len(s.users),
            ActiveUsers: countActive(s.users),
        }
    }, 
        schemabuilder.FieldDesc("Admin statistics (requires ADMIN role)"),
        schemabuilder.FieldDirective("auth", map[string]interface{}{"role": "ADMIN"}),
        schemabuilder.FieldDirective("cache", map[string]interface{}{"ttl": 300}),
    )
}
```

### 8.3 Integration Test Updates

**Update: `example/users/server_test.go`**

Add tests:
```go
// 8. Custom directive tests
func TestCustomDirectives(t *testing.T) {
    h, err := users.GetGraphqlServer()
    require.NoError(t, err)
    
    server := httptest.NewServer(h)
    defer server.Close()
    
    // 8.1. Verify custom directives in introspection
    introData := postQuery(server, `{
        __schema {
            directives {
                name
                description
                locations
                isRepeatable
                args { name description }
            }
        }
    }`)
    
    directives := introData["__schema"].(map[string]interface{})["directives"].([]interface{})
    
    var authDirective, cacheDirective map[string]interface{}
    for _, d := range directives {
        dir := d.(map[string]interface{})
        switch dir["name"] {
        case "auth":
            authDirective = dir
        case "cache":
            cacheDirective = dir
        }
    }
    
    require.NotNil(t, authDirective, "auth directive should exist")
    require.Equal(t, "Requires authentication. Optionally checks for specific role.", authDirective["description"])
    
    require.NotNil(t, cacheDirective, "cache directive should exist")
    require.Equal(t, true, cacheDirective["isRepeatable"], "cache should be repeatable")
    
    // 8.2. Verify directive applied to field
    typeData := postQuery(server, `{
        __type(name: "Query") {
            fields {
                name
                directives {
                    name
                    args
                }
            }
        }
    }`)
    
    // Find adminStats field and verify directives
    fields := typeData["__type"].(map[string]interface{})["fields"].([]interface{})
    var adminStatsField map[string]interface{}
    for _, f := range fields {
        field := f.(map[string]interface{})
        if field["name"] == "adminStats" {
            adminStatsField = field
            break
        }
    }
    
    require.NotNil(t, adminStatsField)
    directives = adminStatsField["directives"].([]interface{})
    require.GreaterOrEqual(t, len(directives), 2, "adminStats should have auth and cache directives")
}
```

---

## 9. SDL Generation

### 9.1 SDL Printer Updates

**Update: `sdl/printer.go`**

```go
// Print generates the SDL string for the schema.
func (p *Printer) Print() string {
    var sb strings.Builder
    
    // Print schema definition
    p.printSchemaDefinition(&sb)
    
    // NEW: Print directive definitions first
    for _, d := range p.schema.Directives {
        if !p.isBuiltInDirective(d.Name) {
            sb.WriteString(p.printDirective(d))
        }
    }
    
    // ... rest of existing code ...
}

// printDirective prints a directive definition.
func (p *Printer) printDirective(d Directive) string {
    var sb strings.Builder
    
    sb.WriteString(p.formatDescription(d.Description, ""))
    sb.WriteString("directive @" + d.Name)
    
    // Print arguments if any
    if len(d.Args) > 0 {
        sb.WriteString("(")
        for i, arg := range d.Args {
            if i > 0 {
                sb.WriteString(", ")
            }
            sb.WriteString(arg.Name + ": " + p.printTypeRef(arg.Type))
        }
        sb.WriteString(")")
    }
    
    // Print locations
    sb.WriteString(" on ")
    for i, loc := range d.Locations {
        if i > 0 {
            sb.WriteString(" | ")
        }
        sb.WriteString(string(loc))
    }
    
    // Print repeatable if applicable
    if d.IsRepeatable {
        sb.WriteString(" repeatable")
    }
    
    sb.WriteString("\n\n")
    return sb.String()
}

// isBuiltInDirective checks if a directive is a built-in.
func (p *Printer) isBuiltInDirective(name string) bool {
    builtins := map[string]bool{
        "skip":         true,
        "include":      true,
        "deprecated":   true,
        "specifiedBy":  true,
        "oneOf":        true,
    }
    return builtins[name]
}

// printField updated to include directives
func (p *Printer) printField(f Field, indent string) string {
    var sb strings.Builder
    sb.WriteString(p.formatDescription(f.Description, indent))
    sb.WriteString(indent + f.Name)
    
    // Print arguments
    if len(f.Args) > 0 {
        sb.WriteString("(")
        for i, arg := range f.Args {
            if i > 0 {
                sb.WriteString(", ")
            }
            sb.WriteString(p.printArgument(arg))
        }
        sb.WriteString(")")
    }
    
    sb.WriteString(": " + p.printTypeRef(f.Type))
    
    // Print deprecation (built-in)
    if f.IsDeprecated {
        reason := ""
        if f.DeprecationReason != nil {
            reason = *f.DeprecationReason
        }
        sb.WriteString(" @deprecated(reason: \"" + reason + "\")")
    }
    
    // NEW: Print custom directives
    for _, d := range f.Directives {
        sb.WriteString(" " + p.printDirectiveApplication(d))
    }
    
    sb.WriteString("\n")
    return sb.String()
}

// printDirectiveApplication prints a directive application (usage).
func (p *Printer) printDirectiveApplication(d DirectiveApplication) string {
    var sb strings.Builder
    sb.WriteString("@" + d.Name)
    
    if len(d.Args) > 0 {
        sb.WriteString("(")
        first := true
        for name, value := range d.Args {
            if !first {
                sb.WriteString(", ")
            }
            sb.WriteString(name + ": " + p.printValue(value))
            first = false
        }
        sb.WriteString(")")
    }
    
    return sb.String()
}
```

### 9.2 SDL Types Updates

**Update: `sdl/types.go`**

```go
// Directive represents a directive definition in SDL.
type Directive struct {
    Name          string              `json:"name"`
    Description   string              `json:"description"`
    Locations     []DirectiveLocation `json:"locations"`
    Args          []InputValue        `json:"args"`
    IsRepeatable  bool                `json:"isRepeatable"`
}

// DirectiveApplication represents a directive applied to an element.
type DirectiveApplication struct {
    Name string                 `json:"name"`
    Args map[string]interface{} `json:"args"`
}

// Add to Field struct
type Field struct {
    Name              string                   `json:"name"`
    Description       string                   `json:"description"`
    Args              []InputValue             `json:"args"`
    Type              TypeRef                  `json:"type"`
    IsDeprecated      bool                     `json:"isDeprecated"`
    DeprecationReason *string                  `json:"deprecationReason"`
    Directives        []DirectiveApplication   `json:"directives"` // NEW
}

// Add to FullType struct
type FullType struct {
    Kind           TypeKind               `json:"kind"`
    Name           string                 `json:"name"`
    Description    string                 `json:"description"`
    Fields         []Field                `json:"fields"`
    InputFields    []InputValue           `json:"inputFields"`
    Interfaces     []TypeRef              `json:"interfaces"`
    EnumValues     []EnumValue            `json:"enumValues"`
    PossibleTypes  []TypeRef              `json:"possibleTypes"`
    SpecifiedByURL *string                `json:"specifiedByURL"`
    Directives     []DirectiveApplication `json:"directives"` // NEW
}
```

---

## 10. Introspection Updates

### 10.1 Introspection Types Updates

**Update: `introspection/introspection.go`**

```go
// Update Directive struct
type Directive struct {
    Name          string              `json:"name"`
    Description   string              `json:"description"`
    Locations     []DirectiveLocation `json:"locations"`
    Args          []InputValue        `json:"args"`
    IsRepeatable  bool                `json:"isRepeatable"` // NEW
}

// Update registerDirective to include isRepeatable
func (s *introspection) registerDirective(schema *schemabuilder.Schema) {
    obj := schema.Object("__Directive", Directive{}, 
        schemabuilder.WithDescription("A directive supported by the schema."))
    
    obj.FieldFunc("name", func(in Directive) string {
        return in.Name
    })
    obj.FieldFunc("description", func(in Directive) string {
        return in.Description
    })
    obj.FieldFunc("locations", func(in Directive) []DirectiveLocation {
        return in.Locations
    })
    obj.FieldFunc("args", func(in Directive, args struct {
        IncludeDeprecated *bool
    }) []InputValue {
        return in.Args
    })
    
    // NEW: isRepeatable field
    obj.FieldFunc("isRepeatable", func(in Directive) bool {
        return in.IsRepeatable
    }, schemabuilder.FieldDesc("Whether this directive can be applied multiple times."))
}

// Update introspection struct
type introspection struct {
    types           map[string]graphql.Type
    query           graphql.Type
    mutation        graphql.Type
    subscription    graphql.Type
    customDirectives []*graphql.DirectiveDefinition  // NEW
}

// Update __schema resolver
func (s *introspection) registerQuery(schema *schemabuilder.Schema) {
    object := schema.Query()
    
    object.FieldFunc("__schema", func() *Schema {
        // ... existing code ...
        
        // Build directives list (built-ins + custom)
        directives := []Directive{
            includeDirective,
            skipDirective,
            specifiedByDirective,
            deprecatedDirective,
            oneOfDirective,
        }
        
        // Add custom directives
        for _, def := range s.customDirectives {
            directives = append(directives, Directive{
                Name:         def.Name,
                Description:  def.Description,
                Locations:    convertLocations(def.Locations),
                Args:         convertArgs(def.Args),
                IsRepeatable: def.IsRepeatable,
            })
        }
        
        return &Schema{
            Types:            types,
            QueryType:        &Type{Inner: s.query},
            MutationType:     &Type{Inner: s.mutation},
            SubscriptionType: &Type{Inner: s.subscription},
            Directives:       directives,
        }
    })
}

// Update AddIntrospectionToSchema
func AddIntrospectionToSchema(schema *graphql.Schema) {
    AddIntrospectionToSchemaWithDirectives(schema, nil)
}

// AddIntrospectionToSchemaWithDirectives adds introspection with custom directives.
func AddIntrospectionToSchemaWithDirectives(schema *graphql.Schema, customDirectives []*graphql.DirectiveDefinition) {
    types := make(map[string]graphql.Type)
    collectTypes(schema.Query, types)
    collectTypes(schema.Mutation, types)
    collectTypes(schema.Subscription, types)
    
    is := &introspection{
        types:            types,
        query:            schema.Query,
        mutation:         schema.Mutation,
        subscription:     schema.Subscription,
        customDirectives: customDirectives,
    }
    
    // ... rest of existing code ...
}
```

---

## 11. Backward Compatibility

### 11.1 Breaking Changes

**NONE** - All changes are additive.

### 11.2 Compatibility Guarantees

| Aspect | Guarantee |
|--------|-----------|
| Existing APIs | No changes to existing function signatures |
| Schema Building | Schemas without custom directives build identically |
| Execution | Queries without custom directives execute identically |
| Introspection | Existing introspection queries work unchanged |
| SDL Output | Existing SDL generation works unchanged |

### 11.3 Migration Path

**Existing Code (continues to work):**
```go
schema := schemabuilder.NewSchema()
query := schema.Query()
query.FieldFunc("hello", func() string { return "hello" })
builtSchema := schema.MustBuild()
```

**New Code (opt-in):**
```go
schema := schemabuilder.NewSchema()

// Register custom directive (optional)
schema.Directive("auth", ...)

query := schema.Query()
query.FieldFunc("hello", func() string { return "hello" })
query.FieldFunc("admin", func() string { return "admin" },
    schemabuilder.FieldDirective("auth", map[string]interface{}{"role": "admin"}))

builtSchema := schema.MustBuild()
```

### 11.4 Internal API Changes

The following internal changes are **not** part of the public API:

| Change | Impact |
|--------|--------|
| `Field.Directives` field | Internal only, not exposed |
| `Object.Directives` field | Internal only, not exposed |
| `Executor` struct changes | Internal, created via constructor |
| `Schema.directives` map | Internal, accessed via methods |

---

## 12. Documentation Updates

### 12.1 README.md Updates

Add section after "OneOf Input Objects":

```markdown
## Custom Directives

Jaal supports custom directives that can modify query execution behavior. Define directives with the `Directive` method and apply them to fields using `FieldDirective`.

### Defining a Custom Directive

```go
schema := schemabuilder.NewSchema()

// Define an authentication directive
schema.Directive("auth",
    schemabuilder.DirectiveDescription("Requires authentication"),
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArg("role", reflect.TypeOf("")),
    schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        role := d.Args["role"].(string)
        userRole := ctx.Value("role")
        if userRole == nil || userRole.(string) != role {
            return nil, fmt.Errorf("unauthorized: requires role %s", role)
        }
        return nil, nil // Continue with normal resolution
    }),
)
```

### Applying Directives to Fields

```go
query := schema.Query()
query.FieldFunc("adminData", func() string {
    return "sensitive data"
}, 
    schemabuilder.FieldDesc("Admin-only data"),
    schemabuilder.FieldDirective("auth", map[string]interface{}{"role": "admin"}),
)
```

### Repeatable Directives

Directives can be marked as repeatable to allow multiple applications:

```go
schema.Directive("tag",
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveRepeatable(),
)

query.FieldFunc("item", resolver,
    schemabuilder.FieldDirective("tag", map[string]interface{}{"name": "featured"}),
    schemabuilder.FieldDirective("tag", map[string]interface{}{"name": "sale"}),
)
```

### Directive Locations

Custom directives can be applied to various schema locations:

- `FIELD_DEFINITION` - Object fields
- `ARGUMENT_DEFINITION` - Field arguments
- `INPUT_FIELD_DEFINITION` - Input object fields
- `OBJECT` - Object types
- `INTERFACE` - Interface types
- `ENUM` - Enum types
- `ENUM_VALUE` - Enum values
- `INPUT_OBJECT` - Input object types
- `SCALAR` - Scalar types
- `FIELD` - Query fields (runtime)
- `FRAGMENT_SPREAD` - Fragment spreads
- `INLINE_FRAGMENT` - Inline fragments

### Built-in Directives

Jaal includes these built-in directives:

- `@skip(if: Boolean!)` - Skip field when condition is true
- `@include(if: Boolean!)` - Include field when condition is true
- `@deprecated(reason: String)` - Mark as deprecated
- `@specifiedBy(url: String!)` - Specify scalar behavior URL
- `@oneOf` - Mark input object as oneOf
```

### 12.2 API Documentation

Add GoDoc comments to all new exported types and functions.

### 12.3 Examples Documentation

Create `docs/DIRECTIVES.md` with comprehensive examples:

- Authentication directive
- Caching directive
- Validation directive
- Logging directive
- Rate limiting directive
- Custom error handling directive

---

## 13. Performance Considerations

### 13.1 Execution Overhead

| Scenario | Overhead |
|----------|----------|
| No custom directives | 0% (no change) |
| Custom directive, not applied | < 1% |
| Custom directive applied, visitor returns nil | < 2% |
| Custom directive applied, visitor processes | Depends on visitor |

### 13.2 Memory Overhead

| Component | Overhead |
|-----------|----------|
| Directive definition | ~200 bytes per directive |
| Directive instance on field | ~50 bytes per application |
| Visitor registration | ~100 bytes per visitor |

### 13.3 Optimization Strategies

1. **Lazy Visitor Lookup:** Store visitors in map, lookup only when directive present
2. **Directive Caching:** Cache parsed directive arguments
3. **Skip Empty Directives:** Fast path when no directives on field/selection

```go
func (e *Executor) resolveAndExecute(...) (interface{}, error) {
    // Fast path: no directives
    if len(field.Directives) == 0 && len(selection.Directives) == 0 {
        return safeExecuteResolver(ctx, field, source, selection.Args, selection.SelectionSet)
    }
    
    // Slow path: process directives
    // ...
}
```

---

## 14. Security Considerations

### 14.1 Directive Validation

- Validate directive locations at schema build time
- Validate directive arguments at schema build time
- Validate repeatable semantics at schema build time

### 14.2 Execution Safety

- Visitor functions run in same context as resolver
- Panics in visitors are recovered and converted to errors
- Directive execution respects context cancellation

### 14.3 Security Best Practices

1. **Input Validation:** Validate directive arguments before use
2. **Error Messages:** Don't leak sensitive information in errors
3. **Rate Limiting:** Consider rate limiting directive execution
4. **Audit Logging:** Log directive execution for security-sensitive directives

---

## 15. Migration Guide

### 15.1 For Framework Users

No migration needed - custom directives are opt-in.

### 15.2 For Framework Contributors

When adding new built-in directives:

1. Define in `introspection/introspection.go`
2. Add visitor if runtime behavior needed
3. Update SDL printer
4. Add tests
5. Update documentation

---

## 16. Success Metrics

### 16.1 Functional Metrics

| Metric | Target | Verification |
|--------|--------|--------------|
| Custom directives can be defined | ✅ | Unit test |
| Directives appear in introspection | ✅ | Integration test |
| SDL output includes directive definitions | ✅ | SDL test |
| Directive visitors execute correctly | ✅ | Execution test |
| Multiple directives execute in order | ✅ | Order test |
| Repeatable directives work | ✅ | Repeatable test |
| Non-repeatable directives rejected when duplicated | ✅ | Validation test |
| Directive location validation works | ✅ | Validation test |

### 16.2 Performance Metrics

| Metric | Target |
|--------|--------|
| Directive registration overhead | < 1ms per directive |
| Directive lookup overhead | < 0.1ms |
| Execution overhead (no directives) | 0% |
| Execution overhead (with directives) | < 5% |
| Memory overhead per directive | < 500 bytes |

### 16.3 Quality Metrics

| Metric | Target |
|--------|--------|
| Test coverage | > 90% |
| Documentation coverage | 100% of public API |
| Backward compatibility | 100% |
| Spec compliance | 100% for directive features |

---

## 17. Implementation Timeline

### Phase 1: Core Infrastructure (Week 1-2) ✅ COMPLETE

- [x] Create `graphql/directive.go` with types
- [x] Create `schemabuilder/directive.go` with registration API
- [x] Update `schemabuilder/types.go` with directive options
- [x] Update `schemabuilder/schema.go` with directive storage
- [x] Write unit tests for core types

### Phase 2: Execution Integration (Week 3) ✅ COMPLETE

- [x] Update `graphql/execute.go` for directive visitors
- [x] Update `graphql/validate.go` for directive validation
- [x] Write execution tests
- [x] Write validation tests

### Phase 3: Introspection & SDL (Week 4) ✅ COMPLETE

- [x] Update `introspection/introspection.go`
- [x] Update `sdl/printer.go`
- [x] Update `sdl/types.go`
- [x] Write introspection tests
- [x] Write SDL tests

### Phase 4: Examples & Documentation (Week 5) ✅ COMPLETE

- [x] Create `example/users/register_directives.go`
- [x] Update example tests
- [x] Update README.md
- [x] Create `docs/DIRECTIVES.md`
- [x] Add GoDoc comments

### Phase 5: Polish & Review (Week 6) ✅ COMPLETE

- [x] Performance optimization (directive lookup is O(1), no overhead without directives)
- [x] Security review (directive visitors run in same context as resolvers)
- [x] Documentation review
- [x] Final testing
- [x] Update phase completion status

---

## 18. Appendix

### 18.1 GraphQL Spec References

- [GraphQL October 2021 - Directives](https://spec.graphql.org/October2021/#sec-Type-System.Directives)
- [GraphQL October 2021 - Directive Locations](https://spec.graphql.org/October2021/#DirectiveLocation)
- [GraphQL October 2021 - Repeatable Directives](https://spec.graphql.org/October2021/#sec-Type-System.Directives)

### 18.2 Related Issues

- Phase 2.1 in GRAPHQL_SPEC_COMPLIANCE_ROADMAP.md
- Feature 1 in FEATURE_COMPARISON.md

### 18.3 Files Changed Summary

| Category | Files | Lines Changed |
|----------|-------|---------------|
| New files | 4 | ~1,200 |
| Modified files | 12 | ~500 |
| Test files | 6 | ~800 |
| Documentation | 3 | ~300 |
| **Total** | **25** | **~2,800** |

### 18.4 Example Usage Patterns

#### Authentication Directive

```go
schema.Directive("auth",
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArg("role", reflect.TypeOf("")),
    schemabuilder.DirectiveVisitorFunc(authVisitor),
)

func authVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
    user := ctx.Value("user")
    if user == nil {
        return nil, errors.New("unauthorized")
    }
    if role := d.Args["role"]; role != "" {
        if !hasRole(user, role.(string)) {
            return nil, errors.New("forbidden")
        }
    }
    return nil, nil
}
```

#### Caching Directive

```go
schema.Directive("cache",
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArg("ttl", reflect.TypeOf(int(0))),
    schemabuilder.DirectiveRepeatable(),
    schemabuilder.DirectiveVisitorFunc(cacheVisitor),
)

func cacheVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
    key := generateCacheKey(f, src)
    if cached := cache.Get(key); cached != nil {
        return cached, nil
    }
    result, err := f.Resolve(ctx, src, nil, nil)
    if err != nil {
        return nil, err
    }
    ttl := d.Args["ttl"].(int)
    cache.Set(key, result, time.Duration(ttl)*time.Second)
    return nil, nil
}
```

#### Validation Directive

```go
schema.Directive("validate",
    schemabuilder.DirectiveLocations(graphql.LocationArgumentDefinition, graphql.LocationInputFieldDefinition),
    schemabuilder.DirectiveArg("min", reflect.TypeOf(int(0))),
    schemabuilder.DirectiveArg("max", reflect.TypeOf(int(0))),
    schemabuilder.DirectiveVisitorFunc(validateVisitor),
)
```

---

*End of Design Document*
