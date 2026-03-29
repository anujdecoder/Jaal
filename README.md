# Jaal - Develop spec compliant GraphQL servers

Jaal is a go framework for building spec compliant GraphQL servers. Jaal has support for all the graphql scalar types and builds the schema from registered objects using reflection. Jaal is inspired from Thunder by Samsara.

## Features

* In-built support for graphQL scalars
* In-built support for maps
* Custom Scalar registration
* Input, Payload and enum registrations
* Custom field registration on objects
* Interface Support
* Union Support
* In build include and skip directives
* Custom directive registration with pre/post resolver execution and configurable fail behaviour
* Automatic batching / DataLoader via `BatchFieldFunc` (N+1 query prevention)
* Protocol buffers API generation
* Out-of-the-box GraphQL Playground (embedded, no CDN): `jaal.HTTPHandler` serves the graphql-playground UI + assets on the *same /graphql route* (GET for interactive explorer; POST for queries). No separate handler, redirect, mount, or example change required.

## Getting Started

The following example depicts how to build a simple graphQL server using jaal.

``` Go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/google/uuid"
    "go.appointy.com/jaal"
    "go.appointy.com/jaal/introspection"
    "go.appointy.com/jaal/schemabuilder"
)

type Server struct {
    Characters []*Character
}

type Character struct {
    Id   string
    Name string
    Type Type
}

type Type int32

const (
    WIZARD Type = iota
    MUGGLE
    GOBLIN
    HOUSE_ELF
)

type CreateCharacterRequest struct {
    Name string
    Type Type
}

func RegisterPayload(schema *schemabuilder.Schema) {
    payload := schema.Object("Character", Character{}, schemabuilder.WithDescription("A character in the story"))
    payload.FieldFunc("id", func(ctx context.Context, in *Character) *schemabuilder.ID {
        return &schemabuilder.ID{Value: in.Id}
    }, schemabuilder.FieldDesc("Unique identifier"))
    payload.FieldFunc("name", func(ctx context.Context, in *Character) string {
        return in.Name
    }, schemabuilder.FieldDesc("Character name"))
    payload.FieldFunc("type", func(ctx context.Context, in *Character) Type {
        return in.Type
    }, schemabuilder.FieldDesc("Character type"))
}

func RegisterInput(schema *schemabuilder.Schema) {
    input := schema.InputObject("CreateCharacterRequest", CreateCharacterRequest{}, schemabuilder.WithDescription("Input payload for creating a character"))
    input.FieldFunc("name", func(target *Character, source string) {
        target.Name = source
    }, schemabuilder.FieldDesc("Character name"))
    input.FieldFunc("type", func(target *Character, source Type) {
        target.Type = source
    }, schemabuilder.FieldDesc("Character type"))
}

func RegisterEnum(schema *schemabuilder.Schema) {
    schema.Enum(Type(0), map[string]interface{}{
        "WIZARD":    Type(0),
        "MUGGLE":    Type(0),
        "GOBLIN":    Type(0),
        "HOUSE_ELF": Type(0),
    }, schemabuilder.WithDescription("Supported character types"))
}

func (s *Server) RegisterOperations(schema *schemabuilder.Schema) {
    schema.Query().FieldFunc("character", func(ctx context.Context, args struct {
        Id *schemabuilder.ID
    }) *Character {
        for _, ch := range s.Characters {
            if ch.Id == args.Id.Value {
                return ch
            }
        }

        return nil
    }, schemabuilder.FieldDesc("Fetch a character by ID"))

    schema.Query().FieldFunc("characters", func(ctx context.Context, args struct{}) []*Character {
        return s.Characters
    }, schemabuilder.FieldDesc("List all characters"))

    schema.Mutation().FieldFunc("createCharacter", func(ctx context.Context, args struct {
        Input *CreateCharacterRequest
    }) *Character {
        ch := &Character{
            Id:   uuid.Must(uuid.NewUUID()).String(),
            Name: args.Input.Name,
            Type: args.Input.Type,
        }
        s.Characters = append(s.Characters, ch)

        return ch
    }, schemabuilder.FieldDesc("Create a new character"))
}

func main() {
    sb := schemabuilder.NewSchema()
    RegisterPayload(sb)
    RegisterInput(sb)
    RegisterEnum(sb)

    s := &Server{
        Characters: []*Character{{
            Id:   "015f13a5-cf9b-49d7-b457-6113bcf8fd56",
            Name: "Harry Potter",
            Type: WIZARD,
        }},
    }

    s.RegisterOperations(sb)
    schema, err := sb.Build()
    if err != nil {
        log.Fatalln(err)
    }

    introspection.AddIntrospectionToSchema(schema)

    http.Handle("/graphql", jaal.HTTPHandler(schema))
    log.Println("Running")
    if err := http.ListenAndServe(":9000", nil); err != nil {
        panic(err)
    }
}
```

