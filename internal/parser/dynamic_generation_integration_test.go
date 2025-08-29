package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TestDynamicDataGeneration_Integration tests the enhanced generateExampleFromSchema function
// with realistic mock data generation capabilities as outlined in dynamic-data-generation.md
func TestDynamicDataGeneration_Integration(t *testing.T) {
	t.Run("EnhancedStringGeneration", testEnhancedStringGeneration)
	t.Run("NumericConstraintsAndRanges", testNumericConstraintsAndRanges)
	t.Run("PatternBasedGeneration", testPatternBasedGeneration)
	t.Run("FieldNameIntelligence", testFieldNameIntelligence)
	t.Run("ArrayLengthVariation", testArrayLengthVariation)
	t.Run("BackwardCompatibility", testBackwardCompatibility)
}

func testEnhancedStringGeneration(t *testing.T) {
	tests := []struct {
		name           string
		schema         *openapi3.Schema
		expectedFormat func(interface{}) bool
		description    string
	}{
		{
			name: "email_format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "email",
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
				return emailRegex.MatchString(str)
			},
			description: "should generate realistic email addresses",
		},
		{
			name: "uuid_format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "uuid",
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
				return uuidRegex.MatchString(str)
			},
			description: "should generate valid UUID format",
		},
		{
			name: "uri_format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "uri",
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
			},
			description: "should generate valid URI format",
		},
		{
			name: "date_format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "date",
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				_, err := time.Parse("2006-01-02", str)
				return err == nil
			},
			description: "should generate valid date format (YYYY-MM-DD)",
		},
		{
			name: "date_time_format",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "date-time",
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				_, err := time.Parse(time.RFC3339, str)
				return err == nil
			},
			description: "should generate valid RFC3339 date-time format",
		},
		{
			name: "string_with_minLength",
			schema: &openapi3.Schema{
				Type:      &openapi3.Types{"string"},
				MinLength: 10,
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				return len(str) >= 10
			},
			description: "should respect minLength constraint",
		},
		{
			name: "string_with_maxLength",
			schema: &openapi3.Schema{
				Type:      &openapi3.Types{"string"},
				MaxLength: openapi3.Uint64Ptr(5),
			},
			expectedFormat: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				return len(str) <= 5
			},
			description: "should respect maxLength constraint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleFromSchema(tt.schema)

			if !tt.expectedFormat(result) {
				t.Errorf("generateExampleFromSchema() = %v, %s", result, tt.description)
			}

			// Ensure it's not the static "string" value
			if str, ok := result.(string); ok && str == "string" {
				t.Errorf("Expected enhanced string generation, but got static 'string' value")
			}
		})
	}
}

func testNumericConstraintsAndRanges(t *testing.T) {
	tests := []struct {
		name        string
		schema      *openapi3.Schema
		validator   func(interface{}) bool
		description string
	}{
		{
			name: "integer_with_minimum",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"integer"},
				Min:  openapi3.Float64Ptr(10),
			},
			validator: func(v interface{}) bool {
				if num, ok := v.(int); ok {
					return num >= 10
				}
				if num, ok := v.(float64); ok {
					return num >= 10
				}
				return false
			},
			description: "should respect minimum constraint for integers",
		},
		{
			name: "integer_with_maximum",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"integer"},
				Max:  openapi3.Float64Ptr(100),
			},
			validator: func(v interface{}) bool {
				if num, ok := v.(int); ok {
					return num <= 100
				}
				if num, ok := v.(float64); ok {
					return num <= 100
				}
				return false
			},
			description: "should respect maximum constraint for integers",
		},
		{
			name: "number_with_range",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"number"},
				Min:  openapi3.Float64Ptr(1.5),
				Max:  openapi3.Float64Ptr(99.9),
			},
			validator: func(v interface{}) bool {
				if num, ok := v.(float64); ok {
					return num >= 1.5 && num <= 99.9
				}
				return false
			},
			description: "should respect min/max range for numbers",
		},
		{
			name: "integer_with_multipleOf",
			schema: &openapi3.Schema{
				Type:       &openapi3.Types{"integer"},
				MultipleOf: openapi3.Float64Ptr(5),
			},
			validator: func(v interface{}) bool {
				if num, ok := v.(int); ok {
					return num%5 == 0
				}
				if num, ok := v.(float64); ok {
					return int(num)%5 == 0
				}
				return false
			},
			description: "should respect multipleOf constraint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleFromSchema(tt.schema)

			if !tt.validator(result) {
				t.Errorf("generateExampleFromSchema() = %v, %s", result, tt.description)
			}

			// Ensure it's not the static 0 value when constraints are specified
			if (result == 0 || result == 0.0) && (tt.schema.Min != nil || tt.schema.Max != nil) {
				t.Errorf("Expected constrained numeric generation, but got static zero value")
			}
		})
	}
}

