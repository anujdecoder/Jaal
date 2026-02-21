# Implementation Plan: `@specifiedBy` Directive

## 1. How `@specifiedBy` Works (Spec Explanation)

### Definition
```graphql
directive @specifiedBy(url: String!) on SCALAR
```

The `@specifiedBy` directive is a **built-in** directive introduced in the GraphQL October 2021 specification. It serves a single purpose: to attach a **scalar specification URL** to a **custom scalar type**, pointing to a human-readable document that describes the scalar's data format, serialization, and coercion rules.

### Rules
1. **Applies only to custom scalar types** — It must **not** be applied to the five built-in scalars (`Int`, `Float`, `String`, `Boolean`, `ID`).
2. **Has a single required argument** — `url: String!` — a non-nullable string containing a URL to the scalar's specification.
3. **The URL should be stable** — Once set, the URL should not change, as tooling may rely on it.
4. **It is a schema-level directive** — It does not affect execution. It is purely metadata for introspection and tooling.

### Example SDL Usage
```graphql
scalar UUID @specifiedBy(url: "https://tools.ietf.org/html/rfc4122")
scalar DateTime @specifiedBy(url: "https://scalars.graphql.org/andimarek/date-time")
```

### Introspection Exposure
The directive's URL is exposed via the `__Type` introspection type through a new field:
```graphql
type __Type {
  # ... existing fields ...
  specifiedByURL: String   # New field
}
```

**Behavior by type kind:**
- For **custom scalars** with `@specifiedBy`: returns the URL string (e.g., `"https://tools.ietf.org/html/rfc4122"`)
- For **built-in scalars** (`Int`, `Float`, `String`, `Boolean`, `ID`): **must return `null`**
- For **all other type kinds** (OBJECT, INTERFACE, UNION, ENUM, INPUT_OBJECT, LIST, NON_NULL): **must return `null`**

Additionally, `@specifiedBy` itself must appear in the `__schema.directives` list alongside `@skip`, `@include`, and `@deprecated`.

### Example Introspection Query & Response
```graphql
{
  __type(name: "UUID") {
    name
    kind
    specifiedByURL
  }
}
```
Response:
```json
{
  "data": {
    "__type": {
      "name": "UUID",
      "kind": "SCALAR",
      "specifiedByURL": "https://tools.ietf.org/html/rfc4122"
    }
  }
}
```

For a built-in scalar:
```graphql
{
  __type(name: "String") {
    name
    kind
    specifiedByURL
  }
}
```
Response:
```json
{
  "data": {
    "__type": {
      "name": "String",
      "kind": "SCALAR",
      "specifiedByURL": null
    }
  }
}
```

---

## 2. Current State in Jaal (What Exists Today)

### Scalar Registration Flow
1. **`schemabuilder/types.go`** — `RegisterScalar(typ reflect.Type, name string, uf UnmarshalFunc) error`
   - Adds the Go type → GraphQL name mapping to the package-level `scalars` map
   - Adds the arg parser to the package-level `scalarArgParsers` map
   - Has **no concept of a URL** or any metadata beyond name and unmarshal function

2. **`schemabuilder/build.go`** — `getType()` and `getScalar()`
   - When building the schema, `getType()` calls `getScalar()` to look up the scalar name
   - Creates `&graphql.Scalar{Type: typeName}` — a fresh `graphql.Scalar` instance with only the `Type` (name) field
   - **No URL is carried through**

3. **`graphql/types.go`** — `Scalar` struct
   ```go
   type Scalar struct {
       Type      string
       Unwrapper func(interface{}) (interface{}, error)
   }
   ```
   - Only has `Type` (the name) and `Unwrapper` — **no `SpecifiedByURL` field**

4. **`introspection/introspection.go`** — `registerType()` for `__Type`
   - Has field funcs for `kind`, `name`, `description`, `interfaces`, `possibleTypes`, `inputFields`, `fields`, `ofType`, `enumValues`
   - **No `specifiedByURL` field func**
   - The `description` field func returns `""` for scalars (not even checking scalar descriptions)

5. **`introspection/introspection.go`** — `__schema.directives`
   - Returns `[]Directive{includeDirective, skipDirective}` — **only two directives**
   - **No `@specifiedBy` directive** and **no `@deprecated` directive** in the list

6. **`introspection/introspection_query.go`** — The built-in introspection query
   - Does **not** include `specifiedByURL` in the `FullType` fragment

