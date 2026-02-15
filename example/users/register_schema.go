package users

import "go.appointy.com/jaal/schemabuilder"

// RegisterSchema orchestrates all registrations (scalars/enums/objects/inputs/
// queries/muts/subs) by calling specific aggregators (per task modularity).
// Allows testing full schema or partial (e.g., RegisterInputs alone); follows
// original RegisterSchema but split for readability. Called by GetGraphqlServer.
func RegisterSchema(sb *schemabuilder.Schema, s *Server) {
	// Order: scalars first (DateTime), then enums/objects/inputs, ops last.
	RegisterScalars(sb)
	RegisterEnums(sb)
	RegisterObjects(sb)
	RegisterInputs(sb)

	RegisterQuery(sb, s)
	RegisterMutation(sb, s)
	RegisterSubscription(sb)
}