# Implementation Plan for @oneOf Directive / Input Unions (Priority 4, Post-June 2018 Spec Feature)

**Note for Review:** This plan is prepared based on code analysis of README.md (queries/muts/unions/interfaces via resolvers/FieldFunc/embed markers like schemabuilder.Union), schemabuilder/input*.go (InputObject/argParsers/FromJSON for coercion), graphql/types.go/validate.go/parser.go (input handling), SPECIFIEDBY_IMPLEMENTATION_PLAN.md & DEPRECATION_ON_INPUT_VALUES_PLAN.md (built-in dir patterns, introspection extensions, non-breaking), SPEC_COMPLIANCE_PLAN.md (align w/ protoc-gen-jaal oneof hack), introspection/ for type dirs. No changes made yet (DO NOT IMPLEMENT). After review/approval, proceed (e.g., via search_replace, terminal go test ./...). Priorities: Mirror Union for input oneOf marker; reuse input coercion (like deprecation tags); minimal impact on resolvers/outputs/exec. 

## How @oneOf Directive is Expected to Work (Per Spec)
From GraphQL Oct 2021 Working Draft (stabilized ~2023; included in Sept 2025 release per SPEC_COMPLIANCE_PLAN.md):

- **Definition**: Built-in directive (definition-time only, like @specifiedBy on SCALAR or @deprecated on inputs):
  ```
  """
  Indicates that an Input Object is a OneOf Input Object (and thus requires exactly one field to be set in a query or mutation).
  """
  directive @oneOf on INPUT_OBJECT
  ```
  - Applies **only to INPUT_OBJECT** type definitions (no arguments; INPUT_OBJECT_LOCATION).
  - Example schema SDL:
    ```
    input ContactInput @oneOf {
      email: String
      phone: String
    }
    ```
