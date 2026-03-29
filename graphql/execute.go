package graphql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"go.appointy.com/jaal/jerrors"
)

type Executor struct {
	iterate bool
}

type computationOutput struct {
	Function  interface{}
	Field     *Field
	Selection *Selection
}

var ErrNoUpdate = errors.New("no update")

func (e *Executor) Execute(ctx context.Context, typ Type, source interface{}, query *Query) (interface{}, error) {
	response, err := e.execute(ctx, typ, source, query.SelectionSet)
	if err != nil {
		return nil, err
	}

	for e.iterate {
		e.iterate = false

		if err := e.lateExecution(ctx, response); err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (e *Executor) execute(ctx context.Context, typ Type, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch typ := typ.(type) {
	case *Scalar:
		if typ.Unwrapper != nil {
			return typ.Unwrapper(source)
		}
		return unwrap(source), nil
	case *Enum:
		val := unwrap(source)
		if mapVal, ok := typ.ReverseMap[val]; ok {
			return mapVal, nil
		}
		return nil, errors.New("enum is not valid")
	case *Union:
		return e.executeUnion(ctx, typ, source, selectionSet)
	case *Interface:
		return e.executeInterface(ctx, typ, source, selectionSet)
	case *Object:
		return e.executeObject(ctx, typ, source, selectionSet)
	case *List:
		return e.executeList(ctx, typ, source, selectionSet)
	case *NonNull:
		return e.execute(ctx, typ.Type, source, selectionSet)
	default:
		panic(typ)
	}
}

// unwrap will return the value associated with a pointer type, or nil if the pointer is nil
func unwrap(v interface{}) interface{} {
	i := reflect.ValueOf(v)
	for i.Kind() == reflect.Ptr && !i.IsNil() {
		i = i.Elem()
	}
	if i.Kind() == reflect.Invalid {
		return nil
	}
	return i.Interface()
}

func (e *Executor) executeUnion(ctx context.Context, typ *Union, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil, nil
	}

	fields := make(map[string]interface{})
	for _, selection := range selectionSet.Selections {
		if selection.Name == "__typename" {
			fields[selection.Alias] = typ.Name
			continue
		}
	}

	// For every inline fragment spread, check if the current concrete type matches and execute that object.
	var possibleTypes []string
	for typString, graphqlTyp := range typ.Types {
		inner := reflect.ValueOf(source)
		if inner.Kind() == reflect.Ptr && inner.Elem().Kind() == reflect.Struct {
			inner = inner.Elem()
		}

		inner = inner.FieldByName(typString)
		if inner.IsNil() {
			continue
		}
		possibleTypes = append(possibleTypes, graphqlTyp.String())

		for _, fragment := range selectionSet.Fragments {
			if fragment.Fragment.On != typString {
				continue
			}
			resolved, err := e.executeObject(ctx, graphqlTyp, inner.Interface(), fragment.Fragment.SelectionSet)
			if err != nil {
				if err == ErrNoUpdate {
					return nil, err
				}
				return nil, jerrors.NestErrorPaths(err, typString)
			}

			for k, v := range resolved.(map[string]interface{}) {
				fields[k] = v
			}
		}
	}

	if len(possibleTypes) > 1 {
		return nil, fmt.Errorf("union type field should only return one value, but received: %s", strings.Join(possibleTypes, " "))
	}
	return fields, nil
}

// executeObject executes an object query
func (e *Executor) executeObject(ctx context.Context, typ *Object, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil, nil
	}

	selections, err := Flatten(selectionSet)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]interface{})

	// for every selection, resolve the value and store it in the output object
	for _, selection := range selections {
		if ok, err := shouldIncludeNode(selection.Directives); err != nil {
			if err == ErrNoUpdate {
				return nil, err
			}
			return nil, jerrors.NestErrorPaths(err, selection.Alias)
		} else if !ok {
			continue
		}

		if selection.Name == "__typename" {
			fields[selection.Alias] = typ.Name
			continue
		}

		field := typ.Fields[selection.Name]
		resolved, err := e.resolveAndExecute(ctx, field, source, selection)
		if err != nil {
			if err == ErrNoUpdate {
				return nil, err
			}
			return nil, jerrors.NestErrorPaths(err, selection.Alias)
		}
		fields[selection.Alias] = resolved
	}

	return fields, nil
}

