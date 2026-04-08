# Custom Directives in Jaal

This document provides comprehensive documentation for custom directive support in Jaal, which implements the GraphQL October 2021 specification for directives.

## Overview

Custom directives allow you to define your own directives that can modify query execution behavior. Common use cases include:

- **Authentication/Authorization**: `@auth`, `@hasRole`
- **Caching**: `@cache`, `@cached`
- **Validation**: `@validate`, `@constraint`
- **Logging/Auditing**: `@log`, `@audit`
- **Rate Limiting**: `@rateLimit`
- **Data Transformation**: `@uppercase`, `@lowercase`, `@format`

## Quick Start

### 1. Define a Custom Directive

```go
import (
    "context"
    "fmt"
    
    "go.appointy.com/jaal/graphql"
    "go.appointy.com/jaal/schemabuilder"
)

// Create a schema
sb := schemabuilder.NewSchema()

// Define an @auth directive
sb.Directive("auth",
    schemabuilder.DirectiveDescription("Requires authentication"),
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArgString("role"),
    schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        user := ctx.Value("user")
        if user == nil {
            return nil, fmt.Errorf("unauthorized")
        }
        return nil, nil // Continue with normal resolution
    }),
)
```

### 2. Apply the Directive to Fields

```go
query := sb.Query()
query.FieldFunc("secretData", func(ctx context.Context) string {
    return "classified"
},
    schemabuilder.FieldDesc("Secret data requiring auth"),
    schemabuilder.FieldDirective("auth", map[string]interface{}{"role": "ADMIN"}),
)
```

### 3. Enable Directive Execution

```go
// Get directive definitions and visitors
directiveDefs := sb.GetDirectiveDefinitions()
directiveVisitors := sb.GetDirectiveVisitors()

// Build schema
schema := sb.MustBuild()

// Add introspection with custom directives
introspection.AddIntrospectionToSchemaWithDirectives(schema, directiveDefs)

// Create handler with directive visitors
handler := jaal.HTTPHandler(schema, jaal.WithDirectiveVisitors(directiveVisitors))
```

## Directive Definition API

### DirectiveRegistration Options

| Option | Description |
|--------|-------------|
| `DirectiveDescription(desc string)` | Sets the directive description |
| `DirectiveLocations(locs ...DirectiveLocation)` | Specifies valid locations |
| `DirectiveRepeatable()` | Marks directive as repeatable |
| `DirectiveArgString(name string)` | Adds a String argument |
| `DirectiveArgInt(name string)` | Adds an Int argument |
| `DirectiveArgFloat(name string)` | Adds a Float argument |
| `DirectiveArgBool(name string)` | Adds a Boolean argument |
| `DirectiveArgID(name string)` | Adds an ID argument |
| `DirectiveVisitor(visitor DirectiveVisitor)` | Sets the execution visitor |
| `DirectiveVisitorFunc(fn)` | Sets visitor from a function |

### Directive Locations

Jaal supports all GraphQL directive locations:

#### Executable Locations
| Constant | Value | Description |
|----------|-------|-------------|
| `LocationQuery` | `QUERY` | On query operations |
| `LocationMutation` | `MUTATION` | On mutation operations |
| `LocationSubscription` | `SUBSCRIPTION` | On subscription operations |
| `LocationField` | `FIELD` | On field selections |
| `LocationFragmentDefinition` | `FRAGMENT_DEFINITION` | On fragment definitions |
| `LocationFragmentSpread` | `FRAGMENT_SPREAD` | On fragment spreads |
| `LocationInlineFragment` | `INLINE_FRAGMENT` | On inline fragments |

#### Type System Locations
| Constant | Value | Description |
|----------|-------|-------------|
| `LocationSchema` | `SCHEMA` | On schema definition |
| `LocationScalar` | `SCALAR` | On scalar definitions |
| `LocationObject` | `OBJECT` | On object type definitions |
| `LocationFieldDefinition` | `FIELD_DEFINITION` | On field definitions |
| `LocationArgumentDefinition` | `ARGUMENT_DEFINITION` | On argument definitions |
| `LocationInterface` | `INTERFACE` | On interface definitions |
| `LocationUnion` | `UNION` | On union definitions |
| `LocationEnum` | `ENUM` | On enum definitions |
| `LocationEnumValue` | `ENUM_VALUE` | On enum value definitions |
| `LocationInputObject` | `INPUT_OBJECT` | On input object definitions |
| `LocationInputFieldDefinition` | `INPUT_FIELD_DEFINITION` | On input field definitions |

## Directive Visitors

### The DirectiveVisitor Interface

```go
type DirectiveVisitor interface {
    VisitField(ctx context.Context, directive *DirectiveInstance, field *Field, source interface{}) (interface{}, error)
}
```

### Return Values

The `VisitField` method controls execution flow:

| Return | Effect |
|--------|--------|
| `(nil, nil)` | Continue with normal field resolution |
| `(result, nil)` | Short-circuit and return `result` |
| `(nil, error)` | Return error, stop execution |
| `(result, error)` | Return both (error takes precedence) |

### DirectiveInstance

```go
type DirectiveInstance struct {
    Name string                    // Directive name
    Args map[string]interface{}    // Argument values
}
```

## Examples

### Authentication Directive

```go
sb.Directive("auth",
    schemabuilder.DirectiveDescription("Requires authentication with optional role"),
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArgString("role"),
    schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        user, ok := ctx.Value("user").(*User)
        if !ok || user == nil {
            return nil, fmt.Errorf("unauthorized: authentication required")
        }
        
        if role, ok := d.Args["role"].(string); ok && role != "" {
            if string(user.Role) != role {
                return nil, fmt.Errorf("forbidden: requires role %s", role)
            }
        }
        
        return nil, nil
    }),
)
```

