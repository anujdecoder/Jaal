# Jaal — GraphQL Spec Compliance Plan (June 2018 → September 2025)

## Executive Summary

Jaal is currently compliant with the **GraphQL June 2018** specification. Two major spec releases have occurred since then:
- **October 2021** — ~100 changes from 35 contributors
- **September 2025** — ~100 changes from 30 contributors (10th anniversary edition)

This document catalogs every specification change, identifies which features Jaal already implements, and lists what needs to be built — ordered by priority.

---

## Part 1: Currently Implemented Features (June 2018 Baseline)

These features are already present in Jaal and are compliant with the June 2018 spec:

| # | Feature | Status | Location in Codebase |
|---|---------|--------|---------------------|
| 1 | **Scalar types** (Int, Float, String, Boolean, ID) | ✅ Implemented | `schemabuilder/build.go`, `schemabuilder/types.go` |
| 2 | **Custom Scalar registration** (with UnmarshalFunc) | ✅ Implemented | `schemabuilder/types.go` — `RegisterScalar()` |
| 3 | **Object types** with field resolvers | ✅ Implemented | `schemabuilder/output.go`, `graphql/types.go` |
| 4 | **Input Object types** | ✅ Implemented | `schemabuilder/input_object.go`, `schemabuilder/input.go` |
| 5 | **Enum types** | ✅ Implemented | `schemabuilder/schema.go` — `Enum()` |
| 6 | **Union types** | ✅ Implemented | `schemabuilder/output.go` — `buildUnionStruct()` |
| 7 | **Interface types** (auto-detected common fields) | ✅ Implemented | `schemabuilder/output.go` — `buildInterfaceStruct()` |
| 8 | **List types** | ✅ Implemented | `schemabuilder/build.go`, `graphql/execute.go` |
| 9 | **Non-Null wrapper types** | ✅ Implemented | `graphql/types.go` — `NonNull` |
| 10 | **Query operations** | ✅ Implemented | `schemabuilder/schema.go` — `Query()` |
| 11 | **Mutation operations** | ✅ Implemented | `schemabuilder/schema.go` — `Mutation()` |
| 12 | **Subscription operations** (via WebSocket + pubsub) | ✅ Implemented | `ws.go`, `schemabuilder/schema.go` — `Subscription()` |
| 13 | **Fragment spreads & inline fragments** | ✅ Implemented | `graphql/parser.go` — `parseSelectionSet()` |
| 14 | **Fragment definitions** | ✅ Implemented | `graphql/parser.go` |
| 15 | **Variables with defaults** | ✅ Implemented | `graphql/parser.go` — variable definition parsing |
| 16 | **`@skip` directive** | ✅ Implemented | `graphql/execute.go` — `shouldIncludeNode()` |
| 17 | **`@include` directive** | ✅ Implemented | `graphql/execute.go` — `shouldIncludeNode()` |
| 18 | **`@deprecated` directive** (on FIELD_DEFINITION, ENUM_VALUE) | ✅ Partial | `introspection/introspection.go` — exposes `isDeprecated`/`deprecationReason` on `__Field` and `__EnumValue` |
| 19 | **`__typename` meta-field** | ✅ Implemented | `graphql/execute.go` — in `executeObject`, `executeUnion`, `executeInterface` |
| 20 | **Introspection** (`__schema`, `__type`, `__Type`, `__Field`, etc.) | ✅ Implemented | `introspection/introspection.go` |
| 21 | **Query validation** (field existence, args, scalar leaf, etc.) | ✅ Implemented | `graphql/validate.go` |
| 22 | **Conflict detection** (same alias, different name/args) | ✅ Implemented | `graphql/parser.go` — `detectConflicts()` |
| 23 | **Cycle & unused fragment detection** | ✅ Implemented | `graphql/parser.go` — `detectCyclesAndUnusedFragments()` |
| 24 | **HTTP POST handler** (application/json) | ✅ Implemented | `http.go` |
| 25 | **Middleware support** | ✅ Implemented | `middleware.go` |
| 26 | **Error handling** (with paths and extensions) | ✅ Implemented | `jerrors/errors.go` |

