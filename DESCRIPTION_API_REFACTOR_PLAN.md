# Refactoring Plan: Improved Description APIs for GraphQL Types

## Executive Summary

The current description API in the Jaal GraphQL framework uses variadic parameters (`description ...string`) for adding descriptions to GraphQL types. While functional, this approach has limitations for future extensibility (e.g., adding deprecation, directives, or other metadata). This plan proposes refactoring to an **options pattern** that provides a cleaner, more extensible API for attaching metadata to GraphQL schema elements.

## Current State Analysis

### Existing Description Support

The framework currently supports descriptions through variadic parameters:

1. **Object registration** (`schema.go`):
   ```go
   func (s *Schema) Object(name string, typ interface{}, description ...string) *Object
   ```

2. **InputObject registration** (`schema.go`):
   ```go
   func (s *Schema) InputObject(name string, typ interface{}, description ...string) *InputObject
   ```

3. **Enum registration** (`schema.go`):
   ```go
   func (s *Schema) Enum(val interface{}, enumMap interface{}, description ...string)
   ```

4. **FieldFunc registration** (`types.go`):
   ```go
   func (s *Object) FieldFunc(name string, f interface{}, description ...string)
   ```

5. **Struct field tags** (`reflect.go`):
   ```go
   type User struct {
       Name string `graphql:"name,description=The user's full name"`
   }
   ```

Note: these APIs are replaced by options in this release (see sections below).
### Limitations of Current Approach

1. **Extensibility**: Adding new metadata (deprecation, directives, etc.) requires adding more variadic parameters, leading to confusing APIs
2. **Type Safety**: Variadic strings are not self-documenting and prone to errors
3. **Discoverability**: Users cannot easily see what options are available
4. **Consistency**: Different registration methods may evolve differently

## Proposed Changes

### 1. New Options Pattern API

Introduce a `TypeOption` functional options pattern for all registration methods:

```go
// TypeOption configures a GraphQL type during registration
type TypeOption func(*typeConfig)

type typeConfig struct {
    description string
    deprecated  string  // for future use
    directives  []Directive  // for future extensibility
}

// WithDescription sets the description for a type
func WithDescription(desc string) TypeOption {
    return func(c *typeConfig) {
        c.description = desc
    }
}

// WithDeprecation marks a type/field as deprecated
func WithDeprecation(reason string) TypeOption {
    return func(c *typeConfig) {
        c.deprecated = reason
    }
}
```

### 2. Updated Registration APIs

#### Schema-level registrations:

```go
// Object registers an output object with options
func (s *Schema) Object(name string, typ interface{}, opts ...TypeOption) *Object

// InputObject registers an input object with options  
func (s *Schema) InputObject(name string, typ interface{}, opts ...TypeOption) *InputObject

// Enum registers an enum with options
func (s *Schema) Enum(val interface{}, enumMap interface{}, opts ...TypeOption)
```

#### Object-level registrations:

```go
// FieldOption configures a field during registration
type FieldOption func(*fieldConfig)

type fieldConfig struct {
    description string
    deprecated  string
    nonNull     bool
}

// FieldDesc sets field description
func FieldDesc(desc string) FieldOption {
    return func(c *fieldConfig) {
        c.description = desc
    }
}

// Deprecated marks a field as deprecated
func Deprecated(reason string) FieldOption {
    return func(c *fieldConfig) {
        c.deprecated = reason
    }
}

// NonNull marks a field as non-nullable
func NonNull() FieldOption {
    return func(c *fieldConfig) {
        c.nonNull = true
    }
}

// Updated FieldFunc signature
func (s *Object) FieldFunc(name string, f interface{}, opts ...FieldOption)
```

### 3. Backward Compatibility Strategy

To maintain backward compatibility during transition:

1. **Phase 1**: Add new options-based APIs alongside existing variadic APIs
2. **Phase 2**: Mark variadic APIs as deprecated with migration guide
3. **Phase 3**: Remove variadic APIs in a future major version

Alternative: Use build tags or feature flags for gradual migration.

## Files to Modify

### Core Registration APIs

