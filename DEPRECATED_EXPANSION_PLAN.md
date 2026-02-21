# Implementation Plan: Expanded `@deprecated` Directive (ARGUMENT_DEFINITION, INPUT_FIELD_DEFINITION)

## 1. How the Expanded `@deprecated` Directive Works (Spec Explanation)

### Definition (September 2025 Spec)
```graphql
directive @deprecated(
  reason: String! = "No longer supported"
) on FIELD_DEFINITION | ARGUMENT_DEFINITION | INPUT_FIELD_DEFINITION | ENUM_VALUE
```

### What Changed from June 2018
In the June 2018 spec, `@deprecated` was limited to two locations:
```graphql
# June 2018 (old):
directive @deprecated(reason: String = "No longer supported") on FIELD_DEFINITION | ENUM_VALUE
```

The September 2025 spec expands this in two ways:
1. **Two new locations**: `ARGUMENT_DEFINITION` and `INPUT_FIELD_DEFINITION`
2. **`reason` is now `String!`** (non-nullable) with a default value of `"No longer supported"` (previously it was nullable `String`)

### What This Means Concretely

#### Deprecating Field Arguments
A field's argument can now be marked deprecated:
```graphql
type Query {
  users(
    limit: Int
    offset: Int @deprecated(reason: "Use `after` cursor instead.")
    after: String
  ): [User]
}
```

#### Deprecating Input Object Fields
A field within an input object can now be marked deprecated:
```graphql
input CreateUserInput {
  name: String!
  email: String @deprecated(reason: "Use `contactEmail` instead.")
  contactEmail: String
}
```

### Rules
1. **Cannot deprecate required inputs** — `@deprecated` must **not** be applied to required arguments or input fields (i.e., non-null types without a default value). To deprecate, first make the field optional (add a default value or make the type nullable).
2. **Deprecated items remain fully functional** — They execute normally. Deprecation is purely metadata for tooling (IDEs, docs, linters).
3. **Reason defaults to `"No longer supported"`** — If the `reason` argument is omitted, the default applies.

### Introspection Changes

#### `__InputValue` gains two new fields
```graphql
type __InputValue {
  name: String!
  description: String
  type: __Type!
  defaultValue: String
  isDeprecated: Boolean!      # NEW
  deprecationReason: String   # NEW
}
```
- `isDeprecated`: `true` if `@deprecated` is present; `false` otherwise
- `deprecationReason`: the `reason` string if deprecated; `null` otherwise

#### `__Field.args` gains `includeDeprecated` argument
```graphql
type __Field {
  # ...
  args(includeDeprecated: Boolean! = false): [__InputValue!]!
}
```

#### `__Type.inputFields` gains `includeDeprecated` argument
```graphql
type __Type {
  # ...
  inputFields(includeDeprecated: Boolean! = false): [__InputValue!]
}
```

#### `@deprecated` directive in `__schema.directives` gains new locations
The directive's `locations` field must include `ARGUMENT_DEFINITION` and `INPUT_FIELD_DEFINITION` in addition to `FIELD_DEFINITION` and `ENUM_VALUE`.

### Example Introspection Queries

**Querying deprecated arguments:**
```graphql
{
  __type(name: "Query") {
    fields {
      name
      args(includeDeprecated: true) {
        name
        isDeprecated
        deprecationReason
      }
    }
  }
}
```

**Querying deprecated input fields:**
```graphql
{
  __type(name: "CreateUserInput") {
    inputFields(includeDeprecated: true) {
      name
      isDeprecated
      deprecationReason
    }
  }
}
```

---

## 2. Current State in Jaal (What Exists Today)

### Deprecation on Fields and Enum Values
- The `field` struct in `introspection/introspection.go` (line 368) has `IsDeprecated bool` and `DeprecationReason string` fields, and these are exposed via `__Field` introspection
- The `EnumValue` struct (line 69) similarly has deprecation fields
- **However**, there is no user-facing API to actually *mark* a field or enum value as deprecated — the fields always default to `false`/`""`. The deprecation metadata exists in the introspection types but is never populated from user schemas.

