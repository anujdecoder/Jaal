# GraphQL Spec Compliance Roadmap for Jaal

**Document Version:** 1.0  
**Last Updated:** 2026-04-05  
**Target Spec:** GraphQL October 2021+ (with September 2025 preview features)

---

## Executive Summary

This document outlines a prioritized implementation plan to achieve full GraphQL specification compliance for the Jaal framework. The framework currently has **~90% compliance** with the October 2021 spec, with the majority of gaps in custom directive support and advanced validation features.

**Current Status:**
- ✅ Core execution engine: Complete
- ✅ Type system (scalars, objects, interfaces, unions): Complete  
- ✅ Introspection: Complete
- ✅ Built-in directives (@skip, @include, @deprecated, @specifiedBy, @oneOf): Complete
- ❌ Custom directives: Not implemented
- ⚠️ Advanced validation: Partial

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-3)
**Goal:** Fix stubbed features and complete partial implementations

#### 1.1 Complete Argument Deprecation Support
**Priority:** P0 (Critical)  
**Complexity:** Low  
**Files:** `introspection/introspection.go`, `schemabuilder/function.go`

**Description:**
Complete the stubbed implementation for `@deprecated` on field arguments (ARGUMENT_DEFINITION) and input fields (INPUT_FIELD_DEFINITION). The introspection fields exist but always return false/nil.

**Implementation Steps:**
1. Extend `graphql.Field.Args` to support deprecation metadata
2. Update `buildFunction` in `schemabuilder/function.go` to capture argument deprecation
3. Modify introspection to read from actual field arg metadata instead of stubs
4. Add `Deprecated()` field option for argument registration

**Usage Example:**
```go
// Field with deprecated argument
query.FieldFunc("user", func(ctx context.Context, args struct {
    ID   *schemabuilder.ID
    Name *string  // Deprecated: use ID instead
}) *User {
    // resolver
}, schemabuilder.FieldDesc("Get user by ID"))

// Marking argument as deprecated (new API needed)
// Option A: Tag-based
// type UserArgs struct {
//     ID   *schemabuilder.ID
//     Name *string `graphql:"deprecated=Use ID instead"`
// }

// Option B: Registration-based
// obj.FieldFuncWithArgs("user", resolver, ArgDeprecated("name", "Use ID instead"))
```

**Acceptance Criteria:**
- [ ] Arguments show `isDeprecated: true` in introspection when marked
- [ ] Arguments show `deprecationReason` in introspection
- [ ] SDL output includes `@deprecated` on arguments
- [ ] Playground shows deprecated arguments with strikethrough

---

#### 1.2 Enum Value Deprecation
**Priority:** P0 (Critical)  
**Complexity:** Low  
**Files:** `schemabuilder/schema.go`, `graphql/types.go`, `introspection/introspection.go`

**Description:**
Implement `@deprecated` support on individual enum values (ENUM_VALUE directive location). Currently stubbed in introspection.

**Implementation Steps:**
1. Extend `EnumMapping` to include per-value deprecation info
2. Update `schema.Enum()` to accept optional deprecation map
3. Modify introspection `enumValues` resolver to return deprecation data
4. Update SDL printer to output deprecated enum values

**Usage Example:**
```go
// Current API
schema.Enum(Status(0), map[string]interface{}{
    "ACTIVE":   Status(0),
    "INACTIVE": Status(1),  // Deprecated
})

// Enhanced API with deprecation
schema.Enum(Status(0), map[string]interface{}{
    "ACTIVE":   Status(0),
    "INACTIVE": Status(1),
}, schemabuilder.EnumValueDeprecated("INACTIVE", "Use ACTIVE instead"))

// Or with descriptions and deprecation
schema.EnumWithConfig(Status(0), []EnumValueConfig{
    {Name: "ACTIVE", Value: Status(0)},
    {Name: "INACTIVE", Value: Status(1), Deprecated: "Use ACTIVE instead"},
})
```

**Acceptance Criteria:**
- [ ] Enum values can be marked as deprecated
- [ ] Introspection shows `isDeprecated` and `deprecationReason` for enum values
- [ ] SDL includes `@deprecated` on enum values
- [ ] Playground renders deprecated enum values correctly

---

### Phase 2: Custom Directives Framework (Weeks 4-8)
**Goal:** Implement the ability to define and use custom directives

#### 2.1 Directive Definition Schema
**Priority:** P1 (High)  
**Complexity:** High  
**Files:** New: `schemabuilder/directive.go`, `graphql/directive.go`

