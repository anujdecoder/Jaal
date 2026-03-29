package graphql_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/schemabuilder"
)

// ---------------------------------------------------------------------------
// domain types used across batch tests
// ---------------------------------------------------------------------------

type batchPost struct {
	ID       string
	Title    string
	AuthorID string
}

type batchAuthor struct {
	ID   string
	Name string
}

type batchTag struct {
	Label string
}

type batchComment struct {
	ID     string
	Body   string
	PostID string
}

// ---------------------------------------------------------------------------
// helper: build schema, execute query, return result
// ---------------------------------------------------------------------------

func batchExec(t *testing.T, sb *schemabuilder.Schema, query string) (interface{}, error) {
	t.Helper()
	return buildAndExec(t, sb, query)
}

func batchExecCtx(t *testing.T, sb *schemabuilder.Schema, ctx context.Context, query string) (interface{}, error) {
	t.Helper()
	return buildAndExecCtx(t, sb, ctx, query)
}

// ---------------------------------------------------------------------------
// 1. Basic batch field – resolves once for all items
// ---------------------------------------------------------------------------

func TestBatch_BasicResolveCalledOnce(t *testing.T) {
	var callCount int32

	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("author", func(ctx context.Context, posts []batchPost) ([]string, error) {
		atomic.AddInt32(&callCount, 1)
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "author-of-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{
			{ID: "1", Title: "A"},
			{ID: "2", Title: "B"},
			{ID: "3", Title: "C"},
		}
	})

	res, err := batchExec(t, sb, `{ posts { title author } }`)
	require.NoError(t, err)

	root := res.(map[string]interface{})
	posts := root["posts"].([]interface{})
	require.Len(t, posts, 3)
	assert.Equal(t, "author-of-1", posts[0].(map[string]interface{})["author"])
	assert.Equal(t, "author-of-2", posts[1].(map[string]interface{})["author"])
	assert.Equal(t, "author-of-3", posts[2].(map[string]interface{})["author"])

	// The batch resolver should be called exactly once.
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

// ---------------------------------------------------------------------------
// 2. Batch field returns struct (nested object)
// ---------------------------------------------------------------------------

func TestBatch_ReturnsObject(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("author", func(posts []batchPost) ([]*batchAuthor, error) {
		out := make([]*batchAuthor, len(posts))
		for i, p := range posts {
			out[i] = &batchAuthor{ID: p.AuthorID, Name: "Author " + p.AuthorID}
		}
		return out, nil
	})

	author := sb.Object("Author", batchAuthor{})
	author.FieldFunc("name", func(a batchAuthor) string { return a.Name })
	author.FieldFunc("id", func(a batchAuthor) string { return a.ID })

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{
			{ID: "1", Title: "First", AuthorID: "a1"},
			{ID: "2", Title: "Second", AuthorID: "a2"},
		}
	})

	res, err := batchExec(t, sb, `{ posts { title author { name id } } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	a0 := posts[0].(map[string]interface{})["author"].(map[string]interface{})
	assert.Equal(t, "Author a1", a0["name"])
	assert.Equal(t, "a1", a0["id"])
}

// ---------------------------------------------------------------------------
// 3. Batch + non-batch fields coexist
// ---------------------------------------------------------------------------

func TestBatch_MixedBatchAndNonBatch(t *testing.T) {
	var batchCalls int32
	var normalCalls int32

	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string {
		atomic.AddInt32(&normalCalls, 1)
		return p.Title
	})
	post.BatchFieldFunc("author", func(posts []batchPost) ([]string, error) {
		atomic.AddInt32(&batchCalls, 1)
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "author-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}, {ID: "2", Title: "B"}}
	})

	res, err := batchExec(t, sb, `{ posts { title author } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "A", posts[0].(map[string]interface{})["title"])
	assert.Equal(t, "author-1", posts[0].(map[string]interface{})["author"])

	assert.Equal(t, int32(1), atomic.LoadInt32(&batchCalls))
	assert.Equal(t, int32(2), atomic.LoadInt32(&normalCalls))
}

// ---------------------------------------------------------------------------
// 4. Batch field with pointer source (func(ctx, []*Post))
// ---------------------------------------------------------------------------