### `InputValue` Has No Deprecation Fields
- The `InputValue` struct (line 46) only has `Name`, `Description`, `Type`, and `DefaultValue`
- **No `IsDeprecated` or `DeprecationReason`** fields exist

### `__Field.args` Has No `includeDeprecated` Argument
- The `args` field func on `__Field` (line 388) returns `[]InputValue` with no filtering argument
- Similarly, `__Type.inputFields` (line 275) has no `includeDeprecated` argument

### `__schema.directives` Does Not Include `@deprecated`
- Currently the directives list only contains `@include`, `@skip`, and `@specifiedBy`
- **`@deprecated` is not listed at all**, which is itself a spec violation even for the June 2018 spec

### `graphql.Field.Args` Is `map[string]Type`
- The `Args` field on `graphql.Field` (types.go line 147) is `map[string]Type` — it maps argument names to their GraphQL types
- **No metadata (deprecation, description) is carried** for individual arguments

### `graphql.InputObject.InputFields` Is `map[string]Type`
- Similarly, `InputFields` on `graphql.InputObject` (types.go line 78) is `map[string]Type`
- **No metadata for individual input fields**

### Key Insight: The Data Model Gap
The core issue is that `graphql.Field.Args` and `graphql.InputObject.InputFields` use `map[string]Type` — they only carry the type, not any metadata like deprecation status. To support deprecation on arguments and input fields, we need to either:
- (a) Change the value type from `Type` to a richer struct that carries metadata, OR
- (b) Store deprecation metadata in a parallel data structure

Option (a) is cleaner and more extensible. We'll introduce a `graphql.InputField` struct (analogous to how `graphql.Field` carries metadata for object fields) and use it as the value type.

---

## 3. Implementation Plan — Exact Code Changes

### Change 1: Introduce `graphql.InputField` struct

**File: `graphql/types.go`**

Add a new struct to carry metadata for input values (arguments and input object fields):

```go
// InputField describes an argument on a field or a field on an input object.
// It carries the type along with optional deprecation metadata.
type InputField struct {
    Type              Type
    IsDeprecated      bool
    DeprecationReason string
}
```

### Change 2: Change `graphql.Field.Args` from `map[string]Type` to `map[string]*InputField`

**File: `graphql/types.go`**

```go
// BEFORE:
type Field struct {
    // ...
    Args           map[string]Type
    // ...
}

// AFTER:
type Field struct {
    // ...
    Args           map[string]*InputField
    // ...
}
```

### Change 3: Change `graphql.InputObject.InputFields` from `map[string]Type` to `map[string]*InputField`

**File: `graphql/types.go`**

```go
// BEFORE:
type InputObject struct {
    Name        string
    InputFields map[string]Type
}

// AFTER:
type InputObject struct {
    Name        string
    InputFields map[string]*InputField
}
```

### Change 4: Update all code that creates or reads `Field.Args` and `InputObject.InputFields`

These maps change from `map[string]Type` to `map[string]*InputField`, so every site that writes to or reads from them must be updated. The changes are mechanical — wrapping `Type` values in `&InputField{Type: ...}` and unwrapping via `.Type`.

**Files affected:**

#### 4a. `schemabuilder/function.go` — `argsTypeMap()`
The function currently returns `map[string]graphql.Type`. It needs to return `map[string]*graphql.InputField`:
```go
// BEFORE:
func (funcCtx *funcContext) argsTypeMap(argType graphql.Type) (map[string]graphql.Type, error) {
    args := make(map[string]graphql.Type)
    // ...
    for name, typ := range inputObject.InputFields {
        args[name] = typ
    }
    return args, nil
}

// AFTER:
func (funcCtx *funcContext) argsTypeMap(argType graphql.Type) (map[string]*graphql.InputField, error) {
    args := make(map[string]*graphql.InputField)
    // ...
    for name, f := range inputObject.InputFields {
        args[name] = f
    }
    return args, nil
}
```

