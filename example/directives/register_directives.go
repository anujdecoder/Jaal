package directives

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.appointy.com/jaal/schemabuilder"
)

// Context keys used by directive handlers.
type ctxKeyRole struct{}
type ctxKeyFeatures struct{}

// RegisterDirectives registers all custom directive definitions on the schema.
// This must be called before Build().
//
// Demonstrates:
//   - @hasRole    : PreResolver + ErrorOnFail  (access control by role)
//   - @featureFlag: PreResolver + SkipOnFail   (feature gate; returns null silently)
//   - @rateLimit  : PreResolver + ErrorOnFail  (simple rate limiter with args)
//   - @audit      : PostResolver + ErrorOnFail (post-resolve audit log)
//   - @cacheControl: metadata-only (nil handler; introspection hints)
//   - @auth       : PreResolver + ErrorOnFail  (used at type level for AdminStats)
func RegisterDirectives(sb *schemabuilder.Schema, s *Server) {

	// -----------------------------------------------------------------------
	// 1. @hasRole — PreResolver + ErrorOnFail (default)
	//
	// Checks that the caller's role (from context) matches the required role.
	// If not, the field returns an error immediately (resolver never runs).
	// -----------------------------------------------------------------------
	sb.Directive("hasRole", &schemabuilder.DirectiveDefinition{
		Description: "Restricts field access to callers with the specified role.",
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationFieldDefinition,
		},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "role", TypeName: "String", Description: "The required role (e.g. ADMIN, EDITOR)."},
		},
		ExecutionOrder: schemabuilder.PreResolver, // default
		OnFail:         schemabuilder.ErrorOnFail, // default
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			required, _ := args["role"].(string)
			actual, _ := ctx.Value(ctxKeyRole{}).(string)
			if actual != required {
				return fmt.Errorf("access denied: requires role %s, got %s", required, actual)
			}
			return nil
		},
	})

	// -----------------------------------------------------------------------
	// 2. @featureFlag — PreResolver + SkipOnFail
	//
	// Guards a field behind a feature flag name stored in context.  If the flag
	// is not enabled, the field silently returns null instead of an error.
	// -----------------------------------------------------------------------
	sb.Directive("featureFlag", &schemabuilder.DirectiveDefinition{
		Description: "Guards a field behind a feature flag; returns null if disabled.",
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationFieldDefinition,
		},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "flag", TypeName: "String", Description: "Feature flag name."},
		},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.SkipOnFail, // silent null
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			flagName, _ := args["flag"].(string)
			enabled, _ := ctx.Value(ctxKeyFeatures{}).(map[string]bool)
			if !enabled[flagName] {
				return errors.New("feature disabled")
			}
			return nil
		},
	})

	// -----------------------------------------------------------------------
	// 3. @rateLimit — PreResolver + ErrorOnFail
	//
	// A simplified per-field rate limiter that tracks the last call time and
	// rejects calls that arrive sooner than the configured window (seconds).
	// -----------------------------------------------------------------------
	var (
		rateMu     sync.Mutex
		rateLimits = map[string]time.Time{} // field-key → last access
	)

	sb.Directive("rateLimit", &schemabuilder.DirectiveDefinition{
		Description: "Rejects the field if called more than once within the window.",
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationFieldDefinition,
		},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "window", TypeName: "Int", Description: "Minimum seconds between calls."},
			{Name: "key", TypeName: "String", Description: "Unique key identifying this limit bucket."},
		},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			key, _ := args["key"].(string)
			windowSec := 1 // default 1 second
			if w, ok := args["window"].(int); ok {
				windowSec = w
			}
			rateMu.Lock()
			defer rateMu.Unlock()

			last, exists := rateLimits[key]
			now := time.Now()
			if exists && now.Sub(last) < time.Duration(windowSec)*time.Second {
				return fmt.Errorf("rate limited: try again in %d seconds", windowSec)
			}
			rateLimits[key] = now
			return nil
		},
	})

	// -----------------------------------------------------------------------
	// 4. @audit — PostResolver + ErrorOnFail
	//
	// Records an audit log entry after the field resolver succeeds.  The handler
	// always passes (returns nil) so the resolved value is returned normally.
	// If you wanted to block a response based on the result you could return an
	// error here.
	// -----------------------------------------------------------------------
	sb.Directive("audit", &schemabuilder.DirectiveDefinition{
		Description: "Records an audit log entry after the field resolves.",
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationFieldDefinition,
		},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "action", TypeName: "String", Description: "Action name for the audit log."},
		},
		ExecutionOrder: schemabuilder.PostResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			action, _ := args["action"].(string)
			s.auditLogs = append(s.auditLogs, AuditLog{
				Action: action,
				At:     time.Now().Format(time.RFC3339),
			})
			return nil // pass — resolved value is returned
		},
	})

	// -----------------------------------------------------------------------
	// 5. @cacheControl — metadata-only (nil handler)
	//
	// Attaches cache-control hints visible in introspection but has no runtime
	// effect on query execution.  Tools or gateway layers can read these hints
	// from the introspection schema.
	// -----------------------------------------------------------------------
	sb.Directive("cacheControl", &schemabuilder.DirectiveDefinition{
		Description: "Provides cache-control hints for gateway/CDN layers (metadata only).",
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationFieldDefinition,
			schemabuilder.LocationObject,
		},
		Args: []schemabuilder.DirectiveArgDef{
			{Name: "maxAge", TypeName: "Int", Description: "Max age in seconds."},
			{Name: "scope", TypeName: "String", Description: "PUBLIC or PRIVATE.", DefaultValue: "PUBLIC"},
		},
		Handler: nil, // metadata only — no runtime handler
	})

	// -----------------------------------------------------------------------
	// 6. @auth — PreResolver + ErrorOnFail  (used at TYPE level)
	//
	// Requires that the context carries any role.  Applied to AdminStats as a
	// type-level directive; propagates to every field on that object.
	// -----------------------------------------------------------------------
	sb.Directive("auth", &schemabuilder.DirectiveDefinition{
		Description: "Requires the caller to be authenticated (any role).",
		Locations: []schemabuilder.DirectiveLocation{
			schemabuilder.LocationObject,
			schemabuilder.LocationFieldDefinition,
		},
		ExecutionOrder: schemabuilder.PreResolver,
		OnFail:         schemabuilder.ErrorOnFail,
		Handler: func(ctx context.Context, args map[string]interface{}) error {
			if ctx.Value(ctxKeyRole{}) == nil {
				return errors.New("unauthenticated: login required")
			}
			return nil
		},
	})
}