Jaal's `HTTPHandler` now supports graphql-playground out of the box *without any CDN or network dependencies*. The UI assets (HTML, CSS, JS, favicon) are embedded in the binary using Go's `embed` package and served on the *same /graphql route* internally (GET for UI; no separate handler/mount/redirect, no example change).

Once running, open `http://localhost:9000/graphql` (or your endpoint) in a browser to launch the graphql-playground UI for exploring the schema, viewing SDL, testing queries/mutations (including resolvers for objects, interfaces, unions, etc.), and introspection. Client requests (e.g., POST) execute unchanged.

In the above example, we registered all the operations, inputs & payloads on the schema. We also registered the fields we wanted to expose on the schema. Field registration allows us to control the way in which a field is exposed. Here we exposed the field Id of Character as the graphQL scalar ID.

### Description Options

Use options to attach descriptions during schema registration:

- `schemabuilder.WithDescription("...")` for objects, inputs, and enums
- `schemabuilder.FieldDesc("...")` for fields

This options pattern is the recommended way to add descriptions going forward.

### Deprecation Options

Mark fields as deprecated using the `Deprecated` option:

```go
// Deprecate an output field
user.FieldFunc("oldField", func(u *User) string {
    return u.OldField
}, schemabuilder.Deprecated("Use newField instead"))

// Deprecate an input field
input.FieldFunc("age", func(target *CreateUserInput, source int32) {
    target.Age = source
}, schemabuilder.FieldDesc("Age in years (deprecated)."), schemabuilder.Deprecated("Use birthdate instead"))
```

### OneOf Input Objects

Mark input objects as OneOf (@oneOf directive per Oct 2021+ spec) using the `MarkOneOf()` method:

```go
type IdentifierInput struct {
    ID    *schemabuilder.ID
    Email *string
}

input := sb.InputObject("IdentifierInput", IdentifierInput{}, 
    schemabuilder.WithDescription("OneOf identifier: exactly one of ID or email"))
input.MarkOneOf()
input.FieldFunc("id", func(target *IdentifierInput, source *schemabuilder.ID) {
    target.ID = source
})
input.FieldFunc("email", func(target *IdentifierInput, source *string) {
    target.Email = source
})
```

Exactly one field must be provided/non-null in queries (enforced during input coercion). This is the recommended way to register oneOf inputs.

## Custom Scalar Registration

Jaal supports custom scalars via reflection. Use the `WithSpecifiedBy` option to attach the `@specifiedBy(url: String!)` directive on scalars (exposed in introspection __Type.specifiedByURL; e.g., for DateTime RFC).

```Go
typ := reflect.TypeOf(time.Time{})
// Register with URL for spec-compliant @specifiedBy (informational; links external type spec).
schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
    v, ok := value.(string)
    if !ok {
        return errors.New("invalid type expected string")
    }

    t, err := time.Parse(time.RFC3339, v)
    if err != nil {
        return err
    }

    dest.Set(reflect.ValueOf(t))

    return nil
}, schemabuilder.WithSpecifiedBy("https://tools.ietf.org/html/rfc3339"))
```

(Backward-compatible: omit URL for pre-existing calls; defaults to null in introspection.)


## Interface Registration