---

## Part 2: Specs Introduced in October 2021 (Not Yet Implemented)

### Priority 1 — Core Type System Changes

#### 2.1 `@specifiedBy` Directive for Custom Scalars
- **Spec Reference**: §3.5.5 (Custom Scalars), §3.13.4 (@specifiedBy)
- **What**: New built-in directive `directive @specifiedBy(url: String!) on SCALAR` that allows custom scalars to reference an external specification URL (e.g., RFC link for UUID format).
- **Impact on Jaal**:
  - Add `SpecifiedByURL string` field to `graphql.Scalar` struct
  - Extend `RegisterScalar()` API to optionally accept a `specifiedByURL` parameter
  - Expose `specifiedByURL: String` in introspection on `__Type` (returns the URL or null)
  - Register `@specifiedBy` as a built-in directive in introspection directives list
- **Files to modify**: `schemabuilder/types.go`, `graphql/types.go`, `introspection/introspection.go`
- **Priority**: **HIGH** — Required for spec compliance; affects introspection output

#### 2.2 Interfaces Implementing Interfaces
- **Spec Reference**: §3.7 (Interfaces)
- **What**: Interfaces can now implement other interfaces, enabling hierarchical type relationships. Example: `interface Resource implements Node { id: ID!, url: String }`. Validation rules include: no cycles, implementing interface must be a superset of implemented interface fields, covariant return types.
- **Impact on Jaal**:
  - Currently, `graphql.Interface` has `Types map[string]*Object` and `Fields map[string]*Field` but no `Interfaces` field
  - Add `Interfaces map[string]*Interface` to `graphql.Interface` struct
  - Update `buildInterfaceStruct()` in `schemabuilder/output.go` to detect and wire up interface-implements-interface relationships
  - Add validation: no cycles in interface implementation, field compatibility checks
  - Update introspection: `__Type.interfaces` should return interfaces for Interface types (currently only returns for Object types)
- **Files to modify**: `graphql/types.go`, `schemabuilder/output.go`, `graphql/validate.go`, `introspection/introspection.go`
- **Priority**: **HIGH** — Major type system feature; needed for complex schema hierarchies

#### 2.3 Repeatable Directives
- **Spec Reference**: §3.13 (Directives)
- **What**: Directives can be declared `repeatable`, allowing multiple instances at the same schema location. Order is semantically significant. Non-repeatable directives must appear at most once per location.
- **Impact on Jaal**:
  - Jaal currently has minimal directive infrastructure — `@skip`, `@include`, `@deprecated` are hard-coded in introspection and execution
  - Need to add a formal `DirectiveDefinition` type with `IsRepeatable bool` field
  - Add `isRepeatable: Boolean!` to `__Directive` introspection type
  - Update validation to enforce uniqueness for non-repeatable directives
- **Files to modify**: `graphql/types.go`, `introspection/introspection.go`
- **Priority**: **HIGH** — Foundation for directive extensibility; affects introspection

#### 2.4 Input Object Circular Reference Validation
- **Spec Reference**: §3.10 (Input Objects — Circular References)
- **What**: Explicit rules: input objects may reference themselves (directly or indirectly), but at least one field in any cycle must be nullable or a List type. Non-null singular cycles are forbidden.
- **Impact on Jaal**:
  - Currently Jaal does not validate circular references in input objects at schema build time
  - Add schema validation in `schemabuilder/build.go` or `schemabuilder/input_object.go` that detects non-null singular cycles
- **Files to modify**: `schemabuilder/input_object.go`, `schemabuilder/build.go`
- **Priority**: **MEDIUM** — Important for schema correctness but unlikely to cause runtime failures in practice

### Priority 2 — Execution & Validation Refinements

#### 2.5 Subscription Execution Refinements
- **Spec Reference**: §6.2 (Executing Subscriptions)
- **What**: Detailed algorithms for `CreateSourceEventStream`, `ResolveFieldEventStream`, and `MapSourceToResponseEvent`. Emphasis on delivery-agnostic transports and proper error handling during event stream creation.
- **Impact on Jaal**:
  - Jaal has a working subscription system via WebSocket + pubsub (`ws.go`), but the internal execution model doesn't formally follow the spec algorithms
  - Review and align `ws.go` with the spec's subscription execution model
  - Ensure proper error handling during event stream creation and propagation