#### 4b. `schemabuilder/input_object.go` — `generateArgParser()`
Where `InputFields` is populated:
```go
// BEFORE:
argType.InputFields[fieldInfo.Name] = fieldArgTyp

// AFTER:
argType.InputFields[fieldInfo.Name] = &graphql.InputField{Type: fieldArgTyp}
```

#### 4c. `schemabuilder/input_object.go` — `generateObjectParserInner()`
Where `InputFields` is populated for registered input objects:
```go
// BEFORE:
argType.InputFields[name] = fieldArgTyp

// AFTER:
argType.InputFields[name] = &graphql.InputField{Type: fieldArgTyp}
```

#### 4d. `graphql/validate.go` — All places that read `field.Args` or `inputObject.InputFields`
Validation code accesses `field.Args[name]` and `typ.InputFields` — these now return `*InputField` instead of `Type`, so we need to access `.Type`:
- `ValidateQuery` for `*Object`: `field.Args` iteration and access
- `ValidateQuery` for `*Interface`: same
- `collectTypes` in introspection: iterates `field.Args` and `typ.InputFields`

#### 4e. `introspection/introspection.go` — Where `InputValue` structs are built from args/inputFields
The `registerType` method builds `InputValue` slices from `field.Args` and `inputObject.InputFields`. These need to read the new struct and populate deprecation fields.

#### 4f. `introspection/introspection.go` — Where `collectTypes` reads args and input fields
`collectTypes` iterates `field.Args` to find nested types — it needs to access `.Type` on the `InputField`.

### Change 5: Add `IsDeprecated` and `DeprecationReason` to introspection `InputValue`

**File: `introspection/introspection.go`**

```go
// BEFORE:
type InputValue struct {
    Name         string
    Description  string
    Type         Type
    DefaultValue *string
}

// AFTER:
type InputValue struct {
    Name              string
    Description       string
    Type              Type
    DefaultValue      *string
    IsDeprecated      bool
    DeprecationReason string
}
```

Register the new fields in `registerInputValue()`:
```go
obj.FieldFunc("isDeprecated", func(in InputValue) bool {
    return in.IsDeprecated
})
obj.FieldFunc("deprecationReason", func(in InputValue) *string {
    if in.DeprecationReason != "" {
        return &in.DeprecationReason
    }
    return nil
})
```

### Change 6: Populate deprecation data when building `InputValue` from `InputField`

**File: `introspection/introspection.go`**

In `registerType()`, where `InputValue` structs are built from `field.Args` and `inputObject.InputFields`, populate the deprecation fields:

For field args (in the `fields` field func):
```go
// BEFORE:
args = append(args, InputValue{
    Name: name,
    Type: Type{Inner: a},
})

// AFTER:
args = append(args, InputValue{
    Name:              name,
    Type:              Type{Inner: a.Type},
    IsDeprecated:      a.IsDeprecated,
    DeprecationReason: a.DeprecationReason,
})
```

For input object fields (in the `inputFields` field func):
```go
// BEFORE:
fields = append(fields, InputValue{
    Name: name,
    Type: Type{Inner: f},
})

// AFTER:
fields = append(fields, InputValue{
    Name:              name,
    Type:              Type{Inner: f.Type},
    IsDeprecated:      f.IsDeprecated,
    DeprecationReason: f.DeprecationReason,
})
```

### Change 7: Add `includeDeprecated` argument to `__Field.args`

**File: `introspection/introspection.go`**

In `registerField()`, change the `args` field func to accept `includeDeprecated`:

```go
// BEFORE:
obj.FieldFunc("args", func(in field) []InputValue {
    return in.Args
})

// AFTER:
obj.FieldFunc("args", func(in field, args struct {
    IncludeDeprecated *bool
}) []InputValue {
    if args.IncludeDeprecated != nil && *args.IncludeDeprecated {
        return in.Args
    }
    var result []InputValue
    for _, arg := range in.Args {
        if !arg.IsDeprecated {
            result = append(result, arg)
        }
    }
    return result
})
```