**Description:**
Implement the core infrastructure for defining custom directives, including directive types, locations, and arguments.

**Implementation Steps:**
1. Create `Directive` type in `graphql/directive.go`:
   ```go
   type DirectiveDefinition struct {
       Name        string
       Description string
       Locations   []DirectiveLocation
       Args        map[string]Type
       IsRepeatable bool  // 2021+ feature
   }
   ```
2. Define all directive locations from spec:
   - QUERY, MUTATION, SUBSCRIPTION, FIELD, FRAGMENT_DEFINITION, FRAGMENT_SPREAD, INLINE_FRAGMENT
   - SCHEMA, SCALAR, OBJECT, FIELD_DEFINITION, ARGUMENT_DEFINITION, INTERFACE, UNION, ENUM, ENUM_VALUE, INPUT_OBJECT, INPUT_FIELD_DEFINITION
3. Add directive registration API to `schemabuilder.Schema`
4. Include directives in introspection (`__Schema.directives`)

**Usage Example:**
```go
// Define a custom directive
sb.Directive("auth", DirectiveConfig{
    Description: "Marks fields requiring authentication",
    Locations:   []DirectiveLocation{FIELD_DEFINITION},
    Args: map[string]graphql.Type{
        "role": &graphql.NonNull{Type: &graphql.Scalar{Type: "String"}},
    },
})

// Use in schema
obj.FieldFunc("adminData", resolver, schemabuilder.Directive("auth", map[string]interface{}{
    "role": "ADMIN",
}))
```

**Acceptance Criteria:**
- [ ] Custom directives can be defined with name, description, locations, args
- [ ] Directives appear in introspection
- [ ] SDL output includes directive definitions
- [ ] Validation rejects directives used in wrong locations

---

#### 2.2 Directive Execution Framework
**Priority:** P1 (High)  
**Complexity:** High  
**Files:** `graphql/execute.go`, `schemabuilder/directive.go`

**Description:**
Implement directive execution hooks that allow directives to transform behavior during query execution.

**Implementation Steps:**
1. Define directive visitor interface:
   ```go
   type DirectiveVisitor interface {
       VisitField(ctx context.Context, directive *Directive, field *Field, source interface{}) (interface{}, error)
       VisitArgument(ctx context.Context, directive *Directive, arg *Argument, value interface{}) (interface{}, error)
       // ... other locations
   }
   ```
2. Modify executor to check and invoke directive visitors
3. Support directive composition (multiple directives on same element)
4. Handle directive arguments during execution

**Usage Example:**
```go
// Implement directive behavior
sb.RegisterDirectiveImplementation("auth", &AuthDirectiveVisitor{})

type AuthDirectiveVisitor struct{}

func (v *AuthDirectiveVisitor) VisitField(ctx context.Context, d *graphql.Directive, f *graphql.Field, src interface{}) (interface{}, error) {
    role := d.Args["role"].(string)
    if !hasRole(ctx, role) {
        return nil, fmt.Errorf("unauthorized: requires role %s", role)
    }
    return f.Resolve(ctx, src, nil, nil)
}
```

**Acceptance Criteria:**
- [ ] Directives can intercept field resolution
- [ ] Directives can transform argument values
- [ ] Multiple directives execute in defined order
- [ ] Directive errors propagate correctly

---

#### 2.3 Repeatable Directives
**Priority:** P1 (High)  
**Complexity:** Medium  
**Files:** `graphql/directive.go`, `graphql/parser.go`

**Description:**
Support the `isRepeatable: true` flag on directive definitions (Oct 2021+ feature).

**Implementation Steps:**
1. Add `IsRepeatable` field to `DirectiveDefinition`
2. Modify parser to allow multiple instances of repeatable directives
3. Update validation to reject duplicate non-repeatable directives
4. Ensure introspection includes `isRepeatable`

**Usage Example:**
```go
// Define repeatable directive
sb.Directive("cache", DirectiveConfig{
    Description:  "Caching configuration",
    Locations:    []DirectiveLocation{FIELD_DEFINITION},
    IsRepeatable: true,  // Can be used multiple times
    Args: map[string]graphql.Type{
        "ttl": &graphql.Scalar{Type: "Int"},
    },
})

// Usage
obj.FieldFunc("data", resolver,
    schemabuilder.Directive("cache", map[string]interface{}{"ttl": 60}),
    schemabuilder.Directive("cache", map[string]interface{}{"ttl": 300}), // Different config
)
```