func TestBatch_PointerSource(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("upper", func(posts []*batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "UPPER:" + p.Title
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "hello"}}
	})

	res, err := batchExec(t, sb, `{ posts { title upper } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "UPPER:hello", posts[0].(map[string]interface{})["upper"])
}

// ---------------------------------------------------------------------------
// 5. Batch field with args
// ---------------------------------------------------------------------------

func TestBatch_WithArgs(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("excerpt", func(posts []batchPost, args struct{ MaxLen int32 }) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			r := []rune(p.Title)
			if int32(len(r)) > args.MaxLen {
				r = r[:args.MaxLen]
			}
			out[i] = string(r)
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{
			{ID: "1", Title: "Hello World"},
			{ID: "2", Title: "Go"},
		}
	})

	res, err := batchExec(t, sb, `{ posts { excerpt(maxLen: 5) } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "Hello", posts[0].(map[string]interface{})["excerpt"])
	assert.Equal(t, "Go", posts[1].(map[string]interface{})["excerpt"])
}

// ---------------------------------------------------------------------------
// 6. Batch field error propagation
// ---------------------------------------------------------------------------

func TestBatch_ErrorPropagation(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("fail", func(posts []batchPost) ([]string, error) {
		return nil, errors.New("batch boom")
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}}
	})

	_, err := batchExec(t, sb, `{ posts { title fail } }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "batch boom")
}

// ---------------------------------------------------------------------------
// 7. Batch field length mismatch
// ---------------------------------------------------------------------------

func TestBatch_LengthMismatch(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("bad", func(posts []batchPost) ([]string, error) {
		// Return wrong number of results.
		return []string{"only-one"}, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1"}, {ID: "2"}, {ID: "3"}}
	})

	_, err := batchExec(t, sb, `{ posts { bad } }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "returned 1 results, expected 3")
}

// ---------------------------------------------------------------------------
// 8. Batch field single-item fallback (non-list context)
// ---------------------------------------------------------------------------

func TestBatch_SingleItemFallback(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "slug-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	// Return a single object (not a list) — the Resolve fallback is used.
	query.FieldFunc("post", func() *batchPost {
		return &batchPost{ID: "42", Title: "Single"}
	})

	res, err := batchExec(t, sb, `{ post { title slug } }`)
	require.NoError(t, err)

	p := res.(map[string]interface{})["post"].(map[string]interface{})
	assert.Equal(t, "Single", p["title"])
	assert.Equal(t, "slug-42", p["slug"])
}

// ---------------------------------------------------------------------------
// 9. Batch field with context propagation
// ---------------------------------------------------------------------------

type batchCtxKey struct{}

func TestBatch_ContextPropagation(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("ctxValue", func(ctx context.Context, posts []batchPost) ([]string, error) {
		val, _ := ctx.Value(batchCtxKey{}).(string)
		out := make([]string, len(posts))
		for i := range posts {
			out[i] = val
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1"}, {ID: "2"}}
	})

	ctx := context.WithValue(context.Background(), batchCtxKey{}, "from-ctx")
	res, err := batchExecCtx(t, sb, ctx, `{ posts { ctxValue } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "from-ctx", posts[0].(map[string]interface{})["ctxValue"])
	assert.Equal(t, "from-ctx", posts[1].(map[string]interface{})["ctxValue"])
}

// ---------------------------------------------------------------------------
// 10. Batch field with no context
// ---------------------------------------------------------------------------

func TestBatch_NoContext(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("doubled", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = p.Title + p.Title
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "Hi"}}
	})

	res, err := batchExec(t, sb, `{ posts { doubled } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "HiHi", posts[0].(map[string]interface{})["doubled"])
}

// ---------------------------------------------------------------------------
// 11. __typename works alongside batch fields
// ---------------------------------------------------------------------------

func TestBatch_Typename(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "s-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}}
	})

	res, err := batchExec(t, sb, `{ posts { __typename title slug } }`)
	require.NoError(t, err)

	p := res.(map[string]interface{})["posts"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "Post", p["__typename"])
	assert.Equal(t, "A", p["title"])
	assert.Equal(t, "s-1", p["slug"])
}

