package graphql

import (
	"context"
	"fmt"
)

// ValidateQuery checks that the given selectionSet matches the schema typ, and parses the args in selectionSet
func ValidateQuery(ctx context.Context, typ Type, selectionSet *SelectionSet) error {
	switch typ := typ.(type) {
	case *Scalar:
		if selectionSet != nil {
			return fmt.Errorf("scalar field must have no selections")
		}
		return nil
	case *Enum:
		if selectionSet != nil {
			return fmt.Errorf("enum field must have no selections")
		}
		return nil
	case *Union:
		if selectionSet == nil {
			return fmt.Errorf("object field must have selections")
		}

		for _, fragment := range selectionSet.Fragments {
			for typString, graphqlTyp := range typ.Types {
				if fragment.Fragment.On != typString {
					continue
				}
				if err := ValidateQuery(ctx, graphqlTyp, fragment.Fragment.SelectionSet); err != nil {
					return err
				}
			}
		}
		for _, selection := range selectionSet.Selections {
			if selection.Name == "__typename" {
				if !isNilArgs(selection.Args) {
					return fmt.Errorf(`error parsing args for "__typename": no args expected`)
				}
				if selection.SelectionSet != nil {
					return fmt.Errorf(`scalar field "__typename" must have no selection`)
				}
				for _, fragment := range selectionSet.Fragments {
					fragment.Fragment.SelectionSet.Selections = append(fragment.Fragment.SelectionSet.Selections, selection)
				}
				continue
			}
			return fmt.Errorf(`unknown field "%s"`, selection.Name)
		}
		return nil

	case *Interface:
		if selectionSet == nil {
			return fmt.Errorf("object field must have selections")
		}
		for _, fragment := range selectionSet.Fragments {
			for typString, graphqlTyp := range typ.Types {
				if fragment.Fragment.On != typString {
					continue
				}
				if err := ValidateQuery(ctx, graphqlTyp, fragment.Fragment.SelectionSet); err != nil {
					return err
				}
			}
		}
		for _, selection := range selectionSet.Selections {
			if selection.Name == "__typename" {
				if !isNilArgs(selection.Args) {
					return fmt.Errorf(`error parsing args for "__typename": no args expected`)
				}
				if selection.SelectionSet != nil {
					return fmt.Errorf(`scalar field "__typename" must have no selection`)
				}
				continue
			}
			field, ok := typ.Fields[selection.Name]
			if !ok {
				return fmt.Errorf(`unknown field "%s"`, selection.Name)
			}

			if !selection.parsed {
				parsed, err := field.ParseArguments(selection.Args)
				if err != nil {
					return fmt.Errorf(`error parsing args for "%s": %s`, selection.Name, err)
				}
				selection.Args = parsed
				selection.parsed = true
			}
			if err := ValidateQuery(ctx, field.Type, selection.SelectionSet); err != nil {
				return err
			}
		}

		return nil
	case *Object:
		if selectionSet == nil {
			return fmt.Errorf("object field must have selections")
		}
		for _, selection := range selectionSet.Selections {
			if selection.Name == "__typename" {
				if !isNilArgs(selection.Args) {
					return fmt.Errorf(`error parsing args for "__typename": no args expected`)
				}
				if selection.SelectionSet != nil {
					return fmt.Errorf(`scalar field "__typename" must have no selection`)
				}
				continue
			}

			field, ok := typ.Fields[selection.Name]
			if !ok {
				return fmt.Errorf(`unknown field "%s"`, selection.Name)
			}

			// Only parse args once for a given selection.
			if !selection.parsed {
				parsed, err := field.ParseArguments(selection.Args)
				if err != nil {
					return fmt.Errorf(`error parsing args for "%s": %s`, selection.Name, err)
				}
				selection.Args = parsed
				selection.parsed = true
			}

			if err := ValidateQuery(ctx, field.Type, selection.SelectionSet); err != nil {
				return err
			}
		}
		for _, fragment := range selectionSet.Fragments {
			if err := ValidateQuery(ctx, typ, fragment.Fragment.SelectionSet); err != nil {
				return err
			}
		}
		return nil

	case *List:
		return ValidateQuery(ctx, typ.Type, selectionSet)

	case *NonNull:
		return ValidateQuery(ctx, typ.Type, selectionSet)

	default:
		panic("unknown type kind")
	}
}

func isNilArgs(args interface{}) bool {
	m, ok := args.(map[string]interface{})
	return args == nil || (ok && len(m) == 0)
}

