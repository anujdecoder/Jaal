# Jaal Missing Features Comparison

**Document Version:** 1.0  
**Last Updated:** 2026-04-07  
**Author:** AI Assistant  
**Purpose:** Compare Jaal's current capabilities against industry-leading GraphQL frameworks (gqlgen, graphql-go, Apollo GraphQL) to identify gaps and guide implementation priorities.

---

## Executive Summary

Jaal is a Go GraphQL server framework with ~90% spec compliance (Oct 2021). It excels at reflection-based schema building and built-in directives but lacks several production-critical features present in mature frameworks like gqlgen and Apollo.

This document details **16 missing or incomplete features**, each with:
1. **Feature Detail** — How the feature works in GraphQL and other frameworks
2. **Reasoning** — Why Jaal should implement it (business/technical value)
3. **Implementation Details** — High-level design and affected files
4. **Success Metrics** — How to measure completion

**Quick Reference Table:**

| Feature | Priority | Complexity | Status |
|---------|----------|------------|--------|
| Custom Directives | P1 | High | Not implemented |
| Query Complexity Analysis | P1 | High | Not implemented |
| DataLoaders / N+1 Prevention | P1 | High | Not implemented |
| Relay-compliant Subscriptions | P1 | Medium | Partial (pubsub-based) |
| Persisted Queries (APQ) | P2 | Medium | Not implemented |
| Tracing / OpenTelemetry | P2 | Medium | Not implemented |
| Schema-First / Code Generation | P3 | Very High | Not applicable (reflection-based) |
| Apollo Federation | P3 | Very High | Not implemented |
| @defer / @stream | P3 | Very High | Not implemented |
| Interface Implements Interface | P2 | Medium | Not implemented |
| Input Field Defaults | P2 | Medium | Not implemented |
| Argument Deprecation | P0 | Low | Stubbed (introspection only) |
| Enum Value Deprecation | P0 | Low | Stubbed |
| Advanced Validation Rules | P2 | High | Partial |
| Plugins / Extensions | P2 | High | Limited (middleware only) |
| Depth Limiting | P2 | Medium | Not implemented |

---

## 1. Custom Directives

### Feature Detail

Custom directives allow users to define their own `@directive` annotations on schema elements (fields, arguments, types) and execute custom logic during query execution, introspection, or schema building.

**How it works:**
- Define a directive with name, description, locations (FIELD_DEFINITION, ARGUMENT_DEFINITION, etc.), arguments, and whether repeatable
- Attach directives to schema elements during registration
- During execution or introspection, the directive's logic (visitor pattern) is invoked
- Examples: `@auth(role: "ADMIN")`, `@cache(ttl: 60)`, `@rateLimit(max: 100)`

**Comparison:**
- **gqlgen**: Full support via `gqlparser` directives, codegen-aware `@goModel`, `@goField`, custom directive implementations via `DirectiveRoot`
- **graphql-go**: Basic support for built-in directives; custom via extensions
- **Apollo**: Full support via `makeExecutableSchema` with directive resolvers

**Jaal Current State:** Only built-in `@skip`, `@include`, `@deprecated`, `@specifiedBy`, `@oneOf` are supported. Custom directives have no registration or execution path.

### Reasoning

1. **Schema enrichment**: Teams need to annotate fields with authorization, caching, logging, rate-limiting, etc.
2. **Ecosystem parity**: All major frameworks support custom directives — users expect this capability
3. **Spec compliance**: GraphQL spec explicitly supports custom directives (Section 3.13)
4. **Extensibility**: Without custom directives, middleware hacks or wrapper patterns are needed — poor DX

### Implementation Details

**Files to modify/create:**
- `graphql/directive.go` — New file with `DirectiveDefinition` struct, `DirectiveLocation` enum, visitor interface
- `schemabuilder/directive.go` — New file with `Directive()` option and registration API
- `graphql/execute.go` — Modify executor to call directive visitors before/after field resolution
- `graphql/types.go` — Extend `Field`, `Object`, etc. to store `Directives []*Directive`
- `introspection/introspection.go` — Add `__Schema.directives` resolver
- `sdl/printer.go` — Output custom directive definitions in SDL

