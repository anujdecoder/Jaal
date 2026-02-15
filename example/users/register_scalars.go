package users

import (
	"errors"
	"reflect"
	"time"

	"go.appointy.com/jaal/schemabuilder"
)

// RegisterScalars registers custom scalars (e.g., DateTime w/ @specifiedBy URL for
// post-2018 spec compliance in introspection __Type.specifiedByURL).
// Called from RegisterSchema aggregator; follows original init in main.go and
// RegisterScalar pattern in schemabuilder/types.go/README.md.
func RegisterScalars(sb *schemabuilder.Schema) {
	typ := reflect.TypeOf(time.Time{})
	// Register DateTime scalar with @specifiedBy URL for full spec compliance
	// (Oct 2021+; exposed in introspection as __Type.specifiedByURL).
	// URL links external spec (RFC3339); backward compat for other scalars.
	if err := schemabuilder.RegisterScalar(typ, "DateTime", func(value interface{}, dest reflect.Value) error {
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
	}, "https://tools.ietf.org/html/rfc3339"); err != nil {
		// Panic on reg fail (pattern from schemabuilder scalar reg).
		panic(err)
	}
}