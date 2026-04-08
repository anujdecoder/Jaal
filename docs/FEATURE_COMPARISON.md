# Jaal Feature Comparison Document

**Version:** 1.0  
**Last Updated:** 2026-04-07  
**Compared Against:** Apollo GraphQL (JS), gqlgen (Go), graphql-go, Thunder (Go), Hasura

---

## Executive Summary

This document provides a comprehensive comparison of features missing from Jaal when compared to other popular GraphQL frameworks. Each feature is analyzed with detailed explanations, implementation reasoning, high-level implementation details, and success metrics.

**Current Jaal Compliance:** ~90% GraphQL October 2021 Spec

---

## Table of Contents

1. [Custom Directives Framework](#1-custom-directives-framework)
2. [Federation Support](#2-federation-support)
3. [Query Complexity Analysis](#3-query-complexity-analysis)
4. [Persisted Queries (APQ)](#4-persisted-queries-apq)
5. [Incremental Delivery (@defer/@stream)](#5-incremental-delivery-deferstream)
6. [Real-time Subscriptions Enhancement](#6-real-time-subscriptions-enhancement)
7. [Plugin/Middleware Ecosystem](#7-pluginmiddleware-ecosystem)
8. [Schema Stitching](#8-schema-stitching)
9. [Advanced Caching](#9-advanced-caching)
10. [File Upload Support](#10-file-upload-support)
11. [Error Handling & Extensions](#11-error-handling--extensions)
12. [Observability & Tracing](#12-observability--tracing)
13. [Security Features](#13-security-features)
14. [Developer Tooling](#14-developer-tooling)
15. [Complete Validation Rules](#15-complete-validation-rules)

---

## 1. Custom Directives Framework

### Feature Detail

Custom directives allow developers to define their own directives that can modify query execution behavior. In GraphQL, directives are annotations like `@auth(role: "admin")` or `@cache(ttl: 300)` that can be applied to various schema locations (fields, arguments, types, etc.).

**How it works in other frameworks:**
- **Apollo:** Full directive support with `mapSchema` and `SchemaDirectiveVisitor`
- **gqlgen:** Directive middleware pattern with type-safe implementations
- **graphql-go:** Directive resolvers that intercept field execution

```graphql
# Example custom directive usage
type Query {
  sensitiveData: Secret @auth(requires: ADMIN)
  cachedData: Data @cache(ttl: 300)
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P1 (High)** | High | High |

1. **Spec Compliance:** Custom directives are part of the GraphQL specification
2. **Extensibility:** Enables framework users to implement cross-cutting concerns (auth, logging, caching) declaratively
3. **Ecosystem Parity:** Other frameworks have this as a core feature
4. **Real-world Necessity:** Most production GraphQL APIs need custom directives for auth, caching, rate limiting

### Implementation Details

**Phase 1: Directive Definition Schema**

```go
// graphql/directive.go
type DirectiveDefinition struct {
    Name          string
    Description  string
    Locations    []DirectiveLocation
    Args         map[string]*Argument
    IsRepeatable bool  // Oct 2021+ feature
}

type DirectiveLocation string

const (
    // Executable directive locations
    LocationQuery              DirectiveLocation = "QUERY"
    LocationMutation           DirectiveLocation = "MUTATION"
    LocationSubscription       DirectiveLocation = "SUBSCRIPTION"
    LocationField              DirectiveLocation = "FIELD"
    LocationFragmentDefinition DirectiveLocation = "FRAGMENT_DEFINITION"
    LocationFragmentSpread     DirectiveLocation = "FRAGMENT_SPREAD"
    LocationInlineFragment     DirectiveLocation = "INLINE_FRAGMENT"
    
    // Type system directive locations
    LocationSchema              DirectiveLocation = "SCHEMA"
    LocationScalar              DirectiveLocation = "SCALAR"
    LocationObject              DirectiveLocation = "OBJECT"
    LocationFieldDefinition     DirectiveLocation = "FIELD_DEFINITION"
    LocationArgumentDefinition  DirectiveLocation = "ARGUMENT_DEFINITION"
    LocationInterface           DirectiveLocation = "INTERFACE"
    LocationUnion               DirectiveLocation = "UNION"
    LocationEnum                DirectiveLocation = "ENUM"
    LocationEnumValue           DirectiveLocation = "ENUM_VALUE"
    LocationInputObject         DirectiveLocation = "INPUT_OBJECT"
    LocationInputFieldDefinition DirectiveLocation = "INPUT_FIELD_DEFINITION"
)
```

**Phase 2: Directive Visitor Interface**

```go
// schemabuilder/directive.go
type DirectiveVisitor interface {
    // Called during field resolution
    VisitField(ctx context.Context, directive *Directive, field *Field, source interface{}) (interface{}, error)
    
    // Called during argument parsing
    VisitArgument(ctx context.Context, directive *Directive, arg *Argument, value interface{}) (interface{}, error)
}

// Registration API
func (s *Schema) Directive(name string, config DirectiveConfig, visitor DirectiveVisitor) {
    // Register directive definition and visitor
}
```

**Phase 3: Execution Integration**

```go
// Modify execute.go to check directives
func (e *Executor) resolveAndExecute(ctx context.Context, field *Field, source interface{}, selection *Selection) (interface{}, error) {
    // Execute directive visitors before resolution
    for _, directive := range selection.Directives {
        if visitor, ok := e.directiveVisitors[directive.Name]; ok {
            result, err := visitor.VisitField(ctx, directive, field, source)
            if err != nil {
                return nil, err
            }
            if result != nil {
                return result, nil  // Directive short-circuited execution
            }
        }
    }
    
    // Continue with normal resolution
    return field.Resolve(ctx, source, selection.Args, selection.SelectionSet)
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Custom directives can be defined | ✅ |
| Directives appear in introspection | ✅ |
| SDL output includes directive definitions | ✅ |
| Directive visitors execute in correct order | ✅ |
| Validation rejects directives in wrong locations | ✅ |
| Performance overhead | < 5% |
| Test coverage | > 90% |

---

## 2. Federation Support

### Feature Detail

Apollo Federation allows multiple GraphQL services to be composed into a single unified graph. Each service defines its own schema and can reference types from other services.

**How it works:**
- Services define "federated" types with `@key`, `@external`, `@requires`, `@provides` directives
- A gateway composes all services into a single schema
- Queries are distributed across services and results are merged

```graphql
# Service A (Users)
type User @key(fields: "id") {
  id: ID!
  name: String!
}

# Service B (Orders)
type Order {
  id: ID!
  user: User @provides(fields: "name")
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P2 (Medium)** | Very High | Very High |

1. **Microservices Architecture:** Essential for organizations with distributed services
2. **Industry Standard:** Apollo Federation is the de-facto standard for GraphQL microservices
3. **Scalability:** Enables independent deployment and scaling of services
4. **Team Autonomy:** Different teams can own different subgraphs

### Implementation Details

**Phase 1: Federation Schema Directives**

```go
// schemabuilder/federation.go

type FederationConfig struct {
    KeyFields     []string  // @key(fields: "id")
    External      bool      // @external
    Requires      []string  // @requires(fields: "email")
    Provides      []string  // @provides(fields: "name")
}

func (s *Schema) FederatedObject(name string, typ interface{}, fedConfig FederationConfig, opts ...TypeOption) *Object {
    obj := s.Object(name, typ, opts...)
    obj.federation = fedConfig
    return obj
}
```

**Phase 2: Federation Entity Resolution**

```go
// Add _entities query for federation
func RegisterFederationQueries(schema *schemabuilder.Schema, resolver EntityResolver) {
    schema.Query().FieldFunc("_entities", func(ctx context.Context, args struct {
        Representations []map[string]interface{}
    }) []Entity {
        return resolver.ResolveEntities(ctx, args.Representations)
    })
    
    schema.Query().FieldFunc("_service", func() ServiceDefinition {
        return ServiceDefinition{Sdl: schema.SDL()}
    })
}
```

**Phase 3: Reference Resolver Interface**

```go
type EntityResolver interface {
    ResolveEntity(ctx context.Context, typename string, key map[string]interface{}) (interface{}, error)
}

// User implements EntityResolver
func (u *User) ResolveEntity(ctx context.Context, typename string, key map[string]interface{}) (interface{}, error) {
    return u.repo.FindByID(key["id"].(string))
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Federation 2.0 spec compliance | 100% |
| Can run as federated subgraph | ✅ |
| `_entities` query works correctly | ✅ |
| `@key`, `@external`, `@requires`, `@provides` supported | ✅ |
| Gateway can compose Jaal services | ✅ |
| Performance overhead vs non-federated | < 10% |

---

## 3. Query Complexity Analysis

### Feature Detail

Query complexity analysis calculates a "cost" for each GraphQL query and rejects queries that exceed a configured threshold. This prevents denial-of-service attacks and resource exhaustion.

**How it works:**
- Each field has an associated complexity cost
- List fields multiply cost by estimated item count
- Custom complexity functions for expensive resolvers
- Query rejected if total complexity exceeds limit

```graphql
# Example: This query might have complexity 105
query {
  user {           # cost: 1
    friends(first: 10) {  # cost: 10 + (10 * 1) = 20
      posts {      # cost: 10 * 1 = 10
        comments { # cost: 10 * 10 = 100
          text
        }
      }
    }
  }
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P1 (High)** | High | Medium |

1. **Security:** Essential for production deployments to prevent DoS
2. **Resource Management:** Prevents runaway queries from consuming server resources
3. **Cost Control:** Important for metered cloud environments
4. **Industry Standard:** Apollo, Hasura, and others include this feature

### Implementation Details

**Phase 1: Complexity Types and Interface**

```go
// graphql/complexity.go

type ComplexityEstimator func(childComplexity int, args map[string]interface{}) int

type FieldComplexity struct {
    BaseComplexity int
    Estimator      ComplexityEstimator
}

type ComplexityConfig struct {
    MaxComplexity     int
    DefaultComplexity int
    ScalarCost        int
    ObjectCost        int
    ListMultiplier    func(args map[string]interface{}) int
}
```

**Phase 2: Complexity Calculation**

```go
func CalculateComplexity(selectionSet *SelectionSet, typ Type, config ComplexityConfig) (int, error) {
    complexity := 0
    
    for _, selection := range selectionSet.Selections {
        field := typ.Fields[selection.Name]
        
        // Base field complexity
        fieldCost := config.DefaultComplexity
        
        // Custom estimator
        if field.ComplexityEstimator != nil {
            fieldCost = field.ComplexityEstimator(fieldCost, selection.Args)
        }
        
        // List multiplication
        if _, isList := field.Type.(*List); isList {
            multiplier := config.ListMultiplier(selection.Args)
            fieldCost *= multiplier
        }
        
        // Recursively calculate child complexity
        childComplexity, _ := CalculateComplexity(selection.SelectionSet, field.Type, config)
        
        complexity += fieldCost + childComplexity
    }
    
    return complexity, nil
}
```

**Phase 3: Schema Builder Integration**

```go
// Register field with custom complexity
obj.FieldFunc("expensiveOperation", resolver,
    schemabuilder.Complexity(func(childComplexity int, args map[string]interface{}) int {
        limit := args["limit"].(int)
        return 10 + (limit * childComplexity)
    }),
)

// Build schema with complexity limit
schema := sb.MustBuild(graphql.WithMaxComplexity(1000))
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Complexity calculated for all queries | ✅ |
| Configurable max complexity per schema | ✅ |
| Custom complexity functions supported | ✅ |
| Complexity reported in response extensions | ✅ |
| Performance overhead | < 1ms for 10KB queries |
| Meaningful error messages | ✅ |

---

## 4. Persisted Queries (APQ)

### Feature Detail

Automatic Persisted Queries (APQ) allows clients to send a query hash instead of the full query text. The server caches the mapping between hash and query, reducing bandwidth usage.

**How it works:**
1. Client sends query with SHA-256 hash in extensions
2. Server checks cache for hash
3. If found: execute cached query
4. If not found: return error, client sends full query
5. Server caches query for future requests

```json
// Client request with APQ
{
  "extensions": {
    "persistedQuery": {
      "version": 1,
      "sha256Hash": "ecf4edb46db40b5132295c0291d62fb65d6759a9eedfa4d5d612dd5ec54a6b38"
    }
  }
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P2 (Medium)** | Medium | Medium |

1. **Bandwidth Reduction:** Up to 90% reduction for large queries
2. **Performance:** Faster parsing for cached queries
3. **Mobile Optimization:** Critical for mobile clients with limited bandwidth
4. **Industry Standard:** Apollo Client uses APQ by default

### Implementation Details

**Phase 1: Query Hash Cache Interface**

```go
// graphql/persisted.go

type PersistedQueryCache interface {
    Get(ctx context.Context, hash string) (string, bool)
    Set(ctx context.Context, hash string, query string)
}

// In-memory implementation
type InMemoryCache struct {
    mu    sync.RWMutex
    store map[string]string
}

// Redis implementation
type RedisCache struct {
    client *redis.Client
}
```

**Phase 2: HTTP Handler Integration**

```go
// http.go modification

type persistedQueryRequest struct {
    Query     string                 `json:"query"`
    Variables map[string]interface{} `json:"variables"`
    Extensions struct {
        PersistedQuery struct {
            Version int    `json:"version"`
            Hash    string `json:"sha256Hash"`
        } `json:"persistedQuery"`
    } `json:"extensions"`
}

func (h *httpHandler) handlePersistedQuery(r *http.Request, req persistedQueryRequest) (string, error) {
    if req.Extensions.PersistedQuery.Hash == "" {
        return req.Query, nil  // Not a persisted query
    }
    
    hash := req.Extensions.PersistedQuery.Hash
    
    // Check cache
    if cached, ok := h.cache.Get(r.Context(), hash); ok {
        return cached, nil
    }
    
    // Cache miss
    if req.Query == "" {
        return "", &PersistedQueryNotFoundError{Hash: hash}
    }
    
    // Verify hash matches query
    if sha256Hash(req.Query) != hash {
        return "", errors.New("query hash mismatch")
    }
    
    // Cache for future
    h.cache.Set(r.Context(), hash, req.Query)
    return req.Query, nil
}
```

**Phase 3: Handler Options**

```go
handler := jaal.HTTPHandler(schema,
    jaal.WithPersistedQueryCache(redisCache),
    jaal.WithAutoPersistedQueries(true),
)
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Queries can be referenced by hash | ✅ |
| Cache hit returns parsed query | ✅ |
| Cache miss returns proper error | ✅ |
| Hash verification works | ✅ |
| Bandwidth reduction for repeated queries | > 80% |
| Cache hit latency | < 1ms |

---

## 5. Incremental Delivery (@defer/@stream)

### Feature Detail

The `@defer` and `@stream` directives enable incremental delivery of query results. Instead of waiting for the entire response, clients receive data as it becomes available.

**@defer:** Defer fragment execution
```graphql
query {
  user {
    name
    ... @defer(label: "friends") {
      friends { name }  # Loaded later
    }
  }
}
```

**@stream:** Stream list items incrementally
```graphql
query {
  user {
    friends @stream(initialCount: 5) {
      name  # First 5 immediately, rest streamed
    }
  }
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P3 (Low)** | High | Very High |

1. **User Experience:** Show initial data quickly, load rest progressively
2. **Performance Perception:** Users see results faster
3. **Large Lists:** Efficient handling of paginated/streaming data
4. **Spec Compliance:** Part of GraphQL October 2021 spec

### Implementation Details

**Phase 1: Directive Definitions**

```go
// Built-in @defer directive
var deferDirective = Directive{
    Name: "defer",
    Args: map[string]*Argument{
        "if":    {Type: &Scalar{Type: "Boolean"}},
        "label": {Type: &Scalar{Type: "String"}},
    },
    Locations: []DirectiveLocation{FRAGMENT_SPREAD, INLINE_FRAGMENT},
}

// Built-in @stream directive
var streamDirective = Directive{
    Name: "stream",
    Args: map[string]*Argument{
        "if":           {Type: &Scalar{Type: "Boolean"}},
        "label":        {Type: &Scalar{Type: "String"}},
        "initialCount": {Type: &Scalar{Type: "Int"}},
    },
    Locations: []DirectiveLocation{FIELD},
}
```

**Phase 2: Multipart Response Format**

```go
// Response format for multipart/mixed
type IncrementalResponse struct {
    Data        interface{} `json:"data,omitempty"`
    Path        []interface{} `json:"path"`
    Label       string      `json:"label,omitempty"`
    HasNext     bool        `json:"hasNext"`
    Errors      []Error     `json:"errors,omitempty"`
}

func writeMultipartResponse(w http.ResponseWriter, initial interface{}, incremental <-chan IncrementalResponse) {
    w.Header().Set("Content-Type", "multipart/mixed; boundary=graphql")
    
    // Write initial response
    w.Write([]byte("--graphql\r\n"))
    w.Write([]byte("Content-Type: application/json\r\n\r\n"))
    json.NewEncoder(w).Encode(initial)
    
    // Stream incremental responses
    for resp := range incremental {
        w.Write([]byte("\r\n--graphql\r\n"))
        w.Write([]byte("Content-Type: application/json\r\n\r\n"))
        json.NewEncoder(w).Encode(resp)
    }
    
    w.Write([]byte("\r\n--graphql--\r\n"))
}
```

**Phase 3: Executor Modification**

```go
func (e *Executor) executeWithDefer(ctx context.Context, typ Type, source interface{}, selectionSet *SelectionSet) (<-chan IncrementalResponse, error) {
    incremental := make(chan IncrementalResponse)
    
    go func() {
        defer close(incremental)
        
        // Execute non-deferred parts first
        result := e.executeSync(ctx, typ, source, selectionSet.NonDeferred())
        
        // Send initial response
        incremental <- IncrementalResponse{
            Data:    result,
            HasNext: true,
        }
        
        // Execute deferred fragments asynchronously
        for _, fragment := range selectionSet.DeferredFragments() {
            deferredResult := e.executeFragment(ctx, typ, source, fragment)
            incremental <- IncrementalResponse{
                Data:    deferredResult,
                Path:    fragment.Path,
                Label:   fragment.Label,
                HasNext: false,
            }
        }
    }()
    
    return incremental, nil
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| @defer directive recognized | ✅ |
| @stream directive works on list fields | ✅ |
| Multipart responses work correctly | ✅ |
| Initial data returns immediately | ✅ |
| Deferred data streams incrementally | ✅ |
| Labels work for matching results | ✅ |
| Error in one item doesn't fail stream | ✅ |

---

## 6. Real-time Subscriptions Enhancement

### Feature Detail

Jaal has basic WebSocket subscription support, but lacks several features present in other frameworks:

**Current Limitations:**
- Only `graphql-ws` protocol (not `graphql-transport-ws`)
- No subscription filtering/multiplexing
- No reconnection handling
- No message batching
- No subscription lifecycle hooks

**Enhanced Features:**
- Multiple protocol support
- Subscription filtering
- Connection keepalive
- Graceful shutdown
- Subscription metrics

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P1 (High)** | High | Medium |

1. **Production Readiness:** Current implementation is basic
2. **Client Compatibility:** Different clients use different protocols
3. **Reliability:** Reconnection and keepalive are essential
4. **Observability:** Need metrics for monitoring

### Implementation Details

**Phase 1: Protocol Abstraction**

```go
// ws/protocol.go

type SubscriptionProtocol interface {
    Name() string
    HandleInit(conn *WebSocket, msg json.RawMessage) error
    HandleSubscribe(conn *WebSocket, msg json.RawMessage) (*Subscription, error)
    HandleComplete(conn *WebSocket, msg json.RawMessage) error
    WriteData(conn *WebSocket, id string, data interface{}) error
    WriteError(conn *WebSocket, id string, err error) error
}

// graphql-ws (legacy)
type GraphQLWSProtocol struct{}

// graphql-transport-ws (recommended)
type GraphQLTransportWSProtocol struct{}

func NegotiateProtocol(protocols []string) SubscriptionProtocol {
    for _, p := range protocols {
        switch p {
        case "graphql-transport-ws":
            return &GraphQLTransportWSProtocol{}
        case "graphql-ws":
            return &GraphQLWSProtocol{}
        }
    }
    return nil
}
```

**Phase 2: Subscription Manager**

```go
// ws/subscription.go

type SubscriptionManager struct {
    mu            sync.RWMutex
    subscriptions map[string]*Subscription
    connections   map[string]*Connection
    hooks         SubscriptionHooks
}

type SubscriptionHooks struct {
    OnConnect    func(ctx context.Context, conn *Connection) context.Context
    OnSubscribe  func(ctx context.Context, sub *Subscription) error
    OnMessage    func(ctx context.Context, sub *Subscription, msg interface{})
    OnComplete   func(ctx context.Context, sub *Subscription)
    OnDisconnect func(ctx context.Context, conn *Connection)
}

func (sm *SubscriptionManager) Subscribe(ctx context.Context, conn *Connection, query string, vars map[string]interface{}) (*Subscription, error) {
    // Parse and validate
    parsed, err := graphql.Parse(query, vars)
    if err != nil {
        return nil, err
    }
    
    // Apply hooks
    sub := &Subscription{Query: parsed, Connection: conn}
    if err := sm.hooks.OnSubscribe(ctx, sub); err != nil {
        return nil, err
    }
    
    // Register
    sm.mu.Lock()
    sm.subscriptions[sub.ID] = sub
    sm.mu.Unlock()
    
    return sub, nil
}
```

**Phase 3: Keepalive and Reconnection**

```go
func (sm *SubscriptionManager) StartKeepalive(conn *Connection, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                if err := conn.WriteKeepalive(); err != nil {
                    conn.Close()
                    return
                }
            case <-conn.Done():
                return
            }
        }
    }()
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Multiple protocol support | ✅ |
| Graceful reconnection | ✅ |
| Keepalive messages | ✅ |
| Subscription lifecycle hooks | ✅ |
| Concurrent subscriptions per connection | > 100 |
| Message throughput | > 10,000 msg/sec |
| Memory per subscription | < 1KB |

---

## 7. Plugin/Middleware Ecosystem

### Feature Detail

A plugin system allows extending Jaal's functionality without modifying core code. This enables community contributions and custom integrations.

**How it works in other frameworks:**
- **Apollo:** Plugin interface with lifecycle hooks
- **gqlgen:** Directive-based plugins
- **Fastify:** Hook-based plugin system

```go
// Example plugin usage
jaal.NewServer(schema,
    jaal.WithPlugin(authPlugin),
    jaal.WithPlugin(loggingPlugin),
    jaal.WithPlugin(tracingPlugin),
)
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P2 (Medium)** | High | Medium |

1. **Extensibility:** Users can add features without forking
2. **Community:** Encourages community contributions
3. **Separation of Concerns:** Keep core minimal
4. **Flexibility:** Different users need different features

### Implementation Details

**Phase 1: Plugin Interface**

```go
// plugin/plugin.go

type Plugin interface {
    Name() string
    Initialize(config *PluginConfig) error
}

type PluginConfig struct {
    Schema       *graphql.Schema
    SchemaBuilder *schemabuilder.Schema
    HTTPHandler  http.Handler
}

// Lifecycle hooks
type PluginHooks struct {
    BeforeParse    func(ctx context.Context, query string) (context.Context, error)
    AfterParse     func(ctx context.Context, query *graphql.Query) error
    BeforeValidate func(ctx context.Context, query *graphql.Query) error
    AfterValidate  func(ctx context.Context, query *graphql.Query) error
    BeforeExecute  func(ctx context.Context, query *graphql.Query) (context.Context, error)
    AfterExecute   func(ctx context.Context, result interface{}, err error) error
    BeforeResponse func(ctx context.Context, response *httpResponse) error
}
```

**Phase 2: Plugin Registry**

```go
// plugin/registry.go

type PluginRegistry struct {
    plugins map[string]Plugin
    hooks   PluginHooks
}

func (r *PluginRegistry) Register(p Plugin) error {
    if _, exists := r.plugins[p.Name()]; exists {
        return fmt.Errorf("plugin %s already registered", p.Name())
    }
    r.plugins[p.Name()] = p
    return nil
}

func (r *PluginRegistry) ApplyHooks(ctx context.Context, phase HookPhase) context.Context {
    // Execute all registered hooks for the phase
    return ctx
}
```

**Phase 3: Built-in Plugins**

```go
// plugin/builtin/logging.go
type LoggingPlugin struct{}

func (p *LoggingPlugin) Name() string { return "logging" }

func (p *LoggingPlugin) Initialize(config *PluginConfig) error {
    // Add logging hooks
    return nil
}

// plugin/builtin/tracing.go
type TracingPlugin struct{}

func (p *TracingPlugin) Name() string { return "tracing" }

// plugin/builtin/auth.go
type AuthPlugin struct{}

func (p *AuthPlugin) Name() string { return "auth" }
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Plugin interface defined | ✅ |
| 3+ built-in plugins | ✅ |
| Plugin hot-reload | ✅ |
| Plugin isolation | ✅ |
| Plugin documentation | ✅ |
| Community plugins | > 5 (within 6 months) |

---

## 8. Schema Stitching

### Feature Detail

Schema stitching combines multiple GraphQL schemas into a single unified schema. Unlike federation (which is distributed), stitching merges schemas at the gateway level.

**How it works:**
1. Remote schemas are introspected
2. Schemas are merged with conflict resolution
3. Resolvers are delegated to appropriate services
4. Single unified schema is exposed

```go
// Example: Stitch multiple schemas
stitchedSchema := stitch.Schemas(
    remoteSchema("http://users-service/graphql"),
    remoteSchema("http://products-service/graphql"),
    remoteSchema("http://orders-service/graphql"),
)
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P3 (Low)** | Medium | Very High |

1. **Legacy Integration:** Combine legacy GraphQL services
2. **Flexibility:** Alternative to federation for simpler setups
3. **Migration Path:** Gradually migrate from monolith to microservices
4. **Third-party Integration:** Combine external GraphQL APIs

### Implementation Details

**Phase 1: Remote Schema Introspection**

```go
// stitch/remote.go

type RemoteSchema struct {
    URL     string
    Headers http.Header
    Schema  *graphql.Schema
}

func IntrospectRemoteSchema(url string, headers http.Header) (*RemoteSchema, error) {
    // Execute introspection query
    resp, err := http.Post(url, "application/json", strings.NewReader(introspectionQuery))
    if err != nil {
        return nil, err
    }
    
    // Parse introspection result into schema
    var result IntrospectionResult
    json.NewDecoder(resp.Body).Decode(&result)
    
    return ConvertIntrospectionToSchema(result)
}
```

**Phase 2: Schema Merger**

```go
// stitch/merge.go

type SchemaMerger struct {
    conflictResolution ConflictResolution
    typeMerging        map[string]MergeConfig
}

type MergeConfig struct {
    TypeName    string
    KeyField    string
    Resolvers   []MergeResolver
}

func (m *SchemaMerger) Merge(schemas ...*graphql.Schema) (*graphql.Schema, error) {
    merged := &graphql.Schema{}
    
    for _, schema := range schemas {
        // Merge types with conflict resolution
        if err := m.mergeTypes(merged, schema); err != nil {
            return nil, err
        }
        
        // Merge query/mutation/subscription
        m.mergeRootTypes(merged, schema)
    }
    
    return merged, nil
}
```

**Phase 3: Delegating Resolvers**

```go
func (m *SchemaMerger) createDelegatingResolver(remote *RemoteSchema) graphql.Resolver {
    return func(ctx context.Context, source, args interface{}, selectionSet *graphql.SelectionSet) (interface{}, error) {
        // Build query for remote service
        query := buildDelegationQuery(selectionSet, args)
        
        // Execute against remote
        return executeRemoteQuery(ctx, remote, query)
    }
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Remote schema introspection | ✅ |
| Schema merging works | ✅ |
| Conflict resolution | ✅ |
| Query delegation | ✅ |
| Type merging | ✅ |
| Performance overhead | < 20% |

---

## 9. Advanced Caching

### Feature Detail

Response caching reduces load on resolvers and improves response times. Advanced caching includes:

- **Response Caching:** Cache full query responses
- **Resolver Caching:** Cache individual field results
- **Partial Query Caching:** Cache sub-selections (like Relay/Apollo)
- **Cache Invalidation:** Smart invalidation based on type/ID

```go
// Example: Cache configuration
schema := sb.MustBuild(
    graphql.WithResponseCache(redisCache, time.Hour),
    graphql.WithResolverCache(localCache),
    graphql.WithCacheScopes(map[string]CacheScope{
        "User.email": CacheScopePrivate,
        "Product.*":  CacheScopePublic,
    }),
)
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P2 (Medium)** | High | High |

1. **Performance:** Dramatically reduce resolver execution
2. **Cost Reduction:** Less database/API calls
3. **Scalability:** Handle more requests with same resources
4. **User Experience:** Faster response times

### Implementation Details

**Phase 1: Cache Interface**

```go
// cache/cache.go

type Cache interface {
    Get(ctx context.Context, key string) (interface{}, bool)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration)
    Delete(ctx context.Context, key string)
    Flush(ctx context.Context)
}

type CacheKey struct {
    Query     string
    Variables map[string]interface{}
    UserID    string  // For private caching
}

func (k CacheKey) Hash() string {
    // Generate deterministic hash
}
```

**Phase 2: Response Cache Middleware**

```go
// cache/response.go

type ResponseCacheMiddleware struct {
    cache Cache
    ttl   time.Duration
}

func (m *ResponseCacheMiddleware) Execute(ctx context.Context, root graphql.Type, query *graphql.Query) (interface{}, error) {
    // Generate cache key
    key := CacheKey{Query: query.String(), Variables: extractVariables(ctx)}
    
    // Check cache
    if cached, ok := m.cache.Get(ctx, key.Hash()); ok {
        return cached, nil
    }
    
    // Execute query
    result, err := m.next.Execute(ctx, root, query)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    m.cache.Set(ctx, key.Hash(), result, m.ttl)
    
    return result, nil
}
```

**Phase 3: Resolver-Level Caching**

```go
// Mark field as cacheable
obj.FieldFunc("expensiveComputation", resolver,
    schemabuilder.CacheField(time.Minute, CacheScopePublic),
)

// Dataloader pattern for batching
func NewUserLoader() *dataloader.Loader {
    return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
        users := fetchUsers(ctx, keys.Keys()...)
        // Return results
    })
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Response caching works | ✅ |
| Cache hit ratio | > 50% (typical workload) |
| Resolver caching works | ✅ |
| Cache invalidation | ✅ |
| TTL support | ✅ |
| Cache scopes (public/private) | ✅ |
| Latency improvement | > 90% for cached queries |

---

## 10. File Upload Support

### Feature Detail

The GraphQL Multipart Request Spec enables file uploads via GraphQL mutations.

**How it works:**
1. Client sends multipart/form-data request
2. Files are mapped to GraphQL variables
3. Mutation receives `Upload` scalar type
4. Server processes uploaded files

```graphql
mutation UploadFile($file: Upload!) {
  uploadFile(file: $file) {
    id
    url
  }
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P2 (Medium)** | Medium | Low |

1. **Common Requirement:** Most APIs need file uploads
2. **Spec Compliance:** Multipart Request Spec is widely adopted
3. **Single Endpoint:** Keep everything in GraphQL
4. **Type Safety:** File uploads are part of the schema

### Implementation Details

**Phase 1: Upload Scalar**

```go
// upload/upload.go

type Upload struct {
    File     multipart.File
    Header   *multipart.FileHeader
    Filename string
    Size     int64
}

func RegisterUploadScalar(schema *schemabuilder.Schema) {
    typ := reflect.TypeOf(Upload{})
    schemabuilder.RegisterScalar(typ, "Upload", func(value interface{}, dest reflect.Value) error {
        upload, ok := value.(*Upload)
        if !ok {
            return errors.New("expected Upload type")
        }
        dest.Set(reflect.ValueOf(*upload))
        return nil
    })
}
```

**Phase 2: Multipart Request Parser**

```go
// upload/parser.go

type MultipartParser struct {
    MaxFileSize int64
    MaxFiles    int
}

func (p *MultipartParser) Parse(r *http.Request) (*GraphQLRequest, error) {
    // Parse multipart form
    if err := r.ParseMultipartForm(p.MaxFileSize); err != nil {
        return nil, err
    }
    
    // Extract operations
    operations := r.FormValue("operations")
    var req GraphQLRequest
    json.Unmarshal([]byte(operations), &req)
    
    // Extract file mappings
    mappings := r.FormValue("map")
    var fileMappings map[string][]string
    json.Unmarshal([]byte(mappings), &fileMappings)
    
    // Replace file placeholders with Upload objects
    for key, paths := range fileMappings {
        file, header, err := r.FormFile(key)
        if err != nil {
            return nil, err
        }
        
        upload := &Upload{
            File:     file,
            Header:   header,
            Filename: header.Filename,
            Size:     header.Size,
        }
        
        // Set upload at each path
        for _, path := range paths {
            setUploadAtPath(&req.Variables, path, upload)
        }
    }
    
    return &req, nil
}
```

**Phase 3: HTTP Handler Integration**

```go
// Modify http.go
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Check for multipart request
    if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
        req, err := h.multipartParser.Parse(r)
        if err != nil {
            writeError(w, err)
            return
        }
        h.executeGraphQL(w, r, req)
        return
    }
    
    // Standard JSON request handling
    // ...
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Upload scalar registered | ✅ |
| Multipart parsing works | ✅ |
| File size limits enforced | ✅ |
| Multiple file uploads | ✅ |
| Integration with mutations | ✅ |
| Max file size | Configurable |

---

## 11. Error Handling & Extensions

### Feature Detail

Enhanced error handling with:
- Structured error codes
- Error extensions
- Error classification
- Partial error responses
- Error transformation

```json
{
  "errors": [{
    "message": "User not found",
    "extensions": {
      "code": "USER_NOT_FOUND",
      "statusCode": 404,
      "timestamp": "2026-04-07T10:00:00Z",
      "traceId": "abc123"
    },
    "path": ["user"]
  }]
}
```

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P1 (High)** | Medium | Low |

1. **Debugging:** Better error messages for developers
2. **Client Handling:** Clients can programmatically handle errors
3. **Observability:** Error tracking and alerting
4. **Standards:** Align with GraphQL error handling best practices

### Implementation Details

**Phase 1: Enhanced Error Type**

```go
// jerrors/errors.go

type Error struct {
    Message    string                 `json:"message"`
    Path       []interface{}          `json:"path,omitempty"`
    Locations  []Location             `json:"locations,omitempty"`
    Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type ErrorBuilder struct {
    err *Error
}

func NewError(message string) *ErrorBuilder {
    return &ErrorBuilder{
        err: &Error{
            Message:    message,
            Extensions: make(map[string]interface{}),
        },
    }
}

func (b *ErrorBuilder) WithCode(code string) *ErrorBuilder {
    b.err.Extensions["code"] = code
    return b
}

func (b *ErrorBuilder) WithStatusCode(code int) *ErrorBuilder {
    b.err.Extensions["statusCode"] = code
    return b
}

func (b *ErrorBuilder) WithExtension(key string, value interface{}) *ErrorBuilder {
    b.err.Extensions[key] = value
    return b
}

func (b *ErrorBuilder) Build() *Error {
    return b.err
}
```

**Phase 2: Error Classification**

```go
type ErrorClassification string

const (
    ClassificationValidationError  ErrorClassification = "VALIDATION_ERROR"
    ClassificationAuthenticationError ErrorClassification = "AUTHENTICATION_ERROR"
    ClassificationAuthorizationError  ErrorClassification = "AUTHORIZATION_ERROR"
    ClassificationNotFoundError     ErrorClassification = "NOT_FOUND"
    ClassificationInternalError     ErrorClassification = "INTERNAL_ERROR"
    ClassificationRateLimitError    ErrorClassification = "RATE_LIMITED"
)

func ClassifyError(err error) *Error {
    switch {
    case errors.Is(err, ErrNotFound):
        return NewError(err.Error()).
            WithCode("NOT_FOUND").
            WithStatusCode(404).
            Build()
    case errors.Is(err, ErrUnauthorized):
        return NewError(err.Error()).
            WithCode("UNAUTHORIZED").
            WithStatusCode(401).
            Build()
    default:
        return NewError(err.Error()).
            WithCode("INTERNAL_ERROR").
            WithStatusCode(500).
            Build()
    }
}
```

**Phase 3: Error Middleware**

```go
func ErrorMiddleware(next HandlerFunc) HandlerFunc {
    return func(ctx context.Context, root graphql.Type, query *graphql.Query) (interface{}, error) {
        result, err := next(ctx, root, query)
        if err == nil {
            return result, nil
        }
        
        // Classify and enhance error
        classified := ClassifyError(err)
        
        // Add trace ID
        if traceID := ctx.Value(TraceIDKey); traceID != nil {
            classified.Extensions["traceId"] = traceID
        }
        
        return nil, classified
    }
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Structured error codes | ✅ |
| Error extensions | ✅ |
| Error classification | ✅ |
| Error middleware | ✅ |
| Trace ID in errors | ✅ |
| Client-friendly messages | ✅ |

---

## 12. Observability & Tracing

### Feature Detail

Full observability support including:
- Distributed tracing (OpenTelemetry)
- Metrics (Prometheus)
- Logging (structured)
- Health checks
- Performance profiling

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P1 (High)** | High | Medium |

1. **Production Necessity:** Essential for debugging production issues
2. **SLA Monitoring:** Track response times and error rates
3. **Performance Analysis:** Identify bottlenecks
4. **Industry Standard:** All major frameworks support this

### Implementation Details

**Phase 1: OpenTelemetry Integration**

```go
// observability/tracing.go

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

type TracingMiddleware struct {
    tracer trace.Tracer
}

func (m *TracingMiddleware) Execute(ctx context.Context, root graphql.Type, query *graphql.Query) (interface{}, error) {
    ctx, span := m.tracer.Start(ctx, "graphql.execute",
        trace.WithAttributes(
            attribute.String("graphql.operation", query.Kind),
            attribute.String("graphql.query", query.Name),
        ),
    )
    defer span.End()
    
    result, err := m.next.Execute(ctx, root, query)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    
    return result, err
}
```

**Phase 2: Prometheus Metrics**

```go
// observability/metrics.go

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
    RequestsTotal   *prometheus.CounterVec
    RequestDuration *prometheus.HistogramVec
    ActiveRequests  prometheus.Gauge
    ErrorsTotal     *prometheus.CounterVec
}

func NewMetrics() *Metrics {
    return &Metrics{
        RequestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "graphql_requests_total",
                Help: "Total GraphQL requests",
            },
            []string{"operation", "name"},
        ),
        RequestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "graphql_request_duration_seconds",
                Help:    "GraphQL request duration",
                Buckets: []float64{.001, .005, .01, .05, .1, .5, 1, 5},
            },
            []string{"operation"},
        ),
    }
}
```

**Phase 3: Structured Logging**

```go
// observability/logging.go

type StructuredLogger struct {
    logger *slog.Logger
}

func (l *StructuredLogger) LogRequest(ctx context.Context, query *graphql.Query, duration time.Duration, err error) {
    level := slog.LevelInfo
    if err != nil {
        level = slog.LevelError
    }
    
    l.logger.Log(ctx, level, "graphql request",
        "operation", query.Kind,
        "name", query.Name,
        "duration_ms", duration.Milliseconds(),
        "error", err,
        "trace_id", ctx.Value(TraceIDKey),
    )
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| OpenTelemetry integration | ✅ |
| Prometheus metrics exposed | ✅ |
| Structured logging | ✅ |
| Health check endpoint | ✅ |
| Request tracing | ✅ |
| Performance profiling | ✅ |

---

## 13. Security Features

### Feature Detail

Built-in security features:
- Query depth limiting
- Query complexity limiting (covered above)
- Rate limiting
- Input sanitization
- Introspection disabling (production)
- CORS configuration
- Request size limits

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P1 (High)** | Critical | Medium |

1. **Security Essential:** GraphQL APIs are vulnerable to specific attacks
2. **DoS Prevention:** Depth and complexity limits prevent abuse
3. **Production Ready:** Security shouldn't be an afterthought
4. **Compliance:** Many regulations require security controls

### Implementation Details

**Phase 1: Query Depth Limiting**

```go
// security/depth.go

type DepthLimiter struct {
    maxDepth int
}

func (l *DepthLimiter) Validate(selectionSet *graphql.SelectionSet, currentDepth int) error {
    if currentDepth > l.maxDepth {
        return fmt.Errorf("query depth %d exceeds maximum %d", currentDepth, l.maxDepth)
    }
    
    for _, selection := range selectionSet.Selections {
        if err := l.Validate(selection.SelectionSet, currentDepth+1); err != nil {
            return err
        }
    }
    
    return nil
}
```

**Phase 2: Rate Limiting**

```go
// security/ratelimit.go

type RateLimiter struct {
    store   RateLimitStore
    limits  map[string]RateLimit
}

type RateLimit struct {
    Requests int
    Window   time.Duration
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := rl.getKey(r) // IP, API key, or user ID
        
        if !rl.store.Allow(key) {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

**Phase 3: Security Configuration**

```go
// Security configuration options
type SecurityConfig struct {
    MaxQueryDepth      int
    MaxQueryComplexity int
    MaxRequestSize     int64
    EnableIntrospection bool
    EnablePlayground    bool
    RateLimits         map[string]RateLimit
    CORS               CORSConfig
}

handler := jaal.HTTPHandler(schema,
    jaal.WithSecurity(SecurityConfig{
        MaxQueryDepth:       10,
        MaxQueryComplexity:  1000,
        MaxRequestSize:      1 << 20, // 1MB
        EnableIntrospection: false,   // Production
        EnablePlayground:    false,
    }),
)
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Query depth limiting | ✅ |
| Rate limiting | ✅ |
| Request size limits | ✅ |
| Introspection toggle | ✅ |
| CORS configuration | ✅ |
| Input sanitization | ✅ |

---

## 14. Developer Tooling

### Feature Detail

Tools to improve developer experience:
- Schema diffing
- Schema migration
- Code generation from schema
- Schema documentation generator
- Query validation CLI
- Mock server

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P3 (Low)** | Medium | Medium |

1. **Developer Experience:** Better tooling = faster development
2. **Documentation:** Auto-generated docs stay in sync
3. **Testing:** Mock server enables frontend development
4. **CI/CD:** Schema diffing for breaking change detection

### Implementation Details

**Phase 1: Schema Diffing**

```go
// tools/diff.go

type SchemaDiff struct {
    BreakingChanges []Change
    SafeChanges     []Change
}

type Change struct {
    Type        ChangeType
    Description string
    Path        string
}

func DiffSchemas(old, new *graphql.Schema) *SchemaDiff {
    diff := &SchemaDiff{}
    
    // Compare types
    diff.compareTypes(old, new)
    
    // Compare fields
    diff.compareFields(old, new)
    
    // Compare arguments
    diff.compareArguments(old, new)
    
    return diff
}
```

**Phase 2: Documentation Generator**

```go
// tools/docs.go

func GenerateDocs(schema *graphql.Schema, config DocConfig) string {
    var doc strings.Builder
    
    // Generate type documentation
    for _, typ := range schema.Types {
        doc.WriteString(fmt.Sprintf("## %s\n\n", typ.Name))
        doc.WriteString(fmt.Sprintf("%s\n\n", typ.Description))
        
        // Document fields
        for _, field := range typ.Fields {
            doc.WriteString(fmt.Sprintf("### %s\n", field.Name))
            doc.WriteString(fmt.Sprintf("```graphql\n%s(%s): %s\n```\n\n",
                field.Name, formatArgs(field.Args), field.Type))
        }
    }
    
    return doc.String()
}
```

**Phase 3: Mock Server**

```go
// tools/mock.go

type MockServer struct {
    schema   *graphql.Schema
    mocks    map[string]interface{}
    resolvers map[string]MockResolver
}

func (s *MockServer) Start(addr string) error {
    // Generate mock resolvers for all fields
    // Return configured mock data or generated fake data
    return http.ListenAndServe(addr, s)
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| Schema diffing works | ✅ |
| Breaking change detection | ✅ |
| Documentation generator | ✅ |
| Mock server | ✅ |
| CLI tool | ✅ |

---

## 15. Complete Validation Rules

### Feature Detail

Implement all GraphQL spec validation rules. Currently missing:

1. Fields on correct type
2. Fragments on composite types
3. Leaf field selections
4. Argument names
5. Argument uniqueness
6. Required arguments
7. Directives in allowed locations
8. Variable usage allowed
9. Variable usage in allowed location
10. All variable usages are defined

### Reasoning

| Priority | Impact | Effort |
|----------|--------|--------|
| **P2 (Medium)** | Medium | Medium |

1. **Spec Compliance:** Required for full GraphQL compliance
2. **Error Messages:** Better developer experience with clear errors
3. **Early Detection:** Catch errors before execution
4. **Security:** Prevent malformed queries

### Implementation Details

**Phase 1: Validation Framework**

```go
// graphql/validate.go

type ValidationRule func(ctx context.Context, schema *Schema, query *Query) []error

type Validator struct {
    rules []ValidationRule
}

func (v *Validator) Validate(ctx context.Context, schema *Schema, query *Query) []error {
    var errors []error
    
    for _, rule := range v.rules {
        if errs := rule(ctx, schema, query); len(errs) > 0 {
            errors = append(errors, errs...)
        }
    }
    
    return errors
}
```

**Phase 2: Individual Rules**

```go
// FieldsOnCorrectType validation
func FieldsOnCorrectType(ctx context.Context, schema *Schema, query *Query) []error {
    var errors []error
    
    // Walk selection set
    walkSelection(query.SelectionSet, func(selection *Selection, typ Type) {
        if typ, ok := typ.(*Object); ok {
            if _, exists := typ.Fields[selection.Name]; !exists {
                errors = append(errors, fmt.Errorf(
                    "Cannot query field \"%s\" on type \"%s\". Did you mean %s?",
                    selection.Name, typ.Name, suggestFields(typ, selection.Name),
                ))
            }
        }
    })
    
    return errors
}

// RequiredArguments validation
func RequiredArguments(ctx context.Context, schema *Schema, query *Query) []error {
    var errors []error
    
    walkSelection(query.SelectionSet, func(selection *Selection, typ Type) {
        if typ, ok := typ.(*Object); ok {
            field := typ.Fields[selection.Name]
            for argName, arg := range field.Args {
                if _, isNonNull := arg.Type.(*NonNull); isNonNull {
                    if selection.Args == nil || selection.Args.(map[string]interface{})[argName] == nil {
                        errors = append(errors, fmt.Errorf(
                            "Field \"%s\" argument \"%s\" of type \"%s\" is required",
                            selection.Name, argName, arg.Type,
                        ))
                    }
                }
            }
        }
    })
    
    return errors
}
```

### Success Metrics

| Metric | Target |
|--------|--------|
| All spec validation rules implemented | ✅ |
| Multiple validation errors returned | ✅ |
| Validation errors include locations | ✅ |
| Validation is extensible | ✅ |
| Test coverage | 100% |

---

## Summary Matrix

| Feature | Priority | Impact | Effort | Spec Compliance | Production Ready |
|---------|----------|--------|--------|-----------------|------------------|
| Custom Directives | P1 | High | High | ✅ Yes | ⚠️ Partial |
| Federation | P2 | Very High | Very High | ❌ No | ❌ No |
| Complexity Analysis | P1 | High | Medium | ✅ Yes | ⚠️ Partial |
| Persisted Queries | P2 | Medium | Medium | ✅ Yes | ❌ No |
| @defer/@stream | P3 | High | Very High | ✅ Yes | ❌ No |
| Subscriptions Enhancement | P1 | High | Medium | ⚠️ Partial | ⚠️ Basic |
| Plugin System | P2 | High | Medium | N/A | ❌ No |
| Schema Stitching | P3 | Medium | Very High | N/A | ❌ No |
| Advanced Caching | P2 | High | High | N/A | ❌ No |
| File Upload | P2 | Medium | Low | ✅ Yes | ❌ No |
| Error Handling | P1 | Medium | Low | ⚠️ Partial | ⚠️ Basic |
| Observability | P1 | High | Medium | N/A | ❌ No |
| Security Features | P1 | Critical | Medium | N/A | ⚠️ Partial |
| Developer Tooling | P3 | Medium | Medium | N/A | ❌ No |
| Validation Rules | P2 | Medium | Medium | ⚠️ Partial | ⚠️ Partial |

---

## Recommended Implementation Order

### Phase 1 (Critical - 2 months)
1. Security Features (depth limiting, rate limiting)
2. Error Handling & Extensions
3. Observability & Tracing
4. Query Complexity Analysis

### Phase 2 (Important - 3 months)
1. Custom Directives Framework
2. Subscriptions Enhancement
3. Plugin/Middleware System
4. File Upload Support

### Phase 3 (Enhancement - 3 months)
1. Persisted Queries (APQ)
2. Advanced Caching
3. Complete Validation Rules
4. Developer Tooling

### Phase 4 (Advanced - 4 months)
1. Federation Support
2. Incremental Delivery (@defer/@stream)
3. Schema Stitching

---

## Comparison with Specific Frameworks

### vs Apollo GraphQL (JavaScript)

| Feature | Jaal | Apollo |
|---------|------|--------|
| Custom Directives | ❌ | ✅ Full support |
| Federation | ❌ | ✅ Federation 2.0 |
| Subscriptions | ⚠️ Basic | ✅ Full + multiple transports |
| Caching | ❌ | ✅ Response + partial query |
| Plugins | ❌ | ✅ Rich ecosystem |
| Apollo Studio | ❌ | ✅ Full integration |
| Performance | ✅ Go speed | ⚠️ Node.js |

### vs gqlgen (Go)

| Feature | Jaal | gqlgen |
|---------|------|--------|
| Code Generation | ❌ | ✅ Type-safe resolvers |
| Custom Directives | ❌ | ✅ |
| Federation | ❌ | ✅ Federation 2.0 |
| Subscriptions | ⚠️ Basic | ✅ |
| Complexity Analysis | ❌ | ✅ |
| Plugin System | ❌ | ✅ |
| Schema-First | ❌ Code-first | ✅ |

### vs Hasura

| Feature | Jaal | Hasura |
|---------|------|--------|
| Database Integration | ❌ | ✅ Auto-generated |
| Real-time | ⚠️ Basic | ✅ Live queries |
| Authorization | ❌ | ✅ Built-in |
| Performance | ⚠️ Manual optimization | ✅ Optimized |
| Admin UI | ❌ | ✅ Console |

---

*This document is a living document and should be updated as features are implemented and new requirements emerge.*
