# Refactoring Plan: OneOf Input API (Function-Based)

## Executive Summary

The current OneOf input API in the Jaal GraphQL framework uses an embedded marker struct (`schemabuilder.OneOfInput`) to denote that an input type should be treated as a oneOf input object (@oneOf directive per Oct 2021+ spec). This plan proposes refactoring to a **function-based API** that aligns with the recent description and deprecation API changes, providing a cleaner and more consistent registration pattern.

## Current State Analysis

### Existing OneOf Support

The framework currently supports OneOf through an embedded marker struct:

```go
type IdentifierInput struct {
    schemabuilder.OneOfInput
    ID    *schemabuilder.ID
    Email *string
}

// Registration
input := sb.InputObject("IdentifierInput", IdentifierInput{})
```

The `OneOfInput` struct is detected via reflection (`hasOneOfMarkerEmbedded`) in `input_object.go`, which sets `graphql.InputObject.OneOf = true`.

### Limitations of Current Approach

1. **Inconsistency**: Descriptions and deprecations moved to options/function-based APIs, but OneOf remains struct-embedding based
2. **Discoverability**: Embedded structs are not self-documenting
3. **Flexibility**: Cannot easily toggle OneOf behavior at runtime or conditionally
4. **Documentation**: The embedded struct approach is less clear in code reviews

## Proposed Changes

### 1. New Function-Based API for OneOf

Add a method on `InputObject` to mark it as OneOf:

```go
// In schemabuilder/types.go

// MarkOneOf marks this input object as a oneOf input (@oneOf directive per spec).
// Exactly one field must be provided/non-null in queries.
func (io *InputObject) MarkOneOf()
```

### 2. Updated Registration Pattern

**Before (embedded struct)**:
```go
type IdentifierInput struct {
    schemabuilder.OneOfInput
    ID    *schemabuilder.ID
    Email *string
}

input := sb.InputObject("IdentifierInput", IdentifierInput{})
```

**After (function-based)**:
```go
type IdentifierInput struct {
    ID    *schemabuilder.ID
    Email *string
}

input := sb.InputObject("IdentifierInput", IdentifierInput{})
input.MarkOneOf()
```

### 3. Internal Changes

#### schemabuilder/types.go
- Add `OneOf bool` field to `InputObject` struct
- Add `MarkOneOf()` method to set the flag

#### schemabuilder/input_object.go
- Update `generateObjectParserInner` to check `obj.OneOf` instead of calling `hasOneOfMarkerEmbedded`
- Remove or deprecate the struct-based detection logic

#### schemabuilder/schema.go (if needed)
- Ensure `InputObject` copying preserves the `OneOf` flag

### 4. Backward Compatibility Strategy

Since this is a major release:

1. **Remove** the embedded struct approach entirely
2. **Replace** with function-based API
3. **Update** all examples and documentation

The `OneOfInput` struct will be removed from the public API.

## Files to Modify

### Core API Files

| File | Changes |
|------|---------|
| `schemabuilder/types.go` | Add `OneOf bool` to `InputObject`; add `MarkOneOf()` method; remove `OneOfInput` struct |
| `schemabuilder/input_object.go` | Check `obj.OneOf` instead of `hasOneOfMarkerEmbedded`; remove struct-based detection |
| `schemabuilder/schema.go` | Ensure `OneOf` flag is copied when cloning `InputObject` |

### GraphQL Types (No Changes Needed)

| File | Status |
|------|--------|
| `graphql/types.go` | Already has `InputObject.OneOf` field |
| `introspection/introspection.go` | Already reads `OneOf` for `@oneOf` directive |

## Test Case Changes

### Unit Tests to Add/Update

1. **schemabuilder/types_test.go** (new or existing):
   ```go
   func TestInputObjectMarkOneOf(t *testing.T) {
       sb := NewSchema()
       input := sb.InputObject("TestInput", TestInput{})
       input.MarkOneOf()
       
       // Verify OneOf flag is set
       require.True(t, input.OneOf)
   }
   ```

2. **schemabuilder/input_object_test.go**:
   - Update tests to use `MarkOneOf()` instead of embedded struct
   - Verify `@oneOf` validation still works

3. **introspection/introspection_test.go**:
   - Update existing OneOf tests to use new API

### Example Test Updates

Update `example/users/register_inputs.go`:

