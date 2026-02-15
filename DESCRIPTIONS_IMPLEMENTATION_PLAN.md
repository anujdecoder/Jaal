# Implementation Plan for Descriptions on Schema Elements

**Note for Review:** This plan is prepared based on code analysis (README.md for reg patterns like Object/InputObject/FieldFunc/Enum; schemabuilder/types.go+output.go/input_object.go/schema.go for Description fields/setters; graphql/types.go/introspection/introspection.go for desc propagation/intro/__Type.description; SPEC_COMPLIANCE_PLAN.md mentioning "Schema Coordinate Extensions" for enhanced descs; prior plans like ONE_OF_DIRECTIVE_IMPLEMENTATION_PLAN.md for structure/tests/non-breaking). Partial support exists (e.g., Object.Description pulled in build); gaps in reg APIs for inputs/fields/enums/schema-level + full Playground/intro exposure. No changes made yet (DO NOT IMPLEMENT). After review/approval, proceed (e.g., search_replace, go build/test). Priorities: non-breaking additions (optional desc param/setter); mirrors Union/InputObject/FieldFunc patterns for readability.

## How Descriptions are Expected to Work (Per Spec + Jaal/Playground)
From GraphQL spec (core since June 2018; "Schema Coordinate Extensions" minor 2021+ per SPEC_COMPLIANCE_PLAN.md):

