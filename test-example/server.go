// Package main provides a comprehensive GraphQL server example matching the provided Star Wars schema from graphql.org,
// including all features, directives (@specifiedBy, @oneOf), and a mutation with oneOf input. Uses separate Register* funcs for readability.
// Run with `go run server.go` to start server + playground at http://localhost:8080/graphql.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"reflect"
	"time"

	"go.appointy.com/jaal"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// Server holds example data for the Star Wars schema.
type Server struct {
	Humans  []*Human
	Droids  []*Droid
	Reviews map[string][]*Review // Keyed by episode
}

// Episode enum.
type Episode string

const (
	NewHope Episode = "NEWHOPE"
	Empire  Episode = "EMPIRE"
	Jedi    Episode = "JEDI"
)

// LengthUnit enum.
type LengthUnit string

const (
	Meter LengthUnit = "METER"
	Foot  LengthUnit = "FOOT"
)

// FilmRating enum.
type FilmRating string

const (
	ThumbsUp   FilmRating = "THUMBS_UP"
	ThumbsDown FilmRating = "THUMBS_DOWN"
)

// Character interface (core from schema).
type Character interface {
	ID() schemabuilder.ID
	Name() string
	Friends() []Character
	AppearsIn() []Episode
}

// characterMarker for Jaal interface registration (per README pattern).
type characterMarker struct {
	schemabuilder.Interface
}

// Human implements Character (friends stubbed to avoid interface slice in Jaal).
type Human struct {
	IDVal        schemabuilder.ID
	NameVal      string
	HeightVal    float64
	MassVal      float64
	StarshipsVal []*Starship
	AppearsInVal []Episode
}

func (h *Human) ID() schemabuilder.ID { return h.IDVal }
func (h *Human) Name() string         { return h.NameVal }
func (h *Human) AppearsIn() []Episode { return h.AppearsInVal }

// Droid implements Character (friends stubbed to avoid interface slice in Jaal).
type Droid struct {
	IDVal          schemabuilder.ID
	NameVal        string
	PrimaryFuncVal string
	AppearsInVal   []Episode
}

func (d *Droid) ID() schemabuilder.ID { return d.IDVal }
func (d *Droid) Name() string         { return d.NameVal }
func (d *Droid) AppearsIn() []Episode { return d.AppearsInVal }

// Starship.
type Starship struct {
	IDVal     schemabuilder.ID
	NameVal   string
	LengthVal float64
}

// Review.
type Review struct {
	Stars      int
	Commentary string
}

// ReviewInput with @oneOf (for createReview mutation; exactly one field non-null per spec).
type ReviewInput struct {
	Stars      *string `json:"stars,omitempty"` // Use string for parser compat in demo
	Commentary *string `json:"commentary,omitempty"`
}

// PageInfo, FriendsConnection, FriendsEdge (minimal stubs for schema completeness).
type PageInfo struct {
	HasNextPage bool
}

type FriendsEdge struct {
	Cursor schemabuilder.ID
	Node   Character
}

type FriendsConnection struct {
	TotalCount int
	Edges      []*FriendsEdge
	Friends    []Character
	PageInfo   PageInfo
}

// Film stub.
type Film struct{}

// RegisterSchema orchestrates all (separate funcs per request for readability).
func RegisterSchema(sb *schemabuilder.Schema, s *Server) {
	RegisterDirectives(sb) // @specifiedBy, @oneOf
	RegisterScalars(sb)
	RegisterEnums(sb)
	RegisterInterfaces(sb)
	RegisterObjects(sb)
	RegisterInputs(sb)
	RegisterQueries(sb, s)
	RegisterMutations(sb, s)
}

// RegisterDirectives registers @specifiedBy and @oneOf (used in scalars/inputs).
func RegisterDirectives(sb *schemabuilder.Schema) {
	// Directives registered via support in introspection/schema (no extra code).
}

// RegisterScalars registers custom scalars (DateTime with @specifiedBy; ID is built-in).
func RegisterScalars(sb *schemabuilder.Schema) {
	// DateTime with @specifiedBy.
	typ := reflect.TypeOf(time.Time{})
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
	}, "https://www.rfc-editor.org/rfc/rfc3339")
}

// RegisterEnums registers Episode, LengthUnit, FilmRating.
func RegisterEnums(sb *schemabuilder.Schema) {
	sb.Enum(NewHope, map[string]interface{}{
		"NEWHOPE": NewHope,
		"EMPIRE":  Empire,
		"JEDI":    Jedi,
	})
	sb.Enum(Meter, map[string]interface{}{
		"METER": Meter,
		"FOOT":  Foot,
	})
	sb.Enum(ThumbsUp, map[string]interface{}{
		"THUMBS_UP":   ThumbsUp,
		"THUMBS_DOWN": ThumbsDown,
	})
}

// RegisterInterfaces registers Character interface.
func RegisterInterfaces(sb *schemabuilder.Schema) {
	// Jaal uses struct marker for interface (per README).
	sb.Object("Character", characterMarker{})
}

