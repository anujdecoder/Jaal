package schemabuilder

import (
	"errors"
	"fmt"
	"reflect"

	"go.appointy.com/jaal/graphql"
)

// makeInputObjectParser constructs an argParser for the passed in args struct i.e. the input struct which contains all the objects to be given as input. For eg:
// obj.fieldFunc("name", func(ctx context.Context, args struct{
// 	A createObjectRequest
// }{}))
func (sb *schemaBuilder) makeInputObjectParser(typ reflect.Type) (*argParser, graphql.Type, error) {
	if typ.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("expected struct but received type %s", typ.Name())
	}

	argType, fields, err := sb.generateArgParser(typ)
	if err != nil {
		return nil, nil, err
	}

	// Capture OneOf for closure (set in generateArgParser if OneOfInput marker embedded;
	// enables @oneOf validation for inline args structs in FieldFunc resolvers, e.g.,
	// schema.Query().FieldFunc(..., func(..., args struct{ In ContactInput })).
	// Matches capture in registered case below.
	oneOf := argType.OneOf

	return &argParser{
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asMap, ok := value.(map[string]interface{})
			if !ok {
				return errors.New("not an object")
			}

			// Validate @oneOf first if flagged (exactly 1 non-null; spec input unions;
			// from validateOneOfInput helper). For arg structs (e.g., README examples'
			// mutation args). Early check before field fill/unknown check; skips for
			// !OneOf (BC for existing).
			if oneOf {
				if err := validateOneOfInput(argType.Name, asMap); err != nil {
					return err
				}
			}

			for name, field := range fields {
				value := asMap[name]
				fieldDest := dest.FieldByIndex(field.field.Index)
				if err := field.parser.FromJSON(value, fieldDest); err != nil {
					return fmt.Errorf("%s: %s", name, err)
				}
			}

			for name := range asMap {
				if _, ok := fields[name]; !ok {
					return fmt.Errorf("unknown arg %s", name)
				}
			}
			return nil
		},
		Type: typ,
	}, argType, nil
}

// generateArgParser generates the parser for each field of args struct
func (sb *schemaBuilder) generateArgParser(typ reflect.Type) (*graphql.InputObject, map[string]argField, error) {
	fields := make(map[string]argField)
	argType := &graphql.InputObject{
		Name:              typ.Name(),
		InputFields:       make(map[string]graphql.Type),
		// FieldDeprecations for INPUT_FIELD_DEFINITION deprecation (spec; from tag parse).
		// Empty string = non-deprecated (compat); populated below per field.
		// OneOf for @oneOf INPUT_OBJECT (set below if marker embedded; default false).
		FieldDeprecations: make(map[string]string),
	}

	// Detect @oneOf marker early (for arg structs in FieldFunc e.g., args{Input: ContactInput}).
	// Mirrors union handling; allows anon OneOfInput embed (spec input unions).
	if hasOneOfMarkerEmbedded(typ) {
		argType.OneOf = true
	}

	// Cache type information ahead of time to catch self-reference
	sb.typeCache[typ] = cachedType{argType, fields}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// Allow anonymous OneOfInput marker embed (for @oneOf; skip like unionType in
		// output.go/buildUnionStruct). Other anon fields still err (non-breaking).
		if field.Anonymous && field.Type == oneOfInputType {
			continue
		}
		if field.Anonymous {
			return nil, nil, fmt.Errorf("bad arg type %s: anonymous fields not supported", typ)
		}

		fieldInfo, err := parseGraphQLFieldInfo(field)
		if err != nil {
			return nil, nil, fmt.Errorf("bad type %s: %s", typ, err.Error())
		}
		if fieldInfo.Skipped {
			continue
		}

		if _, ok := fields[fieldInfo.Name]; ok {
			return nil, nil, fmt.Errorf("bad arg type %s: duplicate field %s", typ, fieldInfo.Name)
		}

		parser, fieldArgTyp, err := sb.generateObjectParser(field.Type)
		if err != nil {
			return nil, nil, err
		}

		// Capture deprecation from tag (for INPUT_FIELD_DEFINITION spec support).
		// Propagated to introspection via FieldDeprecations; empty = non-dep (compat/stubs).
		fields[fieldInfo.Name] = argField{
			field:             field,
			parser:            parser,
			DeprecationReason: fieldInfo.DeprecationReason,
		}
		argType.InputFields[fieldInfo.Name] = fieldArgTyp
		if fieldInfo.DeprecationReason != "" {
			argType.FieldDeprecations[fieldInfo.Name] = fieldInfo.DeprecationReason
		}
	}

	return argType, fields, nil
}

