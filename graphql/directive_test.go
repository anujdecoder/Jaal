package graphql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type roleKeyType struct{}

var roleKey = roleKeyType{}

// buildAndExec builds a schema, validates & executes the query, and returns the
// result plus any error.
func buildAndExec(t *testing.T, sb *schemabuilder.Schema, query string) (interface{}, error) {
	t.Helper()
	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(query, nil)
	require.NoError(t, err)

	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	return e.Execute(context.Background(), schema.Query, nil, q)
}

// buildAndExecCtx is like buildAndExec but accepts a custom context.
func buildAndExecCtx(t *testing.T, sb *schemabuilder.Schema, ctx context.Context, query string) (interface{}, error) {
	t.Helper()
	schema := sb.MustBuild()

	q, err := graphql.Parse(query, nil)
	require.NoError(t, err)

	require.NoError(t, graphql.ValidateQuery(ctx, schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	return e.Execute(ctx, schema.Query, nil, q)
}

// ---------------------------------------------------------------------------
// Test: PreResolver directive — ErrorOnFail (default)
// ---------------------------------------------------------------------------

func TestDirective_PreResolver_ErrorOnFail(t *testing.T) {
	sb := schemabuilder.NewSchema()

	// Register a directive that checks a "role" in context.
	sb.Directive("hasRole", &schemabuilder.DirectiveDefinition{
		Description: "Checks the caller role",
		Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "role", TypeName: "String", Description: "Required role"},
		},
		ExecutionOrder: schemabuilder.PreResolver, // default
		OnFail:         schemabuilder.ErrorOnFail, // default
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			required, _ := args["role"].(string)
			actual, _ := ctx.Value(roleKey).(string)
			if actual != required {
				return errors.New("access denied")
			}
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("public", func() string { return "hello" })
	query.FieldFunc("admin", func() string { return "secret" },
		schemabuilder.WithFieldDirective("hasRole", map[string]interface{}{"role": "ADMIN"}),
	)

	// Public field always works.
	val, err := buildAndExecCtx(t, sb, context.Background(), `{ public }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"public": "hello"}, val)

	// Admin without role → error.
	_, err = buildAndExecCtx(t, sb, context.Background(), `{ admin }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")

	// Admin with correct role → success.
	ctx := context.WithValue(context.Background(), roleKey, "ADMIN")
	val, err = buildAndExecCtx(t, sb, ctx, `{ admin }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"admin": "secret"}, val)
}

// ---------------------------------------------------------------------------
// Test: PreResolver directive — SkipOnFail
// ---------------------------------------------------------------------------

func TestDirective_PreResolver_SkipOnFail(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("featureFlag", &schemabuilder.DirectiveDefinition{
		Description:    "Guards a field behind a feature flag",
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.SkipOnFail, // silently returns nil
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("feature disabled")
		},
	})

	query := sb.Query()
	query.FieldFunc("beta", func() *string {
		s := "beta-data"
		return &s
	}, schemabuilder.WithFieldDirective("featureFlag"))

	// The directive fails but OnFail=SkipOnFail → null, no error.
	val, err := buildAndExecCtx(t, sb, context.Background(), `{ beta }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Nil(t, m["beta"])
}

// ---------------------------------------------------------------------------
// Test: PostResolver directive — ErrorOnFail
// ---------------------------------------------------------------------------

func TestDirective_PostResolver_ErrorOnFail(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("audit", &schemabuilder.DirectiveDefinition{
		Description:    "Runs an audit check after resolution",
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			// Simulates a post-resolve audit that fails.
			return errors.New("audit failed")
		},
	})

	query := sb.Query()
	query.FieldFunc("data", func() string { return "value" },
		schemabuilder.WithFieldDirective("audit"),
	)

	_, err := buildAndExecCtx(t, sb, context.Background(), `{ data }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit failed")
}

// ---------------------------------------------------------------------------
// Test: PostResolver directive — SkipOnFail
// ---------------------------------------------------------------------------

func TestDirective_PostResolver_SkipOnFail(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("mask", &schemabuilder.DirectiveDefinition{
		Description:    "Masks the field value on failure",
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.SkipOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("mask active")
		},
	})

	query := sb.Query()
	query.FieldFunc("secret", func() *string {
		s := "classified"
		return &s
	}, schemabuilder.WithFieldDirective("mask"))

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ secret }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Nil(t, m["secret"])
}

// ---------------------------------------------------------------------------
// Test: PostResolver directive — passes when handler returns nil
// ---------------------------------------------------------------------------

func TestDirective_PostResolver_Pass(t *testing.T) {
	sb := schemabuilder.NewSchema()

	called := false
	sb.Directive("log", &schemabuilder.DirectiveDefinition{
		Description:    "Logs after resolution",
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			called = true
			return nil // pass
		},
	})

	query := sb.Query()
	query.FieldFunc("info", func() string { return "ok" },
		schemabuilder.WithFieldDirective("log"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ info }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"info": "ok"}, val)
	assert.True(t, called, "PostResolver handler should have been called")
}

// ---------------------------------------------------------------------------
// Test: Type-level directive propagates to all fields
// ---------------------------------------------------------------------------

func TestDirective_TypeLevel(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("auth", &schemabuilder.DirectiveDefinition{
		Description:    "Requires authentication",
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationObject},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			if ctx.Value(roleKey) == nil {
				return errors.New("unauthenticated")
			}
			return nil
		},
	})

	type SecretInfo struct{}

	// Apply @auth at the object level — every field should inherit it.
	obj := sb.Object("SecretInfo", SecretInfo{}, schemabuilder.WithDirective("auth"))
	obj.FieldFunc("code", func(_ SecretInfo) string { return "42" })
	obj.FieldFunc("name", func(_ SecretInfo) string { return "x" })

	query := sb.Query()
	query.FieldFunc("secret", func() *SecretInfo { return &SecretInfo{} })

	// Without auth context → error.
	_, err := buildAndExecCtx(t, sb, context.Background(), `{ secret { code name } }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthenticated")

	// With auth context → success.
	ctx := context.WithValue(context.Background(), roleKey, "USER")
	val, err := buildAndExecCtx(t, sb, ctx, `{ secret { code name } }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	inner := m["secret"].(map[string]interface{})
	assert.Equal(t, "42", inner["code"])
	assert.Equal(t, "x", inner["name"])
}

// ---------------------------------------------------------------------------
// Test: Multiple directives on the same field (order matters)
// ---------------------------------------------------------------------------

func TestDirective_MultipleOnField(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var order []string

	sb.Directive("first", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "first")
			return nil
		},
	})
	sb.Directive("second", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "second")
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("multi", func() string { return "ok" },
		schemabuilder.WithFieldDirective("first"),
		schemabuilder.WithFieldDirective("second"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ multi }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"multi": "ok"}, val)
	assert.Equal(t, []string{"first", "second"}, order)
}

// ---------------------------------------------------------------------------
// Test: Metadata-only directive (nil handler) — no runtime effect
// ---------------------------------------------------------------------------

func TestDirective_MetadataOnly(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("cacheControl", &schemabuilder.DirectiveDefinition{
		Description: "Cache control hints",
		Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "maxAge", TypeName: "Int"},
		},
		Handler: nil, // metadata only
	})

	query := sb.Query()
	query.FieldFunc("cached", func() string { return "data" },
		schemabuilder.WithFieldDirective("cacheControl", map[string]interface{}{"maxAge": 300}),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ cached }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"cached": "data"}, val)
}

// ---------------------------------------------------------------------------
// Test: Introspection exposes custom directives
// ---------------------------------------------------------------------------

func TestDirective_Introspection(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("hasRole", &schemabuilder.DirectiveDefinition{
		Description: "Role-based access control",
		Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "role", TypeName: "String", Description: "Required role"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) error { return nil },
	})

	query := sb.Query()
	query.FieldFunc("hello", func() string { return "world" })

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__schema {
			directives {
				name
				description
				locations
				args { name description }
			}
		}
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	root := val.(map[string]interface{})
	schemaData := root["__schema"].(map[string]interface{})
	directives := schemaData["directives"].([]interface{})

	// Find our custom directive.
	found := false
	for _, d := range directives {
		dm := d.(map[string]interface{})
		if dm["name"] == "hasRole" {
			found = true
			assert.Equal(t, "Role-based access control", dm["description"])
			locs := dm["locations"].([]interface{})
			assert.Contains(t, locs, "FIELD_DEFINITION")
			args := dm["args"].([]interface{})
			require.Len(t, args, 1)
			arg := args[0].(map[string]interface{})
			assert.Equal(t, "role", arg["name"])
			assert.Equal(t, "Required role", arg["description"])
			break
		}
	}
	assert.True(t, found, "@hasRole should appear in __schema.directives")
}

// ---------------------------------------------------------------------------
// Test: Unregistered directive panics at Build()
// ---------------------------------------------------------------------------

func TestDirective_UnregisteredPanics(t *testing.T) {
	sb := schemabuilder.NewSchema()
	// No directive registered.
	query := sb.Query()
	query.FieldFunc("bad", func() string { return "" },
		schemabuilder.WithFieldDirective("nonexistent"),
	)

	assert.Panics(t, func() {
		sb.MustBuild()
	})
}

// ===========================================================================
// Comprehensive directive test cases
// ===========================================================================

// ---------------------------------------------------------------------------
// Test: Directive handler receives correct static args
// ---------------------------------------------------------------------------

func TestDirective_HandlerReceivesArgs(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var received map[string]interface{}
	sb.Directive("checkArgs", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			received = args
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("hello", func() string { return "world" },
		schemabuilder.WithFieldDirective("checkArgs", map[string]interface{}{
			"role":  "ADMIN",
			"level": 5,
		}),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ hello }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"hello": "world"}, val)
	assert.Equal(t, "ADMIN", received["role"])
	assert.Equal(t, 5, received["level"])
}

// ---------------------------------------------------------------------------
// Test: Directive with no args (nil Args map)
// ---------------------------------------------------------------------------

func TestDirective_NoArgs(t *testing.T) {
	sb := schemabuilder.NewSchema()

	called := false
	sb.Directive("noArgs", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			called = true
			// args should be nil since no args were provided at registration.
			assert.Nil(t, args)
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("ping", func() string { return "pong" },
		schemabuilder.WithFieldDirective("noArgs"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ ping }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"ping": "pong"}, val)
	assert.True(t, called)
}

// ---------------------------------------------------------------------------
// Test: Directive with empty args map
// ---------------------------------------------------------------------------

func TestDirective_EmptyArgs(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var received map[string]interface{}
	sb.Directive("emptyArgs", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			received = args
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("data", func() string { return "ok" },
		schemabuilder.WithFieldDirective("emptyArgs", map[string]interface{}{}),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ data }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"data": "ok"}, val)
	// Empty map is still a map, not nil.
	assert.NotNil(t, received)
	assert.Len(t, received, 0)
}

// ---------------------------------------------------------------------------
// Test: Context propagation — handler can read values from context
// ---------------------------------------------------------------------------

type userIDKeyType struct{}

func TestDirective_ContextPropagation(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var ctxUserID string
	sb.Directive("captureCtx", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			if uid, ok := ctx.Value(userIDKeyType{}).(string); ok {
				ctxUserID = uid
			}
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("me", func() string { return "self" },
		schemabuilder.WithFieldDirective("captureCtx"),
	)

	ctx := context.WithValue(context.Background(), userIDKeyType{}, "user-42")
	val, err := buildAndExecCtx(t, sb, ctx, `{ me }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"me": "self"}, val)
	assert.Equal(t, "user-42", ctxUserID)
}

// ---------------------------------------------------------------------------
// Test: Mixed PreResolver + PostResolver on the same field
// ---------------------------------------------------------------------------

func TestDirective_MixedPrePost_SameField(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var order []string

	sb.Directive("pre", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "pre")
			return nil
		},
	})
	sb.Directive("post", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "post")
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("mixed", func() string {
		order = append(order, "resolver")
		return "ok"
	},
		schemabuilder.WithFieldDirective("pre"),
		schemabuilder.WithFieldDirective("post"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ mixed }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"mixed": "ok"}, val)
	// Pre runs first, then resolver, then post.
	assert.Equal(t, []string{"pre", "resolver", "post"}, order)
}

