package directives

import (
	"context"
	"fmt"

	"go.appointy.com/jaal/schemabuilder"
)

// RegisterSchema wires all type/query/mutation registrations together.
// Call order: directives first, then objects, then queries/mutations.
func RegisterSchema(sb *schemabuilder.Schema, s *Server) {
	// 1. Register directive definitions (must be before any usage).
	RegisterDirectives(sb, s)

	// 2. Register objects and apply directives.
	RegisterObjects(sb)

	// 3. Register query and mutation fields.
	RegisterQueries(sb, s)
	RegisterMutations(sb, s)
}

// RegisterObjects registers output types and their field resolvers, including
// batch-resolved fields that demonstrate the DataLoader / automatic batching
// feature (N+1 query prevention).
func RegisterObjects(sb *schemabuilder.Schema) {
	// Article — plain object; individual fields carry their own directives.
	article := sb.Object("Article", Article{}, schemabuilder.WithDescription("A published article."))
	article.FieldFunc("id", func(a *Article) schemabuilder.ID { return a.ID },
		schemabuilder.FieldDesc("Unique article ID."))
	article.FieldFunc("title", func(a *Article) string { return a.Title },
		schemabuilder.FieldDesc("Article title."))
	article.FieldFunc("body", func(a *Article) string { return a.Body },
		schemabuilder.FieldDesc("Article body text."))
	article.FieldFunc("author", func(a *Article) string { return a.Author },
		schemabuilder.FieldDesc("Author name."))

	// BatchFieldFunc: resolve author profiles for ALL articles in one call.
	// Instead of N individual lookups (one per article), the batch resolver
	// receives every Article at once and returns one AuthorProfile per item.
	article.BatchFieldFunc("authorProfile", func(ctx context.Context, articles []*Article) ([]*AuthorProfile, error) {
		profiles := make([]*AuthorProfile, len(articles))
		for i, a := range articles {
			profiles[i] = &AuthorProfile{
				Name: a.Author,
				Bio:  fmt.Sprintf("Bio of %s (batch-resolved)", a.Author),
			}
		}
		return profiles, nil
	}, schemabuilder.FieldDesc("Full author profile resolved via automatic batching."))

	// BatchFieldFunc with args: compute a preview of the body for all articles.
	article.BatchFieldFunc("preview", func(articles []*Article, args struct{ MaxLen int32 }) ([]string, error) {
		out := make([]string, len(articles))
		for i, a := range articles {
			r := []rune(a.Body)
			if int32(len(r)) > args.MaxLen {
				r = r[:args.MaxLen]
			}
			out[i] = string(r)
		}
		return out, nil
	}, schemabuilder.FieldDesc("Truncated article preview (batch-resolved with args)."))

	// AuthorProfile — returned by the batch field above.
	profile := sb.Object("AuthorProfile", AuthorProfile{},
		schemabuilder.WithDescription("Author profile resolved via batch/DataLoader."))
	profile.FieldFunc("name", func(p AuthorProfile) string { return p.Name },
		schemabuilder.FieldDesc("Author display name."))
	profile.FieldFunc("bio", func(p AuthorProfile) string { return p.Bio },
		schemabuilder.FieldDesc("Short author biography."))

	// AuditLog — plain object.
	auditLog := sb.Object("AuditLog", AuditLog{}, schemabuilder.WithDescription("An audit log entry."))
	auditLog.FieldFunc("action", func(l AuditLog) string { return l.Action },
		schemabuilder.FieldDesc("The action that was audited."))
	auditLog.FieldFunc("at", func(l AuditLog) string { return l.At },
		schemabuilder.FieldDesc("Timestamp when the audit entry was recorded."))

	// AdminStats — type-level @auth directive: every field on this object
	// automatically inherits the @auth pre-resolver check.
	obj := sb.Object("AdminStats", AdminStats{},
		schemabuilder.WithDescription("Server statistics (admin only)."),
		schemabuilder.WithDirective("auth"), // type-level: propagates to all fields
	)
	obj.FieldFunc("totalArticles", func(_ AdminStats) int32 { return 42 },
		schemabuilder.FieldDesc("Total number of articles."),
	)
	obj.FieldFunc("totalUsers", func(_ AdminStats) int32 { return 7 },
		schemabuilder.FieldDesc("Total number of registered users."),
	)
}