```Go
type server struct {
    dragons []Dragon
    snakes  []Snake
}

type Dragon struct {
    Id           string
    Name         string
    NumberOfLegs int32
}

type Snake struct {
    Id             string
    Name           string
    LengthInMetres float32
}

type MagicalCreature struct {
    schemabuilder.Interface
    *Dragon
    *Snake
}

func (s *server) RegisterInterface(schema *schemabuilder.Schema) {
    schema.Query().FieldFunc("magicalCreature", func(ctx context.Context, args struct {
        Id *schemabuilder.ID
    }) *MagicalCreature {
        for _, d := range s.dragons {
            if d.Id == args.Id.Value {
                return &MagicalCreature{
                    Dragon: &d,
                }
            }
        }

        for _, sn := range s.snakes {
            if sn.Id == args.Id.Value {
                return &MagicalCreature{
                    Snake: &sn,
                }
            }
        }

        return nil
    }, schemabuilder.FieldDesc("Fetch a magical creature by ID."))
}

func RegisterPayloads(schema *schemabuilder.Schema) {
    payload := schema.Object("Dragon", Dragon{}, schemabuilder.WithDescription("Dragon payload."))
    payload.FieldFunc("id", func(ctx context.Context, in *Dragon) schemabuilder.ID {
        return schemabuilder.ID{Value: in.Id}
    }, schemabuilder.FieldDesc("Dragon ID."))
    payload.FieldFunc("name", func(ctx context.Context, in *Dragon) string {
        return in.Name
    }, schemabuilder.FieldDesc("Dragon name."))
    payload.FieldFunc("numberOfLegs", func(ctx context.Context, in *Dragon) int32 {
        return in.NumberOfLegs
    }, schemabuilder.FieldDesc("Number of legs."))

    payload = schema.Object("Snake", Snake{}, schemabuilder.WithDescription("Snake payload."))
    payload.FieldFunc("id", func(ctx context.Context, in *Snake) schemabuilder.ID {
        return schemabuilder.ID{Value: in.Id}
    }, schemabuilder.FieldDesc("Snake ID."))
    payload.FieldFunc("name", func(ctx context.Context, in *Snake) string {
        return in.Name
    }, schemabuilder.FieldDesc("Snake name."))
    payload.FieldFunc("lengthInMetres", func(ctx context.Context, in *Snake) float32 {
        return in.LengthInMetres
    }, schemabuilder.FieldDesc("Snake length in metres."))
}

func main() {
    s := server{
        dragons: []Dragon{
            {
                Id:           "01d823a8-fdcd-4d03-957c-7ca898e2e5fd",
                Name:         "Norbert",
                NumberOfLegs: 2,
            },
        },
        snakes: []Snake{
            {
                Id:             "2631a997-7a73-45b2-a2fc-ae665a383da1",
                Name:           "Nagini",
                LengthInMetres: 1.23,
            },
        },
    }

    sb := schemabuilder.NewSchema()
    RegisterPayloads(sb)
    s.RegisterInterface(sb)

    schema := sb.MustBuild()
    introspection.AddIntrospectionToSchema(schema)

    http.Handle("/graphql", jaal.HTTPHandler(schema))
    fmt.Println("Running")
    if err := http.ListenAndServe(":9000", nil); err != nil {
        panic(err)
    }
}
```

See the Getting Started section above for details on GraphQL Playground support.

The object schemabuilder.Interface acts as a special marker. It indicates that the type is to be registered as an interface. Jaal automatically registers the common fields(Id, Name) of the objects(Dragon & Snake) as the fields of interface (MagicalCreature). While defining a struct for interface, one must remember that all the fields of that struct are anonymous.

## Union Registration

The above example can be converted to a union by replacing schemabuilder.Interface with schemabuilder.Union and RegisterInterface() by RegisterUnion().