// generateObjectParser generates the parser the object in args struct
func (sb *schemaBuilder) generateObjectParser(typ reflect.Type) (*argParser, graphql.Type, error) {
	if typ.Kind() == reflect.Ptr {
		parser, argType, err := sb.generateObjectParserInner(typ.Elem())
		if err != nil {
			return nil, nil, err
		}
		return wrapPtrParser(parser), argType, nil
	}

	parser, argType, err := sb.generateObjectParserInner(typ)
	if err != nil {
		return nil, nil, err
	}
	return parser, argType, nil
}

// generateObjectParserInner generates the parser without having to worry about pointer.
// It creates parser using the registered fields and maps the value from http request into them.
func (sb *schemaBuilder) generateObjectParserInner(typ reflect.Type) (*argParser, graphql.Type, error) {
	if sb.enumMappings[typ] != nil {
		parser, argType := sb.getEnumArgParser(typ)
		return parser, argType, nil
	}

	if isScalarType(typ) {
		return sb.getInputFieldParser(typ)
	}

	if typ.Kind() == reflect.Slice {
		return sb.generateSliceParser(typ)
	}

	if _, ok := sb.inputObjects[typ]; !ok {
		return nil, nil, fmt.Errorf("%s not registered as input object", typ.Name())
	}

	obj := sb.inputObjects[typ]
	fields := make(map[string]argField)
	argType := &graphql.InputObject{
		Name:              obj.Name,
		InputFields:       make(map[string]graphql.Type),
		// FieldDeprecations default empty (for FieldFunc-based inputs; no tag parse).
		// Matches generateArgParser for struct inputs.
		// OneOf for @oneOf INPUT_OBJECT (detected via marker embed below; default false
		// for BC w/ existing registered inputs like CreateUserInput).
		FieldDeprecations: make(map[string]string),
	}

	// For registered InputObject from struct (e.g., CreateUserInput with graphql/json tags
	// for deprecation on fields like age), parse fields to FieldDeprecations.
	// Ensures INPUT_FIELD_DEFINITION deprecation when used as nested input in args struct
	// (FieldFunc case; dummy field bypasses parseGraphQLFieldInfo). Matches reflect.go
	// pattern + bug fix for example/main.go introspection.
	//
	// Also detect OneOfInput marker embed for @oneOf (spec input unions; mirrors
	// structTyp deprecation parse + hasOneOfMarkerEmbedded in arg structs case from
	// generateArgParser).
	if obj.Type != nil {
		structTyp := reflect.TypeOf(obj.Type)
		if structTyp.Kind() == reflect.Ptr {
			structTyp = structTyp.Elem()
		}
		if hasOneOfMarkerEmbedded(structTyp) {
			argType.OneOf = true
		}
		for i := 0; i < structTyp.NumField(); i++ {
			f := structTyp.Field(i)
			fieldInfo, _ := parseGraphQLFieldInfo(f) // skip err; handled in other paths
			if !fieldInfo.Skipped && fieldInfo.DeprecationReason != "" {
				argType.FieldDeprecations[fieldInfo.Name] = fieldInfo.DeprecationReason
			}
		}
	}

	for name, function := range obj.Fields {
		field := reflect.StructField{Name: name}
		funcTyp := reflect.TypeOf(function)
		sourceTyp := funcTyp.In(1)

		parser, fieldArgTyp, err := sb.getInputFieldParser(sourceTyp)
		if err != nil {
			return nil, nil, err
		}

		fields[name] = argField{
			field:  field,
			parser: parser,
		}
		argType.InputFields[name] = fieldArgTyp
	}

	// Capture OneOf for closure (graphql.InputObject.OneOf from marker detect above;
	// enables spec validation w/o changing argType ref).
	oneOf := argType.OneOf

	return &argParser{
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asMap, ok := value.(map[string]interface{})
			if !ok {
				return errors.New("not an object")
			}

			// Validate @oneOf first (exactly 1 non-null field per spec for input unions;
			// from validateOneOfInput in input.go). Runs for registered inputs (e.g.,
			// ContactInput); before field processing to early-error. Non-breaking:
			// skips if !oneOf (existing inputs default false).
			if oneOf {
				if err := validateOneOfInput(argType.Name, asMap); err != nil {
					return err
				}
			}

			target := reflect.New(typ)
			for name, field := range fields {
				value, exists := asMap[name]
				if !exists {
					continue
				}
				function := obj.Fields[name]
				funcTyp := reflect.TypeOf(function)
				sourceTyp := funcTyp.In(1)
				source := reflect.New(sourceTyp).Elem()

				if err := field.parser.FromJSON(value, source); err != nil {
					return fmt.Errorf("%s : %s", name, err)
				}

				output := reflect.ValueOf(function).Call([]reflect.Value{target, source})
				if len(output) > 0 {
					o := output[0].Interface()
					if o != nil {
						return output[0].Interface().(error)
					}
				}

			}

			dest.Set(target.Elem())

			return nil
		},
		Type: typ,
	}, argType, nil
}

