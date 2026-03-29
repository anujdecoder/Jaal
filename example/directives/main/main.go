// Command directives starts a GraphQL server that demonstrates custom
// directive registration in Jaal.
//
// Run:
//
//	go run ./example/directives/main
//
// Then open http://localhost:9000/graphql in a browser to use the Playground.
//
// Example queries to try:
//
//	# Public — no directive
//	{ articles { title author } }
//
//	# @hasRole — will return "access denied" (no role in context)
//	{ secretArticle { title } }
//
//	# @featureFlag — returns null (flag disabled)
//	{ dashboard }
//
//	# @cacheControl — metadata only, works normally
//	{ cachedArticles { title } }
//
//	# @auth (type-level) — returns "unauthenticated" error
//	{ adminStats { totalArticles totalUsers } }
//
//	# @audit (PostResolver) — logs an audit entry
//	{ auditedArticle { title } }
//	{ auditLogs { action at } }
//
//	# Introspection — shows all custom directives
//	{ __schema { directives { name description locations args { name } } } }
//
//	# Batch/DataLoader — authorProfile resolved once for all articles
//	{ articles { title authorProfile { name bio } } }
//
//	# Batch with args — preview truncated for all articles at once
//	{ articles { title preview(maxLen: 20) } }
package main

import (
	"fmt"
	"log"
	"net/http"

	"go.appointy.com/jaal/example/directives"
)

func main() {
	handler, err := directives.GetGraphqlServer()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Directives example server running at http://localhost:9000/graphql")
	log.Fatal(http.ListenAndServe(":9000", handler))
}