**Acceptance Criteria:**
- [ ] Repeatable directives can be applied multiple times
- [ ] Non-repeatable directives rejected when duplicated
- [ ] Introspection shows `isRepeatable: true`

---

### Phase 3: Advanced Type System (Weeks 9-11)
**Goal:** Complete type system features for complex schemas

#### 3.1 Interface Implements Interface
**Priority:** P2 (Medium)  
**Complexity:** Medium  
**Files:** `schemabuilder/output.go`, `graphql/types.go`

**Description:**
Allow GraphQL interfaces to implement other interfaces, creating interface hierarchies.

**Implementation Steps:**
1. Extend `graphql.Interface` to include `Interfaces` field
2. Modify `buildInterfaceStruct` to handle interface inheritance
3. Update validation to check interface compatibility
4. Ensure introspection shows implemented interfaces

**Usage Example:**
```go
// Interface hierarchy
type Node struct {
    schemabuilder.Interface
    ID string
}

type User struct {
    schemabuilder.Interface
    Node  // User implements Node (inherits ID field)
    Name string
}

// Or explicit implementation
sb.Interface("User", User{}, Implements("Node"))
```

**Acceptance Criteria:**
- [ ] Interfaces can declare they implement other interfaces
- [ ] Field validation ensures implementing types have all required fields
- [ ] Introspection shows `interfaces` on interface types
- [ ] SDL shows `interface User implements Node`

---

#### 3.2 Default Values for Input Object Fields
**Priority:** P2 (Medium)  
**Complexity:** Medium  
**Files:** `schemabuilder/input_object.go`, `graphql/parser.go`

**Description:**
Support default values for fields within Input Object types (distinct from variable defaults).

**Implementation Steps:**
1. Extend `graphql.InputObject` to store default values per field
2. Modify input coercion to apply defaults when fields are omitted
3. Update introspection to expose `defaultValue` for input fields
4. Support complex default values (objects, lists)

**Usage Example:**
```go
// Input with defaults
type CreateUserInput struct {
    Name string
    Role string  // Default: "USER"
    Active bool // Default: true
}

input := sb.InputObject("CreateUserInput", CreateUserInput{},
    schemabuilder.WithDescription("Input for creating users"))
input.FieldFunc("role", func(target *CreateUserInput, source string) {
    target.Role = source
}, schemabuilder.DefaultValue("USER"))  // New option

// Or struct tag based
type CreateUserInput struct {
    Name   string
    Role   string `graphql:"default=USER"`
    Active bool   `graphql:"default=true"`
}
```

**Acceptance Criteria:**
- [ ] Input fields can have default values
- [ ] Defaults applied when field not provided in query
- [ ] Defaults appear in introspection
- [ ] SDL shows default values
- [ ] Complex nested defaults work correctly

---

### Phase 4: Validation & Performance (Weeks 12-15)
**Goal:** Add production-ready validation and performance features

#### 4.1 Query Complexity Analysis
**Priority:** P1 (High)  
**Complexity:** High  
**Files:** New: `graphql/complexity.go`, `graphql/validate.go`

**Description:**
Implement query complexity analysis to prevent expensive queries from hitting the server.

**Implementation Steps:**
1. Define complexity calculation rules:
   - Scalar fields: cost 1
   - Object fields: cost 1 + sum(child fields)
   - Connections: cost 1 + (first/last * child cost)
   - Custom complexity functions per field
2. Add complexity limits to schema configuration
3. Implement pre-execution complexity checker
4. Return error if complexity exceeds threshold

**Usage Example:**
```go
// Schema-level complexity limit
schema := sb.MustBuild(WithMaxComplexity(1000))

// Field-level custom complexity
obj.FieldFunc("friends", resolver, 
    schemabuilder.Complexity(func(childComplexity int) int {
        return 10 + childComplexity  // Base cost 10 plus children
    }),
)

// Query with complexity check
query {
    user {           // cost: 1
        friends(first: 100) {  // cost: 10 + (100 * 1) = 110
            name     // cost: 1 each
        }
    }
}
```

**Acceptance Criteria:**
- [ ] Complexity calculated for all queries
- [ ] Configurable max complexity per schema
- [ ] Custom complexity functions supported
- [ ] Meaningful error messages for exceeded complexity
- [ ] Complexity reported in extensions (optional)

---

#### 4.2 Persisted Queries
**Priority:** P2 (Medium)  
**Complexity:** Medium  
**Files:** New: `graphql/persisted.go`, `http.go`

**Description:**
Support Automatic Persisted Queries (APQ) for performance - send query hash instead of full query text.

