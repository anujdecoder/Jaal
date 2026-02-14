# Implementation Plan for @specifiedBy Directive (Post-June 2018 Spec Feature)

**Note for Review:** This plan is prepared based on code analysis (no changes made yet). It details the @specifiedBy directive per GraphQL spec, required code changes in Jaal (focusing on non-breaking additions for compliance), and tests. After review/approval, we can proceed to implement (using tools like search_replace, terminal for go test, etc.). Priorities: Minimal impact on existing scalar reg/introspection; backward compat for built-ins/custom scalars without URL.

## How @specifiedBy Directive is Expected to Work (Per Spec)
From GraphQL Oct 2021 Working Draft (and later drafts up to 2025):
- **Definition**: Built-in directive (like @include/@skip):
  ```
  """
  Exposes a URL that specifies the behaviour of this scalar.
  """
  directive @specifiedBy(
    """
    The URL that specifies the behaviour of this scalar.
    """
    url: String!
  ) on SCALAR
  ```
  - Applies **only to SCALAR** type definitions in schema (e.g., custom scalars like DateTime).
  - Argument: `url: String!` (required, non-null string URL pointing to external spec, e.g., "https://tools.ietf.org/html/rfc3339" for DateTime or ISO8601).
- **Purpose**: Informational/documentation only (no runtime/execution effect). Helps clients/tools understand scalar serialization/validation (e.g., for custom scalars not in core spec: Int/Float/String/Boolean/ID).
  - Example schema SDL:
    ```
    scalar DateTime @specifiedBy(url: "https://www.w3.org/TR/NOTE-datetime")
    ```
  - In introspection (__Type for SCALAR kind):
    - Adds `specifiedByURL: String` field (was `specifiedByUrl` in some early drafts; camelCase per spec).
    - Returns the URL if set; null/omitted otherwise (for built-in scalars).
  - Validation rules:
    - Must only be used on scalars.
    - URL arg must be provided and valid string.
    - Cannot be repeated (unless repeatable directives added later).
    - Appears in __Schema.directives list.
- **Usage in Jaal Context**: Extend custom scalar registration (e.g., DateTime in example/main.go or RegisterScalar). No change to queries/mutations/resolvers (informational). Aligns with existing introspection (which stubs deprecation); enables full spec compliance/UI support in Playground/GraphiQL.
- **Non-Breaking**: Existing scalars default to no URL; new option optional.

(Reference: graphql/graphql-spec sections on scalars/directives/introspection; no runtime parsing/execution beyond schema/intro.)

## Plan of Code Changes Required
**High-Level Approach**: 
- Store URL in Scalar type (graphql/types.go).
- Expose via RegisterScalar in schemabuilder (optional param for backward compat).
- Bubble through schema build/introspection.
- Add built-in directive def (like include/skip in introspection.go).
- Minimal parser/validate updates (reuse Directive handling).
- Update introspection query/JSON for __Type.specifiedByURL.
- Changes confined to: graphql/, schemabuilder/, introspection/ (no breaking existing APIs/examples).

Detailed Step-by-Step Changes (in dependency order):
1. **Update graphql/types.go**:
   - Add `SpecifiedByURL string` field to `Scalar` struct (for holding URL; String() unchanged).
   - (Optional) Add helper if needed for type assertion.
   - Why: Core type rep for scalars; used in exec/validate/intro.

2. **Extend Scalar Registration in schemabuilder/types.go**:
   - Update `RegisterScalar` func sig: `func RegisterScalar(typ reflect.Type, name string, uf UnmarshalFunc, specifiedByURL ...string) error` (variadic/optional for BC; panic/error if >1 URL).
   - Store URL in internal map (add `scalarSpecifiedByURLs map[reflect.Type]string`).
   - In build.go/getScalar (or new func): Attach URL to &graphql.Scalar{Type: name, SpecifiedByURL: url}.
   - Update scalarArgParsers to include URL if needed.
   - Why: Entry point for users (e.g., DateTime); matches RegisterScalar doc/examples.