- **Files to modify**: `ws.go`
- **Priority**: **MEDIUM** — Current implementation works but may not handle edge cases per spec

#### 2.6 Stricter Field Selection Merging Validation
- **Spec Reference**: §5.3.2 (Field Selection Merging)
- **What**: Refined rules for when fields with the same alias can be merged. Beyond matching name and args, fields must also have compatible return types considering their parent type context.
- **Impact on Jaal**:
  - `detectConflicts()` in `graphql/parser.go` checks name and args but does **not** check return type compatibility
  - Need to enhance conflict detection to consider return types when validating field merging
- **Files to modify**: `graphql/parser.go`
- **Priority**: **MEDIUM** — Affects query validation correctness

#### 2.7 Enhanced Variable Usage Validation
- **Spec Reference**: §5.8 (Variables)
- **What**: Stricter rules on variable usage within fragments, including checking that variables used in fragments are defined in the operation and that their types are compatible with the expected input types.
- **Impact on Jaal**:
  - Currently Jaal resolves variables during parsing but doesn't perform comprehensive variable-type compatibility checks
  - Need to add validation that variable types match their usage positions
- **Files to modify**: `graphql/parser.go`, `graphql/validate.go`
- **Priority**: **MEDIUM** — Affects query validation correctness

### Priority 3 — Introspection & Metadata Enhancements

#### 2.8 Expanded Directive Locations
- **Spec Reference**: §3.13 (Directives — Directive Locations)
- **What**: Full set of directive locations including both executable and type system locations: SCHEMA, SCALAR, OBJECT, FIELD_DEFINITION, ARGUMENT_DEFINITION, INTERFACE, UNION, ENUM, ENUM_VALUE, INPUT_OBJECT, INPUT_FIELD_DEFINITION, VARIABLE_DEFINITION.
- **Impact on Jaal**:
  - Jaal's `DirectiveLocation` enum in `introspection/introspection.go` only has 7 values (QUERY, MUTATION, FIELD, FRAGMENT_DEFINITION, FRAGMENT_SPREAD, INLINE_FRAGMENT, SUBSCRIPTION)
  - Need to add all type system directive locations to the enum and the enum registration
- **Files to modify**: `introspection/introspection.go`
- **Priority**: **MEDIUM** — Needed for complete introspection

#### 2.9 Improved Error Formatting
- **Spec Reference**: §7.1.2 (Errors — Extensions)
- **What**: Clarified that errors should have an `extensions` map for custom error metadata. The `path` field (not `paths`) should use integers for list indices and strings for field names.
- **Impact on Jaal**:
  - Jaal's `jerrors.Error` has `Paths []string` — the spec field is `path` (singular), and list indices should be integers, not strings
  - `executeList` in `graphql/execute.go` uses `fmt.Sprint(i)` which produces string "0", "1" etc. — spec requires actual integers
  - Need to change `Paths` to `Path` and use `[]interface{}` (mixed string/int) instead of `[]string`
- **Files to modify**: `jerrors/errors.go`, `graphql/execute.go`
- **Priority**: **LOW** — Mostly cosmetic; current implementation works

#### 2.10 Parser Improvements (Lookahead Restrictions)
- **Spec Reference**: §2 (Language — Lexical Grammar)
- **What**: Lookahead restrictions combined with greedy lexer eliminate grammar ambiguities for better error messages.
- **Impact on Jaal**:
  - Jaal uses `github.com/graphql-go/graphql` for parsing — this is an upstream concern
  - Need to verify the dependency handles these correctly, or upgrade to a newer version
- **Files to modify**: Potentially `go.mod` (dependency upgrade)
- **Priority**: **LOW** — Depends on upstream parser library

---

## Part 3: Specs Introduced in September 2025 (Not Yet Implemented)

### Priority 1 — Major New Features