**Implementation Steps:**
1. Implement query hash calculation (SHA-256)
2. Add cache layer for parsed queries
3. Support `extensions.persistedQuery` in request
4. Handle cache miss (return error with expected hash)
5. Add persisted query registration endpoint

**Usage Example:**
```go
// Client sends hash instead of query
{
    "query": "...",  // Optional if persisted
    "extensions": {
        "persistedQuery": {
            "version": 1,
            "sha256Hash": "ecf4edb46db40b5132295c0291d62fb65d6759a9eedfa4d5d612dd5ec54a6b38"
        }
    }
}

// Server configuration
handler := jaal.HTTPHandler(schema, 
    jaal.WithPersistedQueryCache(redisClient),
    jaal.WithAutoPersistedQueries(true),
)
```

**Acceptance Criteria:**
- [ ] Queries can be referenced by hash
- [ ] Cache hit returns parsed query
- [ ] Cache miss returns proper error
- [ ] Works with existing query cache

---

#### 4.3 Enhanced Validation Rules
**Priority:** P2 (Medium)  
**Complexity:** High  
**Files:** `graphql/validate.go`

**Description:**
Complete implementation of all GraphQL validation rules from the spec.

**Missing Validations:**
1. Fields on correct type
2. Fragments on composite types
3. Leaf field selections
4. Argument names
5. Argument uniqueness
6. Required arguments
7. Directives in allowed locations
8. Variable usage allowed
9. Variable usage in allowed location
10. All variable usages are defined

**Implementation Steps:**
1. Implement each validation rule as separate function
2. Create validation context for error accumulation
3. Run all validations before execution
4. Return all validation errors (not just first)

**Acceptance Criteria:**
- [ ] All spec validation rules implemented
- [ ] Multiple validation errors returned together
- [ ] Validation errors include locations (line/column)
- [ ] Validation is extensible (custom rules)

---

### Phase 5: Incremental Delivery (Weeks 16-18)
**Goal:** Implement @defer and @stream directives

#### 5.1 @defer Directive
**Priority:** P3 (Low)  
**Complexity:** Very High  
**Files:** New: `graphql/defer.go`, `graphql/execute.go`

**Description:**
Implement `@defer` directive for deferred delivery of fragment spreads (Oct 2021+).

**Implementation Steps:**
1. Define `@defer` directive (built-in)
2. Modify executor to support incremental results
3. Implement multipart response format (`multipart/mixed`)
4. Handle deferred fragment execution asynchronously
5. Merge deferred results into main response

**Usage Example:**
```graphql
query {
    user {
        name
        ... @defer(label: "friends") {
            friends {  # Expensive, load later
                name
            }
        }
    }
}

# Response comes in parts:
# Part 1: { data: { user: { name: "Alice" } }, hasNext: true }
# Part 2: { incremental: [{ data: { friends: [...] }, label: "friends" }], hasNext: false }
```

**Acceptance Criteria:**
- [ ] @defer directive recognized
- [ ] Deferred fragments execute asynchronously
- [ ] Multipart responses work correctly
- [ ] Client can merge incremental results
- [ ] Labels work for matching results

---

#### 5.2 @stream Directive
**Priority:** P3 (Low)  
**Complexity:** Very High  
**Files:** `graphql/stream.go`, `graphql/execute.go`

**Description:**
Implement `@stream` directive for incremental delivery of list items.

**Implementation Steps:**
1. Define `@stream` directive with `initialCount` and `label` args
2. Modify list execution to support streaming
3. Send initial items immediately
4. Stream remaining items as they're resolved
5. Handle errors in streamed items

**Usage Example:**
```graphql
query {
    user {
        name
        friends @stream(initialCount: 5, label: "friends") {
            name
        }
    }
}

# Response parts:
# Part 1: { data: { user: { name: "Alice", friends: [{name: "Bob"}, ...] } }, hasNext: true }
# Part 2: { incremental: [{ items: [{name: "Charlie"}], label: "friends" }], hasNext: true }
```

**Acceptance Criteria:**
- [ ] @stream directive works on list fields
- [ ] Initial count of items returned immediately
- [ ] Remaining items streamed incrementally
- [ ] Error in one item doesn't fail entire stream

---

## Implementation Order Summary