// ---------------------------------------------------------------------------
// 12. Empty list — no panic, no batch call
// ---------------------------------------------------------------------------

func TestBatch_EmptyList(t *testing.T) {
	var called int32

	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		atomic.AddInt32(&called, 1)
		return nil, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost { return []batchPost{} })

	res, err := batchExec(t, sb, `{ posts { title slug } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Len(t, posts, 0)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

// ---------------------------------------------------------------------------
// 13. Multiple batch fields on one object
// ---------------------------------------------------------------------------

func TestBatch_MultipleBatchFields(t *testing.T) {
	var batchACalls, batchBCalls int32

	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("fieldA", func(posts []batchPost) ([]string, error) {
		atomic.AddInt32(&batchACalls, 1)
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "A:" + p.ID
		}
		return out, nil
	})
	post.BatchFieldFunc("fieldB", func(posts []batchPost) ([]string, error) {
		atomic.AddInt32(&batchBCalls, 1)
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "B:" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1"}, {ID: "2"}}
	})

	res, err := batchExec(t, sb, `{ posts { title fieldA fieldB } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "A:1", posts[0].(map[string]interface{})["fieldA"])
	assert.Equal(t, "B:2", posts[1].(map[string]interface{})["fieldB"])
	assert.Equal(t, int32(1), atomic.LoadInt32(&batchACalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&batchBCalls))
}

// ---------------------------------------------------------------------------
// 14. Batch field returns nullable (pointer) results
// ---------------------------------------------------------------------------