3. **Update Schema Building in schemabuilder/build.go & output.go/input.go (if scalar refs)**:
   - In `getType`/`getScalar`: Propagate URL to Scalar struct instance.
   - Ensure wrappers (NonNull/List) unwrap to check inner Scalar's URL.
   - Why: Schema construction must embed directive info.

4. **Enhance Directive Handling (if needed) in graphql/parser.go & execute.go**:
   - Extend `parseDirectives` to recognize "@specifiedBy" (minimal; reuse existing for schema def).
   - Add to `shouldIncludeNode`? (No-op, as it's definition-time only).
   - Why: Parser already handles custom/built-in dirs; ensure no validation error for SCALAR locations.

5. **Update Introspection in introspection/introspection.go**:
   - Add `SpecifiedByURL string` to `Type` struct (wrapper for __Type).
   - In `registerType`: Add FieldFunc("specifiedByURL", func(t Type) *string { switch on Scalar: return &url or nil }).
   - Register built-in @specifiedBy Directive (like includeDirective/skipDirective vars; add to Schema.Directives).
   - Update __Directive locations/args to include SCALAR + url arg.
   - In `registerScalar` equiv (via type switch): Inject URL from graphql.Scalar.
   - Why: Core for __Type.specifiedByURL; ensures Playground/intro queries expose it.

6. **Update Introspection Query in introspection/introspection_query.go**:
   - Append `specifiedByURL` to IntrospectionQuery's __Type fragment (to match modern GraphiQL; ensures ComputeSchemaJSON includes it).
   - Why: Current query (copied from old GraphiQL) omits it; needed for tests/compliance.

7. **Minor: schema/schema.proto? (if protobuf impacts)**:
   - No change unless protoc-gen-jaal extension needed (out-of-scope for core).

**Non-Code**: Update README.md examples (e.g., add DateTime with URL); but post-core impl.

## Tests Needed to Verify Changes
**Goal**: Ensure spec compliance, no regressions (existing scalars, build, queries). Use existing test patterns (e.g., introspection_test.go, end_to_end_test.go, schemabuilder tests via Build()).

1. **Unit Tests for Registration/Building** (add to schemabuilder/*_test.go or new):
   - Test RegisterScalar with/without URL: Verify scalar map, graphql.Scalar.SpecifiedByURL set correctly; error on invalid URL/typ.
   - Test schema.Build(): Custom scalar (e.g., DateTime) in __Type has specifiedByURL; built-ins (String) null.

2. **Introspection Tests** (extend introspection/introspection_test.go):
   - TestIntrospectionForScalarWithSpecifiedBy: Register scalar w/ URL; run ComputeSchemaJSON; assert __Type.specifiedByURL matches; check @specifiedBy in directives list.
   - Compare JSON output pre/post (built-ins unchanged).
   - Test for SCALAR only (error if misused on Object?); deprecation stub compat.

3. **Parser/Validator Tests** (graphql/parser_test.go, validate.go tests, end_to_end_test.go):
   - Parse schema/query using scalar w/ @specifiedBy: Ensure no parse/validate err; SelectionSet ignores (def-time).
   - Invalid use: e.g., on FIELD - expect validation err.

4. **End-to-End/Integration Tests** (graphql/end_to_end_test.go, http_test.go, example/main.go run):
   - Query introspection via /graphql: Assert specifiedByURL in response for custom scalar.
   - Playground compat: UI reflects scalar URL (manual/visual).
   - Regression: Existing queries/mutations/scalars (Time, ID, Map) unchanged; full schema roundtrip.

5. **Edge/Compliance Tests**:
   - URL nil/empty: treated as null in intro.
   - Multiple scalars w/ URLs; list in __Schema.types.
   - Go test ./... + go build; cover 80%+ via go test -cover.

**Verification Metrics**: All tests pass; introspection JSON matches spec; no perf impact. Add table-driven tests mirroring existing (e.g., TestIntrospectionForInterface).

This keeps impl isolated (~200-300 LOC est.). Review for adjustments (e.g., API choices for RegisterScalar).