func testPatternBasedGeneration(t *testing.T) {
	tests := []struct {
		name        string
		schema      *openapi3.Schema
		validator   func(interface{}) bool
		description string
	}{
		{
			name: "phone_number_pattern",
			schema: &openapi3.Schema{
				Type:    &openapi3.Types{"string"},
				Pattern: `^\+1-\d{3}-\d{3}-\d{4}$`,
			},
			validator: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				matched, _ := regexp.MatchString(`^\+1-\d{3}-\d{3}-\d{4}$`, str)
				return matched
			},
			description: "should generate strings matching regex pattern for phone numbers",
		},
		{
			name: "product_code_pattern",
			schema: &openapi3.Schema{
				Type:    &openapi3.Types{"string"},
				Pattern: `^[A-Z]{2}\d{4}$`,
			},
			validator: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				matched, _ := regexp.MatchString(`^[A-Z]{2}\d{4}$`, str)
				return matched
			},
			description: "should generate strings matching regex pattern for product codes",
		},
		{
			name: "simple_alphanumeric_pattern",
			schema: &openapi3.Schema{
				Type:    &openapi3.Types{"string"},
				Pattern: `^[a-zA-Z0-9]{8}$`,
			},
			validator: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				matched, _ := regexp.MatchString(`^[a-zA-Z0-9]{8}$`, str)
				return matched
			},
			description: "should generate strings matching alphanumeric pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleFromSchema(tt.schema)

			if !tt.validator(result) {
				t.Errorf("generateExampleFromSchema() = %v, %s", result, tt.description)
			}

			// Ensure it's not the static "string" value
			if str, ok := result.(string); ok && str == "string" {
				t.Errorf("Expected pattern-based generation, but got static 'string' value")
			}
		})
	}
}

func testFieldNameIntelligence(t *testing.T) {
	// Create schemas with specific property names that should trigger intelligent generation
	tests := []struct {
		name         string
		propertyName string
		schema       *openapi3.Schema
		validator    func(interface{}) bool
		description  string
	}{
		{
			name:         "firstName_field",
			propertyName: "firstName",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			},
			validator: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				// Should be a realistic first name (not "string")
				return str != "string" && len(str) > 1 && cases.Title(language.English).String(str) == str
			},
			description: "should generate realistic first names",
		},
		{
			name:         "email_field",
			propertyName: "email",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			},
			validator: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
				return emailRegex.MatchString(str)
			},
			description: "should generate realistic email addresses based on field name",
		},
		{
			name:         "phone_field",
			propertyName: "phone",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			},
			validator: func(v interface{}) bool {
				str, ok := v.(string)
				if !ok {
					return false
				}
				// Should be a phone number format, not "string"
				return str != "string" && (strings.Contains(str, "-") || strings.Contains(str, "(") || len(str) >= 10)
			},
			description: "should generate realistic phone numbers based on field name",
		},
		{
			name:         "age_field",
			propertyName: "age",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"integer"},
			},
			validator: func(v interface{}) bool {
				if num, ok := v.(int); ok {
					return num >= 18 && num <= 80
				}
				return false
			},
			description: "should generate realistic age values (18-80)",
		},
		{
			name:         "price_field",
			propertyName: "price",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"number"},
			},
			validator: func(v interface{}) bool {
				if num, ok := v.(float64); ok {
					return num >= 10.0 && num <= 999.99
				}
				return false
			},
			description: "should generate realistic price values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an object schema with the specific property name
			objectSchema := &openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: map[string]*openapi3.SchemaRef{
					tt.propertyName: {Value: tt.schema},
				},
			}

			result := generateExampleFromSchema(objectSchema)

			// Extract the property value from the generated object
			if obj, ok := result.(map[string]interface{}); ok {
				if propValue, exists := obj[tt.propertyName]; exists {
					if !tt.validator(propValue) {
						t.Errorf("generateExampleFromSchema() property %s = %v, %s", tt.propertyName, propValue, tt.description)
					}
				} else {
					t.Errorf("Property %s not found in generated object", tt.propertyName)
				}
			} else {
				t.Errorf("Expected object, got %T", result)
			}
		})
	}
}

