# Refactoring Plan: Deprecation API for Input Fields (Options Pattern)

## Executive Summary

The current deprecation API for input fields in the Jaal GraphQL framework uses struct tags (e.g., `graphql:"name,deprecated=Use newField instead"`). This plan proposes refactoring to an **options pattern** that aligns with the recent description API changes, providing a cleaner, more extensible, and consistent API for marking input fields as deprecated.

## Current State Analysis

### Existing Deprecation Support

The framework currently supports deprecation through struct tags in `reflect.go`:

```go
type CreateUserInput struct {
    Name string `graphql:"name"`
    Age  int32  `graphql:"age,deprecated=Use birthDate instead"`
}
```

The tag is parsed in `reflect.go` and the deprecation reason is stored in `graphQLFieldInfo.DeprecationReason`, then propagated to `graphql.InputObject.FieldDeprecations`.

### Limitations of Current Approach

1. **Inconsistency**: Descriptions moved to options pattern, but deprecation remains tag-based
2. **Discoverability**: Tags are not self-documenting and prone to syntax errors
3. **Flexibility**: Cannot easily combine with other metadata (directives, extensions)
4. **Dynamic deprecation**: Cannot conditionally deprecate fields at runtime
5. **InputObject.FieldFunc limitation**: No way to deprecate fields registered via `InputObject.FieldFunc()`

## Proposed Changes

### 1. New Options Pattern API for Input Fields

Extend `FieldOption` to support deprecation on input fields:

```go
// In schemabuilder/types.go - FieldOption already exists for output fields
// We extend it to work with InputObject.FieldFunc as well

// Deprecated marks a field as deprecated (works for both output and input fields)
func Deprecated(reason string) FieldOption {
    return func(cfg *fieldConfig) {
        cfg.deprecated = reason
    }
}
```

### 2. Updated InputObject.FieldFunc Signature

The `InputObject.FieldFunc` should accept `FieldOption` variadic parameter:

```go
// Current (already updated for descriptions)
func (io *InputObject) FieldFunc(name string, function interface{}, opts ...FieldOption)

// Usage with deprecation:
input.FieldFunc("age", func(target *CreateUserInput, source int32) {
    target.Age = source
}, schemabuilder.Deprecated("Use birthDate instead"))
```

### 3. Internal Propagation Changes

#### schemabuilder/types.go
- `InputObject` struct needs `FieldDeprecations map[string]string` field (already added)
- `FieldFunc` should populate `FieldDeprecations` when `Deprecated` option is used

#### schemabuilder/input_object.go
- Update `makeInputObjectParser` to read from `obj.FieldDeprecations` 
- Populate `argType.FieldDeprecations[name]` from the options

#### schemabuilder/reflect.go
- Keep tag-based parsing for backward compatibility during transition
- Tag-based deprecation should be merged with options-based deprecation
- Options take precedence over tags if both are present

### 4. Backward Compatibility Strategy

Since this is a major release (per previous changes):

1. **Phase 1 (Current)**: Support both tag-based and options-based deprecation
2. **Phase 2 (Future major)**: Remove tag-based deprecation

For now, we'll support both with options taking precedence.

## Files to Modify

### Core API Files

| File | Changes |
|------|---------|
| `schemabuilder/types.go` | Ensure `Deprecated()` function works for input fields; update `InputObject.FieldFunc` to store deprecations |
| `schemabuilder/input_object.go` | Read `FieldDeprecations` from `InputObject` and populate `graphql.InputObject.FieldDeprecations` |
| `schemabuilder/reflect.go` | Merge tag-based and options-based deprecation; options take precedence |

### GraphQL Types (No Changes Needed)

| File | Status |
|------|--------|
| `graphql/types.go` | Already has `InputObject.FieldDeprecations` |
| `introspection/introspection.go` | Already reads `FieldDeprecations` for `__Type.inputFields` |

## Test Case Changes

### Unit Tests to Add/Update

1. **schemabuilder/types_test.go** (new or existing):
   ```go
   func TestInputObjectFieldFuncWithDeprecation(t *testing.T) {
       sb := NewSchema()
       input := sb.InputObject("TestInput", TestInput{})
       input.FieldFunc("oldField", func(target *TestInput, source string) {
           target.OldField = source
       }, Deprecated("Use newField instead"))
       
       // Verify FieldDeprecations is populated
       require.Equal(t, "Use newField instead", input.FieldDeprecations["oldField"])
   }
   ```

2. **schemabuilder/reflect_test.go**:
   - Test that tag-based deprecation still works
   - Test that options-based deprecation takes precedence over tags
   - Test combining options and tags (different fields)