// RegisterQueries registers Query fields demonstrating various directive usages.
func RegisterQueries(sb *schemabuilder.Schema, s *Server) {
	q := sb.Query()

	// Public field — no directive; always accessible.
	q.FieldFunc("articles", func(ctx context.Context) []*Article {
		return s.articles
	}, schemabuilder.FieldDesc("List all articles (public)."))

	// @hasRole — only ADMIN can access.
	q.FieldFunc("secretArticle", func(ctx context.Context) *Article {
		if len(s.articles) > 0 {
			return s.articles[0]
		}
		return nil
	},
		schemabuilder.FieldDesc("A secret article visible only to admins."),
		schemabuilder.WithFieldDirective("hasRole", map[string]interface{}{"role": "ADMIN"}),
	)

	// @featureFlag — guarded by "new-dashboard" flag; null if disabled.
	q.FieldFunc("dashboard", func(ctx context.Context) *string {
		msg := "Welcome to the new dashboard!"
		return &msg
	},
		schemabuilder.FieldDesc("New dashboard (feature-flagged)."),
		schemabuilder.WithFieldDirective("featureFlag", map[string]interface{}{"flag": "new-dashboard"}),
	)

	// @cacheControl — metadata-only; no runtime impact.
	q.FieldFunc("cachedArticles", func(ctx context.Context) []*Article {
		return s.articles
	},
		schemabuilder.FieldDesc("Cached article list with cache hints for CDN."),
		schemabuilder.WithFieldDirective("cacheControl", map[string]interface{}{"maxAge": 300, "scope": "PUBLIC"}),
	)

	// AdminStats — type-level @auth (every field auto-guarded).
	q.FieldFunc("adminStats", func(ctx context.Context) *AdminStats {
		return &AdminStats{}
	},
		schemabuilder.FieldDesc("Server statistics (requires authentication)."),
	)

	// @audit — PostResolver records an audit entry after resolution.
	q.FieldFunc("auditedArticle", func(ctx context.Context) *Article {
		if len(s.articles) > 0 {
			return s.articles[0]
		}
		return nil
	},
		schemabuilder.FieldDesc("Returns an article and logs an audit entry."),
		schemabuilder.WithFieldDirective("audit", map[string]interface{}{"action": "read_article"}),
	)

	// Audit log viewer.
	q.FieldFunc("auditLogs", func(ctx context.Context) []AuditLog {
		return s.auditLogs
	}, schemabuilder.FieldDesc("View recorded audit logs."))

	// Combined: @hasRole (PreResolver) + @audit (PostResolver) on same field.
	q.FieldFunc("adminAuditedArticle", func(ctx context.Context) *Article {
		if len(s.articles) > 0 {
			return s.articles[0]
		}
		return nil
	},
		schemabuilder.FieldDesc("Admin-only article that also logs an audit entry."),
		schemabuilder.WithFieldDirective("hasRole", map[string]interface{}{"role": "ADMIN"}),
		schemabuilder.WithFieldDirective("audit", map[string]interface{}{"action": "admin_read"}),
	)
}

// RegisterMutations registers Mutation fields demonstrating directive usage.
func RegisterMutations(sb *schemabuilder.Schema, s *Server) {
	m := sb.Mutation()

	// @hasRole on mutation — only EDITOR can create articles.
	// @rateLimit — at most once per 2 seconds.
	m.FieldFunc("createArticle", func(ctx context.Context, args struct {
		Title  string
		Body   string
		Author string
	}) (*Article, error) {
		a := &Article{
			ID:     schemabuilder.ID{Value: fmt.Sprintf("a%d", len(s.articles)+1)},
			Title:  args.Title,
			Body:   args.Body,
			Author: args.Author,
		}
		s.articles = append(s.articles, a)
		return a, nil
	},
		schemabuilder.FieldDesc("Create a new article (requires EDITOR role, rate limited)."),
		schemabuilder.WithFieldDirective("hasRole", map[string]interface{}{"role": "EDITOR"}),
		schemabuilder.WithFieldDirective("rateLimit", map[string]interface{}{"window": 2, "key": "create-article"}),
	)
}