- **Definition**: Optional `description` string on schema elements (SDL: """ ... """ or # comments; code-first equiv).
  - Applies to: SCHEMA (root), OBJECT/INTERFACE/UNION/ENUM/INPUT_OBJECT/SCALAR (types), FIELD_DEFINITION (fields), ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION (args/inputs), ENUM_VALUE, etc.
  - Example SDL:
    ```
    """
    The root query.
    """
    type Query {
      """
      Fetch user by ID.
      """
      user(id: ID! @deprecated(reason: "Example")): User
    }
    """
    User payload.
    """
    type User { ... }
    ```
- **Purpose**: Documentation for tools/clients; no runtime effect (informational).
  - In introspection (__Type.description, __Field.description, __InputValue.description, __EnumValue.description, etc.): Returned as string (""/null if unset).
  - Playground/GraphiQL/UI: Shows descs in docs pane, tooltips, schema explorer (e.g., for fields/types/args; embedded assets in jaal.HTTPHandler use IntrospectionQuery w/ description fields).
  - Validation: None strict; Markdown OK (CommonMark).
- **Usage in Jaal Context**: Extend reg APIs (schemabuilder.Object(name, typ, desc?), .FieldFunc w/ desc opt, InputObject/Enum similar) to set descs (propagates to graphql.Type/intro like Description in Object/Union). Aligns w/ existing partial (e.g., Object.Description in output.go); enables full Playground (http.go) + oneOf/deprecation/specifiedBy compat. Non-breaking: default "".
- **Jaal Alignment**: Builds on desc support in graphql/types.go (Object etc), introspection.Type.description FieldFunc, output.go build (pulls object.Description); extend for inputs/fields/enums (no SDL parser, code-first like README examples).

(Reference: graphql-spec "Descriptions"; jaal/introspection_query.go FullType/InputValue/etc fragments already include description.)

## Plan of Code Changes Required
**High-Level Approach** (non-breaking, like OneOf/deprecation/specifiedBy plans):
- Add desc param/setter to schemabuilder regs (Object.InputObject.Enum; FieldFunc opts; mirror Object.Description).
- Propagate to graphql.Type structs (add to InputObject/Enum/Field if missing).
- Ensure in build/output/input/enum paths + introspection (already mostly supports via __Type etc).
- Update intro query/tests minimally if needed for schema-level.
- Changes confined to: schemabuilder/types.go+*.go, graphql/types.go, introspection/ (no exec impact; Playground auto-benefits).
- (Schema-level desc via custom __schema override if spec; minor.)

Detailed Step-by-Step Changes (dependency order, following patterns e.g., Description string, FieldFunc, build funcs, type switches):

1. **Update graphql/types.go**:
   - Add `Description string` to `InputObject` (like Object/Union/Interface); to `Enum` if missing (spec __Enum.description).
   - (Field already has via deprecation; Scalar/others ok.)
   - Why: Core rep for inputs/enums; used in introspection/validate/build (mirrors Scalar.SpecifiedByURL, InputObject.OneOf).

2. **Extend Registration in schemabuilder/types.go + schema.go**:
   - types.go: Add `Description string` setter/methods to InputObject (e.g., func (io *InputObject) Description(d string) { io.description = d }); update Object/Enum if needed (Object already has; make consistent).
   - schema.go: Update Object(name, typ interface{}, description ...string) *Object (variadic optional for BC); set .Description; similar for InputObject(name, typ, desc...) , Enum w/ desc.
   - Why: User entry (e.g., obj := schema.Object("User", User{}, "User payload"); input.Description("..."); like README Object reg; non-breaking defaults "").

3. **Update Schema Building in schemabuilder/**:
   - output.go: Already pulls description = object.Description in buildObject/buildUnion/buildInterface - ok.
   - input_object.go: In generateArgParser/make.../generateObjectParserInner: set argType.Description = obj.description (from schemabuilder.InputObject) or from struct comment/reflect if advanced.
   - build.go/schema.go: Propagate in getType/input reg (copyInputObject etc).
   - reflect.go: Optional parse desc from struct tags/comments (e.g., `graphql:",description=..."`; minimal for now).
   - enum in input.go/types.go: Add desc to EnumMapping/graphql.Enum.
   - Why: Ensure desc in graphql.InputObject/Enum (for intro; mirrors OneOf/FieldDeprecations).

4. **Enhance Introspection (if needed) in introspection/introspection.go**:
   - registerType: description FieldFunc already switches + pulls t.Description for Object/InputObject/Union/Interface - extend for *graphql.InputObject case if missing.
   - For enums/fields/inputs: already in registerEnumValue/registerField/inputFields (uses .Description).
   - Schema-level: Optional (add to __Schema if spec coord); minor.
   - Why: Ensure __Type.description etc returns set values (Playground uses; like oneOfDirective desc).

5. **Update Introspection Query (minor) in introspection/introspection_query.go**:
   - FullType/InputValue/enumValues fragments already include `description` - ok (no change).
   - Why: Matches prior like directives/specifiedByURL.

6. **Minor: Other**:
   - No change to parser/execute/validate (def-time only).
   - README.md/examples: Update post for usage (e.g., add desc to Object/InputObject).
   - users/ in example/ or main.go: no (out-of-scope).

**Non-Code**: Update SPEC_COMPLIANCE_PLAN.md (mark schema descs); tests cover.

## Tests Needed to Verify Changes
**Goal**: Ensure descs set/returned in intro/Playground, no regressions (existing regs/objects/inputs like in example/users/; oneOf/deprecation). Use patterns: table-driven introspection_test.go, end_to_end; DeepEqual JSON; go test -cover.

1. **Unit Tests for Registration/Building** (schemabuilder/*_test.go or new; types.go/schema.go):
   - Test Object/InputObject/Enum w/ desc param/setter: Verify .Description set in graphql.Type; build err none.
   - Test build/copy funcs propagate desc (e.g., InputObject in makeInputObjectParser).

2. **Introspection Tests** (extend introspection/introspection_test.go, e.g., TestIntrospectionForInterface/Test_Directives/TestComputeSchemaJSON):
   - TestDescriptionOnType: Register obj/input/enum w/ desc; ComputeSchemaJSON; assert __Type.description matches; __Field/__InputValue/__EnumValue for fields/args.
   - Test w/ no desc ("" default); Playground compat (JSON); oneOf/deprecation compat.

3. **Parser/Validator/Other Tests** (graphql/parser_test.go/end_to_end_test.go; no direct but):
   - Full query/intro parse: descs in response; no err.

4. **End-to-End/Integration** (http_test.go, example/users/server.go test via GetGraphqlServer):
   - Query /graphql introspection: descs in JSON (__type etc); UI visual (manual).
   - Regression: Existing (User/ContactByInput) unchanged.

5. **Edge/Compliance Tests**:
   - Markdown/special chars in desc; schema-level if added; cover >80%; spec match (e.g., __schema description).

**Verification Metrics**: Tests pass; intro JSON has descs (Playground shows); no perf/break. Add e.g., TestDescriptions mirroring TestOneOfInputIntrospection.

This adds enhanced desc support (minor spec ext). Review for: desc source (param vs tag/comment), InputObject setter, schema-level.

After review, implement changes.
