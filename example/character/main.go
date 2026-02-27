package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

func init() {
	var typ = reflect.TypeOf(time.Time{})
	// Register DateTime with @specifiedBy for spec compliance (as in main example).
	_ = schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
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

}

type Server struct {
	Characters []*Character
}

type Character struct {
	Id          string
	Name        string
	Type        Type
	DateOfBirth time.Time
	Metadata    map[string]string
}

type Type int32

const (
	WIZARD Type = iota
	MUGGLE
	GOBLIN
	HOUSE_ELF
)

type CreateCharacterRequest struct {
	Name        string
	Type        Type
	DateOfBirth time.Time
	Metadata    map[string]string
}

func RegisterPayload(schema *schemabuilder.Schema) {
	payload := schema.Object("Character", Character{}, schemabuilder.WithDescription("A character in the system."))
	payload.FieldFunc("id", func(ctx context.Context, in *Character) *schemabuilder.ID {
		return &schemabuilder.ID{Value: in.Id}
	}, schemabuilder.FieldDesc("Unique identifier for the character."))
	payload.FieldFunc("name", func(ctx context.Context, in *Character) string {
		return in.Name
	}, schemabuilder.FieldDesc("Character name."))
	payload.FieldFunc("type", func(ctx context.Context, in *Character) Type {
		return in.Type
	}, schemabuilder.FieldDesc("Character type."))
	payload.FieldFunc("dateOfBirth", func(ctx context.Context, in *Character) time.Time {
		return in.DateOfBirth
	}, schemabuilder.FieldDesc("Birth date of the character."))
	payload.FieldFunc("metadata", func(ctx context.Context, in *Character) (*schemabuilder.Map, error) {
		data, err := json.Marshal(in.Metadata)
		if err != nil {
			return nil, err
		}

		return &schemabuilder.Map{Value: string(data)}, nil
	}, schemabuilder.FieldDesc("Additional metadata for the character."))
}

func RegisterInput(schema *schemabuilder.Schema) {
	input := schema.InputObject("CreateCharacterRequest", CreateCharacterRequest{}, schemabuilder.WithDescription("Input payload for creating a character."))
	input.FieldFunc("name", func(target *CreateCharacterRequest, source string) {
		target.Name = source
	}, schemabuilder.FieldDesc("Name of the character."))
	input.FieldFunc("type", func(target *CreateCharacterRequest, source Type) {
		target.Type = source
	}, schemabuilder.FieldDesc("Type of the character."))
	input.FieldFunc("dateOfBirth", func(target *CreateCharacterRequest, source time.Time) {
		target.DateOfBirth = source
	}, schemabuilder.FieldDesc("Birth date of the character."))
	input.FieldFunc("metadata", func(target *CreateCharacterRequest, source schemabuilder.Map) error {
		v := source.Value

		decodedValue, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}

		data := make(map[string]string)
		if err := json.Unmarshal(decodedValue, &data); err != nil {
			return err
		}

		target.Metadata = data
		return nil
	}, schemabuilder.FieldDesc("Metadata values for the character."))
}

func RegisterEnum(schema *schemabuilder.Schema) {
	schema.Enum(Type(0), map[string]interface{}{
		"WIZARD":    Type(0),
		"MUGGLE":    Type(1),
		"GOBLIN":    Type(2),
		"HOUSE_ELF": Type(3),
	}, schemabuilder.WithDescription("Supported character types."))
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
	}, schemabuilder.FieldDesc("Fetch a character by ID."))

	schema.Query().FieldFunc("characters", func(ctx context.Context, args struct{}) []*Character {
		return s.Characters
	}, schemabuilder.FieldDesc("List all characters."))

	schema.Mutation().FieldFunc("createCharacter", func(ctx context.Context, args struct {
		Input *CreateCharacterRequest
	}) *Character {
		ch := &Character{
			Id:          uuid.Must(uuid.NewUUID()).String(),
			Name:        args.Input.Name,
			Type:        args.Input.Type,
			DateOfBirth: args.Input.DateOfBirth,
			Metadata:    args.Input.Metadata,
		}
		s.Characters = append(s.Characters, ch)

		return ch
	}, schemabuilder.FieldDesc("Create a new character."))
}

func main() {
	sb := schemabuilder.NewSchema()
	RegisterPayload(sb)
	RegisterInput(sb)
	RegisterEnum(sb)

	s := &Server{
		Characters: []*Character{{
			Id:          "015f13a5-cf9b-49d7-b457-6113bcf8fd56",
			Name:        "Harry Potter",
			Type:        WIZARD,
			DateOfBirth: time.Date(1980, time.July, 31, 0, 0, 0, 0, time.Local),
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