```go
// Before (embedded struct):
type IdentifierInput struct {
    schemabuilder.OneOfInput
    ID    *schemabuilder.ID
    Email *string
}

// After (function-based):
type IdentifierInput struct {
    ID    *schemabuilder.ID
    Email *string
}

func RegisterIdentifierInput(sb *schemabuilder.Schema) {
    input := sb.InputObject("IdentifierInput", IdentifierInput{}, 
        schemabuilder.WithDescription("OneOf identifier: exactly one of ID or email"))
    input.MarkOneOf()
    // ... FieldFunc registrations
}
```

## Example Directory Changes

### example/users/types.go

**Current (embedded struct)**:
```go
type IdentifierInput struct {
    schemabuilder.OneOfInput
    ID    *schemabuilder.ID
    Email *string
}

type ContactByInput struct {
    schemabuilder.OneOfInput
    Email *string
    Phone *string
}
```

**Proposed (plain struct)**:
```go
type IdentifierInput struct {
    ID    *schemabuilder.ID
    Email *string
}

type ContactByInput struct {
    Email *string
    Phone *string
}
```

### example/users/register_inputs.go

**Current**:
```go
func RegisterIdentifierInput(sb *schemabuilder.Schema) {
    identifierInput := sb.InputObject("IdentifierInput", IdentifierInput{}, 
        schemabuilder.WithDescription("OneOf identifier: exactly one of ID or email"))
    identifierInput.FieldFunc("id", ...)
    identifierInput.FieldFunc("email", ...)
}
```

**Proposed**:
```go
func RegisterIdentifierInput(sb *schemabuilder.Schema) {
    identifierInput := sb.InputObject("IdentifierInput", IdentifierInput{}, 
        schemabuilder.WithDescription("OneOf identifier: exactly one of ID or email"))
    identifierInput.MarkOneOf()
    identifierInput.FieldFunc("id", ...)
    identifierInput.FieldFunc("email", ...)
}
```

## Documentation Changes

### README.md

Update the OneOf section:

```markdown
### OneOf Input Objects

Mark input objects as OneOf (@oneOf directive) using `MarkOneOf()`:

```go
type IdentifierInput struct {
    ID    *schemabuilder.ID
    Email *string
}

input := sb.InputObject("IdentifierInput", IdentifierInput{})
input.MarkOneOf()
```

Exactly one field must be provided/non-null in queries (enforced by schema).
```

### ONE_OF_API_REFACTOR_PLAN.md (this file)

Will be created as the canonical reference for this refactor.

### Migration Guide

Add to README or separate MIGRATION.md:

```markdown
## Migrating from Embedded Struct to Function-Based OneOf

### Before (Embedded Struct)
```go
type Input struct {
    schemabuilder.OneOfInput
    Field1 *string
    Field2 *string
}
```

### After (Function-Based)
```go
type Input struct {
    Field1 *string
    Field2 *string
}

input := sb.InputObject("Input", Input{})
input.MarkOneOf()
```
```

## Implementation Checklist

- [ ] Add `OneOf bool` field to `InputObject` struct in `types.go`
- [ ] Add `MarkOneOf()` method to `InputObject` in `types.go`
- [ ] Remove `OneOfInput` struct from `types.go`
- [ ] Update `generateObjectParserInner` in `input_object.go` to check `obj.OneOf`
- [ ] Remove `hasOneOfMarkerEmbedded` function from `input_object.go`
- [ ] Remove struct-based OneOf detection from `generateArgParser` in `input_object.go`
- [ ] Ensure `InputObject` copying in `schema.go` preserves `OneOf` flag
- [ ] Add unit tests for `MarkOneOf()`
- [ ] Update `example/users/` to use function-based OneOf
- [ ] Update `example/character/` (if applicable)
- [ ] Update `example/test-1/` (if applicable)
- [ ] Update README.md with OneOf examples
- [ ] Verify introspection shows `@oneOf` directive correctly

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing code | Major version release with migration guide |
| User confusion | Clear documentation and examples |
| Inconsistent with other APIs | This change brings OneOf in line with descriptions/deprecations |
| Validation logic breakage | Keep existing validation, just change how flag is set |

## Conclusion

This refactor aligns the OneOf API with the recently implemented options-based description and deprecation APIs, providing consistency across the schemabuilder package. The function-based approach is clearer and more discoverable than the embedded struct pattern.