func TestBatch_NullableResults(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("id", func(p batchPost) string { return p.ID })
	post.BatchFieldFunc("maybeAuthor", func(posts []batchPost) ([]*batchAuthor, error) {
		out := make([]*batchAuthor, len(posts))
		for i, p := range posts {
			if p.AuthorID != "" {
				out[i] = &batchAuthor{ID: p.AuthorID, Name: "Author " + p.AuthorID}
			}
			// else nil
		}
		return out, nil
	})

	author := sb.Object("Author", batchAuthor{})
	author.FieldFunc("name", func(a batchAuthor) string { return a.Name })

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{
			{ID: "1", AuthorID: "a1"},
			{ID: "2", AuthorID: ""},
		}
	})

	res, err := batchExec(t, sb, `{ posts { id maybeAuthor { name } } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	a0 := posts[0].(map[string]interface{})["maybeAuthor"]
	assert.NotNil(t, a0)
	assert.Equal(t, "Author a1", a0.(map[string]interface{})["name"])

	a1 := posts[1].(map[string]interface{})["maybeAuthor"]
	assert.Nil(t, a1)
}

// ---------------------------------------------------------------------------
// 15. Nested batch: batch field returning list that also has batch fields
// ---------------------------------------------------------------------------

func TestBatch_NestedBatch(t *testing.T) {
	var postBatchCalls, commentBatchCalls int32

	sb := schemabuilder.NewSchema()

	comment := sb.Object("Comment", batchComment{})
	comment.FieldFunc("body", func(c batchComment) string { return c.Body })
	comment.BatchFieldFunc("upperBody", func(comments []batchComment) ([]string, error) {
		atomic.AddInt32(&commentBatchCalls, 1)
		out := make([]string, len(comments))
		for i, c := range comments {
			out[i] = "UPPER:" + c.Body
		}
		return out, nil
	})

	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("comments", func(ctx context.Context, posts []batchPost) ([][]*batchComment, error) {
		atomic.AddInt32(&postBatchCalls, 1)
		out := make([][]*batchComment, len(posts))
		for i, p := range posts {
			out[i] = []*batchComment{
				{ID: fmt.Sprintf("c%s-1", p.ID), Body: "comment1-" + p.ID, PostID: p.ID},
				{ID: fmt.Sprintf("c%s-2", p.ID), Body: "comment2-" + p.ID, PostID: p.ID},
			}
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "P1"}, {ID: "2", Title: "P2"}}
	})

	res, err := batchExec(t, sb, `{ posts { title comments { body upperBody } } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	require.Len(t, posts, 2)

	// Post 1 comments
	c1 := posts[0].(map[string]interface{})["comments"].([]interface{})
	require.Len(t, c1, 2)
	assert.Equal(t, "comment1-1", c1[0].(map[string]interface{})["body"])
	assert.Equal(t, "UPPER:comment1-1", c1[0].(map[string]interface{})["upperBody"])

	// Post 2 comments
	c2 := posts[1].(map[string]interface{})["comments"].([]interface{})
	require.Len(t, c2, 2)
	assert.Equal(t, "comment2-2", c2[1].(map[string]interface{})["body"])
	assert.Equal(t, "UPPER:comment2-2", c2[1].(map[string]interface{})["upperBody"])

	// Post-level batch called once (all posts batched together).
	assert.Equal(t, int32(1), atomic.LoadInt32(&postBatchCalls))
	// Comment-level batch called once per inner list (one per post's comments).
	assert.Equal(t, int32(2), atomic.LoadInt32(&commentBatchCalls))
}

// ---------------------------------------------------------------------------
// 16. Batch field with directive (PreResolver + SkipOnFail)
// ---------------------------------------------------------------------------

func TestBatch_DirectiveSkipOnFail(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("blocked", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.SkipOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("blocked")
		},
	})

	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("secret", func(posts []batchPost) ([]string, error) {
		t.Fatal("should not be called when directive skips")
		return nil, nil
	}, schemabuilder.WithFieldDirective("blocked"))

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}, {ID: "2", Title: "B"}}
	})

	res, err := batchExec(t, sb, `{ posts { title secret } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "A", posts[0].(map[string]interface{})["title"])
	assert.Nil(t, posts[0].(map[string]interface{})["secret"])
	assert.Nil(t, posts[1].(map[string]interface{})["secret"])
}

// ---------------------------------------------------------------------------
// 17. Batch field with directive (PreResolver + ErrorOnFail)
// ---------------------------------------------------------------------------

func TestBatch_DirectiveErrorOnFail(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("deny", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			return errors.New("access denied")
		},
	})

	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("admin", func(posts []batchPost) ([]string, error) {
		t.Fatal("should not be called")
		return nil, nil
	}, schemabuilder.WithFieldDirective("deny"))

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1"}}
	})

	_, err := batchExec(t, sb, `{ posts { title admin } }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// ---------------------------------------------------------------------------
// 18. Batch field with PostResolver directive
// ---------------------------------------------------------------------------

func TestBatch_PostResolverDirective(t *testing.T) {
	var postDirCalled int32

	sb := schemabuilder.NewSchema()

	sb.Directive("audit", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationFieldDefinition},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			atomic.AddInt32(&postDirCalled, 1)
			return nil
		},
	})

	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("audited", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "val-" + p.ID
		}
		return out, nil
	}, schemabuilder.WithFieldDirective("audit"))

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1"}, {ID: "2"}}
	})

	res, err := batchExec(t, sb, `{ posts { audited } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "val-1", posts[0].(map[string]interface{})["audited"])
	assert.Equal(t, int32(1), atomic.LoadInt32(&postDirCalled))
}

// ---------------------------------------------------------------------------
// 19. Clone preserves batch flag
// ---------------------------------------------------------------------------

func TestBatch_ClonePreservesBatch(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "s-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "T"}}
	})

	clone := sb.Clone()

	res, err := batchExec(t, clone, `{ posts { title slug } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "s-1", posts[0].(map[string]interface{})["slug"])
}

// ---------------------------------------------------------------------------
// 20. Batch introspection — field type is still the element type
// ---------------------------------------------------------------------------

func TestBatch_IntrospectionFieldType(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("author", func(posts []batchPost) ([]*batchAuthor, error) {
		out := make([]*batchAuthor, len(posts))
		return out, nil
	})

	author := sb.Object("Author", batchAuthor{})
	author.FieldFunc("name", func(a batchAuthor) string { return a.Name })

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return nil
	})

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__type(name: "Post") {
			fields {
				name
				type { name kind ofType { name } }
			}
		}
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	res, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	typ := res.(map[string]interface{})["__type"].(map[string]interface{})
	fields := typ["fields"].([]interface{})

	// Find "author" field.
	var authorField map[string]interface{}
	for _, f := range fields {
		fm := f.(map[string]interface{})
		if fm["name"] == "author" {
			authorField = fm
			break
		}
	}
	require.NotNil(t, authorField, "author field should exist in introspection")

	// The type should be Author (nullable pointer = Author, not [Author]).
	ftype := authorField["type"].(map[string]interface{})
	assert.Equal(t, "Author", ftype["name"])
}

