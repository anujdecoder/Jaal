# Jaal GraphQL Spec Compliance Plan (June 2018 -> September 2025)

## Background
- Jaal is currently compliant with the June 2018 GraphQL spec (see schemabuilder/schema.go comments, graphql/ package).
- Explored README.md (resolvers via FieldFunc for queries/mutations; interfaces/unions via marker embeds like schemabuilder.Interface/Union + auto-common-fields; examples in example/main.go and test-1/server).
- Goal: Full compliance with Sept 2025 spec (includes Oct 2021, 2022 drafts, etc.).
- **DO NOT implement yet**—this is planning only. (From prior analysis.)

## Already Implemented (June 2018 Core + Some Overlaps)
- Queries, mutations, subscriptions (via schemabuilder.Schema.Query()/Mutation()/Subscription() + FieldFunc resolvers).
- Type system: Objects, Interfaces, Unions, Enums, InputObjects, Lists, NonNull, Scalars (built-in/custom + maps).
- Directives: @skip/@include (parser/execute.go + introspection), @deprecated (introspection support).
- Fragments, variables, selection sets, basic validation/execution/errors.
- Introspection (__schema, __type, etc.).
- Custom fields, inputs, enums, scalars registration.

## Missing/New Specs from Post-2018 to Sept 2025
(Based on spec.graphql.org diffs: Oct 2021 edition added @specifiedBy/isRepeatable; later drafts add oneOf, defer/stream, validation updates.)
- @specifiedBy directive (custom scalars spec linking).
- Repeatable directives + `isRepeatable` in __Directive introspection.
- oneOf input objects (exactly one field required).
- @defer and @stream (incremental delivery).
- Updated validation rules (coercion, fragments, directives, variables).
- Enhanced errors (extensions, paths).
- Parser/syntax for new features (e.g., oneOf keyword).
- Subscription enhancements (incremental events).
- Misc: Schema extensions/descriptions, full custom scalar validation.

## Prioritized Implementation List (Order of Priority)
**Priority 1 (High - Impacts core scalar users):** @specifiedBy + scalar linking.
**Priority 2 (High - Type system):** oneOf InputObjects (build on existing protoc support).
**Priority 3 (High - Introspection):** Repeatable directives + isRepeatable.
**Priority 4 (Medium - Execution):** @defer/@stream support.
**Priority 5 (Medium - Robustness):** Validation/parser updates + error enhancements.
**Priority 6 (Low - Polish):** Subscription increments + misc.

## Detailed Per-Spec Plan (No Code Changes Yet)
For **each** spec (in priority order):
1. **Research step**: Diff exact spec sections (e.g., Directives, Type System, Execution, Validation, Introspection). Note examples/tests from spec.
2. **Files to modify**:
   - schemabuilder/ (schema.go, types.go, input_object.go, reflect.go, function.go): Add registration (e.g., new markers for oneOf, directive support in FieldFunc).
   - graphql/ (types.go, parser.go, execute.go, validate.go): Extend AST/Directive/Selection, parsing logic, execution for new directives/validations, error formats.
   - introspection/ (introspection.go, introspection_query.go): Update __Directive struct/fields (add isRepeatable), register new built-ins like @specifiedBy/@defer.
   - http.go / ws.go: Handle streaming responses if needed for defer/stream.