#### 3.1 OneOf Input Objects (`@oneOf` Directive)
- **Spec Reference**: §3.10.1 (OneOf Input Objects), §3.13.5 (@oneOf)
- **What**: New built-in directive `directive @oneOf on INPUT_OBJECT`. Marks an input object where exactly one field must be provided and non-null. All fields must be nullable with no defaults. Introspection exposes `isOneOf: Boolean` on `__Type`.
- **Example**:
  ```graphql
  input UserUniqueCondition @oneOf {
    id: ID
    username: String
    organizationAndEmail: OrganizationAndEmailInput
  }
  ```
- **Impact on Jaal**:
  - Add `IsOneOf bool` field to `graphql.InputObject`
  - Add `@oneOf` directive to built-in directives
  - Implement OneOf input coercion rules: exactly one non-null field, error otherwise
  - Add schema validation: OneOf fields must be nullable, no default values
  - Add `isOneOf: Boolean` to `__Type` introspection
  - Extend `InputObject` registration API in schemabuilder (e.g., `schema.OneOfInputObject(...)`)
- **Files to modify**: `graphql/types.go`, `schemabuilder/schema.go`, `schemabuilder/types.go`, `schemabuilder/input_object.go`, `schemabuilder/input.go`, `introspection/introspection.go`
- **Priority**: **HIGH** — Major new feature for input polymorphism

#### 3.2 Expanded `@deprecated` Directive (ARGUMENT_DEFINITION, INPUT_FIELD_DEFINITION)
- **Spec Reference**: §3.13.1 (@deprecated)
- **What**: `@deprecated` now applies to `ARGUMENT_DEFINITION` and `INPUT_FIELD_DEFINITION` in addition to `FIELD_DEFINITION` and `ENUM_VALUE`. The `reason` argument is now `String!` (non-nullable) with default `"No longer supported"`. Cannot deprecate required (non-null without default) input fields or arguments.
- **Impact on Jaal**:
  - Add `IsDeprecated bool` and `DeprecationReason string` to `InputValue` introspection struct
  - Register `isDeprecated` and `deprecationReason` field funcs on `__InputValue`
  - Update `__Directive` introspection for `@deprecated` to include ARGUMENT_DEFINITION and INPUT_FIELD_DEFINITION locations
  - Provide API for marking input fields and arguments as deprecated
- **Files to modify**: `introspection/introspection.go`, `schemabuilder/types.go`, `graphql/types.go`
- **Priority**: **HIGH** — Important for schema evolution

#### 3.3 Schema Coordinates
- **Spec Reference**: §2.14 (Schema Coordinates)
- **What**: Human-readable, unique strings identifying schema elements. Formats:
  - Type: `Business`
  - Field: `Business.name`
  - Input field: `SearchCriteria.filter`
  - Enum value: `SearchFilter.OPEN_NOW`
  - Field argument: `Query.searchBusiness(criteria:)`
  - Directive: `@private`
  - Directive argument: `@private(scope:)`
- **Impact on Jaal**:
  - Add a schema coordinates utility module
  - Implement coordinate generation for all schema elements
  - Use in error messages and logging for better developer experience
  - Not part of introspection; purely a utility format
- **Files to modify**: New file (e.g., `graphql/coordinates.go`)
- **Priority**: **MEDIUM** — Useful for tooling and error messages but not required for execution

### Priority 2 — Language & Parser Changes

#### 3.4 Full Unicode Range Support
- **Spec Reference**: §2.1 (Source Text), §2.4.7 (String Value)
- **What**: Full Unicode scalar value support beyond BMP (U+10000–U+10FFFF). New variable-width escape sequences `\u{XXXXX}` for supplementary characters. Legacy surrogate pairs `\uD83D\uDCA9` allowed but discouraged.
- **Impact on Jaal**:
  - Jaal delegates parsing to `github.com/graphql-go/graphql` — need to check if it handles `\u{XXXXX}` escapes
  - May need to upgrade or patch the parser dependency, or switch to a spec-compliant parser
  - Ensure string values can contain supplementary Unicode characters