func (e *Executor) resolveAndExecute(ctx context.Context, field *Field, source interface{}, selection *Selection) (interface{}, error) {
	// Execute pre-resolver schema directives.
	for _, d := range field.SchemaDirectives {
		if d.ExecutionOrder != 0 { // 0 = PreResolver
			continue
		}
		if d.Handler == nil {
			continue
		}
		if err := d.Handler(ctx, d.Args); err != nil {
			if d.OnFail == 1 { // 1 = SkipOnFail
				return nil, nil
			}
			return nil, err
		}
	}

	value, err := safeExecuteResolver(ctx, field, source, selection.Args, selection.SelectionSet)
	if err != nil {
		return nil, err
	}

	// Execute post-resolver schema directives.
	for _, d := range field.SchemaDirectives {
		if d.ExecutionOrder != 1 { // 1 = PostResolver
			continue
		}
		if d.Handler == nil {
			continue
		}
		if err := d.Handler(ctx, d.Args); err != nil {
			if d.OnFail == 1 { // 1 = SkipOnFail
				return nil, nil
			}
			return nil, err
		}
	}

	// If a field returns function, then do not execute the function at the moment
	if field.LazyExecution {
		e.iterate = true
		return &computationOutput{
			Function:  value,
			Field:     field,
			Selection: selection,
		}, nil
	}

	return e.execute(ctx, field.Type, value, selection.SelectionSet)
}

func safeExecuteResolver(ctx context.Context, field *Field, source, args interface{}, selectionSet *SelectionSet) (result interface{}, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			result, err = nil, fmt.Errorf("graphql: panic: %v\n%s", panicErr, buf)
		}
	}()
	return field.Resolve(ctx, source, args, selectionSet)
}

var emptyList = []interface{}{}

// unwrapToObject peels through NonNull wrappers and returns the underlying
// *Object if there is one, or nil otherwise.
func unwrapToObject(typ Type) *Object {
	switch t := typ.(type) {
	case *Object:
		return t
	case *NonNull:
		return unwrapToObject(t.Type)
	default:
		return nil
	}
}

// objectHasBatchFields returns true if any field on obj has a BatchResolve set.
func objectHasBatchFields(obj *Object) bool {
	for _, f := range obj.Fields {
		if f.BatchResolve != nil {
			return true
		}
	}
	return false
}

// executeList executes a set query
func (e *Executor) executeList(ctx context.Context, typ *List, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	if reflect.ValueOf(source).IsNil() {
		return emptyList, nil
	}

	// iterate over arbitrary slice types using reflect
	slice := reflect.ValueOf(source)
	n := slice.Len()

	// If the inner type is an Object with batch fields, use the optimised
	// batched execution path that calls BatchResolve once per batch field
	// for all N items instead of N times.
	if obj := unwrapToObject(typ.Type); obj != nil && selectionSet != nil && n > 0 {
		if objectHasBatchFields(obj) {
			return e.executeBatchedList(ctx, typ.Type, obj, slice, selectionSet)
		}
	}

	items := make([]interface{}, n)

	// resolve every element in the slice
	for i := 0; i < n; i++ {
		value := slice.Index(i)
		resolved, err := e.execute(ctx, typ.Type, value.Interface(), selectionSet)
		if err != nil {
			if err == ErrNoUpdate {
				return nil, err
			}
			return nil, jerrors.NestErrorPaths(err, fmt.Sprint(i))
		}
		items[i] = resolved
	}

	return items, nil
}