// ---------------------------------------------------------------------------
// Test: PreResolver fails → resolver never runs, PostResolver never runs
// ---------------------------------------------------------------------------

func TestDirective_PreFails_ResolverAndPostSkipped(t *testing.T) {
	sb := schemabuilder.NewSchema()

	resolverCalled := false
	postCalled := false

	sb.Directive("blockPre", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("blocked")
		},
	})
	sb.Directive("postCheck", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			postCalled = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("guarded", func() string {
		resolverCalled = true
		return "should not run"
	},
		schemabuilder.WithFieldDirective("blockPre"),
		schemabuilder.WithFieldDirective("postCheck"),
	)

	_, err := buildAndExecCtx(t, sb, context.Background(), `{ guarded }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
	assert.False(t, resolverCalled, "resolver should not run after pre-resolver failure")
	assert.False(t, postCalled, "post-resolver should not run after pre-resolver failure")
}

// ---------------------------------------------------------------------------
// Test: PreResolver SkipOnFail → resolver never runs, PostResolver never runs,
// field returns null (no error)
// ---------------------------------------------------------------------------

func TestDirective_PreSkip_ResolverAndPostSkipped(t *testing.T) {
	sb := schemabuilder.NewSchema()

	resolverCalled := false
	postCalled := false

	sb.Directive("skipPre", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.SkipOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("skip")
		},
	})
	sb.Directive("postTracker", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			postCalled = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("skippable", func() *string {
		resolverCalled = true
		s := "data"
		return &s
	},
		schemabuilder.WithFieldDirective("skipPre"),
		schemabuilder.WithFieldDirective("postTracker"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ skippable }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Nil(t, m["skippable"])
	assert.False(t, resolverCalled, "resolver should not run when pre-resolver skips")
	assert.False(t, postCalled, "post-resolver should not run when pre-resolver skips")
}

// ---------------------------------------------------------------------------
// Test: Multiple PreResolver directives — first fails, second never executes
// ---------------------------------------------------------------------------

func TestDirective_MultiplePreResolvers_FirstFailsSecondSkipped(t *testing.T) {
	sb := schemabuilder.NewSchema()

	secondCalled := false

	sb.Directive("failFirst", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("first failed")
		},
	})
	sb.Directive("secondPre", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			secondCalled = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("chain", func() string { return "ok" },
		schemabuilder.WithFieldDirective("failFirst"),
		schemabuilder.WithFieldDirective("secondPre"),
	)

	_, err := buildAndExecCtx(t, sb, context.Background(), `{ chain }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "first failed")
	assert.False(t, secondCalled, "second pre-resolver should not run after first fails")
}