### Key Observation: Scalar Instances Are Not Shared
In `getType()` (build.go lines 38-44), every call creates a **new** `graphql.Scalar` instance:
```go
if typeName, ok := getScalar(nodeType); ok {
    return &graphql.NonNull{Type: &graphql.Scalar{Type: typeName}}, nil
}
```
However, `collectTypes()` in introspection deduplicates by name — only the **first** `*graphql.Scalar` encountered for a given name is kept. This means we need to ensure the `SpecifiedByURL` is set on the scalar instance that ends up in the introspection type map.

The cleanest approach: store the URL in a **package-level map** alongside the existing `scalars` map, and populate it onto every `graphql.Scalar` instance at creation time.

---

## 3. Implementation Plan — Exact Code Changes

### Change 1: Add `SpecifiedByURL` field to `graphql.Scalar` struct

**File: `graphql/types.go`**

```go
// BEFORE (lines 20-23):
type Scalar struct {
    Type      string
    Unwrapper func(interface{}) (interface{}, error)
}

// AFTER:
type Scalar struct {
    Type           string
    Unwrapper      func(interface{}) (interface{}, error)
    SpecifiedByURL string
}
```

This is a purely additive change. The zero value `""` means "no URL specified", which is correct for built-in scalars.

### Change 2: Add a package-level map to store scalar URLs and update `RegisterScalar`

**File: `schemabuilder/types.go`**

**2a. Add a new package-level map** (near the existing `scalars` map, but in this file since `RegisterScalar` lives here):

```go
// scalarSpecifiedByURLs maps Go reflect.Type to the specifiedBy URL for custom scalars.
var scalarSpecifiedByURLs = map[reflect.Type]string{}
```

**2b. Add a new registration function** that accepts the URL. We keep the old `RegisterScalar` signature intact for backward compatibility and add a new function:

```go
// RegisterScalarWithURL is used to register custom scalars with a specifiedBy URL.
// It behaves identically to RegisterScalar but additionally records the scalar
// specification URL per the GraphQL @specifiedBy directive (Oct 2021 spec).
func RegisterScalarWithURL(typ reflect.Type, name string, url string, uf UnmarshalFunc) error {
    if err := RegisterScalar(typ, name, uf); err != nil {
        return err
    }
    if url != "" {
        scalarSpecifiedByURLs[typ] = url
    }
    return nil
}
```

**2c. Add a getter function** so the `build.go` can look up the URL:

```go
// GetScalarSpecifiedByURL returns the specifiedBy URL for a scalar Go type, if registered.
func GetScalarSpecifiedByURL(typ reflect.Type) string {
    return scalarSpecifiedByURLs[typ]
}
```

### Change 3: Propagate `SpecifiedByURL` when creating `graphql.Scalar` instances

**File: `schemabuilder/build.go`**

In the `getType()` function, where `graphql.Scalar` instances are created, we need to populate `SpecifiedByURL`. There are two places:

**3a. Non-pointer scalar (line 38-39):**
```go
// BEFORE:
if typeName, ok := getScalar(nodeType); ok {
    return &graphql.NonNull{Type: &graphql.Scalar{Type: typeName}}, nil
}

// AFTER:
if typeName, ok := getScalar(nodeType); ok {
    return &graphql.NonNull{Type: &graphql.Scalar{Type: typeName, SpecifiedByURL: GetScalarSpecifiedByURL(nodeType)}}, nil
}
```

**3b. Pointer-to-scalar (lines 41-44):**
```go
// BEFORE:
if nodeType.Kind() == reflect.Ptr {
    if typeName, ok := getScalar(nodeType.Elem()); ok {
        return &graphql.Scalar{Type: typeName}, nil
    }
}

// AFTER:
if nodeType.Kind() == reflect.Ptr {
    if typeName, ok := getScalar(nodeType.Elem()); ok {
        return &graphql.Scalar{Type: typeName, SpecifiedByURL: GetScalarSpecifiedByURL(nodeType.Elem())}, nil
    }
}
```

**3c. In `getScalarArgParser` (file: `schemabuilder/input.go`, line 132):**
This creates `graphql.Scalar` for argument types. This is used for introspection of argument types, so it should also carry the URL:
```go
// BEFORE:
return argParser, &graphql.Scalar{Type: name}, true

// AFTER:
return argParser, &graphql.Scalar{Type: name, SpecifiedByURL: GetScalarSpecifiedByURL(typ)}, true
```

### Change 4: Add `specifiedByURL` field func to `__Type` introspection

**File: `introspection/introspection.go`**

In the `registerType()` method, add a new field func after the existing `description` field func (after line 222):

```go
object.FieldFunc("specifiedByURL", func(t Type) *string {
    switch t := t.Inner.(type) {
    case *graphql.Scalar:
        if t.SpecifiedByURL != "" {
            return &t.SpecifiedByURL
        }
        return nil
    default:
        return nil
    }
})
```