### Change 8: Add `includeDeprecated` argument to `__Type.inputFields`

**File: `introspection/introspection.go`**

```go
// BEFORE:
object.FieldFunc("inputFields", func(t Type) []InputValue {

// AFTER:
object.FieldFunc("inputFields", func(t Type, args struct {
    IncludeDeprecated *bool
}) []InputValue {
```

And add filtering logic to exclude deprecated input fields when `includeDeprecated` is false/nil.

### Change 9: Add `@deprecated` directive to `__schema.directives` list

**File: `introspection/introspection.go`**

Add two new directive location constants:
```go
DIRECTIVE_LOC_FIELD_DEFINITION      DirectiveLocation = "FIELD_DEFINITION"
DIRECTIVE_LOC_ARGUMENT_DEFINITION   DirectiveLocation = "ARGUMENT_DEFINITION"
DIRECTIVE_LOC_INPUT_FIELD_DEFINITION DirectiveLocation = "INPUT_FIELD_DEFINITION"
DIRECTIVE_LOC_ENUM_VALUE            DirectiveLocation = "ENUM_VALUE"
```

Add these to the `DirectiveLocation` enum registration.

Define the `deprecatedDirective` var:
```go
var deprecatedDirective = Directive{
    Name:        "deprecated",
    Description: "Marks an element of a GraphQL schema as no longer supported.",
    Locations: []DirectiveLocation{
        DIRECTIVE_LOC_FIELD_DEFINITION,
        DIRECTIVE_LOC_ARGUMENT_DEFINITION,
        DIRECTIVE_LOC_INPUT_FIELD_DEFINITION,
        DIRECTIVE_LOC_ENUM_VALUE,
    },
    Args: []InputValue{
        {
            Name:        "reason",
            Type:        Type{Inner: &graphql.NonNull{Type: &graphql.Scalar{Type: "String"}}},
            Description: "Explains why this element was deprecated, usually also including a suggestion for how to access supported similar data. Formatted using the Markdown syntax, as specified by [CommonMark](https://commonmark.org/).",
        },
    },
}
```

Add to the directives list:
```go
Directives: []Directive{includeDirective, skipDirective, deprecatedDirective, specifiedByDirective},
```

### Change 10: Add user-facing API to mark arguments and input fields as deprecated

**File: `schemabuilder/types.go`**

Add a `FieldFuncWithDeprecation` method on `Object` for marking field arguments as deprecated. Since Jaal uses Go structs for args (e.g., `args struct { Offset int }`), deprecation is applied per-field after schema build. We need a mechanism to mark specific args as deprecated.

Add a `DeprecatedFields` map to `schemabuilder.Object` and `schemabuilder.InputObject`:

```go
// On Object:
type Object struct {
    Name             string
    Description      string
    Type             interface{}
    Methods          Methods
    key              string
    DeprecatedArgs   map[string]map[string]string  // method name -> arg name -> reason
}

// On InputObject:
type InputObject struct {
    Name             string
    Type             interface{}
    Fields           map[string]interface{}
    DeprecatedFields map[string]string  // field name -> reason
}
```

Add API methods:
```go
// DeprecateArg marks an argument on a field as deprecated with the given reason.
func (s *Object) DeprecateArg(fieldName, argName, reason string) {
    if s.DeprecatedArgs == nil {
        s.DeprecatedArgs = make(map[string]map[string]string)
    }
    if s.DeprecatedArgs[fieldName] == nil {
        s.DeprecatedArgs[fieldName] = make(map[string]string)
    }
    s.DeprecatedArgs[fieldName][argName] = reason
}

// DeprecateField marks an input field as deprecated with the given reason.
func (io *InputObject) DeprecateField(fieldName, reason string) {
    if io.DeprecatedFields == nil {
        io.DeprecatedFields = make(map[string]string)
    }
    io.DeprecatedFields[fieldName] = reason
}
```

### Change 11: Apply deprecation metadata during schema build

**File: `schemabuilder/output.go`**