// ---------------------------------------------------------------------------
// Test: Multiple PostResolver directives — first fails, second never executes
// ---------------------------------------------------------------------------

func TestDirective_MultiplePostResolvers_FirstFailsSecondSkipped(t *testing.T) {
	sb := schemabuilder.NewSchema()

	secondCalled := false

	sb.Directive("failPost1", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("post1 failed")
		},
	})
	sb.Directive("passPost2", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			secondCalled = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("chain", func() string { return "ok" },
		schemabuilder.WithFieldDirective("failPost1"),
		schemabuilder.WithFieldDirective("passPost2"),
	)

	_, err := buildAndExecCtx(t, sb, context.Background(), `{ chain }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "post1 failed")
	assert.False(t, secondCalled, "second post-resolver should not run after first fails")
}

// ---------------------------------------------------------------------------
// Test: Type-level directive with args propagates to all fields
// ---------------------------------------------------------------------------

func TestDirective_TypeLevel_WithArgs(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var capturedRoles []string
	sb.Directive("requireRole", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationObject},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "role", TypeName: "String"},
		},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			role, _ := args["role"].(string)
			capturedRoles = append(capturedRoles, role)
			// Allow for this test.
			return nil
		},
	})

	type Stats struct{}
	sb.Object("Stats", Stats{},
		schemabuilder.WithDirective("requireRole", map[string]interface{}{"role": "MANAGER"}),
	)
	obj, _ := sb.GetObject("Stats", Stats{})
	obj.FieldFunc("count", func(_ Stats) int32 { return 10 })
	obj.FieldFunc("avg", func(_ Stats) float64 { return 5.5 })

	query := sb.Query()
	query.FieldFunc("stats", func() *Stats { return &Stats{} })

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ stats { count avg } }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	inner := m["stats"].(map[string]interface{})
	assert.Equal(t, int32(10), inner["count"])
	// Each field triggers the type-level directive → 2 calls.
	assert.Len(t, capturedRoles, 2)
	assert.Equal(t, "MANAGER", capturedRoles[0])
	assert.Equal(t, "MANAGER", capturedRoles[1])
}

// ---------------------------------------------------------------------------
// Test: Type-level + field-level directive combined ordering
// ---------------------------------------------------------------------------

func TestDirective_TypeLevel_PlusFieldLevel_Ordering(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var order []string

	sb.Directive("typeDir", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationObject},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "type-dir")
			return nil
		},
	})
	sb.Directive("fieldDir", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "field-dir")
			return nil
		},
	})

	type Item struct{}
	sb.Object("Item", Item{}, schemabuilder.WithDirective("typeDir"))
	obj, _ := sb.GetObject("Item", Item{})
	obj.FieldFunc("name", func(_ Item) string { return "x" },
		schemabuilder.WithFieldDirective("fieldDir"),
	)

	query := sb.Query()
	query.FieldFunc("item", func() *Item { return &Item{} })

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ item { name } }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Equal(t, "x", m["item"].(map[string]interface{})["name"])
	// Type-level directive runs first (prepended), then field-level.
	assert.Equal(t, []string{"type-dir", "field-dir"}, order)
}

// ---------------------------------------------------------------------------
// Test: Multiple type-level directives on same object
// ---------------------------------------------------------------------------

func TestDirective_MultipleTypeLevelDirectives(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var order []string

	sb.Directive("typeA", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationObject},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "typeA")
			return nil
		},
	})
	sb.Directive("typeB", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationObject},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			order = append(order, "typeB")
			return nil
		},
	})

	type Stuff struct{}
	sb.Object("Stuff", Stuff{},
		schemabuilder.WithDirective("typeA"),
		schemabuilder.WithDirective("typeB"),
	)
	obj, _ := sb.GetObject("Stuff", Stuff{})
	obj.FieldFunc("val", func(_ Stuff) string { return "v" })

	query := sb.Query()
	query.FieldFunc("stuff", func() *Stuff { return &Stuff{} })

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ stuff { val } }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Equal(t, "v", m["stuff"].(map[string]interface{})["val"])
	// Both type-level directives should fire.
	assert.Equal(t, []string{"typeA", "typeB"}, order)
}

// ---------------------------------------------------------------------------
// Test: Handler that always succeeds — data flows through
// ---------------------------------------------------------------------------

func TestDirective_AlwaysPass(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("alwaysOK", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return nil // always pass
		},
	})

	query := sb.Query()
	query.FieldFunc("value", func() int32 { return 42 },
		schemabuilder.WithFieldDirective("alwaysOK"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ value }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"value": int32(42)}, val)
}

// ---------------------------------------------------------------------------
// Test: Multiple metadata-only directives on same field — no runtime impact
// ---------------------------------------------------------------------------

func TestDirective_MultipleMetadataOnly(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("deprecated2", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Args:      []schemabuilder.DirectiveArgDef{{Name: "reason", TypeName: "String"}},
		Handler:   nil,
	})
	sb.Directive("tag", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Args:      []schemabuilder.DirectiveArgDef{{Name: "name", TypeName: "String"}},
		Handler:   nil,
	})

	query := sb.Query()
	query.FieldFunc("old", func() string { return "legacy" },
		schemabuilder.WithFieldDirective("deprecated2", map[string]interface{}{"reason": "use newField"}),
		schemabuilder.WithFieldDirective("tag", map[string]interface{}{"name": "v1"}),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ old }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"old": "legacy"}, val)
}

// ---------------------------------------------------------------------------
// Test: Multiple fields, each with different directives (isolation)
// ---------------------------------------------------------------------------

func TestDirective_MultipleFields_DifferentDirectives(t *testing.T) {
	sb := schemabuilder.NewSchema()

	var callLog []string

	sb.Directive("dirA", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			callLog = append(callLog, "dirA")
			return nil
		},
	})
	sb.Directive("dirB", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			callLog = append(callLog, "dirB")
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("fieldA", func() string { return "a" },
		schemabuilder.WithFieldDirective("dirA"),
	)
	query.FieldFunc("fieldB", func() string { return "b" },
		schemabuilder.WithFieldDirective("dirB"),
	)
	query.FieldFunc("fieldPlain", func() string { return "p" })

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ fieldA fieldB fieldPlain }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Equal(t, "a", m["fieldA"])
	assert.Equal(t, "b", m["fieldB"])
	assert.Equal(t, "p", m["fieldPlain"])
	// dirA and dirB each called once; order depends on field resolution order.
	assert.Len(t, callLog, 2)
	assert.Contains(t, callLog, "dirA")
	assert.Contains(t, callLog, "dirB")
}

// ---------------------------------------------------------------------------
// Test: PostResolver SkipOnFail — resolver runs, but result is nullified
// ---------------------------------------------------------------------------

func TestDirective_PostResolver_SkipOnFail_ResolverRuns(t *testing.T) {
	sb := schemabuilder.NewSchema()

	resolverRan := false
	sb.Directive("postMask", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.SkipOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("mask it")
		},
	})

	query := sb.Query()
	query.FieldFunc("sensitive", func() *string {
		resolverRan = true
		s := "secret"
		return &s
	}, schemabuilder.WithFieldDirective("postMask"))

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ sensitive }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Nil(t, m["sensitive"], "post-resolver skip should null out the value")
	assert.True(t, resolverRan, "resolver should have run before post-resolver")
}

// ---------------------------------------------------------------------------
// Test: Directive on nested object fields
// ---------------------------------------------------------------------------

func TestDirective_NestedObjectField(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("guard", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			if ctx.Value(roleKey) == nil {
				return errors.New("not allowed")
			}
			return nil
		},
	})

	type Inner struct{}
	obj := sb.Object("InnerObj", Inner{})
	obj.FieldFunc("public", func(_ Inner) string { return "pub" })
	obj.FieldFunc("private", func(_ Inner) string { return "priv" },
		schemabuilder.WithFieldDirective("guard"),
	)

	query := sb.Query()
	query.FieldFunc("nested", func() *Inner { return &Inner{} })

	// Without auth → accessing private errors.
	_, err := buildAndExecCtx(t, sb, context.Background(), `{ nested { private } }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	// Public field works fine.
	val, err := buildAndExecCtx(t, sb, context.Background(), `{ nested { public } }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Equal(t, "pub", m["nested"].(map[string]interface{})["public"])

	// With auth → private works.
	ctx := context.WithValue(context.Background(), roleKey, "USER")
	val, err = buildAndExecCtx(t, sb, ctx, `{ nested { public private } }`)
	require.NoError(t, err)
	m = val.(map[string]interface{})
	inner := m["nested"].(map[string]interface{})
	assert.Equal(t, "pub", inner["public"])
	assert.Equal(t, "priv", inner["private"])
}

// ---------------------------------------------------------------------------
// Test: Duplicate directive registration panics
// ---------------------------------------------------------------------------

func TestDirective_DuplicateRegistrationPanics(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("dup", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler:   nil,
	})

	assert.Panics(t, func() {
		sb.Directive("dup", &schemabuilder.DirectiveDefinition{
			Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
			Handler:   nil,
		})
	})
}

// ---------------------------------------------------------------------------
// Test: Clone preserves directives
// ---------------------------------------------------------------------------

func TestDirective_ClonePreservesDirectives(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("cloneDir", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler:   nil,
	})

	query := sb.Query()
	query.FieldFunc("field1", func() string { return "a" },
		schemabuilder.WithFieldDirective("cloneDir"),
	)

	// Clone the schema.
	cloned := sb.Clone()

	// The cloned schema should build successfully (directive still registered).
	schema, err := cloned.Build()
	require.NoError(t, err)
	require.NotNil(t, schema)

	// CustomDirectives present.
	found := false
	for _, cd := range schema.CustomDirectives {
		if cd.Name == "cloneDir" {
			found = true
			break
		}
	}
	assert.True(t, found, "cloned schema should retain directive definitions")
}

// ---------------------------------------------------------------------------
// Test: Introspection with multiple custom directives and their args
// ---------------------------------------------------------------------------

func TestDirective_Introspection_MultipleWithArgs(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("auth", &schemabuilder.DirectiveDefinition{
		Description: "Authentication check",
		Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition, schemabuilder.LocationObject},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "role", TypeName: "String", Description: "Required role"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) error { return nil },
	})
	sb.Directive("rateLimit", &schemabuilder.DirectiveDefinition{
		Description: "Rate limiter",
		Locations:   []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "max", TypeName: "Int", Description: "Max calls"},
			{Name: "window", TypeName: "Int", Description: "Time window in seconds"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) error { return nil },
	})

	query := sb.Query()
	query.FieldFunc("hello", func() string { return "world" })

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__schema {
			directives {
				name description locations
				args { name description }
			}
		}
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	root := val.(map[string]interface{})
	schemaData := root["__schema"].(map[string]interface{})
	directives := schemaData["directives"].([]interface{})

	// Find custom directives.
	dirMap := map[string]map[string]interface{}{}
	for _, d := range directives {
		dm := d.(map[string]interface{})
		dirMap[dm["name"].(string)] = dm
	}

	// auth directive
	authDir, ok := dirMap["auth"]
	require.True(t, ok, "@auth should appear in introspection")
	assert.Equal(t, "Authentication check", authDir["description"])
	authLocs := authDir["locations"].([]interface{})
	assert.Contains(t, authLocs, "FIELD_DEFINITION")
	assert.Contains(t, authLocs, "OBJECT")
	authArgs := authDir["args"].([]interface{})
	require.Len(t, authArgs, 1)
	assert.Equal(t, "role", authArgs[0].(map[string]interface{})["name"])

	// rateLimit directive
	rlDir, ok := dirMap["rateLimit"]
	require.True(t, ok, "@rateLimit should appear in introspection")
	rlArgs := rlDir["args"].([]interface{})
	require.Len(t, rlArgs, 2)
	argNames := []string{
		rlArgs[0].(map[string]interface{})["name"].(string),
		rlArgs[1].(map[string]interface{})["name"].(string),
	}
	assert.Contains(t, argNames, "max")
	assert.Contains(t, argNames, "window")
}

// ---------------------------------------------------------------------------
// Test: Schema with no directives builds normally
// ---------------------------------------------------------------------------

func TestDirective_NoDirectives_BuildsClean(t *testing.T) {
	sb := schemabuilder.NewSchema()
	// No directives registered at all.

	query := sb.Query()
	query.FieldFunc("hello", func() string { return "world" })

	schema, err := sb.Build()
	require.NoError(t, err)
	require.NotNil(t, schema)
	assert.Nil(t, schema.CustomDirectives)
}

// ---------------------------------------------------------------------------
// Test: Directive on scalar-returning field
// ---------------------------------------------------------------------------

func TestDirective_ScalarField(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("scalarGuard", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return nil // pass
		},
	})

	query := sb.Query()
	query.FieldFunc("number", func() int32 { return 99 },
		schemabuilder.WithFieldDirective("scalarGuard"),
	)
	query.FieldFunc("text", func() string { return "hello" },
		schemabuilder.WithFieldDirective("scalarGuard"),
	)
	query.FieldFunc("flag", func() bool { return true },
		schemabuilder.WithFieldDirective("scalarGuard"),
	)
	query.FieldFunc("score", func() float64 { return 3.14 },
		schemabuilder.WithFieldDirective("scalarGuard"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ number text flag score }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Equal(t, int32(99), m["number"])
	assert.Equal(t, "hello", m["text"])
	assert.Equal(t, true, m["flag"])
	assert.Equal(t, 3.14, m["score"])
}

// ---------------------------------------------------------------------------
// Test: Directive on list-returning field
// ---------------------------------------------------------------------------

func TestDirective_ListField(t *testing.T) {
	sb := schemabuilder.NewSchema()

	called := false
	sb.Directive("listCheck", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			called = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("items", func() []string { return []string{"a", "b", "c"} },
		schemabuilder.WithFieldDirective("listCheck"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ items }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	items := m["items"].([]interface{})
	assert.Len(t, items, 3)
	assert.True(t, called)
}

// ---------------------------------------------------------------------------
// Test: Directive does not affect sibling fields
// ---------------------------------------------------------------------------

func TestDirective_DoesNotAffectSiblings(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("blocker", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.SkipOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("nope")
		},
	})

	query := sb.Query()
	query.FieldFunc("blocked", func() *string {
		s := "blocked"
		return &s
	}, schemabuilder.WithFieldDirective("blocker"))
	query.FieldFunc("open", func() string { return "visible" })

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ blocked open }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Nil(t, m["blocked"], "blocked field should be null")
	assert.Equal(t, "visible", m["open"], "sibling field should be unaffected")
}

// ---------------------------------------------------------------------------
// Test: Directive with multiple locations in definition
// ---------------------------------------------------------------------------

func TestDirective_MultipleLocationsInDefinition(t *testing.T) {
	sb := schemabuilder.NewSchema()

	callCount := 0
	sb.Directive("multiLoc", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationFieldDefinition,
			schemabuilder.LocationObject,
		},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			callCount++
			return nil
		},
	})

	// Use at field level.
	query := sb.Query()
	query.FieldFunc("fieldLevel", func() string { return "f" },
		schemabuilder.WithFieldDirective("multiLoc"),
	)

	// Use at type level.
	type Obj struct{}
	sb.Object("Obj", Obj{}, schemabuilder.WithDirective("multiLoc"))
	o, _ := sb.GetObject("Obj", Obj{})
	o.FieldFunc("val", func(_ Obj) string { return "v" })
	query.FieldFunc("objLevel", func() *Obj { return &Obj{} })

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ fieldLevel objLevel { val } }`)
	require.NoError(t, err)
	m := val.(map[string]interface{})
	assert.Equal(t, "f", m["fieldLevel"])
	assert.Equal(t, "v", m["objLevel"].(map[string]interface{})["val"])
	// 1 for field-level + 1 for type-level (propagated to val)
	assert.Equal(t, 2, callCount)
}

// ---------------------------------------------------------------------------
// Test: Introspection with no custom directives — only built-ins appear
// ---------------------------------------------------------------------------

func TestDirective_Introspection_NoCustom(t *testing.T) {
	sb := schemabuilder.NewSchema()
	query := sb.Query()
	query.FieldFunc("hello", func() string { return "world" })

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__schema { directives { name } }
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	val, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	root := val.(map[string]interface{})
	dirs := root["__schema"].(map[string]interface{})["directives"].([]interface{})

	names := map[string]bool{}
	for _, d := range dirs {
		names[d.(map[string]interface{})["name"].(string)] = true
	}

	// Only built-in directives.
	assert.True(t, names["include"])
	assert.True(t, names["skip"])
	assert.True(t, names["deprecated"])
	assert.True(t, names["specifiedBy"])
	assert.True(t, names["oneOf"])
}

// ---------------------------------------------------------------------------
// Test: Field-level directive on a field of the Query root (not nested)
// ---------------------------------------------------------------------------

func TestDirective_OnQueryRootField(t *testing.T) {
	sb := schemabuilder.NewSchema()

	handlerCalled := false
	sb.Directive("rootGuard", &schemabuilder.DirectiveDefinition{
		Locations: []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			handlerCalled = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("rootField", func() string { return "root" },
		schemabuilder.WithFieldDirective("rootGuard"),
	)

	val, err := buildAndExecCtx(t, sb, context.Background(), `{ rootField }`)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"rootField": "root"}, val)
	assert.True(t, handlerCalled)
}

// ---------------------------------------------------------------------------
// Test: Directive interaction with field returning error
// ---------------------------------------------------------------------------

func TestDirective_FieldReturnsError(t *testing.T) {
	sb := schemabuilder.NewSchema()

	preCalled := false
	postCalled := false

	sb.Directive("preOK", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			preCalled = true
			return nil
		},
	})
	sb.Directive("postOK", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			postCalled = true
			return nil
		},
	})

	query := sb.Query()
	query.FieldFunc("failing", func() (string, error) {
		return "", errors.New("resolver error")
	},
		schemabuilder.WithFieldDirective("preOK"),
		schemabuilder.WithFieldDirective("postOK"),
	)

	_, err := buildAndExecCtx(t, sb, context.Background(), `{ failing }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolver error")
	assert.True(t, preCalled, "pre-resolver should have run before the resolver")
	// Post-resolver should not run because the resolver itself errored.
	assert.False(t, postCalled, "post-resolver should not run when resolver errors")
}
