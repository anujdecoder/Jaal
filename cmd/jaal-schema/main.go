package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"go.appointy.com/jaal"
)

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func main() {
	url := flag.String("url", "http://localhost:8080/graphql", "URL of the GraphQL server")
	outputJSON := flag.String("output-json", "", "Output introspection JSON file path (optional)")
	outputSDL := flag.String("output-sdl", "schema.graphql", "Output SDL file path")
	
	var headers headerFlags
	flag.Var(&headers, "header", "Custom HTTP headers in Key:Value format (can be specified multiple times)")
	
	flag.Parse()

	h := http.Header{}
	h.Set("User-Agent", "jaal-schema/1.0")

	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid header format: %s. Expected Key:Value", header)
		}
		h.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	client := jaal.NewHttpClient(http.DefaultClient, *url, h)

	var response IntrospectionResponse
	var err error

	queries := []struct {
		name  string
		query string
	}{
		{"September 2025", introspectionQuery2025},
		{"2021", introspectionQuery2021},
		{"2018", introspectionQuery2018},
	}

	success := false
	for _, q := range queries {
		fmt.Printf("Attempting introspection using %s spec...\n", q.name)
		err = client.Do(q.query, nil, &response)
		if err == nil {
			success = true
			fmt.Printf("Successfully fetched introspection using %s spec.\n", q.name)
			break
		}
		fmt.Printf("Failed to fetch introspection using %s spec: %v\n", q.name, err)
	}

	if !success {
		log.Fatalf("Failed to fetch introspection after trying all spec versions.")
	}

	if *outputJSON != "" {
		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal introspection response: %v", err)
		}

		err = os.WriteFile(*outputJSON, data, 0644)
		if err != nil {
			log.Fatalf("Failed to write JSON to file: %v", err)
		}
		fmt.Printf("Introspection JSON saved to %s\n", *outputJSON)
	}

	sdl := ConvertToSDL(response)
	err = os.WriteFile(*outputSDL, []byte(sdl), 0644)
	if err != nil {
		log.Fatalf("Failed to write SDL to file: %v", err)
	}

	fmt.Printf("SDL saved to %s\n", *outputSDL)
}