// ValidateDirectiveLocations checks that directives are used in valid locations.
// It validates that each directive in the selection set is allowed at the FIELD location.
func ValidateDirectiveLocations(selectionSet *SelectionSet, definitions map[string]*DirectiveDefinition) []error {
	var errors []error

	if selectionSet == nil {
		return errors
	}

	// Validate directives on selections (FIELD location)
	for _, selection := range selectionSet.Selections {
		for _, directive := range selection.Directives {
			def, ok := definitions[directive.Name]
			if !ok {
				// Unknown directive - could be a built-in like @skip, @include
				continue
			}

			if !def.HasLocation(LocationField) {
				errors = append(errors, fmt.Errorf(
					"directive @%s cannot be used on FIELD (allowed: %v)",
					directive.Name, def.Locations,
				))
			}
		}

		// Recurse into nested selections
		if selection.SelectionSet != nil {
			errors = append(errors, ValidateDirectiveLocations(selection.SelectionSet, definitions)...)
		}
	}

	// Validate directives on fragment spreads (FRAGMENT_SPREAD location)
	for _, fragment := range selectionSet.Fragments {
		for _, directive := range fragment.Directives {
			def, ok := definitions[directive.Name]
			if !ok {
				continue
			}

			if !def.HasLocation(LocationFragmentSpread) {
				errors = append(errors, fmt.Errorf(
					"directive @%s cannot be used on FRAGMENT_SPREAD (allowed: %v)",
					directive.Name, def.Locations,
				))
			}
		}

		// Recurse into fragment selections
		if fragment.Fragment.SelectionSet != nil {
			errors = append(errors, ValidateDirectiveLocations(fragment.Fragment.SelectionSet, definitions)...)
		}
	}

	return errors
}

// ValidateRepeatableDirectives checks that non-repeatable directives are not duplicated.
// Per GraphQL spec, non-repeatable directives can only appear once per location.
func ValidateRepeatableDirectives(selectionSet *SelectionSet, definitions map[string]*DirectiveDefinition) []error {
	var errors []error

	if selectionSet == nil {
		return errors
	}

	// Check selections
	for _, selection := range selectionSet.Selections {
		seen := make(map[string]bool)
		for _, directive := range selection.Directives {
			def, ok := definitions[directive.Name]
			if !ok {
				continue
			}

			if seen[directive.Name] && !def.IsRepeatable {
				errors = append(errors, fmt.Errorf(
					"directive @%s is not repeatable but was applied multiple times",
					directive.Name,
				))
			}
			seen[directive.Name] = true
		}

		// Recurse
		if selection.SelectionSet != nil {
			errors = append(errors, ValidateRepeatableDirectives(selection.SelectionSet, definitions)...)
		}
	}

	// Check fragment spreads
	for _, fragment := range selectionSet.Fragments {
		seen := make(map[string]bool)
		for _, directive := range fragment.Directives {
			def, ok := definitions[directive.Name]
			if !ok {
				continue
			}

			if seen[directive.Name] && !def.IsRepeatable {
				errors = append(errors, fmt.Errorf(
					"directive @%s is not repeatable but was applied multiple times",
					directive.Name,
				))
			}
			seen[directive.Name] = true
		}

		if fragment.Fragment.SelectionSet != nil {
			errors = append(errors, ValidateRepeatableDirectives(fragment.Fragment.SelectionSet, definitions)...)
		}
	}

	return errors
}

// ValidateFieldDirectives checks that directives applied to fields at schema build time
// are valid for FIELD_DEFINITION location.
func ValidateFieldDirectives(field *Field, definitions map[string]*DirectiveDefinition) []error {
	var errors []error

	for _, directive := range field.Directives {
		def, ok := definitions[directive.Name]
		if !ok {
			// Unknown directive - could be a built-in
			continue
		}

		if !def.HasLocation(LocationFieldDefinition) {
			errors = append(errors, fmt.Errorf(
				"directive @%s cannot be used on FIELD_DEFINITION (allowed: %v)",
				directive.Name, def.Locations,
			))
		}
	}

	return errors
}

// ValidateSchemaDirectives validates all directives in a schema.
// This should be called during schema building to ensure directive validity.
func ValidateSchemaDirectives(schema *Schema, definitions map[string]*DirectiveDefinition) []error {
	var errors []error

	// Validate directives on query type
	if obj, ok := schema.Query.(*Object); ok {
		errors = append(errors, validateObjectDirectives(obj, definitions)...)
	}

	// Validate directives on mutation type
	if obj, ok := schema.Mutation.(*Object); ok {
		errors = append(errors, validateObjectDirectives(obj, definitions)...)
	}

	// Validate directives on subscription type
	if obj, ok := schema.Subscription.(*Object); ok {
		errors = append(errors, validateObjectDirectives(obj, definitions)...)
	}

	return errors
}

func validateObjectDirectives(obj *Object, definitions map[string]*DirectiveDefinition) []error {
	var errors []error

	for _, field := range obj.Fields {
		errors = append(errors, ValidateFieldDirectives(field, definitions)...)
	}

	return errors
}