| File | Changes |
|------|---------|
| `schemabuilder/schema.go` | Add `TypeOption` type and `WithDescription()`; update `Object()`, `InputObject()`, `Enum()` signatures |
| `schemabuilder/types.go` | Add `FieldOption` type and helpers; update `FieldFunc()` signature; update `InputObject.FieldFunc()` |
| `schemabuilder/reflect.go` | Keep tag-based description parsing (already implemented) |

### Internal Propagation

| File | Changes |
|------|---------|
| `schemabuilder/build.go` | No changes needed (description already propagated) |
| `schemabuilder/output.go` | No changes needed (description already used in build) |
| `schemabuilder/input_object.go` | No changes needed (description already propagated) |
| `schemabuilder/input.go` | No changes needed (enum description already handled) |

### GraphQL Types (Core)

| File | Changes |
|------|---------|
| `graphql/types.go` | No changes needed (Description fields already exist) |

### Introspection

| File | Changes |
|------|---------|
| `introspection/introspection.go` | No changes needed (description already exposed) |
| `introspection/introspection_query.go` | No changes needed |

## Test Case Changes

### Unit Tests

1. **schemabuilder/*_test.go** (new or existing):
   - Test `Object()` with `WithDescription()` option
   - Test `InputObject()` with `WithDescription()` option  
   - Test `Enum()` with `WithDescription()` option
   - Test `FieldFunc()` with `FieldDesc()` option
   - Test combination of multiple options
   - Test backward compatibility with variadic params

2. **Test Examples**:
   ```go
   func TestObjectWithOptions(t *testing.T) {
       sb := NewSchema()
       obj := sb.Object("User", User{}, WithDescription("A user in the system"))
       require.Equal(t, "A user in the system", obj.Description)
   }
   
   func TestFieldFuncWithOptions(t *testing.T) {
       sb := NewSchema()
       obj := sb.Object("User", User{})
       obj.FieldFunc("name", func(u *User) string { return u.Name }, 
           FieldDesc("The user's full name"),
           Deprecated("Use displayName instead"),
       )
       // Verify field has description and deprecation
   }
   ```

### Integration Tests

1. **introspection/introspection_test.go**:
   - Verify descriptions appear in introspection with new API
   - Test that options-based and variadic APIs produce same introspection output

2. **graphql/end_to_end_test.go**:
   - Add end-to-end test using new options API

## Example Directory Changes

### example/users/ Directory

Update all registration files to use new options pattern:

**register_objects.go**:
```go
// Before:
user := sb.Object("User", User{}, "User payload representing a person in the system.")
user.FieldFunc("id", func(u *User) schemabuilder.ID { return u.ID }, schemabuilder.FieldDesc("Unique identifier for the user."))

// After:
user := sb.Object("User", User{}, schemabuilder.WithDescription("User payload representing a person in the system."))
user.FieldFunc("id", func(u *User) schemabuilder.ID { return u.ID },
    schemabuilder.FieldDesc("Unique identifier for the user."))
```

**register_inputs.go**:
```go
// Before:
input := sb.InputObject("CreateUserInput", CreateUserInput{}, "Input for creating a new user...")

// After:
input := sb.InputObject("CreateUserInput", CreateUserInput{},
    schemabuilder.WithDescription("Input for creating a new user..."))
```

**register_enums.go**:
```go
// Before:
sb.Enum(RoleMember, map[string]interface{}{...}, "Role for user access control...")

// After:
sb.Enum(RoleMember, map[string]interface{}{...},
    schemabuilder.WithDescription("Role for user access control..."))
```

**register_queries.go**:
```go
// Before:
q.FieldFunc("me", func(ctx context.Context) *User { ... }, "Returns the current authenticated user...")

// After:
q.FieldFunc("me", func(ctx context.Context) *User { ... },
    schemabuilder.FieldDesc("Returns the current authenticated user..."))
```

### example/main.go

No changes needed (delegates to users package).

### example/character/ and example/test-1/

Update to use new options pattern (follow same pattern as users/). These examples should be updated to demonstrate the new recommended approach.

## Documentation Changes

### README.md

Update all code examples to use the new options pattern:

**Before**:
```go
payload := schema.Object("Character", Character{})
payload.FieldFunc("id", func(ctx context.Context, in *Character) *schemabuilder.ID {
    return &schemabuilder.ID{Value: in.Id}
})
```

**After**:
```go
payload := schema.Object("Character", Character{}, 
    schemabuilder.WithDescription("A character in the story"))
payload.FieldFunc("id", func(ctx context.Context, in *Character) *schemabuilder.ID {
    return &schemabuilder.ID{Value: in.Id}
}, schemabuilder.FieldDesc("Unique identifier"))
```

Add section explaining:
1. The options pattern and why it's preferred
2. Available options (WithDescription, FieldDesc, Deprecated, etc.)
3. Backward compatibility note about variadic params
4. Migration guide from old to new API

### CONTRIBUTING.md

Add guidelines for:
1. Using options pattern for new metadata features
2. Maintaining backward compatibility
3. Deprecation process for old APIs

### SPEC_COMPLIANCE_PLAN.md

Update to mark description support as fully implemented with the new options pattern.

### DEPRECATION_ON_INPUT_VALUES_PLAN.md

Note that the options pattern will be the recommended way to add deprecation in the future.

### ONE_OF_DIRECTIVE_IMPLEMENTATION_PLAN.md

Reference the options pattern as the model for future directive implementations.

## Migration Strategy

### For Framework Users

1. **Immediate**: Options-only API (variadic APIs removed in this major release)
2. **Removal**: Variadic APIs are removed; update callers to use options

### Migration Script (Optional)

Provide a simple `sed` or `go fix` style script to automate migration:

```bash
# Example transformation
# sb.Object("X", X{}, "desc") -> sb.Object("X", X{}, WithDescription("desc"))
# obj.FieldFunc("f", fn, "desc") -> obj.FieldFunc("f", fn, FieldDesc("desc"))
```

## Benefits of New Approach

1. **Extensibility**: Easy to add new metadata without breaking changes
2. **Discoverability**: IDE autocomplete shows available options
3. **Type Safety**: Options are typed, not raw strings
4. **Consistency**: Same pattern across all registration methods
5. **Future-Proof**: Can support directives, metadata, extensions cleanly

## Future Enhancements (Post-Refactor)

With the options pattern in place, future enhancements become trivial:

```go
// Example future extensions
sb.Object("User", User{},
    WithDescription("A user"),
    WithDirective("auth", map[string]interface{}{"role": "admin"}),
    WithMetadata("team", "platform"),
)
```

## Implementation Checklist

- [ ] Create `TypeOption` and `FieldOption` types in schemabuilder
- [ ] Implement option functions (WithDescription, FieldDesc, Deprecated, NonNull)
- [ ] Update `Object()` to accept `...TypeOption`
- [ ] Update `InputObject()` to accept `...TypeOption`
- [ ] Update `Enum()` to accept `...TypeOption`
- [ ] Update `FieldFunc()` to accept `...FieldOption`
- [ ] Update `InputObject.FieldFunc()` to accept `...FieldOption`
- [ ] Add unit tests for new options
- [ ] Update example/users/ to use new API
- [ ] Update example/character/ to use new API
- [ ] Update example/test-1/ to use new API
- [ ] Update README.md with new examples
- [ ] Add migration guide
- [ ] Remove old variadic APIs (major release)
- [ ] Verify all existing tests pass

## Non-Goals

1. **Changing graphql/types.go**: Core types already have Description fields
2. **Changing SDL/schema parsing**: Out of scope (code-first framework)

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing code | Major version release with migration guide |
| User confusion | Clear documentation and migration guide |
| Increased API surface | Options are discoverable and self-documenting |
| Performance overhead | Options evaluated at schema build time, not runtime |

## Conclusion

This refactoring modernizes the Jaal framework's API to use the options pattern, providing a foundation for future extensibility while maintaining full backward compatibility. The changes are localized to the schemabuilder package and examples, with no changes needed to core GraphQL types or introspection.
