package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"go.appointy.com/jaal/introspection"
	"go.appointy.com/jaal/sdl"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		return nil
	}

	command := args[0]

	switch command {
	case "introspect":
		return runIntrospect(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printUsage() {
	fmt.Println(`jaal - CLI tool for GraphQL schema generation

Usage:
  jaal introspect <url> [options]

Description:
  Fetches introspection data from a GraphQL server and generates a schema.graphql file.

Options:
  -H, --header <key:value>       Add custom header (can be repeated)
  -o, --output <file>            Output schema.graphql file (default: schema.graphql)
  --introspection <file>         Save introspection JSON to file (optional)
  --stdout                       Output schema to stdout instead of file
  --spec <version>               GraphQL spec version: 2018, 2021, or 2025 (default: 2025)

Examples:
  jaal introspect http://localhost:8080/graphql
  jaal introspect http://localhost:8080/graphql -o my-schema.graphql
  jaal introspect http://localhost:8080/graphql -H "Authorization:Bearer token"
  jaal introspect http://localhost:8080/graphql --introspection introspection.json
  jaal introspect http://localhost:8080/graphql --stdout
  jaal introspect http://localhost:8080/graphql --spec 2018
  jaal introspect http://localhost:8080/graphql --spec 2021`)
}

// introspectConfig holds configuration for the introspect command
type introspectConfig struct {
	url           string
	headers       map[string]string
	output        string
	introspection string
	stdout        bool
	specVersion   introspection.SpecVersion
}

func runIntrospect(args []string) error {
	config, err := parseIntrospectArgs(args)
	if err != nil {
		return err
	}

	// Fetch introspection JSON using the appropriate query for the spec version
	introspectionJSON, err := fetchIntrospection(config.url, config.headers, config.specVersion)
	if err != nil {
		return fmt.Errorf("failed to fetch introspection: %w", err)
	}

	// Optionally save introspection JSON to file
	if config.introspection != "" {
		if err := os.WriteFile(config.introspection, introspectionJSON, 0644); err != nil {
			return fmt.Errorf("failed to write introspection file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Introspection JSON saved to %s\n", config.introspection)
	}

	// Convert to SDL
	sdlOutput, err := sdl.IntrospectionJSONToSDL(introspectionJSON)
	if err != nil {
		return fmt.Errorf("failed to convert to SDL: %w", err)
	}

	// Output the schema
	if config.stdout {
		fmt.Println(sdlOutput)
	} else {
		outputFile := config.output
		if outputFile == "" {
			outputFile = "schema.graphql"
		}
		if err := os.WriteFile(outputFile, []byte(sdlOutput), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Schema saved to %s\n", outputFile)
	}

	return nil
}

func parseIntrospectArgs(args []string) (*introspectConfig, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing URL argument\n\nUsage: jaal introspect <url> [options]")
	}

	config := &introspectConfig{
		headers:     make(map[string]string),
		specVersion: introspection.Spec2025,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "-H" || arg == "--header":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			headerValue := args[i+1]
			key, value, err := parseHeader(headerValue)
			if err != nil {
				return nil, err
			}
			config.headers[key] = value
			i += 2

		case arg == "-o" || arg == "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			config.output = args[i+1]
			i += 2

		case arg == "--introspection":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			config.introspection = args[i+1]
			i += 2

		case arg == "--stdout":
			config.stdout = true
			i++

		case arg == "--spec":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			specStr := args[i+1]
			config.specVersion = introspection.ParseSpecVersion(specStr)
			i += 2

		case strings.HasPrefix(arg, "-"):
			return nil, fmt.Errorf("unknown flag: %s", arg)

		default:
			// This is the URL
			if config.url == "" {
				config.url = arg
			} else {
				return nil, fmt.Errorf("unexpected argument: %s", arg)
			}
			i++
		}
	}

	if config.url == "" {
		return nil, fmt.Errorf("missing URL argument")
	}

	return config, nil
}

func parseHeader(headerValue string) (string, string, error) {
	// Support both "Key:Value" and "Key: Value" formats
	parts := strings.SplitN(headerValue, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid header format, expected 'Key:Value' or 'Key: Value', got: %s", headerValue)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", fmt.Errorf("header key cannot be empty")
	}
	return key, value, nil
}

func fetchIntrospection(url string, headers map[string]string, specVersion introspection.SpecVersion) ([]byte, error) {
	// Get the appropriate introspection query for the spec version
	query := introspection.GetIntrospectionQuery(specVersion)

	// Create the introspection query request
	requestBody := map[string]interface{}{
		"query": query,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, strings.NewReader(string(requestJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response to extract the data field
	var response struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", response.Errors)
	}

	// Pretty print the introspection data
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, response.Data, "", "  "); err != nil {
		return nil, fmt.Errorf("failed to format JSON: %w", err)
	}

	return prettyJSON.Bytes(), nil
}