func testArrayLengthVariation(t *testing.T) {
	tests := []struct {
		name        string
		schema      *openapi3.Schema
		validator   func(interface{}) bool
		description string
	}{
		{
			name: "array_with_minItems",
			schema: &openapi3.Schema{
				Type:     &openapi3.Types{"array"},
				MinItems: 3,
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
				},
			},
			validator: func(v interface{}) bool {
				if arr, ok := v.([]interface{}); ok {
					return len(arr) >= 3
				}
				return false
			},
			description: "should respect minItems constraint",
		},
		{
			name: "array_with_maxItems",
			schema: &openapi3.Schema{
				Type:     &openapi3.Types{"array"},
				MaxItems: openapi3.Uint64Ptr(2),
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
				},
			},
			validator: func(v interface{}) bool {
				if arr, ok := v.([]interface{}); ok {
					return len(arr) <= 2
				}
				return false
			},
			description: "should respect maxItems constraint",
		},
		{
			name: "array_with_uniqueItems",
			schema: &openapi3.Schema{
				Type:        &openapi3.Types{"array"},
				UniqueItems: true,
				MinItems:    3,
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}},
				},
			},
			validator: func(v interface{}) bool {
				if arr, ok := v.([]interface{}); ok {
					if len(arr) < 3 {
						return false
					}
					// Check for uniqueness
					seen := make(map[interface{}]bool)
					for _, item := range arr {
						if seen[item] {
							return false // Duplicate found
						}
						seen[item] = true
					}
					return true
				}
				return false
			},
			description: "should generate unique items when uniqueItems is true",
		},
		{
			name: "array_without_constraints",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"array"},
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
				},
			},
			validator: func(v interface{}) bool {
				if arr, ok := v.([]interface{}); ok {
					// Should generate 1-3 items by default (not just 1)
					return len(arr) >= 1 && len(arr) <= 3
				}
				return false
			},
			description: "should generate 1-3 items when no constraints specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleFromSchema(tt.schema)

			if !tt.validator(result) {
				t.Errorf("generateExampleFromSchema() = %v, %s", result, tt.description)
			}
		})
	}
}

func testBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		schema      *openapi3.Schema
		expected    interface{}
		description string
	}{
		{
			name: "schema_with_explicit_example",
			schema: &openapi3.Schema{
				Type:    &openapi3.Types{"string"},
				Example: "explicit_example_value",
			},
			expected:    "explicit_example_value",
			description: "should return explicit example when provided",
		},
		{
			name: "enum_schema",
			schema: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
				Enum: []interface{}{"option1", "option2", "option3"},
			},
			expected:    "option1", // Current implementation returns first enum value
			description: "should return enum value (current behavior)",
		},
		{
			name:        "null_schema",
			schema:      nil,
			expected:    nil,
			description: "should handle null schema gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleFromSchema(tt.schema)

			if result != tt.expected {
				t.Errorf("generateExampleFromSchema() = %v, expected %v, %s", result, tt.expected, tt.description)
			}
		})
	}
}

// TestDynamicDataGeneration_EndToEnd tests the complete integration with actual OpenAPI specs
func TestDynamicDataGeneration_EndToEnd(t *testing.T) {
	parser, err := New("../../examples/petstore.yaml")
	if err != nil {
		t.Skipf("Skipping end-to-end test due to parser error: %v", err)
	}

	routes := parser.GetRoutes()
	if len(routes) == 0 {
		t.Skip("No routes found in petstore.yaml")
	}

	// Test various status codes to ensure enhanced generation works in real scenarios
	statusCodes := []string{"200", "201", "400", "404"}

	for _, route := range routes[:min(len(routes), 3)] { // Test first 3 routes only
		for _, code := range statusCodes {
			t.Run(fmt.Sprintf("route_%s_status_%s", route.Path, code), func(t *testing.T) {
				result, err := parser.GetExampleResponse(route.Operation, code)
				if err != nil {
					t.Logf("No example for status %s on route %s: %v", code, route.Path, err)
					return
				}

				// Validate that the result is not using static values
				resultJSON, _ := json.Marshal(result)
				resultStr := string(resultJSON)

				// Check that we're not generating too many static "string" values
				staticStringCount := strings.Count(resultStr, `"string"`)
				if staticStringCount > 2 {
					t.Logf("Warning: Generated response contains %d static 'string' values for %s %s. Enhanced generation may not be working.",
						staticStringCount, route.Path, code)
				}

				// Check for static numeric values
				if strings.Contains(resultStr, `"0"`) || strings.Contains(resultStr, `:0,`) || strings.Contains(resultStr, `:0}`) {
					t.Logf("Warning: Generated response contains static zero values for %s %s", route.Path, code)
				}

				t.Logf("Generated example for %s %s: %s", route.Path, code, resultStr)
			})
		}
	}
}

// Helper function for Go versions that don't have min function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