```Go
type MagicalCreature struct {
    schemabuilder.Union
    *Dragon
    *Snake
}

func (s *server) RegisterUnion(schema *schemabuilder.Schema) {
    schema.Query().FieldFunc("magicalCreature", func(ctx context.Context, args struct {
        Id *schemabuilder.ID
    }) *MagicalCreature {
        for _, d := range s.dragons {
            if d.Id == args.Id.Value {
                return &MagicalCreature{
                    Dragon: &d,
                }
            }
        }

        for _, sn := range s.snakes {
            if sn.Id == args.Id.Value {
                return &MagicalCreature{
                    Snake: &sn,
                }
            }
        }

        return nil
    }, schemabuilder.FieldDesc("Fetch a magical creature by ID."))
}
```

## Automatic Batching / DataLoader

Jaal provides built-in DataLoader-style automatic batching via `BatchFieldFunc`. Instead of resolving a field once per item in a list (the N+1 problem), a batch resolver is called **once** with all source objects and must return a result slice of the same length.

### Basic usage

```Go
type Article struct {
    ID       string
    AuthorID string
}

type Author struct {
    ID   string
    Name string
}

// Register batch-resolved field on Article.
article := sb.Object("Article", Article{})
article.FieldFunc("id", func(a Article) string { return a.ID })

// BatchFieldFunc: called once with ALL articles instead of N times.
article.BatchFieldFunc("author", func(ctx context.Context, articles []Article) ([]*Author, error) {
    // Collect all author IDs, do a single DB query.
    ids := make([]string, len(articles))
    for i, a := range articles {
        ids[i] = a.AuthorID
    }
    return db.BatchGetAuthors(ctx, ids) // returns []*Author, error
})
```

When the query `{ articles { id author { name } } }` executes on a list of 100 articles, the `author` batch resolver is called **exactly once** with all 100 articles, instead of 100 individual calls.

### Supported function signatures

```
func([ctx context.Context,] sources []SourceType [, args struct{}] [, *graphql.SelectionSet]) ([]ResultType, [error])
```

- `SourceType` must match the object's Go type (value or pointer).
- `ResultType` is the per-item return value (scalar, struct, pointer, slice, etc.).
- The result slice must have the **same length** as the sources slice.

### Batch with arguments

```Go
article.BatchFieldFunc("preview", func(articles []Article, args struct{ MaxLen int32 }) ([]string, error) {
    out := make([]string, len(articles))
    for i, a := range articles {
        r := []rune(a.Body)
        if int32(len(r)) > args.MaxLen {
            r = r[:args.MaxLen]
        }
        out[i] = string(r)
    }
    return out, nil
})
// Query: { articles { preview(maxLen: 100) } }
```

### Mixing batch and normal fields

Batch and non-batch fields coexist on the same object. Batch fields are resolved once in bulk; non-batch fields resolve per item as usual:

```Go
article.FieldFunc("title", func(a Article) string { return a.Title })        // per-item
article.BatchFieldFunc("author", func(articles []Article) ([]*Author, error) { // batched
    // ...
})
```

### Single-item fallback

Batch fields also work outside of list contexts (e.g., `{ article { author { name } } }`). A single-item `Resolve` fallback is automatically generated that wraps the batch call with a slice of one.

### Options

`BatchFieldFunc` supports the same `FieldOption` values as `FieldFunc`:

```Go
article.BatchFieldFunc("slug", batchSlugResolver,
    schemabuilder.FieldDesc("URL-friendly slug"),
    schemabuilder.Deprecated("Use canonicalUrl instead"),
    schemabuilder.NonNull(),
    schemabuilder.WithFieldDirective("hasRole", map[string]interface{}{"role": "ADMIN"}),
)
```

### Nested batching

When a batch field returns a list of objects that themselves have batch fields, the nested batch resolvers fire when the inner lists are executed. This is fully automatic.

## Custom Directive Registration

Jaal supports custom schema-level directives that execute handler functions at resolve time. This is ideal for access control (roles, rights, feature flags), auditing, rate limiting, and more. Directives are defined once on the schema, then attached to individual fields or entire object types.

### Defining and registering a directive