// ---------------------------------------------------------------------------
// 21. Batch with field alias
// ---------------------------------------------------------------------------

func TestBatch_Alias(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "s-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}, {ID: "2", Title: "B"}}
	})

	res, err := batchExec(t, sb, `{ posts { title mySlug: slug } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "s-1", posts[0].(map[string]interface{})["mySlug"])
	assert.Equal(t, "s-2", posts[1].(map[string]interface{})["mySlug"])
}

// ---------------------------------------------------------------------------
// 22. Batch with FieldDesc option
// ---------------------------------------------------------------------------

func TestBatch_FieldDescription(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		return out, nil
	}, schemabuilder.FieldDesc("URL-friendly slug"))

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost { return nil })

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__type(name: "Post") {
			fields { name description }
		}
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	res, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	fields := res.(map[string]interface{})["__type"].(map[string]interface{})["fields"].([]interface{})
	for _, f := range fields {
		fm := f.(map[string]interface{})
		if fm["name"] == "slug" {
			assert.Equal(t, "URL-friendly slug", fm["description"])
			return
		}
	}
	t.Fatal("slug field not found")
}

// ---------------------------------------------------------------------------
// 23. Batch with Deprecated option
// ---------------------------------------------------------------------------

func TestBatch_Deprecated(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("oldSlug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		return out, nil
	}, schemabuilder.Deprecated("use slug instead"))

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost { return nil })

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__type(name: "Post") {
			fields(includeDeprecated: true) { name isDeprecated deprecationReason }
		}
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	res, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	fields := res.(map[string]interface{})["__type"].(map[string]interface{})["fields"].([]interface{})
	for _, f := range fields {
		fm := f.(map[string]interface{})
		if fm["name"] == "oldSlug" {
			assert.Equal(t, true, fm["isDeprecated"])
			assert.Equal(t, "use slug instead", fm["deprecationReason"])
			return
		}
	}
	t.Fatal("oldSlug field not found")
}

// ---------------------------------------------------------------------------
// 24. Batch returns int scalar slice
// ---------------------------------------------------------------------------

func TestBatch_ScalarIntReturn(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("titleLen", func(posts []batchPost) ([]int32, error) {
		out := make([]int32, len(posts))
		for i, p := range posts {
			out[i] = int32(len(p.Title))
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "Hello"}, {ID: "2", Title: "Go"}}
	})

	res, err := batchExec(t, sb, `{ posts { title titleLen } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, int32(5), posts[0].(map[string]interface{})["titleLen"])
	assert.Equal(t, int32(2), posts[1].(map[string]interface{})["titleLen"])
}

// ---------------------------------------------------------------------------
// 25. Batch with NonNull option
// ---------------------------------------------------------------------------

func TestBatch_NonNullOption(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "s-" + p.ID
		}
		return out, nil
	}, schemabuilder.NonNull())

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}}
	})

	schema := sb.MustBuild()
	introspection.AddIntrospectionToSchema(schema)

	q, err := graphql.Parse(`{
		__type(name: "Post") {
			fields { name type { kind ofType { name } } }
		}
	}`, nil)
	require.NoError(t, err)
	require.NoError(t, graphql.ValidateQuery(context.Background(), schema.Query, q.SelectionSet))

	e := graphql.Executor{}
	res, err := e.Execute(context.Background(), schema.Query, nil, q)
	require.NoError(t, err)

	fields := res.(map[string]interface{})["__type"].(map[string]interface{})["fields"].([]interface{})
	for _, f := range fields {
		fm := f.(map[string]interface{})
		if fm["name"] == "slug" {
			ft := fm["type"].(map[string]interface{})
			assert.Equal(t, "NON_NULL", fmt.Sprint(ft["kind"]))
			return
		}
	}
	t.Fatal("slug field not found")
}

// ---------------------------------------------------------------------------
// 26. Batch with type-level directive
// ---------------------------------------------------------------------------