After building each method's field in `buildStruct()`, check if any args are marked deprecated and set `IsDeprecated`/`DeprecationReason` on the corresponding `InputField` entries in `field.Args`.

**File: `schemabuilder/input_object.go`**

In `generateObjectParserInner()`, after building input fields for registered input objects, check if any fields are marked deprecated and set the metadata on the corresponding `InputField` entries.

### Change 12: Update the built-in introspection query

**File: `introspection/introspection_query.go`**

Add `isDeprecated` and `deprecationReason` to the `InputValue` fragment:
```graphql
fragment InputValue on __InputValue {
    name
    description
    type { ...TypeRef }
    defaultValue
    isDeprecated
    deprecationReason
}
```

Update `fields` and `inputFields` to use `includeDeprecated: true`:
```graphql
fields(includeDeprecated: true) {
    name
    description
    args(includeDeprecated: true) {
        ...InputValue
    }
    ...
}
inputFields(includeDeprecated: true) {
    ...InputValue
}
```

---

## 4. Summary of Files Changed

| File | Type of Change |
|------|---------------|
| `graphql/types.go` | Add `InputField` struct; change `Field.Args` to `map[string]*InputField`; change `InputObject.InputFields` to `map[string]*InputField` |
| `graphql/validate.go` | Update all reads of `field.Args` and `inputObject.InputFields` to access `.Type` |
| `graphql/execute.go` | Update `collectTypes` reads if any (none expected — execution uses `Field.Resolve`, not `Args` directly) |
| `schemabuilder/types.go` | Add `DeprecatedArgs` to `Object`, `DeprecatedFields` to `InputObject`, add `DeprecateArg()` and `DeprecateField()` methods |
| `schemabuilder/function.go` | Update `argsTypeMap()` return type to `map[string]*graphql.InputField` |
| `schemabuilder/input_object.go` | Wrap types in `&graphql.InputField{...}` when building `InputFields`; apply deprecation metadata from `DeprecatedFields` |
| `schemabuilder/output.go` | Apply deprecation metadata from `DeprecatedArgs` after building field args |
| `schemabuilder/schema.go` | Propagate `DeprecatedArgs` in `copyObject()` for `Clone()` |
| `introspection/introspection.go` | Add `IsDeprecated`/`DeprecationReason` to `InputValue`; register new field funcs; add `includeDeprecated` to `args`/`inputFields`; add `@deprecated` directive with new locations; add new location constants and enum entries |
| `introspection/introspection_query.go` | Add `isDeprecated`/`deprecationReason` to `InputValue` fragment; add `includeDeprecated: true` to `args` and `inputFields` |

---

## 5. Backward Compatibility

- **`graphql.Field.Args` type change** — This is the most impactful change. Any external code directly accessing `field.Args` will need to be updated. However, since Jaal is a framework (users register via `FieldFunc`, not by constructing `graphql.Field` directly), the impact is limited to internal code and introspection.
- **`graphql.InputObject.InputFields` type change** — Same as above. Internal-only impact.
- **`Object.FieldFunc()` signature is unchanged** — Existing user code continues to work.
- **`InputObject.FieldFunc()` signature is unchanged** — Existing user code continues to work.
- **New methods are additive** — `DeprecateArg()` and `DeprecateField()` are new methods that don't affect existing code.
- **Introspection gains new fields** — `isDeprecated` and `deprecationReason` on `__InputValue` default to `false`/`null`, which is correct for non-deprecated items.
- **`includeDeprecated` defaults to `false`** via `*bool` nil check — when not provided, deprecated items are excluded (matching spec default). Existing introspection queries that don't pass `includeDeprecated` will see no change since nothing is deprecated by default.
- **`@deprecated` directive appears in `__schema.directives`** — Additive, no breakage.
- **New directive location enum values** — Additive to the enum.

---

## 6. Test Plan