```Go
sb := schemabuilder.NewSchema()

// Register a @hasRole directive — PreResolver (default), ErrorOnFail (default).
sb.Directive("hasRole", &schemabuilder.DirectiveDefinition{
    Description: "Restricts field access to the specified role",
    Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
    Args: []schemabuilder.DirectiveArgDef{
        {Name: "role", TypeName: "String", Description: "Required role"},
    },
    // ExecutionOrder: schemabuilder.PreResolver,  // default — runs before the resolver
    // OnFail:         schemabuilder.ErrorOnFail,   // default — returns the error to the client
    Handler: func(ctx context.Context, args map[string]interface{}) error {
        requiredRole := args["role"].(string)
        userRole, _ := ctx.Value("userRole").(string)
        if userRole != requiredRole {
            return fmt.Errorf("access denied: requires role %s", requiredRole)
        }
        return nil
    },
})
```

### Attaching a directive to a field

Attach directives using `WithFieldDirective` in `FieldFunc`. The static args are supplied at registration time (not in the query):

```Go
query := sb.Query()
query.FieldFunc("adminUsers", func(ctx context.Context) []*User {
    return getAllAdminUsers(ctx)
},
    schemabuilder.FieldDesc("List of admin users"),
    schemabuilder.WithFieldDirective("hasRole", map[string]interface{}{"role": "ADMIN"}),
)
```

### Attaching a directive to an entire object type

Use `WithDirective` on `Object()` to apply the directive to every field of that type:

```Go
obj := sb.Object("SecretData", SecretData{}, schemabuilder.WithDirective("hasRole", map[string]interface{}{"role": "ADMIN"}))
obj.FieldFunc("code",  func(s SecretData) string { return s.Code })
obj.FieldFunc("label", func(s SecretData) string { return s.Label })
// Both "code" and "label" are now wrapped by @hasRole(role: "ADMIN").
```

### Execution order

By default directives run **before** the field resolver (`PreResolver`). Set `PostResolver` to run **after** the resolver:

```Go
sb.Directive("audit", &schemabuilder.DirectiveDefinition{
    Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
    ExecutionOrder: schemabuilder.PostResolver,
    Handler: func(ctx context.Context, args map[string]interface{}) error {
        log.Println("field resolved — audit pass")
        return nil
    },
})
```

### Fail behaviour

By default, if a directive handler returns an error the error is propagated to the client (`ErrorOnFail`). Set `SkipOnFail` to silently return `null` for the field instead:

```Go
sb.Directive("featureFlag", &schemabuilder.DirectiveDefinition{
    Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
    OnFail:         schemabuilder.SkipOnFail, // null instead of error
    Handler: func(ctx context.Context, args map[string]interface{}) error {
        if !isFeatureEnabled(ctx, args["flag"].(string)) {
            return errors.New("disabled") // silently returns null
        }
        return nil
    },
})
```

### Metadata-only directives

Pass `Handler: nil` for directives that only need to appear in introspection (e.g. cache hints):

```Go
sb.Directive("cacheControl", &schemabuilder.DirectiveDefinition{
    Description: "Cache control hints",
    Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
    Args:        []schemabuilder.DirectiveArgDef{{Name: "maxAge", TypeName: "Int"}},
    Handler:     nil, // no runtime effect; visible in __Schema.directives
})
```

Custom directives automatically appear in GraphQL introspection (`__schema { directives { ... } }`), alongside the built-in `@skip`, `@include`, `@deprecated`, `@specifiedBy`, and `@oneOf` directives.

## protoc-gen-jaal - Develop relay compliant GraphQL servers

[protoc-gen-jaal](https://github.com/appointy/protoc-gen-jaal) is a protoc plugin which is used to generate jaal APIs. The server built from these APIs is graphQL spec compliant as well as relay compliant. It also handles oneOf by registering it as a Union on the schema.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for reporting bugs and issues, the process for submitting pull requests to us, and roadmap. This project has adopted [Contributor Covenant Code of Conduct](code-of-conduct.md).

## Contributors

* Souvik Mandal (mandalsouvik76@gmail.com) - Implemented protoc-gen-jaal for creating jaal APIs.
* Bitan Paul (bitanpaul1@gmail.com) - Implemented relay compliant graphql subscriptions.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

* **Samsara** - *Initial work* - [Thunder](https://github.com/samsarahq/thunder)