3. **introspection/introspection_test.go**:
   - Update existing tests to use options pattern
   - Verify `isDeprecated` and `deprecationReason` appear in introspection

### Example Test Updates

Update `example/users/register_inputs.go`:

```go
// Before (tag-based in types.go):
type CreateUserInput struct {
    Age int32 `graphql:"age,deprecated=Use birthDate instead"`
}

// After (options-based in register_inputs.go):
input.FieldFunc("age", func(target *CreateUserInput, source int32) {
    target.Age = source
}, schemabuilder.Deprecated("Use birthDate instead"))
```

## Example Directory Changes

### example/users/register_inputs.go

**Current (tag-based)**:
```go
// In types.go
// Age  int32  `graphql:"age,deprecated=Use birthDate instead"`

// In register_inputs.go
input.FieldFunc("age", func(target *CreateUserInput, source int32) { 
    target.Age = source 
})
```

**Proposed (options-based)**:
```go
// In register_inputs.go
input.FieldFunc("age", func(target *CreateUserInput, source int32) { 
    target.Age = source 
}, schemabuilder.Deprecated("Use birthDate instead"))
```

### example/users/types.go

Remove or keep the tag for backward compatibility during transition:

```go
// Option 1: Remove tag entirely (breaking change)
type CreateUserInput struct {
    Name string
    Age  int32  // No tag
}

// Option 2: Keep tag for backward compatibility
// (options will take precedence if both present)
type CreateUserInput struct {
    Name string
    Age  int32  `graphql:"age,deprecated=Use birthDate instead"`
}
```

## Documentation Changes

### README.md

Add deprecation section alongside description section:

```markdown
### Deprecation Options

Mark fields as deprecated using the `Deprecated` option:

```go
// Deprecate an output field
user.FieldFunc("oldField", func(u *User) string { 
    return u.OldField 
}, schemabuilder.Deprecated("Use newField instead"))

// Deprecate an input field
input.FieldFunc("oldField", func(target *Input, source string) {
    target.OldField = source
}, schemabuilder.Deprecated("Use newField instead"))
```
```

### DEPRECATION_API_REFACTOR_PLAN.md (this file)

Will be created as the canonical reference for this refactor.

### Migration Guide

Add to README or separate MIGRATION.md:

```markdown
## Migrating from Tag-Based to Options-Based Deprecation

### Before (Tag-Based)
```go
type Input struct {
    OldField string `graphql:"oldField,deprecated=Use newField instead"`
}
```

### After (Options-Based)
```go
type Input struct {
    OldField string
}

input.FieldFunc("oldField", func(target *Input, source string) {
    target.OldField = source
}, schemabuilder.Deprecated("Use newField instead"))
```
```

## Implementation Checklist

- [ ] Add `Deprecated()` function to `schemabuilder/types.go` (if not already present)
- [ ] Update `InputObject.FieldFunc()` to store deprecations in `FieldDeprecations` map
- [ ] Update `makeInputObjectParser()` in `input_object.go` to read from `FieldDeprecations`
- [ ] Update `reflect.go` to merge tag-based and options-based deprecation
- [ ] Add unit tests for options-based deprecation
- [ ] Update `example/users/` to use options-based deprecation
- [ ] Update `example/character/` to use options-based deprecation (if applicable)
- [ ] Update `example/test-1/` to use options-based deprecation (if applicable)
- [ ] Update README.md with deprecation examples
- [ ] Verify introspection shows deprecation correctly

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing tag-based code | Support both during transition; options take precedence |
| User confusion | Clear migration guide and examples |
| Inconsistent with description API | This change brings deprecation in line with descriptions |
| Performance overhead | Negligible (map lookup at schema build time) |

## Future Enhancements

With the options pattern in place:

```go
// Future: Conditional deprecation
input.FieldFunc("oldField", fn, 
    schemabuilder.Deprecated("Use newField instead"),
    schemabuilder.DeprecatedIf(func() bool { return featureFlagEnabled }),
)

// Future: Deprecation with metadata
input.FieldFunc("oldField", fn,
    schemabuilder.Deprecated("Use newField instead"),
    schemabuilder.WithDeprecationInfo("1.0", "2.0"), // deprecated since, removal in
)
```

## Conclusion

This refactor aligns the deprecation API with the recently implemented options-based description API, providing consistency and better extensibility. The changes are minimal and focused on the `schemabuilder` package, with no changes needed to core GraphQL types or introspection (which already support the underlying data structures).