We return `*string` (nullable) because:
- For custom scalars with a URL → returns the URL string
- For built-in scalars / all other types → returns `nil` (null in GraphQL)

### Change 5: Add `@specifiedBy` to the `__schema.directives` list

**File: `introspection/introspection.go`**

**5a. Define the directive constant** (near the existing `includeDirective` and `skipDirective` vars, around line 457):

```go
var specifiedByDirective = Directive{
    Name:        "specifiedBy",
    Description: "Exposes a URL that specifies the behavior of this scalar.",
    Locations: []DirectiveLocation{
        SCALAR,
    },
    Args: []InputValue{
        {
            Name:        "url",
            Type:        Type{Inner: &graphql.NonNull{Type: &graphql.Scalar{Type: "String"}}},
            Description: "The URL that specifies the behavior of this scalar.",
        },
    },
}
```

**5b. Add `SCALAR` to the `DirectiveLocation` enum** (currently missing). In the const block (line 22-30) and the enum registration (line 124-132):

Constants (add `SCALAR`):
```go
const (
    QUERY               DirectiveLocation = "QUERY"
    MUTATION                              = "MUTATION"
    FIELD                                 = "FIELD"
    FRAGMENT_DEFINITION                   = "FRAGMENT_DEFINITION"
    FRAGMENT_SPREAD                       = "FRAGMENT_SPREAD"
    INLINE_FRAGMENT                       = "INLINE_FRAGMENT"
    SUBSCRIPTION                          = "SUBSCRIPTION"
    SCALAR              DirectiveLocation = "SCALAR"    // NEW
)
```

Enum registration (add `"SCALAR"` entry):
```go
schema.Enum(DirectiveLocation("QUERY"), map[string]interface{}{
    "QUERY":               DirectiveLocation("QUERY"),
    "MUTATION":            DirectiveLocation("MUTATION"),
    "FIELD":               DirectiveLocation("FIELD"),
    "FRAGMENT_DEFINITION": DirectiveLocation("FRAGMENT_DEFINITION"),
    "FRAGMENT_SPREAD":     DirectiveLocation("FRAGMENT_SPREAD"),
    "INLINE_FRAGMENT":     DirectiveLocation("INLINE_FRAGMENT"),
    "SUBSCRIPTION":        DirectiveLocation("SUBSCRIPTION"),
    "SCALAR":              DirectiveLocation("SCALAR"),    // NEW
})
```

**5c. Add the directive to the `__schema` response** (line 507):

```go
// BEFORE:
Directives: []Directive{includeDirective, skipDirective},

// AFTER:
Directives: []Directive{includeDirective, skipDirective, specifiedByDirective},
```

### Change 6: Update the built-in introspection query to include `specifiedByURL`

**File: `introspection/introspection_query.go`**

In the `FullType` fragment, add `specifiedByURL` after `description`:

```go
// BEFORE:
fragment FullType on __Type {
    kind
    name
    description
    fields(includeDeprecated: true) {

// AFTER:
fragment FullType on __Type {
    kind
    name
    description
    specifiedByURL
    fields(includeDeprecated: true) {
```

---

## 4. Summary of Files Changed

| File | Type of Change |
|------|---------------|
| `graphql/types.go` | Add `SpecifiedByURL string` field to `Scalar` struct |
| `schemabuilder/types.go` | Add `scalarSpecifiedByURLs` map, `RegisterScalarWithURL()` function, `GetScalarSpecifiedByURL()` getter |
| `schemabuilder/build.go` | Pass `SpecifiedByURL` when creating `graphql.Scalar` instances in `getType()` |
| `schemabuilder/input.go` | Pass `SpecifiedByURL` when creating `graphql.Scalar` in `getScalarArgParser()` |
| `introspection/introspection.go` | Add `specifiedByURL` field func on `__Type`, add `SCALAR` directive location, add `specifiedByDirective` var, include it in `__schema.directives` |
| `introspection/introspection_query.go` | Add `specifiedByURL` to `FullType` fragment |

**No new files are created.** All changes are modifications to existing files.

---

## 5. Backward Compatibility

- **`RegisterScalar()` signature is unchanged** — Existing callers continue to work exactly as before. Scalars registered without a URL will have `SpecifiedByURL: ""`, which translates to `null` in introspection.
- **`graphql.Scalar` struct gains a new field** — The zero value `""` is safe. Existing code creating `graphql.Scalar{Type: "..."}` will compile and work correctly (URL defaults to empty/null).
- **Introspection gains a new field** — `specifiedByURL` is a nullable field returning `null` for all existing types. Existing introspection queries that don't request this field are unaffected. Queries that do request it will get `null` for built-in scalars.
- **`__schema.directives` gains a new entry** — The `@specifiedBy` directive appears in the list. This is additive and does not break existing consumers.

