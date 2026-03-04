package sdl

import (
	"encoding/json"
	"fmt"
)

// IntrospectionJSONToSDL converts introspection JSON to SDL format.
// The input should be the JSON response from a GraphQL introspection query.
func IntrospectionJSONToSDL(jsonData []byte) (string, error) {
	var response IntrospectionResponse
	if err := json.Unmarshal(jsonData, &response); err != nil {
		return "", fmt.Errorf("failed to parse introspection JSON: %w", err)
	}

	printer := NewPrinter(response.Schema)
	return printer.Print(), nil
}

// IntrospectionDataToSDL converts parsed introspection data to SDL format.
func IntrospectionDataToSDL(schema Schema) string {
	printer := NewPrinter(schema)
	return printer.Print()
}
