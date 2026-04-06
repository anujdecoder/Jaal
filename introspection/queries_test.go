package introspection

import (
	"strings"
	"testing"
)

func TestGetIntrospectionQuery(t *testing.T) {
	tests := []struct {
		version  SpecVersion
		contains []string
	}{
		{
			version: Spec2018,
			contains: []string{
				"query IntrospectionQuery",
				"__schema",
				"queryType",
				"mutationType",
				"subscriptionType",
				"types",
				"directives",
			},
		},
		{
			version: Spec2021,
			contains: []string{
				"query IntrospectionQuery",
				"specifiedByURL",
				"isDeprecated",
				"deprecationReason",
			},
		},
		{
			version: Spec2025,
			contains: []string{
				"query IntrospectionQuery",
				"specifiedByURL",
				"isDeprecated",
				"deprecationReason",
				"directives {",
				"locations",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			query := GetIntrospectionQuery(tt.version)
			for _, expected := range tt.contains {
				if !strings.Contains(query, expected) {
					t.Errorf("expected query to contain %q, but it didn't", expected)
				}
			}
		})
	}
}

func TestParseSpecVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected SpecVersion
	}{
		{"2018", Spec2018},
		{"2021", Spec2021},
		{"2025", Spec2025},
		{"", Spec2025},
		{"invalid", Spec2025},
		{"2019", Spec2025},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseSpecVersion(tt.input)
			if result != tt.expected {
				t.Errorf("ParseSpecVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test that 2018 query does NOT contain version-specific fields
func TestIntrospectionQuery2018Exclusions(t *testing.T) {
	query := IntrospectionQuery2018

	// These fields should NOT be in the 2018 query
	exclusions := []string{
		"specifiedByURL",
	}

	for _, excluded := range exclusions {
		if strings.Contains(query, excluded) {
			t.Errorf("2018 query should not contain %q, but it does", excluded)
		}
	}
}

// Test that 2021 query contains specifiedByURL but not directives on __Type
func TestIntrospectionQuery2021Features(t *testing.T) {
	query := IntrospectionQuery2021

	// Should have specifiedByURL
	if !strings.Contains(query, "specifiedByURL") {
		t.Error("2021 query should contain specifiedByURL")
	}

	// Should have isDeprecated on InputValue
	if !strings.Contains(query, "isDeprecated") {
		t.Error("2021 query should contain isDeprecated")
	}
}

// Test that 2025 query contains all features
func TestIntrospectionQuery2025Features(t *testing.T) {
	query := IntrospectionQuery

	// Should have all features
	required := []string{
		"specifiedByURL",
		"isDeprecated",
		"deprecationReason",
	}

	for _, feature := range required {
		if !strings.Contains(query, feature) {
			t.Errorf("2025 query should contain %q", feature)
		}
	}
}
