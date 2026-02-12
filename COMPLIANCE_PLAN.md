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