// executeBatchedList handles list execution when the inner object type has
// batch-resolved fields.  It:
//  1. Flattens the selection set.
//  2. Calls each batch field's BatchResolve once with ALL source items.
//  3. Resolves non-batch fields per item as usual.
//  4. Stitches everything into the response.
func (e *Executor) executeBatchedList(ctx context.Context, innerType Type, obj *Object, slice reflect.Value, selectionSet *SelectionSet) (interface{}, error) {
	n := slice.Len()

	selections, err := Flatten(selectionSet)
	if err != nil {
		return nil, err
	}

	// Collect all source values.
	sources := make([]interface{}, n)
	for i := 0; i < n; i++ {
		sources[i] = slice.Index(i).Interface()
	}

	// Pre-resolve every batch field once for all items.
	// Key = selection Alias (unique per flattened selection).
	batchResults := make(map[string][]interface{})

	for _, sel := range selections {
		if sel.Name == "__typename" {
			continue
		}

		if ok, err := shouldIncludeNode(sel.Directives); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		field := obj.Fields[sel.Name]
		if field == nil || field.BatchResolve == nil {
			continue
		}

		// Parse args once.
		if !sel.parsed {
			parsedArgs, err := field.ParseArguments(sel.Args)
			if err != nil {
				return nil, jerrors.NestErrorPaths(err, sel.Alias)
			}
			sel.Args = parsedArgs
			sel.parsed = true
		}

		// Execute pre-resolver schema directives.
		skipped := false
		for _, d := range field.SchemaDirectives {
			if d.ExecutionOrder != 0 { // 0 = PreResolver
				continue
			}
			if d.Handler == nil {
				continue
			}
			if err := d.Handler(ctx, d.Args); err != nil {
				if d.OnFail == 1 { // SkipOnFail → null for every item
					nilResults := make([]interface{}, n)
					batchResults[sel.Alias] = nilResults
					skipped = true
					break
				}
				return nil, err
			}
		}
		if skipped {
			continue
		}

		// Call the batch resolver.
		results, err := field.BatchResolve(ctx, sources, sel.Args, sel.SelectionSet)
		if err != nil {
			return nil, jerrors.NestErrorPaths(err, sel.Alias)
		}
		if len(results) != n {
			return nil, fmt.Errorf("batch resolver for field %q returned %d results, expected %d", sel.Name, len(results), n)
		}

		// Execute post-resolver schema directives.
		postSkipped := false
		for _, d := range field.SchemaDirectives {
			if d.ExecutionOrder != 1 { // 1 = PostResolver
				continue
			}
			if d.Handler == nil {
				continue
			}
			if err := d.Handler(ctx, d.Args); err != nil {
				if d.OnFail == 1 { // SkipOnFail
					nilResults := make([]interface{}, n)
					batchResults[sel.Alias] = nilResults
					postSkipped = true
					break
				}
				return nil, err
			}
		}
		if postSkipped {
			continue
		}

		batchResults[sel.Alias] = results
	}

	// Now resolve each item, using pre-resolved batch results where available.
	items := make([]interface{}, n)
	for i := 0; i < n; i++ {
		source := sources[i]
		value := reflect.ValueOf(source)

		// Respect NonNull: if the inner type is nullable and source is nil ptr → null.
		if value.Kind() == reflect.Ptr && value.IsNil() {
			items[i] = nil
			continue
		}

		fields := make(map[string]interface{})

		for _, sel := range selections {
			if ok, err := shouldIncludeNode(sel.Directives); err != nil {
				return nil, jerrors.NestErrorPaths(jerrors.NestErrorPaths(err, sel.Alias), fmt.Sprint(i))
			} else if !ok {
				continue
			}

			if sel.Name == "__typename" {
				fields[sel.Alias] = obj.Name
				continue
			}

			field := obj.Fields[sel.Name]

			if batchRes, ok := batchResults[sel.Alias]; ok {
				// Use the pre-resolved batch result — just execute sub-selections.
				resolved, err := e.execute(ctx, field.Type, batchRes[i], sel.SelectionSet)
				if err != nil {
					if err == ErrNoUpdate {
						return nil, err
					}
					return nil, jerrors.NestErrorPaths(jerrors.NestErrorPaths(err, sel.Alias), fmt.Sprint(i))
				}
				fields[sel.Alias] = resolved
			} else {
				// Non-batch field: resolve per item normally.
				resolved, err := e.resolveAndExecute(ctx, field, source, sel)
				if err != nil {
					if err == ErrNoUpdate {
						return nil, err
					}
					return nil, jerrors.NestErrorPaths(jerrors.NestErrorPaths(err, sel.Alias), fmt.Sprint(i))
				}
				fields[sel.Alias] = resolved
			}
		}

		items[i] = fields
	}

	return items, nil
}

