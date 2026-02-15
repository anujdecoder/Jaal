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

	// me: simple resolver returning User (from Server).
	q.FieldFunc("me", func(ctx context.Context) *User {
		if len(s.users) > 0 {
			return s.users[0]
		}
		return nil
	})

	// user: arg by ID, resolver loop (error if not found).
	q.FieldFunc("user", func(ctx context.Context, args struct {
		ID schemabuilder.ID
	}) (*User, error) {
		for _, u := range s.users {
			if u.ID.Value == args.ID.Value {
				return u, nil
			}
		}
		return nil, fmt.Errorf("user not found")
	})

	// allUsers: returns slice.
	q.FieldFunc("allUsers", func(ctx context.Context) []*User {
		return s.users
	})
}