---

## 6. Test Plan

### Test 1: `RegisterScalarWithURL` registers URL correctly
**File: `schemabuilder/types_test.go` (new test in existing or new file)**

- Register a custom scalar using `RegisterScalarWithURL` with a URL
- Verify `GetScalarSpecifiedByURL` returns the correct URL for that type
- Verify `GetScalarSpecifiedByURL` returns `""` for a type not registered with a URL
- Verify `RegisterScalarWithURL` with empty URL does not store anything
- Verify `RegisterScalarWithURL` returns error for pointer types (same as `RegisterScalar`)

### Test 2: `specifiedByURL` in introspection for custom scalar WITH URL
**File: `introspection/introspection_test.go` (add test case)**

- Register a custom scalar (e.g., `DateTime`) using `RegisterScalarWithURL` with URL `"https://scalars.graphql.org/andimarek/date-time"`
- Build schema, add introspection
- Execute query:
  ```graphql
  { __type(name: "DateTime") { name kind specifiedByURL } }
  ```
- Assert result:
  ```json
  { "__type": { "name": "DateTime", "kind": "SCALAR", "specifiedByURL": "https://scalars.graphql.org/andimarek/date-time" } }
  ```

### Test 3: `specifiedByURL` returns `null` for built-in scalars
**File: `introspection/introspection_test.go` (add test case)**

- Use an existing schema that has `String` and `ID` scalars
- Execute query:
  ```graphql
  { __type(name: "String") { name kind specifiedByURL } }
  ```
- Assert `specifiedByURL` is `nil`
- Repeat for `"ID"`, `"Boolean"`, `"Int"`, `"Float"`

### Test 4: `specifiedByURL` returns `null` for non-scalar types
**File: `introspection/introspection_test.go` (add test case)**

- Query `__type` for an Object, Enum, Interface, Union, InputObject
- Assert `specifiedByURL` is `nil` for each

### Test 5: `specifiedByURL` returns `null` for custom scalar WITHOUT URL
**File: `introspection/introspection_test.go` (add test case)**

- Register a custom scalar using the existing `RegisterScalar()` (no URL)
- Query `__type` for that scalar
- Assert `specifiedByURL` is `nil`

### Test 6: `@specifiedBy` appears in `__schema.directives`
**File: `introspection/introspection_test.go` (add test case or modify existing "Test Directives")**

- Execute query:
  ```graphql
  {
    __schema {
      directives {
        name
        description
        locations
        args { name type { name kind } }
      }
    }
  }
  ```
- Assert that the result includes a directive with:
  - `name: "specifiedBy"`
  - `locations: ["SCALAR"]`
  - `args` containing `{ name: "url", type: { name: "String", kind: "NON_NULL" } }` (or the NonNull wrapper around String)

### Test 7: `ComputeSchemaJSON` includes `specifiedByURL`
**File: `introspection/introspection_test.go` (modify existing `TestComputeSchemaJSON`)**

- Register a custom scalar with URL in the schema used by `TestComputeSchemaJSON`
- Call `ComputeSchemaJSON`
- Parse the result and verify the custom scalar's type entry includes `specifiedByURL` with the correct value
- Verify built-in scalars have `specifiedByURL: null`

### Test 8: Backward compatibility — existing `RegisterScalar` still works
**File: `schemabuilder/types_test.go`**

- Register a scalar with `RegisterScalar()` (old API)
- Build schema successfully
- Verify introspection shows `specifiedByURL: null` for that scalar
- This ensures no regression

---

## 7. Implementation Order

1. **`graphql/types.go`** — Add `SpecifiedByURL` field to `Scalar` (1 line change)
2. **`schemabuilder/types.go`** — Add map, `RegisterScalarWithURL`, `GetScalarSpecifiedByURL` (~20 lines)
3. **`schemabuilder/build.go`** — Propagate URL in `getType()` (2 lines changed)
4. **`schemabuilder/input.go`** — Propagate URL in `getScalarArgParser()` (1 line changed)
5. **`introspection/introspection.go`** — Add `specifiedByURL` field func, `SCALAR` location, `specifiedByDirective`, update directives list (~25 lines)
6. **`introspection/introspection_query.go`** — Add `specifiedByURL` to fragment (1 line)
7. **Tests** — All test cases described above
8. **Run existing tests** — Ensure no regressions
