package users

import (
	"context"
	"time"

	"go.appointy.com/jaal/schemabuilder"
)

// RegisterSubscription registers subs (e.g., currentTime). Specific + aggregator
// per task. Pattern from original RegisterSubscription + README functional
// resolver returning func() for chan-like.
func RegisterSubscription(sb *schemabuilder.Schema) {
	s := sb.Subscription()

	// currentTime: resolver returns func() time.Time (Jaal sub pattern).
	s.FieldFunc("currentTime", func(ctx context.Context) func() time.Time {
		return time.Now
	})
}