**High-level design:**
```go
// 1. Define directive
sb.Directive("auth", schemabuilder.DirectiveConfig{
    Description: "Requires authentication with role",
    Locations:   []graphql.DirectiveLocation{graphql.FieldDefinition},
    Args: map[string]graphql.Type{
        "role": &graphql.NonNull{Type: &graphql.Scalar{Type: "String"}},
    },
})

// 2. Use on field
obj.FieldFunc("adminData", resolver, schemabuilder.Directive("auth", map[string]interface{}{"role": "ADMIN"}))

// 3. Register visitor (execution-time logic)
graphql.RegisterDirective("auth", &AuthVisitor{})

type AuthVisitor struct{}
func (v *AuthVisitor) VisitField(ctx context.Context, d *graphql.Directive, f *graphql.Field, src interface{}) (interface{}, error) {
    // Check auth, return error or call f.Resolve
}
```

**Phases:**
1. Directive definition schema (registration, introspection)
2. Directive execution framework (visitor pattern)
3. Repeatable directives support

### Success Metrics

- [ ] `sb.Directive("name", config)` registers without error
- [ ] Custom directives appear in introspection (`__Schema.directives`)
- [ ] SDL printer outputs directive definitions
- [ ] Directive visitor is invoked during `Execute()` for matching locations
- [ ] Multiple directives compose correctly (order preserved)
- [ ] Validation rejects directive on wrong location
- [ ] Test coverage: unit + end-to-end with custom `@auth` directive
- [ ] Benchmark: <5% overhead for directive execution on hot path

---

## 2. Query Complexity Analysis

### Feature Detail

Query complexity analysis calculates a "cost" for each query before execution and rejects queries exceeding a configurable threshold. Prevents DoS via expensive nested queries.

**How it works:**
- Each field has a base complexity (default 1)
- Object fields add child complexity recursively
- List fields multiply by `first`/`last` argument (connection pattern)
- Custom complexity functions per field override defaults
- Pre-execution: walk selection set, sum costs, compare to `maxComplexity`

**Comparison:**
- **gqlgen**: Built-in via `ComplexityRoot`, `ComplexityLimit()` option, `@complexity` directive
- **graphql-go**: Manual via middleware
- **Apollo**: `graphql-query-complexity` npm package

**Jaal Current State:** No complexity analysis. Roadmap Phase 4.1.

### Reasoning

1. **Security**: Prevent attackers from crafting expensive queries (`users { posts { comments { replies } } }`)
2. **Performance**: Protect database from N+1 or cartesian explosion
3. **Production necessity**: Standard in any public GraphQL API
4. **User experience**: Clear error message instead of timeout

### Implementation Details

**Files to create/modify:**
- `graphql/complexity.go` — New file with `ComplexityCalculator` struct
- `graphql/execute.go` — Add pre-execution complexity check (before `execute()`)
- `schemabuilder/output.go` — Add `Complexity()` field option
- `graphql/schema.go` — Add `MaxComplexity` field to `Schema`

**High-level design:**
```go
// Schema config
schema := sb.MustBuild(WithMaxComplexity(1000))

// Field-level custom complexity
obj.FieldFunc("friends", resolver,
    schemabuilder.Complexity(func(childComplexity int, args map[string]interface{}) int {
        first := args["first"].(int)
        return 10 + first*childComplexity  // base 10 + N children
    }),
)

// Execution flow
func (e *Executor) Execute(...) {
    if complexity := e.CalculateComplexity(query); complexity > schema.MaxComplexity {
        return nil, fmt.Errorf("query complexity %d exceeds limit %d", complexity, schema.MaxComplexity)
    }
    return e.execute(...)
}
```

**Complexity rules:**
- Scalar/Enum: 1
- Object field: 1 + child complexity
- List: 1 + (first or last or defaultLimit) × child complexity
- Custom: user function result

### Success Metrics

- [ ] `WithMaxComplexity(N)` on `MustBuild` sets limit
- [ ] Query exceeding limit returns error with complexity value
- [ ] Default complexity per field type works (scalar=1, object=1+child)
- [ ] `Complexity()` option overrides per field
- [ ] List fields multiply correctly (respects `first`/`last`)
- [ ] Complexity reported in `extensions.complexity` (optional)
- [ ] Test: 10+ nested queries rejected; simple queries pass
- [ ] Benchmark: complexity calculation <0.5ms for 10KB query

---

## 3. DataLoaders / N+1 Prevention

### Feature Detail

DataLoaders batch and cache data fetches within a single request. Solves the classic N+1 query problem where resolving a list of parent objects triggers N child queries.

**How it works:**
- `DataLoader` wraps a fetch function: `(keys) -> (values)`
- During field resolution, instead of calling `fetch(key)` immediately, call `loader.Load(key)`
- Loader collects keys during the synchronous phase, then batches them into one DB call
- Cache prevents duplicate fetches for the same key within the request
- Typically uses `context` to attach loaders per-request