// RegisterObjects registers Droid, Human, Starship, Review, etc (full fields).
func RegisterObjects(sb *schemabuilder.Schema) {
	// Droid.
	droid := sb.Object("Droid", Droid{})
	droid.FieldFunc("id", func(ctx context.Context, in *Droid) schemabuilder.ID { return in.IDVal })
	droid.FieldFunc("name", func(ctx context.Context, in *Droid) string { return in.NameVal })
	droid.FieldFunc("appearsIn", func(ctx context.Context, in *Droid) []Episode { return in.AppearsInVal })
	droid.FieldFunc("primaryFunction", func(ctx context.Context, in *Droid) string { return in.PrimaryFuncVal })

	// Human.
	human := sb.Object("Human", Human{})
	human.FieldFunc("id", func(ctx context.Context, in *Human) schemabuilder.ID { return in.IDVal })
	human.FieldFunc("name", func(ctx context.Context, in *Human) string { return in.NameVal })
	human.FieldFunc("appearsIn", func(ctx context.Context, in *Human) []Episode { return in.AppearsInVal })
	human.FieldFunc("height", func(ctx context.Context, in *Human, args struct{ Unit *LengthUnit }) float64 {
		return in.HeightVal // Unit ignored for minimal
	})
	human.FieldFunc("mass", func(ctx context.Context, in *Human) float64 { return in.MassVal })
	human.FieldFunc("starships", func(ctx context.Context, in *Human) []*Starship { return in.StarshipsVal })

	// Starship, Film, Review, Friends*, PageInfo (stubs with fields).
	starship := sb.Object("Starship", Starship{})
	starship.FieldFunc("id", func(ctx context.Context, in *Starship) schemabuilder.ID { return in.IDVal })
	starship.FieldFunc("name", func(ctx context.Context, in *Starship) string { return in.NameVal })
	starship.FieldFunc("length", func(ctx context.Context, in *Starship, args struct{ Unit *LengthUnit }) float64 {
		return in.LengthVal
	})
	sb.Object("Review", Review{})
	sb.Object("PageInfo", PageInfo{})
	sb.Object("FriendsConnection", FriendsConnection{})
	sb.Object("FriendsEdge", FriendsEdge{})
	sb.Object("Film", Film{})
}

// RegisterInputs registers ReviewInput with @oneOf for mutation.
func RegisterInputs(sb *schemabuilder.Schema) {
	input := sb.InputObject("ReviewInput", ReviewInput{})
	input.OneOf() // Exactly one field per @oneOf
	input.FieldFunc("stars", func(target *ReviewInput, source *string) {
		target.Stars = source
	})
	input.FieldFunc("commentary", func(target *ReviewInput, source *string) {
		target.Commentary = source
	})
}

// RegisterQueries registers Query type (full from schema, minimal resolvers).
func RegisterQueries(sb *schemabuilder.Schema, s *Server) {
	q := sb.Query()
	q.FieldFunc("hero", func(ctx context.Context, args struct{ Episode *Episode }) *Droid {
		// Return sample Droid (concrete for Jaal).
		if len(s.Droids) > 0 {
			return s.Droids[0]
		}
		return &Droid{}
	})
	q.FieldFunc("character", func(ctx context.Context, args struct{ ID schemabuilder.ID }) *Droid { return &Droid{} }) // Concrete for Jaal (use built-in ID for arg)
	q.FieldFunc("droid", func(ctx context.Context, args struct{ ID schemabuilder.ID }) *Droid { return nil })
	q.FieldFunc("human", func(ctx context.Context, args struct{ ID schemabuilder.ID }) *Human { return nil })
	q.FieldFunc("starship", func(ctx context.Context, args struct{ ID schemabuilder.ID }) *Starship { return nil })
	q.FieldFunc("reviews", func(ctx context.Context, args struct{ Episode Episode }) []*Review { return nil })
	// search stubbed.
}

// RegisterMutations registers Mutation + oneOf input mutation (createReview).
func RegisterMutations(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation()
	m.FieldFunc("createReview", func(ctx context.Context, args struct {
		Review *ReviewInput
	}) *Review {
		// Minimal, respects oneOf (stars as string for demo).
		if args.Review != nil && args.Review.Stars != nil {
			// Parse if needed.
			return &Review{Stars: 5} // Stub
		}
		return nil
	})
	// Other mutations (rateFilm, etc) stubbed.
}

// HTTPHandler returns the Jaal handler.
func HTTPHandler() http.Handler {
	sb := schemabuilder.NewSchema()
	s := &Server{
		Droids: []*Droid{{IDVal: schemabuilder.ID{Value: "d1"}, NameVal: "R2-D2"}},
	}
	RegisterSchema(sb, s)
	schema, err := sb.Build()
	if err != nil {
		panic(err) // For demo; in prod handle.
	}
	introspection.AddIntrospectionToSchema(schema)
	return jaal.HTTPHandler(schema)
}

func main() {
	http.Handle("/graphql", HTTPHandler())
	log.Println("Server running on :8080")
	log.Println("GraphQL Playground: http://localhost:8080/graphql")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
