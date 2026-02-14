# Jaal GraphQL Spec Compliance Plan

## Overview
This project (Jaal) is currently compliant with the GraphQL specification up to the June 2018 release, as stated in README.md. This includes core features like:
- Queries, Mutations, and basic Subscriptions
- Objects, Interfaces, and Unions (implemented via resolvers and struct markers like schemabuilder.Interface/Union)
- Scalars (built-in + custom registration)
- Enums, Input Objects, Output Objects
- Built-in directives: @include, @skip
- Introspection (via introspection package)
- Schema building from Go structs using reflection (in schemabuilder/)
- Support for maps, custom fields, as per examples and README

See README.md and examples/ (e.g., example/main.go for queries/mutations/subscriptions, character example for interfaces/unions) for implementation details.

The goal is to update compliance to the September 2025 spec release (encompassing ratified updates and working drafts from June 2018 through ongoing evolutions up to late 2025, based on graphql.org spec and working group progress).

## All Latest Specifications/Features (Post-June 2018 to Sept 2025)
Based on GraphQL spec evolution (June 2018 ratified; Oct 2021 Working Draft; subsequent drafts/proposals up to 2025):
1. **Deprecations on Input Values** (Oct 2021): Allow `@deprecated` directive on arguments and input object fields (with deprecationReason).
2. **@specifiedBy Directive** (Oct 2021): For custom scalars to link to external type specs (e.g., `@specifiedBy(url: "...")`).
3. **Repeatable Directives** (Oct 2021): Support applying the same directive multiple times on a location (directive definition with `repeatable` keyword).
4. **Input Unions / OneOf Input Objects** (Oct 2021, stabilized ~2023): Input objects marked with `@oneOf` directive for exclusive fields (like protobuf oneof; already partially handled in protoc-gen-jaal as Union, but needs full spec input support).
5. **Incremental Delivery** (2023+ drafts): 
   - `@defer` directive for deferred fragment resolution.
   - `@stream` directive for streaming list items.
6. **Client-Controlled Nullability (CCN)** (~2024-2025 proposal): New `!` syntax in queries for nullability assertions (experimental in drafts).
7. **Schema Coordinate Extensions** (minor, 2021+): Enhanced descriptions and annotations on schema elements.
8. **Other minor** (2021+): Updated validation rules for nulls, improved error paths, directive argument validation enhancements, better support for federation-like but stick to core spec.

(Note: Exact Sept 2025 cutoff assumes inclusion of ratified Incremental Delivery and CCN if finalized by then; monitor graphql/graphql-spec repo for drafts.)

## Already Implemented Specs (June 2018 Baseline + Some Partial)
- Core Execution: Queries, Mutations, Subscriptions (see graphql/execute.go, ws.go for WS subs).
- Type System: Objects, Interfaces, Unions, Scalars, Enums, Inputs (schemabuilder/, graphql/types.go).
- Directives: Built-in @include/@skip (graphql/execute.go).
- Introspection, Parsing, Validation (graphql/parser.go, validate.go, introspection/).
- Playground/HTTP handling (http.go).

**Gaps/Partials**: Deprecation support is stubbed in introspection.go (forces IsDeprecated=false); oneOf hack in tests/protoc; no support for newer directives.

## Implementation Plan
**DO NOT IMPLEMENT YET** - this is planning only. Prioritized list of specs to implement (order by: backward compatibility, ease of integration, user impact, spec stability):
1. **High Priority: Deprecations on Input Values** - Extend directive parsing/registration in graphql/parser.go and schemabuilder/ to support @deprecated on args/input fields. Update introspection.go to properly reflect deprecationReason. (Affects schema building, no breaking changes).
2. **High Priority: @specifiedBy Directive** - Add to scalar registration in schemabuilder/reflect.go and types.go; include in introspection for custom scalars (e.g., DateTime).
3. **Medium: Repeatable Directives** - Modify directive handling in parser.go, types.go, validate.go to allow multiples; update directive structs.
4. **Medium: Input Unions/OneOf** - Full support for @oneOf on input objects (build on existing Union in output.go, extend to input_object.go; align with protoc-gen-jaal).
5. **Lower: Incremental Delivery (@defer, @stream)** - Requires changes to executor in execute.go for async/deferred responses; update validation, parsing. High complexity due to streaming.
6. **Lowest: Client-Controlled Nullability & Misc** - Experimental; add to parser/validate if spec finalizes by 2025; minor for others.

This plan ensures step-by-step compliance. Next step (after approval): Detailed specs per item, tests via graphql/*_test.go, updates to README/examples. Track via issues/PRs per CONTRIBUTING.md.

Priorities chosen: Start with non-breaking (deprecation, specifiedBy) for quick wins and existing users.