**Comparison:**
- **gqlgen**: First-class via `dataloader` package + examples; `dataloaden` codegen
- **graphql-go**: Manual implementation via context middleware
- **Apollo**: `dataloader` npm package (Facebook)

**Jaal Current State:** `BatchResolver` type exists in `graphql/types.go` but is commented out in `schemabuilder/output.go`. No loader abstraction.

### Reasoning

1. **Performance**: 100 users → 100 DB queries for posts becomes 1 query (100× improvement)
2. **Database protection**: Prevents connection pool exhaustion
3. **Common pattern**: Expected in any production GraphQL server
4. **Ecosystem**: gqlgen's popularity partly comes from excellent dataloader patterns

### Implementation Details

**Files to create:**
- `graphql/dataloader.go` — New file with `DataLoader` interface and `NewDataLoader(batchFn, opts)`
- `schemabuilder/output.go` — Expose `DataLoader` registration on fields
- `graphql/execute.go` — Context-aware loader injection

**High-level design:**
```go
// Define loader
userLoader := dataloader.NewDataLoader(func(ctx context.Context, keys []interface{}) ([]interface{}, error) {
    ids := make([]string, len(keys))
    for i, k := range keys { ids[i] = k.(string) }
    return db.LoadUsersByIDs(ids)
})

// Register on field
obj.FieldFunc("author", func(ctx context.Context, post *Post) *User {
    return dataloader.Load(ctx, "userLoader", post.AuthorID).(*User)
}, schemabuilder.WithDataLoader("userLoader", userLoader))

// Per-request context
ctx = dataloader.Attach(ctx, "userLoader", userLoader)
result, _ := executor.Execute(ctx, ...)
```

