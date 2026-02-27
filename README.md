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
* Protocol buffers API generation
* Out-of-the-box GraphQL Playground (embedded, no CDN): `jaal.HTTPHandler` serves full UI + assets on the *same /graphql route* (GET for interactive explorer; POST for queries). No separate handler, redirect, mount, or example change required.

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

Jaal's `HTTPHandler` now supports GraphQL Playground out of the box *without any CDN or network dependencies*. The UI assets (HTML, CSS, JS, favicon) are embedded in the binary using Go's `embed` package and served on the *same /graphql route* internally (GET for UI; no separate handler/mount/redirect, no example change).

Once running, open `http://localhost:9000/graphql` (or your endpoint) in a browser to launch the interactive Playground UI for exploring the schema, testing queries/mutations (including resolvers for objects, interfaces, unions, etc.), and introspection. Client requests (e.g., POST) execute unchanged.

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

Deprecation is now options-based. The previous tag-based approach (`graphql:",deprecated=..."`) has been removed in favor of the cleaner options pattern.

## Custom Scalar Registration

Jaal supports custom scalars via reflection. Post-June 2018 spec compliance adds optional `specifiedByURL` (4th arg) for `@specifiedBy(url: String!)` directive on SCALAR (exposed in introspection __Type.specifiedByURL; e.g., for DateTime RFC).

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
}, "https://tools.ietf.org/html/rfc3339")
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