- **Files to modify**: Potentially `go.mod` (dependency upgrade), or custom parser patches in `graphql/parser.go`
- **Priority**: **MEDIUM** — Important for internationalization

#### 3.5 Descriptions on Executable Definitions
- **Spec Reference**: §2.3 (Document — Executable Definitions)
- **What**: Operations (queries, mutations, subscriptions) and fragment definitions can now have triple-quoted string descriptions (block strings). Ignored at execution but useful for tooling and AI.
- **Example**:
  ```graphql
  """Fetch hero status by ID or name."""
  query HeroStatus($id: ID!) {
    hero(id: $id) { name status }
  }
  ```
- **Impact on Jaal**:
  - Add `Description string` field to `graphql.Query` struct
  - Add `Description string` field to `graphql.FragmentDefinition` struct
  - Parse descriptions from operation/fragment definitions (requires parser support)
- **Files to modify**: `graphql/types.go`, `graphql/parser.go`
- **Priority**: **LOW** — Informational only; no execution impact

### Priority 3 — Introspection Updates

#### 3.6 `includeDeprecated` Argument Now Non-Nullable
- **Spec Reference**: §4.2 (Introspection)
- **What**: The `includeDeprecated` argument on `fields`, `enumValues`, and `inputFields` introspection queries is now `Boolean!` (non-nullable) with default `false`. When `false`, deprecated items should be filtered out.
- **Impact on Jaal**:
  - Currently `includeDeprecated` is `*bool` (nullable pointer) in Jaal's introspection field funcs
  - Change to `bool` (non-nullable) with default behavior
  - Implement actual filtering: when `includeDeprecated: false`, exclude deprecated fields/enum values/input fields from results
  - Currently the filtering is not implemented — all items are always returned regardless of the argument
- **Files to modify**: `introspection/introspection.go`
- **Priority**: **MEDIUM** — Affects introspection correctness

---

## Part 4: Implementation Priority Order

Based on impact, dependency relationships, and spec compliance importance:

### Phase 1: Directive Foundation (October 2021 Core)
| Order | Item | Ref | Priority | Effort |
|-------|------|-----|----------|--------|
| 1 | Repeatable Directives | 2.3 | HIGH | Medium |
| 2 | `@specifiedBy` Directive | 2.1 | HIGH | Low |
| 3 | Expanded Directive Locations in Introspection | 2.8 | MEDIUM | Low |

### Phase 2: Type System (October 2021 Advanced)
| Order | Item | Ref | Priority | Effort |
|-------|------|-----|----------|--------|
| 4 | Interfaces Implementing Interfaces | 2.2 | HIGH | High |
| 5 | Input Object Circular Reference Validation | 2.4 | MEDIUM | Medium |

### Phase 3: Validation & Execution (October 2021)
| Order | Item | Ref | Priority | Effort |
|-------|------|-----|----------|--------|
| 6 | Stricter Field Selection Merging | 2.6 | MEDIUM | Medium |
| 7 | Enhanced Variable Usage Validation | 2.7 | MEDIUM | Medium |
| 8 | Subscription Execution Refinements | 2.5 | MEDIUM | Medium |

### Phase 4: September 2025 Major Features
| Order | Item | Ref | Priority | Effort |
|-------|------|-----|----------|--------|
| 9 | Expanded `@deprecated` (args + input fields) | 3.2 | HIGH | Medium |
| 10 | OneOf Input Objects (`@oneOf`) | 3.1 | HIGH | High |
| 11 | `includeDeprecated` Non-Nullable + Filtering | 3.6 | MEDIUM | Low |

### Phase 5: September 2025 Tooling & Language
| Order | Item | Ref | Priority | Effort |
|-------|------|-----|----------|--------|
| 12 | Schema Coordinates | 3.3 | MEDIUM | Medium |
| 13 | Full Unicode Range Support | 3.4 | MEDIUM | Medium-High |
| 14 | Descriptions on Executable Definitions | 3.5 | LOW | Low |

### Phase 6: Infrastructure & Polish
| Order | Item | Ref | Priority | Effort |
|-------|------|-----|----------|--------|
| 15 | Error Formatting Alignment (`path` field) | 2.9 | LOW | Low |
| 16 | Parser Dependency Upgrade | 2.10 | LOW | Variable |

