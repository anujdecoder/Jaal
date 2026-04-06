package users

import (
	"context"
	"fmt"

	"go.appointy.com/jaal/schemabuilder"
)

// RegisterQuery registers query fields (e.g., me/user/allUsers). Specific funcs
// can be added (e.g., RegisterMeQuery); aggregator per task. Pattern from
// original RegisterQuery + README.md FieldFunc resolvers (ctx, args struct).
func RegisterQuery(sb *schemabuilder.Schema, s *Server) {
	q := sb.Query()

	// me: simple resolver returning User (from Server; w/ description for FIELD_DEFINITION).
	q.FieldFunc("me", func(ctx context.Context) *User {
		if len(s.users) > 0 {
			return s.users[0]
		}
		return nil
	}, schemabuilder.FieldDesc("Returns the current authenticated user (if any)."))

	// user: arg by ID, resolver loop (error if not found; field desc).
	q.FieldFunc("user", func(ctx context.Context, args struct {
		ID schemabuilder.ID
	}) (*User, error) {
		for _, u := range s.users {
			if u.ID.Value == args.ID.Value {
				return u, nil
			}
		}
		return nil, fmt.Errorf("user not found")
	}, schemabuilder.FieldDesc("Fetch user by ID."))

	// allUsers: returns slice (field desc).
	q.FieldFunc("allUsers", func(ctx context.Context) []*User {
		return s.users
	}, schemabuilder.FieldDesc("Returns all users in the system."))

	// searchUsers: demonstrates argument deprecation - oldFilter is deprecated, use filter instead
	q.FieldFunc("searchUsers", func(ctx context.Context, args struct {
		OldFilter *string
		Filter    *string
		Limit     int32
	}) []*User {
		// Use filter if provided, otherwise fall back to oldFilter
		searchTerm := args.Filter
		if searchTerm == nil {
			searchTerm = args.OldFilter
		}

		var results []*User
		for _, u := range s.users {
			if searchTerm == nil {
				results = append(results, u)
			} else if contains(u.Name, *searchTerm) || contains(u.Email, *searchTerm) {
				results = append(results, u)
			}
			if int32(len(results)) >= args.Limit && args.Limit > 0 {
				break
			}
		}
		return results
	}, schemabuilder.FieldDesc("Search users by name or email."),
		schemabuilder.ArgDeprecation("oldFilter", "Use 'filter' instead. The oldFilter parameter will be removed in v2.0."))
}

// contains is a helper function for substring matching
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