3. **Tests**: Add to graphql/*_test.go, end_to_end_test.go, introspection_test.go. Use spec-provided query examples. Ensure coverage for resolvers/interfaces/unions.
4. **Examples/Usage**: Update example/main.go + README.md with new features (e.g., oneOf example).
5. **Dependencies**: Possibly update graphql-go parser dep; add validation helpers.
6. **Validation/Compat**: Keep June2018 backward-compat; add feature flags if needed. Run full tests + GraphiQL validation.
7. **Docs**: Update README.md with compliance table + usage for new features.

## Overall Timeline/Risks
- Phase 1: Priorities 1-3 (2-3 weeks, test heavily).
- Phase 2: 4-6.
- Risks: Parser breakage, client compat (test Apollo/GraphiQL), perf for incremental.
- Use todos for tracking; run `go test ./...` post-each.
- Contributing: Follow CONTRIBUTING.md; add to roadmap.

This plan ensures systematic compliance without starting implementation.

## Detailed Plan for Priority 1: @specifiedBy Directive (Custom Scalars Spec Linking)
### Explanation of How @specifiedBy is Expected to Work
Per GraphQL spec (Oct 2021+ editions, Section 3.9 "Scalars" and 6 "Directives"):
- **Purpose**: Links a custom SCALAR to its formal specification URL. This helps clients, tools, and validators understand the scalar's exact serialization, deserialization, and validation rules (e.g., beyond just "String" or "Int"). It does **not** affect runtime execution/resolution—it's purely declarative for schema introspection and documentation.
- **Usage in Schema** (SDL example, for reference; Jaal is code-first):
  ```
  scalar DateTime @specifiedBy(url: "https://example.com/rfc3339-spec")
  ```
  - Only applicable to SCALAR definitions.
  - Argument: `url` (non-null String).
  - Can appear once per scalar (not repeatable by default).
  - If absent, `specifiedByURL` is null in introspection.
- **Introspection**: 
  - Added `specifiedByURL` field to `__Type` (for SCALAR kinds only): Returns the URL string or null.
  - Example query: `{ __type(name: "DateTime") { ... specifiedByURL } }`
- **Validation/Behavior**: 
  - Server must validate directive usage (e.g., only on scalars, valid URL arg).
  - No impact on query execution, resolvers, or custom scalar funcs (like Jaal's `RegisterScalar`).
  - Errors if misused (e.g., on non-scalar).
- **Why needed for Jaal**: Jaal supports custom scalars (e.g., DateTime in examples); this makes them fully spec-compliant for tools like GraphiQL/Apollo.

### Plan of Code Changes Required (High-Level, NO Implementation Yet)
1. **schemabuilder/ layer** (for registration, as Jaal is code-first):
   - Extend `RegisterScalar` (in schemabuilder/types.go or new scalar.go) to optionally accept `specifiedByURL` string param (e.g., `RegisterScalar(typ, name, unwrap, specifiedByURL string)`).
   - Store it in internal Scalar type (update reflect.go, output.go for type caching).
   - Add support in `schemaBuilder.getType` / `buildScalar` to attach the URL.
   - Handle in Object/Field registration if scalars are used there.
   - Backward-compat: Make URL optional (default "").

2. **graphql/ layer** (core types/execution):
   - Update `Scalar` struct in graphql/types.go to include `SpecifiedByURL string`.
   - Minimal changes to parser.go/execute.go (since directive is schema-only, not query-executable; but ensure directive parsing skips/ignores if in queries).
   - Update validation (validate.go) to enforce @specifiedBy only on scalars + arg checks.

3. **introspection/ layer** (key for compliance):
   - Update `__Type` struct/registration in introspection.go to include `SpecifiedByURL *string` field + FieldFunc.
   - Register @specifiedBy as a built-in directive (like @skip) in registerDirective() + introspection_query.go.
   - Extend collectTypes / schema registration to propagate URL from custom scalars.
   - Update __Type kind handling for SCALAR.

4. **Other**:
   - http.go / middleware: No change (introspection-driven).
   - go.mod: No dep change needed (uses existing parser).
   - Error handling: Add spec-compliant errors for invalid @specifiedBy usage.

Changes must preserve June 2018 compat (e.g., no breaking RegisterScalar calls).

### Tests Needed to Verify Changes
- **Unit tests** (introspection_test.go, graphql/execute_test.go):
  - Register custom scalar with/without @specifiedBy (e.g., DateTime with RFC3339 URL).
  - Introspection query for __type on scalar: Assert `specifiedByURL` matches or is null.
  - Test built-in scalars (String/Int/etc.) return null for specifiedByURL.
- **End-to-end** (end_to_end_test.go, http_test.go):
  - Full schema build + GraphQL introspection query execution.
  - Query with custom scalar fields; verify no runtime breakage.
  - Error cases: Misuse @specifiedBy on non-scalar (e.g., object) → validation error.
- **Integration**:
  - Update example/main.go temporarily for test (revert after); run server + curl introspection.
  - Compatibility: Re-run existing scalar tests (e.g., DateTime in example) to ensure no regression.
  - Spec compliance: Use introspection query from spec examples; test with GraphiQL playground.
- **Coverage**: >90% for new paths; include negative tests for invalid URLs/args.

### Noting Plan for Review
- This plan appended to COMPLIANCE_PLAN.md (see above).
- Review steps: Check explanation vs. spec, validate file changes, approve test scope.
- Post-review: Follow exactly (use todos, implement in order, tests first, no shortcuts).
- Risks: Introspection breakage for clients; ensure URL is optional.

(End of @specifiedBy section. Ready for review before any code changes.)

## Detailed Plan for Priority 2: oneOf InputObjects (build on existing protoc support)
### Explanation of How oneOf is Expected to Work
Per GraphQL spec (Oct 2021 draft + later ratification in 2022/2025 editions, Section 3.10 "Input Objects" and directives):
- **Purpose**: Marks an Input Object as "oneOf" to enforce that **exactly one** of its fields is provided in input args (no more, no less; like a tagged union or discriminated union for inputs). Prevents invalid states (e.g., multiple fields set for mutually exclusive options like "create via ID or via email").
- **Usage in Schema** (SDL example):
  ```
  input CreateUserInput @oneOf {
    id: ID
    email: String
  }
  ```
  - Applied via `@oneOf` directive (or code marker in Jaal).
  - Validation: At query parse/execution, if input provided with 0 or >1 fields set → error (spec-compliant "oneOf" violation).
  - Jaal's protoc-gen already maps protobuf oneof to Union, so build on that for native code-first support.
- **Introspection**: 
  - __InputObject type gains `isOneOf: Boolean` field (true if marked).
  - Shows in __type for INPUT_OBJECT.
- **Validation/Behavior**:
  - Only on InputObjects; args to fields/queries/mutations using it must satisfy exactly-one.
  - No change to output/resolvers; affects input parsing in schemabuilder/input.go and graphql/validate.go/execute.go.
  - Errors: Spec-defined message for invalid oneOf inputs.
- **Why for Jaal**: Enhances InputObject (used heavily in mutations); aligns with existing Union/protoc support for full type system compliance.

### Plan of Code Changes Required (High-Level, NO Implementation Yet)
1. **schemabuilder/ layer**:
   - Extend InputObject struct/registration (input_object.go, schema.go) to support oneOf marker (e.g., `InputObject.OneOf bool` or @oneOf via FieldFunc/ new method like `.OneOf()`).
   - Update reflect/input parsing to track fields set and enforce exactly 1 in FromJSON.
   - Handle in build/getType for inputs; add to Object/InputObject clone etc.
   - Backward compat: Default false for existing inputs.

2. **graphql/ layer**:
   - Update InputObject in types.go to include `IsOneOf bool`.
   - Extend validate.go for input arg validation (check field count ==1 if oneOf).
   - Update execute/parser for input coercion to trigger oneOf checks.
   - Minimal parser update if SDL support added later.

3. **introspection/ layer**:
   - Add `isOneOf` FieldFunc to __Type for INPUT_OBJECT in registerType.
   - Include in collectTypes, __InputValue if needed.
   - Register @oneOf directive (like @specifiedBy) in registerDirective + schema.Directives.

4. **Other**:
   - schemabuilder/schema.go: Support in Build for oneOf inputs.
   - Error handling: Use jerrors or spec errors for violations.
   - Docs/examples: Update README with oneOf InputObject example (build on protoc mention).

Changes must preserve existing InputObject usage/compat.

### Tests Needed to Verify Changes
- **Unit tests** (schemabuilder/input*_test.go if exist, or add to build_test; graphql/validate_test.go, execute_test.go):
  - Register InputObject with .OneOf(true); test input parsing succeeds only for exactly 1 field.
  - Test failures: 0 fields or 2+ fields → specific error.
  - Existing inputs remain non-oneOf (multi-fields OK).
- **Introspection tests** (introspection_test.go, introspection_test.go):
  - Introspect __type on oneOf InputObject: Assert `isOneOf: true`; false for normal.
  - Include in __schema types/directives test.
- **End-to-end** (end_to_end_test.go, http_test.go):
  - Mutation/query with oneOf input arg; valid/invalid cases.
  - Full schema + execution; ensure no regression on normal inputs/unions.
- **Compatibility**: Re-run all existing input/mutation tests (e.g., in example, end_to_end); test with protoc-generated oneOf if applicable.
- **Coverage**: Edge cases (nested, optional fields, nulls); spec example queries.

### Noting Plan for Review
- This plan appended to COMPLIANCE_PLAN.md (see above).
- Review steps: Verify explanation vs. spec (focus on input validation), check code impact on existing InputObjects, approve tests.
- Post-review: Follow exactly (use todos, start with tests, etc.).
- Risks: Input parsing breakage; ensure protoc compat; performance on large inputs.

(End of oneOf plan section. Ready for review before implementation.)