func TestBatch_TypeLevelDirective(t *testing.T) {
	sb := schemabuilder.NewSchema()

	sb.Directive("auth", &schemabuilder.DirectiveDefinition{
		Locations:      []schemabuilder.DirectiveLocation{schemabuilder.LocationObject},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			if ctx.Value(batchCtxKey{}) != "allowed" {
				return errors.New("unauthenticated")
			}
			return nil
		},
	})

	post := sb.Object("Post", batchPost{}, schemabuilder.WithDirective("auth"))
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("slug", func(posts []batchPost) ([]string, error) {
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = "s-" + p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}}
	})

	// Without auth → error
	_, err := batchExec(t, sb, `{ posts { title slug } }`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthenticated")

	// With auth → success
	ctx := context.WithValue(context.Background(), batchCtxKey{}, "allowed")
	res, err := batchExecCtx(t, sb, ctx, `{ posts { title slug } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "s-1", posts[0].(map[string]interface{})["slug"])
}

// ---------------------------------------------------------------------------
// 27. Batch duplicate method panics
// ---------------------------------------------------------------------------

func TestBatch_DuplicateMethodPanics(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })

	assert.Panics(t, func() {
		post.BatchFieldFunc("title", func(posts []batchPost) ([]string, error) {
			return nil, nil
		})
	})
}

// ---------------------------------------------------------------------------
// 28. Batch only fields — all fields batch, no normal
// ---------------------------------------------------------------------------

func TestBatch_AllFieldsBatch(t *testing.T) {
	var calls int32

	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.BatchFieldFunc("title", func(posts []batchPost) ([]string, error) {
		atomic.AddInt32(&calls, 1)
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = p.Title
		}
		return out, nil
	})
	post.BatchFieldFunc("id", func(posts []batchPost) ([]string, error) {
		atomic.AddInt32(&calls, 1)
		out := make([]string, len(posts))
		for i, p := range posts {
			out[i] = p.ID
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "T1"}, {ID: "2", Title: "T2"}}
	})

	res, err := batchExec(t, sb, `{ posts { title id } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	assert.Equal(t, "T1", posts[0].(map[string]interface{})["title"])
	assert.Equal(t, "2", posts[1].(map[string]interface{})["id"])
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls)) // two batch fields, each called once
}

// ---------------------------------------------------------------------------
// 29. Batch field returning list of scalars (e.g. tags)
// ---------------------------------------------------------------------------

func TestBatch_ReturnsListScalar(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("tags", func(posts []batchPost) ([][]string, error) {
		out := make([][]string, len(posts))
		for i, p := range posts {
			out[i] = []string{"tag1-" + p.ID, "tag2-" + p.ID}
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1", Title: "A"}, {ID: "2", Title: "B"}}
	})

	res, err := batchExec(t, sb, `{ posts { title tags } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	tags0 := posts[0].(map[string]interface{})["tags"].([]interface{})
	assert.Equal(t, "tag1-1", tags0[0])
	assert.Equal(t, "tag2-1", tags0[1])
}

// ---------------------------------------------------------------------------
// 30. Batch field with SelectionSet parameter
// ---------------------------------------------------------------------------

func TestBatch_SelectionSetParam(t *testing.T) {
	sb := schemabuilder.NewSchema()
	post := sb.Object("Post", batchPost{})
	post.FieldFunc("title", func(p batchPost) string { return p.Title })
	post.BatchFieldFunc("meta", func(posts []batchPost, ss *graphql.SelectionSet) ([]string, error) {
		// Just confirm we receive a non-nil selection set.
		out := make([]string, len(posts))
		for i := range posts {
			if ss != nil {
				out[i] = "has-ss"
			} else {
				out[i] = "no-ss"
			}
		}
		return out, nil
	})

	query := sb.Query()
	query.FieldFunc("posts", func() []batchPost {
		return []batchPost{{ID: "1"}}
	})

	res, err := batchExec(t, sb, `{ posts { meta } }`)
	require.NoError(t, err)

	posts := res.(map[string]interface{})["posts"].([]interface{})
	// For a scalar return type, selection set should be nil.
	assert.Equal(t, "no-ss", posts[0].(map[string]interface{})["meta"])
}