---

## Part 5: Dependency Graph

```
Phase 1: Directive Foundation
  ├── 2.3 Repeatable Directives
  │     ├── 2.1 @specifiedBy (uses directive system)
  │     └── 2.8 Expanded Directive Locations
  │
Phase 2: Type System
  ├── 2.2 Interfaces Implementing Interfaces (standalone)
  └── 2.4 Input Object Circular Ref Validation
  │
Phase 3: Validation & Execution
  ├── 2.6 Field Selection Merging (standalone)
  ├── 2.7 Variable Usage Validation (standalone)
  └── 2.5 Subscription Refinements (standalone)
  │
Phase 4: Sept 2025 Major Features
  ├── 3.2 Expanded @deprecated
  │     └── depends on: 2.3 (directive locations), 2.8 (locations enum)
  ├── 3.1 OneOf Input Objects
  │     └── depends on: 2.3 (directive system), 2.4 (input validation)
  └── 3.6 includeDeprecated filtering
        └── depends on: 3.2 (deprecation on input values)
  │
Phase 5: Tooling & Language
  ├── 3.3 Schema Coordinates (standalone)
  ├── 3.4 Full Unicode (depends on parser upgrade 2.10)
  └── 3.5 Descriptions on Executables (depends on parser upgrade 2.10)
  │
Phase 6: Polish
  ├── 2.9 Error Formatting (standalone)
  └── 2.10 Parser Upgrade (standalone, enables 3.4 + 3.5)
```

---

## Part 6: Files Impact Summary

| File | Changes Needed |
|------|---------------|
| `graphql/types.go` | Add `SpecifiedByURL` to `Scalar`, `IsOneOf` to `InputObject`, `Interfaces` to `Interface`, formal `DirectiveDefinition` type |
| `graphql/parser.go` | Unicode escapes, descriptions on executables, enhanced variable validation |
| `graphql/validate.go` | Field selection merging, variable usage, OneOf coercion validation |
| `graphql/execute.go` | Subscription execution refinements |
| `schemabuilder/types.go` | `RegisterScalar` with specifiedByURL, OneOf support, deprecation on input fields/args |
| `schemabuilder/schema.go` | OneOf InputObject registration API |
| `schemabuilder/build.go` | Interface-implements-interface resolution |
| `schemabuilder/output.go` | Interface hierarchy building, interface-implements-interface wiring |
| `schemabuilder/input_object.go` | Circular reference validation, OneOf field constraints |
| `schemabuilder/input.go` | OneOf input coercion rules |
| `introspection/introspection.go` | `specifiedByURL`, `isRepeatable`, `isOneOf`, expanded directive locations, `isDeprecated`/`deprecationReason` on `__InputValue`, `includeDeprecated` filtering, `@specifiedBy` and `@oneOf` in directives list |
| `jerrors/errors.go` | Rename `Paths` → `Path`, use `[]interface{}` for mixed string/int path segments |
| `ws.go` | Subscription execution algorithm alignment |
| `go.mod` | Potential parser dependency upgrade |
| **New: `graphql/coordinates.go`** | Schema Coordinates utility |

---

## Part 7: Testing Strategy

Each feature implementation should include:

1. **Unit tests** for the new type/validation/execution logic
2. **Integration tests** (end-to-end query execution with the new feature)
3. **Introspection tests** verifying new fields appear correctly in introspection queries
4. **Negative tests** (invalid schemas should fail to build, invalid queries should fail validation)
5. **Backward compatibility tests** ensuring existing schemas and queries continue to work unchanged

### Key Test Files to Update/Create
- `graphql/execute_test.go` — execution of new features
- `graphql/parser_test.go` — parser changes (Unicode, descriptions)
- `graphql/validate_test.go` — new validation rules (could be created)
- `graphql/union_test.go` — already exists, may need updates
- `introspection/introspection_test.go` — new introspection fields
- `graphql/clone_test.go` — ensure cloning works with new fields
- New test files for OneOf, interface hierarchies, schema coordinates
