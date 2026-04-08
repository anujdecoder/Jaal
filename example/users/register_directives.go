package users

import (
	"context"
	"fmt"
	"strings"

	"go.appointy.com/jaal/graphql"
	"go.appointy.com/jaal/schemabuilder"
)

// RegisterDirectives registers custom directives for the users example.
// This demonstrates how to create and use custom directives in a real application.
func RegisterDirectives(sb *schemabuilder.Schema) {
	// @auth directive - requires authentication with optional role check
	sb.Directive("auth",
		schemabuilder.DirectiveDescription("Requires authentication. Optionally checks for a specific role."),
		schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
		schemabuilder.DirectiveArgString("role"),
		schemabuilder.DirectiveVisitorFunc(authDirectiveVisitor),
	)

	// @cache directive - caches field results (repeatable for multiple cache configs)
	sb.Directive("cache",
		schemabuilder.DirectiveDescription("Caches the field result for the specified TTL in seconds."),
		schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
		schemabuilder.DirectiveArgInt("ttl"),
		schemabuilder.DirectiveRepeatable(),
		schemabuilder.DirectiveVisitorFunc(cacheDirectiveVisitor),
	)

	// @uppercase directive - transforms string output to uppercase
	sb.Directive("uppercase",
		schemabuilder.DirectiveDescription("Transforms string output to uppercase."),
		schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
		schemabuilder.DirectiveVisitorFunc(uppercaseDirectiveVisitor),
	)

	// @log directive - logs field access (demonstrates side-effect directives)
	sb.Directive("log",
		schemabuilder.DirectiveDescription("Logs field access for debugging."),
		schemabuilder.DirectiveLocations(graphql.LocationFieldDefinition),
		schemabuilder.DirectiveArgString("message"),
		schemabuilder.DirectiveVisitorFunc(logDirectiveVisitor),
	)
}

// authDirectiveVisitor implements authentication checking.
// It validates that the user is authenticated and optionally has the required role.
func authDirectiveVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
	// Get user from context (set by middleware)
	user, ok := ctx.Value("user").(*User)
	if !ok || user == nil {
		return nil, fmt.Errorf("unauthorized: authentication required")
	}

	// Check role if specified
	if role, ok := d.Args["role"].(string); ok && role != "" {
		if string(user.Role) != role {
			return nil, fmt.Errorf("forbidden: requires role %s", role)
		}
	}

	// Continue with normal resolution
	return nil, nil
}

// cacheDirectiveVisitor implements caching (simplified in-memory cache).
// In a real application, this would use Redis or similar.
var cacheStore = make(map[string]interface{})

func cacheDirectiveVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
	// Generate cache key (simplified - in real app, include args)
	key := fmt.Sprintf("%p:%v", f, src)

	// Check cache
	if cached, ok := cacheStore[key]; ok {
		return cached, nil
	}

	// Execute resolver and cache result
	result, err := f.Resolve(ctx, src, nil, nil)
	if err != nil {
		return nil, err
	}

	cacheStore[key] = result
	return nil, nil // Continue with normal resolution
}

// uppercaseDirectiveVisitor transforms string output to uppercase.
func uppercaseDirectiveVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
	// Execute the resolver first
	result, err := f.Resolve(ctx, src, nil, nil)
	if err != nil {
		return nil, err
	}

	// Transform if string
	if str, ok := result.(string); ok {
		return strings.ToUpper(str), nil
	}

	// Return as-is if not a string
	return result, nil
}

// logDirectiveVisitor logs field access.
func logDirectiveVisitor(ctx context.Context, d *graphql.DirectiveInstance, f *graphql.Field, src interface{}) (interface{}, error) {
	message := "field access"
	if msg, ok := d.Args["message"].(string); ok && msg != "" {
		message = msg
	}

	fmt.Printf("[LOG] %p: %s\n", f, message)

	// Continue with normal resolution
	return nil, nil
}