func (sb *schemaBuilder) getInputFieldParser(typ reflect.Type) (*argParser, graphql.Type, error) {
	if sb.enumMappings[typ] != nil {
		parser, argType := sb.getEnumArgParser(typ)
		return parser, argType, nil
	}

	if parser, argType, ok := getScalarArgParser(typ); ok {
		return parser, argType, nil
	}

	switch typ.Kind() {
	case reflect.Struct:
		parser, argType, err := sb.generateObjectParser(typ)
		if err != nil {
			return nil, nil, err
		}
		if argType.(*graphql.InputObject).Name == "" {
			return nil, nil, fmt.Errorf("bad type %s: should have a name", typ)
		}
		return parser, argType, nil
	case reflect.Slice:
		return sb.generateSliceParser(typ)
	case reflect.Ptr:
		parser, argType, err := sb.getInputFieldParser(typ.Elem())
		if err != nil {
			return nil, nil, err
		}
		return wrapPtrParser(parser), argType, nil
	default:
		return nil, nil, fmt.Errorf("bad arg type %s: should be struct, scalar, pointer, or a slice", typ)
	}
}

// generateSliceParser generates the parser for a slice input by generating the parser for underlying object and using it to fill the values in list
func (sb *schemaBuilder) generateSliceParser(typ reflect.Type) (*argParser, graphql.Type, error) {
	inner, argType, err := sb.generateObjectParser(typ.Elem())
	if err != nil {
		return nil, nil, err
	}

	return &argParser{
		FromJSON: func(value interface{}, dest reflect.Value) error {
			asSlice, ok := value.([]interface{})
			if !ok {
				return errors.New("not a list")
			}

			sourceTyp := typ.Elem()
			sourceSlice := reflect.MakeSlice(typ, len(asSlice), len(asSlice))

			for i, value := range asSlice {
				source := reflect.New(sourceTyp).Elem()
				if err := inner.FromJSON(value, source); err != nil {
					return err
				}
				sourceSlice.Index(i).Set(source)
			}

			dest.Set(sourceSlice)

			return nil
		},
		Type: typ,
	}, &graphql.List{Type: argType}, nil
}

// hasOneOfMarkerEmbedded determines if a struct has an embedded schemabuilder.OneOfInput
// (for @oneOf directive support on INPUT_OBJECT per spec). Mirrors
// hasUnionMarkerEmbedded in output.go (anon embed only; used for input symmetry
// w/ unions/interfaces from README.md and protoc oneof hack).
func hasOneOfMarkerEmbedded(typ reflect.Type) bool {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.Type == oneOfInputType {
			return true
		}
	}
	return false
}