**Alternative (simpler for Jaal's reflection model):**
```go
// In resolver, use batch hint
obj.FieldFunc("posts", func(ctx context.Context, user *User) []*Post {
    return db.GetPostsForUser(user.ID)  // Still N+1 unless...
}, schemabuilder.UseBatch(true))  // Future: auto-batch sibling resolvers
```

### Success Metrics

- [ ] `DataLoader` interface with `Load(key) -> Promise` semantics
- [ ] Batch function called once per request with all collected keys
- [ ] Cache prevents duplicate loads for same key
- [ ] Per-request via context (no global state)
- [ ] Example: resolving 100 users' posts in 1 DB query (not 100)
- [ ] Benchmark: 1000 items batched in <10ms vs naive 1000× single fetches
- [ ] Documentation with common patterns (user→posts, post→author)

---

## 4. Relay-compliant Subscriptions

### Feature Detail

GraphQL subscriptions enable real-time updates via WebSocket. Relay compliance adds cursor-based pagination patterns, connection types, and stable subscription semantics per the Relay spec.

**How it works:**
- Client subscribes: `subscription { messageAdded(roomId: "1") { id text } }`
- Server maintains WebSocket, pushes events matching subscription
- Relay adds: `connection` wrapper, `pageInfo`, `edges` for paginated live data
- Subscription payload includes `operationId` for multi-subscription management

**Comparison:**
- **gqlgen**: WebSocket via `gorilla/websocket` + subscription resolver pattern; docs for Relay
- **graphql-go**: Basic subscription support
- **Apollo**: `graphql-subscriptions` + `subscriptions-transport-ws`

**Jaal Current State:** `ws.go` implements WebSocket + pubsub-based subscriptions. Not Relay-compliant (no connection types, no cursor semantics). Roadmap: "Work in progress."

### Reasoning

1. **Real-time apps**: Chat, notifications, live dashboards need subscriptions
2. **Relay ecosystem**: Many frontend tools (Relay, Apollo Client) expect Relay patterns
3. **Production parity**: All major frameworks support subscriptions
4. **Current gap**: Jaal's pubsub approach works but lacks standard patterns

### Implementation Details

**Files to modify:**
- `ws.go` — Enhance subscription protocol (support `connection_init`, `start`, `stop`, `data`, `error`)
- `schemabuilder/schema.go` — Add subscription field registration helpers
- `graphql/types.go` — Add `Subscription` type with connection semantics
- `example/users/register_subscriptions.go` — Reference Relay-compliant example

**High-level design:**
```go
// Relay-compliant subscription
schema.Subscription().FieldFunc("messageAdded", func(ctx context.Context, args struct {
    RoomID string
}) *MessageConnection {
    // Return a connection that streams edges
    return &MessageConnection{
        Edges:   make(chan *MessageEdge),
        PageInfo: &PageInfo{},
    }
}, schemabuilder.FieldDesc("New messages in room"))

// WS protocol
// Client: {"id":"1","type":"start","payload":{"query":"subscription { messageAdded { ... } }"}}
// Server: {"id":"1","type":"data","payload":{"data":{"messageAdded":{...}}}}
```

### Success Metrics

- [ ] WebSocket upgrade succeeds, protocol matches graphql-ws/subscriptions-transport-ws
- [ ] Multiple clients receive same event (broadcast)
- [ ] Client can subscribe/unsubscribe independently
- [ ] Connection types (`edges`, `pageInfo`) work with `first`/`last`
- [ ] Error in one subscription doesn't kill connection
- [ ] Test: 2 clients subscribe, 1 publishes, both receive
- [ ] Benchmark: 100 concurrent subscribers, <50ms fan-out

---

## 5. Persisted Queries (Automatic Persisted Queries)

### Feature Detail

APQ lets clients send a SHA-256 hash of the query instead of the full text. Server looks up query by hash in cache. Saves bandwidth, enables query allowlisting.

**How it works:**
- Client computes `sha256(query)` → sends `extensions.persistedQuery.sha256Hash`
- Server checks cache; if hit, executes cached query; if miss, returns error with hash
- Client retries with full query text; server caches it
- Subsequent requests only send hash

**Comparison:**
- **gqlgen**: Via middleware + cache (Redis/memory); no built-in but patterns exist
- **graphql-go**: No built-in
- **Apollo**: Built into Apollo Server with Redis

**Jaal Current State:** Not implemented. Roadmap Phase 4.2.

### Reasoning

1. **Bandwidth**: Large queries (with fragments) are costly on mobile
2. **Security**: Allowlist known queries, reject unknown (prevents injection)
3. **Caching**: Server can cache parsed AST, skip parsing step
4. **CDN-friendly**: Hashed queries can be GET requests (cacheable)

### Implementation Details

**Files to modify:**
- `http.go` — Check `extensions.persistedQuery` in POST body
- `graphql/parser.go` — Cache parsed queries by hash
- New: `graphql/persisted.go` — APQ logic, cache interface

**High-level design:**
```go
type PersistedQueryCache interface {
    Get(hash string) (*Query, bool)
    Set(hash string, q *Query)
}

// In HTTP handler
func (h *httpHandler) ServeHTTP(...) {
    var body httpPostBody
    json.Unmarshal(...)
    
    if ext := body.Extensions; ext != nil {
        if pq := ext.PersistedQuery; pq != nil {
            if q, ok := cache.Get(pq.Sha256Hash); ok {
                // Execute cached
            } else {
                // Return error: { "errors": [{ "message": "PersistedQueryNotFound" }] }
            }
        }
    }
}
```

### Success Metrics

- [ ] Client sends only hash → server executes from cache
- [ ] Cache miss returns `PersistedQueryNotFound` error
- [ ] Client retries with full query → server caches and executes
- [ ] Cache hit bypasses parsing (measure via benchmark)
- [ ] Works with variables (hash includes variables or separate)
- [ ] Test: 1000 requests with hash, 0 parses
- [ ] Benchmark: hash lookup <0.1ms; parse + cache >1ms

---

## 6. Tracing / OpenTelemetry

### Feature Detail

Tracing captures timing and metadata for each field resolution, enabling distributed tracing (Jaeger, Zipkin) and performance profiling.

**How it works:**
- Wrap resolver execution with span start/end
- Capture: field name, type, arguments (sanitized), duration, errors
- Propagate trace context via `context` (traceparent header)
- Export spans to OTLP collector

**Comparison:**
- **gqlgen**: `opentelemetry` extension; `WithTracing()` middleware
- **graphql-go**: Via `extensions` or custom middleware
- **Apollo**: `apollo-tracing` extension (deprecated in favor of OTEL)

**Jaal Current State:** No tracing hooks. Middleware exists but no span integration.

### Reasoning

1. **Observability**: Know which resolvers are slow in production
2. **Debugging**: Correlate GraphQL errors with DB calls, network latency
3. **SLA compliance**: Track p99 latency per field
4. **Modern standard**: OpenTelemetry is the de facto observability spec

### Implementation Details

**Files to create:**
- `graphql/tracing.go` — New file with `Tracer` interface, `StartSpan(ctx, name, attrs)`
- `graphql/execute.go` — Wrap `executeObject` / `executeList` with spans
- `http.go` — Extract trace context from headers

**High-level design:**
```go
type Tracer interface {
    StartSpan(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, Span)
}

type Span interface {
    End()
    SetError(err error)
}

// In executor
func (e *Executor) executeField(ctx, field, source) {
    ctx, span := tracer.StartSpan(ctx, "graphql.resolve."+field.Name, map[string]interface{}{
        "graphql.field": field.Name,
        "graphql.type":  field.Type.String(),
    })
    defer span.End()
    
    result, err := field.Resolve(ctx, source, args, selectionSet)
    if err != nil { span.SetError(err) }
    return result
}
```

### Success Metrics

- [ ] Spans emitted for each field resolution
- [ ] Span attributes: field name, type, duration, error
- [ ] Trace context propagated via context
- [ ] Integration with OTEL SDK (export to collector)
- [ ] No overhead when tracing disabled
- [ ] Test: end-to-end trace visible in Jaeger UI
- [ ] Benchmark: tracing adds <2% overhead

---

## 7. Schema-First / Code Generation (gqlgen-style)

### Feature Detail

Schema-first means defining GraphQL schema in SDL (`.graphqls` files), then generating Go types and resolver stubs via `go generate`. Opposite of Jaal's reflection-based code-first.

**How it works:**
- Write `schema.graphqls` with types, queries, mutations
- Run `gqlgen generate` → produces `generated.go` with interfaces
- Implement resolver interfaces in your code
- Types are generated, no reflection at runtime

**Comparison:**
- **gqlgen**: This is its core model
- **graphql-go**: Code-first only
- **Apollo**: Schema-first (SDL) standard

**Jaal Current State:** Reflection-based code-first only. No code generation.

### Reasoning

1. **Type safety**: Generated types catch errors at compile time
2. **Schema as contract**: Frontend and backend agree on SDL
3. **DX**: No manual `FieldFunc` registration boilerplate
4. **Ecosystem fit**: Many teams prefer SDL-first workflow

### Implementation Details

**Files to create (major effort):**
- `codegen/` — New package with SDL parser, Go code generator
- Templates for: types, resolvers, enums, inputs
- CLI: `jaal generate` (or integrate with existing `jaal` binary)

**Note:** This is a **paradigm shift**, not a feature addition. Jaal's reflection model is intentional for struct-first teams. Consider this lower priority unless user demand is high.

### Success Metrics

- [ ] `jaal generate --schema schema.graphqls --out generated/` produces valid Go
- [ ] Generated types match SDL (scalars, objects, enums, inputs)
- [ ] Resolver interfaces match field signatures
- [ ] `go build` succeeds; resolvers compile
- [ ] Example project in `example/codegen/`
- [ ] Docs: "Schema-first with Jaal" tutorial

---

## 8. Apollo Federation

### Feature Detail

Federation allows splitting a GraphQL API across multiple services (subgraphs). A gateway composes them into a single unified schema. Each subgraph contributes types/fields; `@key`, `@extends`, `@external` directives coordinate.

**How it works:**
- Subgraph defines entity: `type User @key(fields: "id") { id name }`
- Another subgraph extends: `extend type User @key(fields: "id") { posts: [Post] }`
- Gateway fetches from each service, merges results
- Supports federation v1 (SDL) and v2 (new directives)

**Comparison:**
- **gqlgen**: Official federation support via `@federation` plugin
- **graphql-go**: No built-in; community forks exist
- **Apollo**: Native (Apollo Server + Apollo Gateway/Router)

**Jaal Current State:** No federation support. Not on roadmap.

### Reasoning

1. **Microservices**: Large orgs split GraphQL across teams
2. **Independent deploy**: Each service deploys its subgraph independently
3. **Ecosystem**: Federation is the de facto standard for distributed GraphQL
4. **Enterprise adoption**: Without federation, Jaal won't be chosen for large projects

### Implementation Details

**Files to create:**
- `federation/` — New package with `@key`, `@extends`, `@external`, `@provides`, `@requires`
- `graphql/federation.go` — Entity resolution, `_service` and `_entities` resolvers
- `schemabuilder/federation.go` — Federation directive options

**High-level design:**
```go
// Subgraph A
obj := sb.Object("User", User{})
obj.Key("id")  // @key(fields: "id")
obj.FieldFunc("id", ...)
obj.FieldFunc("name", ...)

// Subgraph B (extends)
obj := sb.ExtendObject("User")
obj.FieldFunc("posts", func(ctx, user) []*Post { ... }, schemabuilder.External(false))

// Gateway (separate service) composes subgraphs
```

### Success Metrics

- [ ] SDL includes federation directives
- [ ] `_service { sdl }` introspection works
- [ ] `_entities(representations: [...])` resolves entities
- [ ] Multi-subgraph query returns merged data
- [ ] Compatible with Apollo Router (fetches subgraph SDL)
- [ ] Test: 2 subgraphs, gateway, single query spans both

---

## 9. @defer / @stream (Incremental Delivery)

### Feature Detail

Incremental delivery sends query results in parts: initial payload immediately, then deferred fragments or streamed list items later. Reduces time-to-first-byte for large queries.

**How it works:**
- `@defer` on fragment: `query { user { name ...@defer { friends } } }`
- Server returns: `{ data: { user: { name: "A" } }, hasNext: true }` then `{ incremental: [{ data: { friends: [...] } }] }`
- Response uses `multipart/mixed` content type
- `@stream` on lists: sends first N items immediately, rest streamed

**Comparison:**
- **gqlgen**: No built-in (spec is new; experimental)
- **graphql-go**: No
- **Apollo**: Supports via `@defer` / `@stream` with Apollo Client

**Jaal Current State:** Not implemented. Roadmap Phase 5.

### Reasoning

1. **Perceived performance**: Users see data faster (name loads before friends)
2. **Large queries**: Avoid waiting for entire response
3. **Spec compliance**: Oct 2021+ spec includes these (RFC 742)
4. **Modern UX**: Expected for interactive apps

### Implementation Details

**Files to create:**
- `graphql/defer.go` — Defer handling, multipart response
- `graphql/stream.go` — Stream handling for lists
- `graphql/execute.go` — Modify to support incremental results

**High-level design:**
```go
// In executor, detect @defer on selection
if hasDirective(selection, "defer") {
    go func() {
        result := e.execute(ctx, typ, src, selection.SelectionSet)
        incremental <- IncrementalResult{Label: label, Data: result}
    }()
    // Continue with main response (omit deferred field)
}

// Response format
Content-Type: multipart/mixed; boundary="-"
--\r\n
Content-Type: application/json\r\n
\r\n
{"data":{"user":{"name":"Alice"}},"hasNext":true}
--\r\n
Content-Type: application/json\r\n
\r\n
{"incremental":[{"label":"friends","data":{"friends":[...]}}],"hasNext":false}
--\r\n
```

### Success Metrics

- [ ] `@defer` on fragment returns initial response immediately
- [ ] Deferred data arrives in subsequent parts
- [ ] `hasNext: true/false` signals end of stream
- [ ] `@stream(initialCount: N)` on list sends first N then rest
- [ ] Error in one incremental part doesn't fail whole response
- [ ] Client (e.g., Apollo) can merge incremental results
- [ ] Test: query with 1s defer returns in <100ms, full data in ~1s

---

## 10. Interface Implements Interface

### Feature Detail

GraphQL interfaces can implement other interfaces, forming hierarchies (e.g., `Node` → `User` → `Admin`).

**How it works:**
- `interface Node { id: ID! }`
- `interface User implements Node { id: ID! name: String }`
- `type Admin implements User { ... }` — Admin has Node's fields too
- Introspection: `__Type.interfaces` lists implemented interfaces

**Comparison:**
- **gqlgen**: Supported via `implements` in SDL
- **graphql-go**: Supported
- **Jaal**: Not currently supported

**Jaal Current State:** Interfaces exist (`schemabuilder.Interface` marker) but no interface inheritance.

### Reasoning

1. **Type reuse**: Common fields (e.g., `id`, `createdAt`) defined once
2. **Polymorphism**: Query `node(id: "1") { ... on Node { id } }` works for all
3. **Spec compliance**: Oct 2021 spec Section 3.6.1

### Implementation Details

**Files to modify:**
- `schemabuilder/output.go` — `Interface()` option accepts `Implements("ParentInterface")`
- `graphql/types.go` — `Interface` struct gains `Interfaces []string`
- `graphql/validate.go` — Check implementing types have all parent fields
- `introspection/introspection.go` — `__Type.interfaces` resolver

### Success Metrics

- [ ] `sb.Interface("User", User{}, Implements("Node"))` registers
- [ ] SDL outputs `interface User implements Node`
- [ ] Introspection shows `interfaces: [{ name: "Node" }]`
- [ ] Query selecting parent interface fields on child type works
- [ ] Validation: implementing type missing parent field → error

---

## 11. Input Field Defaults

### Feature Detail

Input object fields can have default values used when the client omits the field.

**How it works:**
- `input CreateUserInput { role: String = "USER" active: Boolean = true }`
- Client sends `{ name: "Alice" }` → server sees `{ name: "Alice", role: "USER", active: true }`

**Comparison:**
- **gqlgen**: Supported in SDL (`= "default"`)
- **graphql-go**: Supported
- **Jaal**: Not implemented

**Jaal Current State:** `graphql.InputObject` has no `DefaultValue` per field. Roadmap Phase 3.2.

### Reasoning

1. **Ergonomics**: Clients don't send obvious defaults
2. **Backward compatibility**: Add new required-with-default fields without breaking clients
3. **Spec compliance**: Oct 2021 spec Section 3.10

### Implementation Details

**Files to modify:**
- `graphql/types.go` — `InputObject.InputFields` → map to struct with `DefaultValue`
- `schemabuilder/input_object.go` — `FieldFunc` accepts `DefaultValue(v)` option
- `graphql/parser.go` — Apply defaults during input coercion

### Success Metrics

- [ ] `input.FieldFunc("role", ..., schemabuilder.DefaultValue("USER"))`
- [ ] Omitted field in query gets default value
- [ ] Explicit `null` overrides default (or per spec, errors if non-nullable)
- [ ] Introspection shows `defaultValue: "USER"`
- [ ] SDL outputs default

---

## 12. Argument Deprecation (Complete)

### Feature Detail

Mark individual field arguments as deprecated (e.g., `user(id: ID, legacyName: String @deprecated)`).

**How it works:**
- Introspection: `__Field.args[].isDeprecated`, `__Field.args[].deprecationReason`
- SDL: `field(arg: Type @deprecated(reason: "..."))`

**Jaal Current State:** Introspection fields exist but always return `false`/`nil` (stubbed). See `DESIGN_ARG_ENUM_DEPRECATION.md`.

### Reasoning

1. **Migration**: Deprecate old args while new ones stabilize
2. **Spec compliance**: ARGUMENT_DEFINITION location for `@deprecated`
3. **Playground**: Deprecated args shown with strikethrough

### Implementation Details

**Files to modify:**
- `graphql/types.go` — `Field.ArgIsDeprecated`, `Field.ArgDeprecationReason`
- `schemabuilder/function.go` — Capture `ArgDeprecation("name", "reason")` option
- `introspection/introspection.go` — Read from actual metadata

See full design in `docs/DESIGN_ARG_ENUM_DEPRECATION.md`.

### Success Metrics

- [ ] `FieldFunc("user", fn, ArgDeprecation("name", "Use ID"))`
- [ ] Introspection: `args[1].isDeprecated == true`
- [ ] SDL: `user(name: String @deprecated(reason: "Use ID"))`
- [ ] Playground shows strikethrough

---

## 13. Enum Value Deprecation (Complete)

### Feature Detail

Mark specific enum values as deprecated.

**How it works:**
- `enum Status { ACTIVE INACTIVE @deprecated(reason: "...") }`
- Introspection: `__EnumValue.isDeprecated`, `__EnumValue.deprecationReason`

**Jaal Current State:** Stubbed in `introspection/introspection.go:470` (always false/nil).

### Reasoning

1. **Enum evolution**: Phase out values without removing them
2. **Spec compliance**: ENUM_VALUE location for `@deprecated`

### Implementation Details

**Files to modify:**
- `schemabuilder/schema.go` — `Enum()` accepts `EnumValueDeprecation("VAL", "reason")`
- `graphql/types.go` — `Enum.ValueDeprecations map[string]string`
- `introspection/introspection.go` — Read from `ValueDeprecations`

### Success Metrics

- [ ] `schema.Enum(Status(0), map{...}, EnumValueDeprecation("OLD", "Use NEW"))`
- [ ] Introspection: enum value shows `isDeprecated: true`
- [ ] SDL: `OLD @deprecated(reason: "Use NEW")`

---

## 14. Advanced Validation Rules

### Feature Detail

GraphQL spec defines ~20 validation rules (e.g., "Fields on correct type", "No unused variables", "Unique argument names"). Jaal implements only a subset.

**How it works:**
- `ValidateQuery()` in `graphql/validate.go` checks selection sets
- Missing rules: argument names, variable usage, directive locations, etc.

**Jaal Current State:** Partial. Core rules exist; many spec rules missing.

### Reasoning

1. **Spec compliance**: Full validation is required for "spec compliant"
2. **Better errors**: Catch mistakes early with precise locations
3. **Security**: Prevent malformed queries from reaching resolvers

### Implementation Details

**Files to modify:**
- `graphql/validate.go` — Add remaining rules from spec Section 5

**Missing rules (per roadmap):**
- Fields on correct type
- Fragments on composite types
- Leaf field selections
- Argument names / uniqueness / required
- Directives in allowed locations
- Variable usage allowed / defined

### Success Metrics

- [ ] All 20+ validation rules from spec implemented
- [ ] Multiple errors returned (not just first)
- [ ] Errors include line/column
- [ ] Test against official GraphQL spec examples

---

## 15. Plugins / Extensions System

### Feature Detail

Extensible architecture allowing third-party code to hook into parsing, validation, execution, and response phases.

**How it works:**
- Define extension points: `OnParse`, `OnValidate`, `OnExecute`, `OnResponse`
- Plugins register handlers; framework calls them in order
- Example: tracing plugin, logging plugin, auth plugin

**Jaal Current State:** Basic middleware (`WithMiddlewares`) exists but no structured plugin system.

### Reasoning

1. **Extensibility**: Users add features without forking
2. **Ecosystem**: Plugins for auth, rate limiting, caching, etc.
3. **Comparison**: gqlgen has strong plugin hooks

### Implementation Details

**Files to create:**
- `graphql/extension.go` — `Extension` interface with lifecycle hooks
- `graphql/execute.go` — Call extension hooks around execution

### Success Metrics

- [ ] `Extension` interface with `OnExecuteStart`, `OnExecuteEnd`, etc.
- [ ] Multiple extensions compose correctly
- [ ] Example: logging extension prints field timings
- [ ] Benchmark: extension overhead <1%

---

## 16. Depth Limiting

### Feature Detail

Reject queries deeper than N levels to prevent stack overflow or expensive traversals.

**How it works:**
- Pre-execution: walk selection set, track nesting depth
- If `depth > maxDepth`, return error

**Jaal Current State:** Not implemented.

### Reasoning

1. **Security**: `a { b { c { d { ... } } } }` with 1000 levels crashes server
2. **Simpler than complexity**: Easier to implement, catches many DoS cases
3. **Common pattern**: Most GraphQL servers have this

### Implementation Details

**Files to modify:**
- `graphql/validate.go` — Add `MaxDepth` check
- `graphql/schema.go` — Add `MaxDepth` field

### Success Metrics

- [ ] `WithMaxDepth(10)` on schema
- [ ] Query with depth 11 rejected
- [ ] Query with depth 10 allowed
- [ ] Error message: "query depth 11 exceeds max 10"

---

## Summary & Prioritization

| Feature | Priority | Est. Effort | Dependencies |
|---------|----------|-------------|--------------|
| Argument Deprecation | P0 | Low | — |
| Enum Value Deprecation | P0 | Low | — |
| Custom Directives | P1 | High | — |
| Query Complexity | P1 | High | — |
| DataLoaders | P1 | High | — |
| Relay Subscriptions | P1 | Medium | — |
| Interface Implements Interface | P2 | Medium | — |
| Input Field Defaults | P2 | Medium | — |
| Persisted Queries | P2 | Medium | — |
| Tracing / OTEL | P2 | Medium | — |
| Advanced Validation | P2 | High | — |
| Plugins / Extensions | P2 | High | — |
| Depth Limiting | P2 | Medium | — |
| @defer / @stream | P3 | Very High | Complexity |
| Apollo Federation | P3 | Very High | — |
| Code Generation | P3 | Very High | — |

**Recommended first steps (P0 + P1):**
1. Complete deprecation (argument + enum) — unblocks introspection parity
2. Custom directives — unblocks auth, caching, rate-limiting patterns
3. Query complexity — critical security feature
4. DataLoaders — performance essential for real-world use

---

## Appendix: Current Jaal Feature Matrix

| Feature | Jaal | gqlgen | graphql-go | Apollo |
|---------|------|--------|------------|--------|
| Core execution | ✅ | ✅ | ✅ | ✅ |
| Introspection | ✅ | ✅ | ✅ | ✅ |
| Built-in directives | ✅ | ✅ | ✅ | ✅ |
| Custom directives | ❌ | ✅ | ⚠️ | ✅ |
| Subscriptions | ⚠️ | ✅ | ✅ | ✅ |
| Federation | ❌ | ✅ | ❌ | ✅ |
| DataLoaders | ⚠️ | ✅ | ⚠️ | ✅ |
| Complexity | ❌ | ✅ | ⚠️ | ✅ |
| Tracing | ❌ | ✅ | ⚠️ | ✅ |
| Persisted Queries | ❌ | ⚠️ | ❌ | ✅ |
| Code generation | ❌ | ✅ | ❌ | N/A |
| Middleware | ✅ | ✅ | ⚠️ | ✅ |
| Reflection-based | ✅ | ❌ | ❌ | ❌ |

Legend: ✅ full, ⚠️ partial, ❌ none

---

*This document is a living reference. Update as features are implemented or priorities shift.*
