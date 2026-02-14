# Implementation Plan for Deprecations on Input Values (@deprecated on Args/Input Fields)

**Note for Review:** This plan is prepared based on code analysis of stubs (in introspection.go), input/arg handling (schemabuilder/input*.go, function.go), and directive patterns. References: SPEC_COMPLIANCE_PLAN.md (lists as high-priority #1) and SPECIFIEDBY_IMPLEMENTATION_PLAN.md (similar non-breaking, built-in dir + introspection extension for post-2018 spec; used for structure/comments, e.g., FieldFunc switches, directive vars, test JSON updates). No changes made yet. After review/approval, proceed to implement (e.g., via search_replace, go build/test).

## How @deprecated Directive on Input Values is Expected to Work (Per Spec)
From GraphQL Oct 2021 Working Draft (and later to 2025):
- **Definition**: Built-in directive (extends pre-2018 @deprecated on fields/enums):
  ```
  """
  Marks an element of a GraphQL schema as no longer supported.
  """
  directive @deprecated(
    """
    Explains why this element was deprecated, usually also including a suggestion for how to
    access supported similar data. Formatted using the Markdown syntax, as specified by
    [CommonMark](https://commonmark.org/).
    """
    reason: String = "No longer supported"
  ) on FIELD_DEFINITION | ENUM_VALUE | ARGUMENT_DEFINITION | INPUT_FIELD_DEFINITION
  ```
  - Applies to **input values**: ARGUMENT_DEFINITION (args on fields/queries/mutations) and INPUT_FIELD_DEFINITION (fields on InputObject).
  - Argument: `reason: String` (optional, defaults "No longer supported").
- **Purpose**: Marks inputs as deprecated for deprecation warnings in tools/clients (e.g., GraphiQL/Playground); informational only (no runtime enforcement/blocking; execution still works). Allows graceful API evolution.
  - Example schema (Jaal context):
    - Field arg: `user(id: ID! @deprecated(reason: "Use username")): User`
    - Input field: `input CreateUserInput { age: Int @deprecated(reason: "Use birthdate") }`
  - In introspection:
    - __InputValue (for input fields/args): adds `isDeprecated: Boolean!`, `deprecationReason: String`.
    - __Field.args[] also reflects (extends existing field deprecation stub).
    - Appears in __Schema.directives; __Type.inputFields/fields.args include deprecation.
  - Validation rules:
    - Only on input locations (ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION); reason string.
    - Can combine with @specifiedBy etc (if repeatable added later).
    - Definition-time only (like @specifiedBy; not query-exec like @skip).
- **Usage in Jaal Context**: Extend input registration (schemabuilder.InputObject.FieldFunc, function args in schema ops) to support @deprecated tag/directive. Aligns with existing stubs in introspection.go (forces false/nil to avoid UI bugs); enables full spec (no breaking for outputs). No runtime change to arg parsing/exec (schemabuilder/argParser/graphql/execute).
- **Non-Breaking**: Defaults match stubs (non-deprecated); backward compat for existing inputs/args.

(Reference: graphql/graphql-spec "Input Values" + introspection; matches @specifiedBy as definition-time scalar/input enhancement.)

## Plan of Code Changes Required
**High-Level Approach** (non-breaking, like @specifiedBy plan):
- Reuse/extend existing deprecation stubs (InputValue/field structs with IsDeprecated/DeprecationReason) to support input locations.
- Add to input object/arg building (schemabuilder) for @deprecated parsing/registration (e.g., via tags or FieldFunc opts).
- Add built-in dir def (introspection.go, extending include/skip/specifiedBy).
- Update graphql types/intro for inputFields/args deprecation.
- Minimal parser/validate (definition-time; reuse Directive).
- Changes confined to: introspection/, schemabuilder/input*.go + function.go/output.go, graphql/types.go (no exec impact).

Detailed Step-by-Step Changes (dependency order, following codebase patterns e.g., FieldFunc, type switches, directive vars, omitempty for JSON):
1. **Update graphql/types.go**:
   - Extend Field struct: add `IsDeprecated bool`, `DeprecationReason *string` (omitempty; for args on objects).
   - (InputObject/args already map[string]Type; deprecation in introspection wrapper.)
   - Why: Args deprecation flows from Field (per function.go/argsTypeMap); matches @specifiedBy Scalar extension.

2. **Extend Input/Arg Support in schemabuilder/**:
   - input_object.go: Update generateArgParser/generateInputFieldParser to parse @deprecated (e.g., struct tags like `graphql:"age,deprecated=Use birthdate"` or FieldFunc opts; store in graphql.InputObject? but since input type, mainly for intro).
   - function.go: In buildFunction/argsTypeMap: propagate deprecation to Field.Args map (extend to hold deprecation metadata).
   - output.go: Update buildField to support deprecation for args.
   - types.go/reflect.go: Add helpers (e.g., parseGraphQLFieldInfo for deprecated tag; similar to Skipped/Name).
   - Why: Entry for users (inputs/args in ops like RegisterInputs, FieldFunc args struct); matches scalar reg.

3. **Enhance Directive Handling in graphql/**:
   - parser.go/types.go: Ensure parseDirectives recognizes @deprecated (generic already; add to Directive if args like reason).
   - validate.go/execute.go: No-op (definition-time; stub validate for input locs if needed).
   - Why: Reuse from @specifiedBy (minimal; only skip/include exec-special).

4. **Update Introspection in introspection/introspection.go**:
   - Extend InputValue struct: ensure `IsDeprecated bool`, `DeprecationReason *string` (already partial; remove stubs).
   - registerInputValue: Add FieldFunc for isDeprecated/deprecationReason (like EnumValue/field).
   - registerField/registerType: Update for args (in __Field.args, __Type.inputFields) to pull deprecation from schema (remove force-false/nil stubs; e.g., if graphql.InputField has dep data).
   - Add ARGUMENT_DEFINITION, INPUT_FIELD_DEFINITION to DirectiveLocation const/enum.
   - Add var deprecatedDirective = Directive{...} (locations include input ones, default reason arg; like specifiedByDirective).
   - registerSchema/registerQuery: Include in __Schema.directives (extend []Directive{include, skip, specifiedBy, deprecated}).
   - Why: Core for __InputValue/__Field deprecation (stubs reference this); matches specifiedBy FieldFunc + built-in var.

5. **Update Introspection Query + Related in introspection/introspection_query.go**:
   - Ensure InputValue fragment includes `isDeprecated deprecationReason` (already does); add if missing for args/inputFields.
   - Why: Like FullType.specifiedByURL addition; ensures ComputeSchemaJSON captures for tests.

6. **Minor: schema/schema.proto?**:
   - No (protobuf extension out-of-scope; align if needed post).

**Non-Code**: Update README/examples (e.g., deprecate arg in RegisterQuery/mutation); reference in SPEC_COMPLIANCE_PLAN.md.

## Tests Needed to Verify Changes
**Goal**: Ensure spec compliance for input deprecation (introspection reflects), no regressions (existing fields/inputs/args/stubs; like specifiedBy tests). Use table-driven patterns from introspection_test.go, end_to_end_test.go, input tests in schemabuilder/graphql tests.

1. **Unit Tests for Registration** (schemabuilder/input*_test.go or new; function.go):
   - Test InputObject/FieldFunc arg with @deprecated tag/reason: Verify parsed deprecation in graphql.InputObject/Field; default reason if omitted.

2. **Introspection Tests** (extend introspection/introspection_test.go, matching TestIntrospectionForInterface/Test_Directives):
   - TestInputValueDeprecation: Register input/arg w/ deprecation; run ComputeSchemaJSON; assert __InputValue.isDeprecated=true, deprecationReason matches; check @deprecated in directives (new locs).
   - Test for args in __Field (e.g., query arg); scalars/outputs unchanged.
   - Compare JSON pre/post (remove stub forces); specifiedBy compat.

3. **Parser/Validator Tests** (graphql/parser_test.go, validate.go tests, end_to_end_test.go):
   - Parse schema w/ deprecated input arg/field: No err; SelectionSet/args ignore (def-time).
   - Invalid loc (e.g., on FIELD): Validate err.
   - Query using deprecated arg: Still executes OK.

4. **End-to-End/Integration** (http_test.go, examples with deprecated input):
   - Introspect /graphql endpoint: Deprecated input shows in JSON (e.g., CreateUserInput field).
   - Playground/UI: Deprecation warning renders; no breakage.
   - Regression: Existing non-dep inputs/args (e.g., main.go user funcs) pass.

5. **Edge/Compliance Tests**:
   - Default reason; nil reason; enumInput compat; cover >80% (go test -cover).
   - DeepEqual JSON updates like specifiedBy test fixes.

**Verification Metrics**: Tests pass; introspection matches spec (e.g., ARGUMENT_DEFINITION); no perf/exec impact. Add to TestIntrospectionForInterface for inputs.

This ensures quick-win compliance (2-300 LOC est.; isolated like @specifiedBy). Review for tag vs dir opts etc.

Priorities: Build on stubs for minimal disruption.