// executeInterface resolves an interface query
func (e *Executor) executeInterface(ctx context.Context, typ *Interface, source interface{}, selectionSet *SelectionSet) (interface{}, error) {
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return nil, nil
	}
	fields := make(map[string]interface{})
	var possibleTypes []string
	for typString, graphqlTyp := range typ.Types {
		inner := reflect.ValueOf(source)
		if inner.Kind() == reflect.Ptr && inner.Elem().Kind() == reflect.Struct {
			inner = inner.Elem()
		}
		inner = inner.FieldByName(typString)
		if inner.IsNil() {
			continue
		}
		possibleTypes = append(possibleTypes, graphqlTyp.String())

		// modifiedSelectionSet selection set contains fragments on typString
		modifiedSelectionSet := &SelectionSet{
			Selections: selectionSet.Selections,
			Fragments:  []*FragmentSpread{},
		}
		for _, f := range selectionSet.Fragments {
			if f.Fragment.On == typString {
				modifiedSelectionSet.Fragments = append(modifiedSelectionSet.Fragments, f)
			}
		}

		selections, err := Flatten(modifiedSelectionSet)
		if err != nil {
			return nil, err
		}
		// for every selection, resolve the value and store it in the output object
		for _, selection := range selections {
			if selection.Name == "__typename" {
				fields[selection.Alias] = graphqlTyp.Name
				continue
			}
			field, ok := graphqlTyp.Fields[selection.Name]
			if !ok {
				continue
			}
			value := reflect.ValueOf(source).Elem()
			value = value.FieldByName(typString)
			resolved, err := e.resolveAndExecute(ctx, field, value.Interface(), selection)
			if err != nil {
				if err == ErrNoUpdate {
					return nil, err
				}
				return nil, jerrors.NestErrorPaths(err, selection.Alias)
			}
			fields[selection.Alias] = resolved
		}
	}

	return fields, nil
}

func findDirectiveWithName(directives []*Directive, name string) *Directive {
	for _, directive := range directives {
		if directive.Name == name {
			return directive
		}
	}
	return nil
}

func shouldIncludeNode(directives []*Directive) (bool, error) {
	parseIf := func(d *Directive) (bool, error) {
		args := d.Args.(map[string]interface{})
		if args["if"] == nil {
			return false, fmt.Errorf("required argument not provided: if")
		}

		if _, ok := args["if"].(bool); !ok {
			return false, fmt.Errorf("expected type Boolean, found %v", args["if"])
		}

		return args["if"].(bool), nil
	}

	skipDirective := findDirectiveWithName(directives, "skip")
	if skipDirective != nil {
		b, err := parseIf(skipDirective)
		if err != nil {
			return false, err
		}
		if b {
			return false, nil
		}
	}

	includeDirective := findDirectiveWithName(directives, "include")
	if includeDirective != nil {
		return parseIf(includeDirective)
	}

	return true, nil
}

func (e *Executor) lateExecution(ctx context.Context, response interface{}) error {
	list, ok := response.([]interface{})
	if ok {
		for _, element := range list {
			if err := e.lateExecution(ctx, element); err != nil {
				return err
			}
		}
	}

	data, ok := response.(map[string]interface{})
	if !ok {
		return nil
	}

	for key, value := range data {
		output, ok := value.(*computationOutput)
		if !ok {
			if err := e.lateExecution(ctx, value); err != nil {
				return err
			}
			continue
		}

		resolved, err := e.resolveAndExecuteFunction(ctx, output)
		if err != nil {
			return err
		}

		data[key] = resolved
	}

	return nil
}

func (e *Executor) resolveAndExecuteFunction(ctx context.Context, output *computationOutput) (interface{}, error) {
	value, err := output.Field.LazyResolver(ctx, output.Function)
	if err != nil {
		return nil, err
	}

	return e.execute(ctx, output.Field.Type, value, output.Selection.SelectionSet)
}