| Phase | Feature | Priority | Complexity | Est. Days | Depends On |
|-------|---------|----------|------------|-----------|------------|
| **Phase 1** | | | | **15 days** | |
| 1.1 | Argument Deprecation | P0 | Low | 3 | - |
| 1.2 | Enum Value Deprecation | P0 | Low | 3 | - |
| 1.3 | Bug fixes & polish | P1 | Low | 5 | 1.1, 1.2 |
| 1.4 | Test coverage | P1 | Medium | 4 | 1.1, 1.2 |
| **Phase 2** | | | | **30 days** | |
| 2.1 | Directive Definition Schema | P1 | High | 8 | - |
| 2.2 | Directive Execution Framework | P1 | High | 10 | 2.1 |
| 2.3 | Repeatable Directives | P1 | Medium | 5 | 2.1 |
| 2.4 | Built-in directive migration | P2 | Medium | 4 | 2.1, 2.2 |
| 2.5 | Tests & documentation | P2 | Medium | 3 | All Phase 2 |
| **Phase 3** | | | | **18 days** | |
| 3.1 | Interface Implements Interface | P2 | Medium | 8 | - |
| 3.2 | Default Values for Input Fields | P2 | Medium | 7 | - |
| 3.3 | Integration tests | P2 | Low | 3 | 3.1, 3.2 |
| **Phase 4** | | | | **24 days** | |
| 4.1 | Query Complexity Analysis | P1 | High | 10 | - |
| 4.2 | Persisted Queries | P2 | Medium | 7 | - |
| 4.3 | Enhanced Validation | P2 | High | 7 | - |
| **Phase 5** | | | | **21 days** | |
| 5.1 | @defer Directive | P3 | Very High | 12 | 4.1 |
| 5.2 | @stream Directive | P3 | Very High | 9 | 4.1 |

**Total Estimated Time:** ~4 months (with 1 developer, parallel work possible)

---

## Dependencies Graph

```
Phase 1 (Foundation)
├── Argument Deprecation
└── Enum Value Deprecation
    └── Phase 1.3 (Polish)

Phase 2 (Custom Directives)
├── Directive Definition
│   ├── Directive Execution
│   │   └── Phase 2.4 (Migration)
│   └── Repeatable Directives
└── Phase 2.5 (Tests)

Phase 3 (Type System)
├── Interface Hierarchies
└── Input Defaults

Phase 4 (Validation/Perf)
├── Complexity Analysis
│   └── Phase 5.1 (@defer)
├── Persisted Queries
└── Enhanced Validation

Phase 5 (Incremental)
├── @defer
└── @stream
```

---

## Testing Strategy

Each phase must include:

1. **Unit Tests:** Every new function/method
2. **Integration Tests:** End-to-end scenarios  
3. **Introspection Tests:** Verify schema reflection
4. **SDL Tests:** Verify schema printing
5. **Spec Compliance Tests:** Against GraphQL spec examples

**Test Files to Create:**
- `graphql/directive_test.go`
- `graphql/complexity_test.go`
- `graphql/persisted_test.go`
- `graphql/defer_test.go`
- `graphql/stream_test.go`

---

## Migration Guide for Users

### Breaking Changes
None expected - all changes are additive.

### Deprecations
- Old scalar registration without `WithSpecifiedBy` (still works, but deprecated)

### New Recommended Patterns
```go
// After Phase 1
schema.Enum(Status(0), values, 
    schemabuilder.EnumValueDeprecated("OLD", "Use NEW"))

// After Phase 2
sb.Directive("auth", DirectiveConfig{...})
obj.FieldFunc("data", resolver, 
    schemabuilder.Directive("auth", args))

// After Phase 3
type Input struct {
    Field string `graphql:"default=value"`
}

// After Phase 4
schema := sb.MustBuild(WithMaxComplexity(1000))
```

---

## Success Metrics

**Spec Compliance Targets:**
- Phase 1 Complete: 92% compliant
- Phase 2 Complete: 96% compliant  
- Phase 3 Complete: 97% compliant
- Phase 4 Complete: 99% compliant
- Phase 5 Complete: 100% compliant (Oct 2021 spec)

**Performance Targets:**
- Query parsing: <1ms for 10KB queries
- Complexity analysis: <0.5ms overhead
- Directive execution: <5% overhead

---

## Appendix: Spec References

- [GraphQL June 2018 Spec](https://spec.graphql.org/June2018/)
- [GraphQL October 2021 Spec](https://spec.graphql.org/October2021/)
- [OneOf Input Objects RFC](https://github.com/graphql/graphql-spec/pull/825)
- [Incremental Delivery RFC](https://github.com/graphql/graphql-spec/pull/742)

---

*This roadmap is a living document. Priorities may shift based on community feedback and evolving spec requirements.*