### Caching Directive

```go
var cache = make(map[string]cachedItem)

type cachedItem struct {
    value     interface{}
    expiresAt time.Time
}

sb.Directive("cache",
    schemabuilder.DirectiveDescription("Caches field results"),
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArgInt("ttl"),
    schemabuilder.DirectiveRepeatable(),
    schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        key := fmt.Sprintf("%v:%v", f, src)
        
        if cached, ok := cache[key]; ok && time.Now().Before(cached.expiresAt) {
            return cached.value, nil
        }
        
        result, err := f.Resolve(ctx, src, nil, nil)
        if err != nil {
            return nil, err
        }
        
        ttl, _ := d.Args["ttl"].(int)
        cache[key] = cachedItem{
            value:     result,
            expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
        }
        
        return nil, nil // Continue with normal resolution
    }),
)
```

### Transformation Directive

```go
sb.Directive("uppercase",
    schemabuilder.DirectiveDescription("Transforms string output to uppercase"),
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        result, err := f.Resolve(ctx, src, nil, nil)
        if err != nil {
            return nil, err
        }
        
        if str, ok := result.(string); ok {
            return strings.ToUpper(str), nil
        }
        
        return result, nil
    }),
)
```

### Logging Directive

```go
sb.Directive("log",
    schemabuilder.DirectiveDescription("Logs field access"),
    schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
    schemabuilder.DirectiveArgString("message"),
    schemabuilder.DirectiveVisitorFunc(func(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
        message := "field access"
        if msg, ok := d.Args["message"].(string); ok && msg != "" {
            message = msg
        }
        
        log.Printf("[%s] %s", f.Name, message)
        
        return nil, nil
    }),
)
```

## Validation

Jaal provides validation functions for directives:

### ValidateDirectiveLocations

Validates that directives are used in their allowed locations:

```go
errors := graphql.ValidateDirectiveLocations(selectionSet, definitions)
```

### ValidateRepeatableDirectives

Validates that non-repeatable directives aren't duplicated:

```go
errors := graphql.ValidateRepeatableDirectives(selectionSet, definitions)
```

### ValidateSchemaDirectives

Validates all directives in a schema:

```go
errors := graphql.ValidateSchemaDirectives(schema, definitions)
```

## Introspection

Custom directives are exposed via introspection:

```graphql
query {
  __schema {
    directives {
      name
      description
      locations
      isRepeatable
      args {
        name
        description
        type {
          name
          kind
        }
        defaultValue
      }
    }
  }
}
```

## SDL Generation

Custom directives appear in the generated SDL:

```graphql
directive @auth(
  "Requires authentication with optional role"
  role: String
) on FIELD_DEFINITION

directive @cache(
  "Caches field results"
  ttl: Int
) repeatable on FIELD_DEFINITION

directive @uppercase on FIELD_DEFINITION
```

## Best Practices

1. **Use descriptive names**: Directive names should clearly indicate their purpose
2. **Provide descriptions**: Always add descriptions for better documentation
3. **Validate arguments**: Check argument types and values in your visitor
4. **Handle errors gracefully**: Return meaningful error messages
5. **Use context**: Pass authentication info and other context through `context.Context`
6. **Consider performance**: Directive execution adds overhead; keep visitors efficient
7. **Test thoroughly**: Test directive behavior in various scenarios

## Built-in Directives

Jaal includes these built-in directives:

| Directive | Locations | Description |
|-----------|-----------|-------------|
| `@skip` | FIELD, FRAGMENT_SPREAD, INLINE_FRAGMENT | Skip if `if: true` |
| `@include` | FIELD, FRAGMENT_SPREAD, INLINE_FRAGMENT | Include if `if: true` |
| `@deprecated` | FIELD, ARGUMENT_DEFINITION, INPUT_FIELD_DEFINITION | Mark as deprecated |
| `@specifiedBy` | SCALAR | Specify scalar specification URL |
| `@oneOf` | INPUT_OBJECT | Mark as oneOf input object |

## Migration Guide

### From Built-in Only to Custom Directives

If you're migrating from a version without custom directive support:

1. No code changes required for existing code
2. Custom directives are opt-in
3. Built-in directives continue to work as before
4. Add `jaal.WithDirectiveVisitors()` to enable custom directive execution

### Adding Directive Visitors to Existing Schemas

```go
// Before
schema := sb.MustBuild()
introspection.AddIntrospectionToSchema(schema)
handler := jaal.HTTPHandler(schema)

// After
directiveDefs := sb.GetDirectiveDefinitions()
directiveVisitors := sb.GetDirectiveVisitors()
schema := sb.MustBuild()
introspection.AddIntrospectionToSchemaWithDirectives(schema, directiveDefs)
handler := jaal.HTTPHandler(schema, jaal.WithDirectiveVisitors(directiveVisitors))
```

## Troubleshooting

### Directive Not Executing

1. Ensure directive visitors are passed to the handler
2. Check that the directive is applied to a field
3. Verify the directive location matches the application

### Directive Not in Introspection

1. Use `AddIntrospectionToSchemaWithDirectives()` instead of `AddIntrospectionToSchema()`
2. Pass directive definitions before building the schema

### Validation Errors

1. Check directive locations match the usage
2. For non-repeatable directives, ensure only one application per element
3. Verify argument types match the definition