- **Purpose**: Enables "input unions" / exclusive fields for polymorphic inputs (like protobuf oneof, discriminated unions, similar to output Unions/Interfaces in README.md's registration via embed pointers/resolvers). Allows API to accept exactly one variant (e.g., search by email OR phone, not both/none).
  - Semantics (spec rules: Input Object Field Coercion, OneOf Input Objects validation):
    - Exactly one field must be **present and non-null** in input value; others must be absent (omitted; nulls count as absent in some but strictly enforced).
    - Violation: Error during input coercion/validation (e.g., "Exactly one key must be set and non-null for oneOf input object 'ContactInput'; found 0/2").
    - Informational/no runtime block beyond validate (execution proceeds only if valid).
    - Complements: Outputs use schemabuilder.Union/Interface (resolvers return wrapper); inputs now @oneOf for symmetry (protoc-gen-jaal handles proto oneof as Union output - extend to input).
  - Introspection (__Type for INPUT_OBJECT kind):
    - `directives: [__Directive!]!` includes [{name: "oneOf", ...}].
    - In __Schema.directives list; INPUT_OBJECT in locations.
    - No extra fields (unlike scalar.specifiedByURL).
  - Validation rules:
    - Only on INPUT_OBJECT; no args/repeat (until repeatable dir).
    - Applies to top-level args or nested input fields in queries/muts/subs.
    - Definition-time marker + runtime coercion (vs query dirs like @skip in execute.go).
- **Usage in Jaal Context**: Extend InputObject reg (e.g., in RegisterInput like README/example/main.go's CreateCharacterRequest; FieldFunc for fields) with @oneOf marker (embed or method). Affects arg parsing for ops resolvers (no change to queries/muts/unions/interfaces resolvers). Non-breaking: Existing inputs default false.
- **Jaal Alignment**: Builds on output unions (output.go hasUnionMarkerEmbedded/buildUnionStruct); input deprecation (reflect.go tags); ensures spec-compliant inputs in schema.Build()/execute.

(Reference: graphql-spec "OneOf Input Objects"; no effect on README examples' FieldFunc/resolvers beyond inputs.)

## Plan of Code Changes Required
**High-Level Approach** (non-breaking, like specifiedBy/deprecation plans):
- Add OneOf marker/flag to InputObject (schemabuilder/graphql types; like Union in output.go).
- Set in schema build/input parsers; add spec validation in FromJSON/coercion (input_object.go/input.go).
- Add built-in dir + loc to introspection (extend DirectiveLocation/oneOfDirective var, schema.Directives; add __Type.directives FieldFunc for type-level dirs like @oneOf).
- Minimal graphql/ parser/validate updates (coercion-focused; reuse ParseArguments from function.go).
- Changes confined to: schemabuilder/input*.go/output.go (for marker reuse), graphql/types.go/validate.go, introspection/ (no breaking APIs/resolvers/examples; align protoc).
- (Assumes post-deprecation: INPUT_OBJECT loc exists; build on that.)

Detailed Step-by-Step Changes (in dependency order, following patterns e.g., FieldFunc, embeds, type switches, FromJSON, directive vars, JSON/intro updates):

1. **Update graphql/types.go**:
   - Add `OneOf bool `json:"-"`` to `InputObject` struct (for flag; update isType/String if needed; like SpecifiedByURL on Scalar).
   - Why: Core type rep for inputs; flows to parsers/exec/intro (used in function.go/validate.go).

2. **Extend Input/Arg Support in schemabuilder/**:
   - types.go: Add `type OneOfInput struct{}` marker (anon embed like Union); `var oneOfInputType = reflect.TypeOf(OneOfInput{})`; doc example (similar Union).
   - input_object.go: Add `hasOneOfMarkerEmbedded(typ reflect.Type) bool` (loop fields, anon == oneOfInputType); `func (io *InputObject) OneOf()` setter for registered objs.
   - In `makeInputObjectParser`/`generateArgParser`/`copyInputObject`: detect marker/flag, set `argType.OneOf = true`; update registered InputObject FromJSON: if oneOf, count present (non-nil/non-zero? per spec) fields in value map ==1 else err (spec-compliant msg; e.g., check len(nonNullKeys)==1).
   - input.go: Enhance `argParser`/`wrap*` if needed for oneOf (propagate flag).
   - reflect.go/build.go/schema.go: If tag support (e.g., json/graphql tag for oneOf), parse; handle in input obj reg/build like unions (error if bad embed).
   - output.go: (Minor reuse helpers if shared.)
   - Why: User entry (e.g., type MyInput struct { schemabuilder.OneOfInput; Email string; Phone string } or input.OneOf()); mirrors README Union/Interface + input FieldFunc; enables protoc oneof input.

3. **Enhance Directive/Input Validation in graphql/**:
   - types.go: ok (flag set).
   - parser.go: argsToJson(map) ok for oneOf inputs; no change.
   - validate.go: Extend `ValidateQuery` (or add validateInputValue/ for args): if inputObj, ok := typ.(*graphql.InputObject); ok && inputObj.OneOf { validate exactly 1 field in selection.Args } (called in arg parse; like enum/scalar).
   - execute.go: No (coercion via field's ParseArguments in resolve).
   - Why: Enforce spec at validate/coerce time for input args (e.g., in mut/query FieldFunc args); reuses patterns from shouldIncludeNode/directive parse; def-time ok since no SDL dir.

4. **Update Introspection in introspection/introspection.go**:
   - Add `INPUT_OBJECT DirectiveLocation = "INPUT_OBJECT"` (for @oneOf; extend comment).
   - In `registerDirective` enum: include it.
   - Add `var oneOfDirective = Directive{ Name: "oneOf", Description: "...", Locations: []DirectiveLocation{INPUT_OBJECT}, Args: []InputValue{}, IsRepeatable: false }` (like specifiedByDirective/deprecatedDirective vars; no args).
   - In `registerSchema`: include in `Directives: []Directive{..., oneOfDirective}` (extend comment for input unions).
   - Extend `type Type`: add `FieldFunc("directives", func(t Type) []Directive { switch inner := t.Inner.(type) { case *graphql.InputObject: if inner.OneOf { return []Directive{oneOfDirective} }; ... default: return nil } })` (per-type for __Type.directives; note: may be missing currently for @specifiedBy etc - add for full compliance).
   - Update `registerType` switch for INPUT_OBJECT case.
   - Why: Expose in __Schema.directives & __Type (spec); matches built-in pattern; ensures Playground/intro shows @oneOf on input objs.

5. **Update Introspection Query + Related in introspection/introspection_query.go**:
   - Add `directives { name }` (etc) to __Type fragments (to capture type dirs in ComputeSchemaJSON/intro tests; similar to specifiedByURL/deprecation fields addition).
   - Why: Like prior plans; full __Type introspection for oneOf inputs.

6. **Minor: Other**:
   - schema/schema.proto: no (protobuf out-of-scope).
   - README.md/SPEC_*.md: post-impl doc (e.g., add input oneOf example mirroring Union).

**Non-Code**: Update examples if needed. Est ~200-300 LOC; isolated to input path.

## Tests Needed to Verify Changes
**Goal**: Ensure spec compliance (oneOf validation/intro), no regressions (existing inputs like README's CreateCharacterRequest, deprecation/specifiedBy, unions/interfaces resolvers, queries/muts). Mirror test patterns: table-driven in introspection_test.go/end_to_end_test.go; schemabuilder/input tests; cover coercion errors + intro JSON; go test -cover.

1. **Unit Tests for Registration/Building/Parsers** (add to schemabuilder/input_object.go tests or input.go; function.go):
   - Test OneOfInput marker / .OneOf(): Verify set in graphql.InputObject; build err for invalid (non-anon, dup like Union); compat w/ deprecation tags.
   - Test argParser.FromJSON/makeInputObjectParser for oneOf: exactly 1 non-null field -> parsed struct ok; 0/both/nulls -> err (spec msg); scalar/enum/list/nested fields; vars.

2. **Introspection Tests** (extend introspection/introspection_test.go e.g., TestIntrospectionForInputObject/Test_Directives/ nodeInput):
   - TestOneOfInputIntrospection: Register oneOf input (e.g., ContactInput); ComputeSchemaJSON; assert __Type(name:"ContactInput", kind:INPUT_OBJECT).directives=[{name:"oneOf"}], __Schema.directives includes it w/ INPUT_OBJECT loc; compare JSON.
   - Test __Type.directives field; oneOf + specifiedBy/deprecation compat; no effect on Union/outputs.

3. **Parser/Validator Tests** (graphql/parser_test.go, validate.go tests/end_to_end_test.go, clone_test.go):
   - Valid oneOf input in query/mut: parse/validate ok.
   - Invalid: none/both fields -> err (in ValidateQuery/ParseArguments); wrong type loc (build err).
   - Edge: optional fields, defaults=null, deep input obj.

4. **End-to-End/Integration Tests** (http_test.go, graphql/end_to_end_test.go, example/main.go + test w/ oneOf input mut/query):
   - Exec valid/invalid via /graphql: success/GraphQL err; introspect endpoint shows @oneOf.
   - Playground/UI: input validation; regression on existing (Characters, deprecations).
   - Protoc/proto oneof compat (if test).

5. **Edge/Compliance Tests**:
   - Spec err msgs/paths; repeatable=false; combine w/ other dirs on fields; cover >80% (go test ./... -cover).
   - Full schema roundtrip; no perf hit.

**Verification Metrics**: Tests pass; introspection JSON matches spec (__Type.directives, oneOf loc); valid inputs execute (resolvers unchanged); invalid rejected early. Add e.g., TestValidateOneOfInput mirroring TestSkipDirectives.

This completes medium-priority for Sept 2025 compliance (builds on unions/resolvers from README). Review for: marker choice (embed vs method), validation placement (schemabuilder.FromJSON vs graphql/validate), __Type.directives necessity (if spec requires for all type dirs).

After review, we will follow this plan to implement the changes.