### Test 1: `DeprecateArg` marks argument as deprecated in introspection
- Register a query field with an optional argument
- Call `DeprecateArg` on the object for that field/arg
- Build schema, add introspection
- Query `__type(name: "Query") { fields { name args(includeDeprecated: true) { name isDeprecated deprecationReason } } }`
- Assert the deprecated arg has `isDeprecated: true` and the correct `deprecationReason`

### Test 2: `DeprecateField` marks input field as deprecated in introspection
- Register an input object with an optional field
- Call `DeprecateField` on the input object
- Build schema, add introspection
- Query `__type(name: "MyInput") { inputFields(includeDeprecated: true) { name isDeprecated deprecationReason } }`
- Assert the deprecated field has `isDeprecated: true` and the correct `deprecationReason`

### Test 3: `includeDeprecated: false` (default) excludes deprecated args
- Same setup as Test 1
- Query `__type(name: "Query") { fields { name args { name } } }` (no `includeDeprecated`)
- Assert the deprecated arg is **not** in the results

### Test 4: `includeDeprecated: false` (default) excludes deprecated input fields
- Same setup as Test 2
- Query `__type(name: "MyInput") { inputFields { name } }` (no `includeDeprecated`)
- Assert the deprecated field is **not** in the results

### Test 5: Non-deprecated args/input fields have `isDeprecated: false`
- Query args and input fields that are NOT deprecated
- Assert `isDeprecated: false` and `deprecationReason: null`

### Test 6: `@deprecated` directive appears in `__schema.directives` with correct locations
- Query `__schema { directives { name locations } }`
- Assert `deprecated` directive exists with locations `["FIELD_DEFINITION", "ARGUMENT_DEFINITION", "INPUT_FIELD_DEFINITION", "ENUM_VALUE"]`

### Test 7: Deprecated args/input fields still execute normally
- Register a field with a deprecated argument
- Execute a query that uses the deprecated argument
- Assert the query executes successfully and returns correct data

### Test 8: Backward compatibility — schemas without deprecation work unchanged
- Build a schema with no `DeprecateArg`/`DeprecateField` calls
- All existing introspection queries work
- All args/input fields have `isDeprecated: false`

### Test 9: `ComputeSchemaJSON` includes deprecation fields in `InputValue`
- Register schema with deprecated arg and input field
- Call `ComputeSchemaJSON`
- Parse result and verify `isDeprecated`/`deprecationReason` appear correctly

### Test 10: Existing tests pass without modification (beyond expected introspection changes)
- The existing `TestIntrospectionForInterface` "Test Directives" test will need updating to include the `@deprecated` directive
- The existing "Test __Schema" test will need the `@deprecated` directive in the expected directives list

---

## 7. Dependency Graph & Implementation Order

```
Phase 1: Core Data Model
  1. graphql/types.go — Add InputField struct, change Args/InputFields types
  2. graphql/validate.go — Update all reads to use .Type
  3. schemabuilder/function.go — Update argsTypeMap return type
  4. schemabuilder/input_object.go — Wrap types in InputField

Phase 2: Introspection Data
  5. introspection/introspection.go — Add deprecation to InputValue, includeDeprecated args,
     @deprecated directive, new location constants
  6. introspection/introspection_query.go — Update InputValue fragment

Phase 3: User-Facing API
  7. schemabuilder/types.go — Add DeprecateArg, DeprecateField
  8. schemabuilder/output.go — Apply DeprecatedArgs during build
  9. schemabuilder/input_object.go — Apply DeprecatedFields during build
  10. schemabuilder/schema.go — Update Clone() for new fields

Phase 4: Tests
  11. Update existing introspection tests
  12. Add new deprecation-specific tests
```

---

## 8. Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Changing `map[string]Type` to `map[string]*InputField` is a breaking internal API change | All usages are internal to the framework; users interact via `FieldFunc`/`InputObject` APIs which don't change |
| `collectTypes` in introspection iterates `Args` and `InputFields` | Straightforward mechanical update to access `.Type` |
| Existing tests rely on exact introspection output | Tests will be updated as part of implementation |
| `Clone()` in schema.go copies objects | Must be updated to copy `DeprecatedArgs` map |
