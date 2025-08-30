package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGenerator tests the creation of a new Generator.
func TestNewGenerator(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		g := New(Config{})
		assert.NotNil(t, g)
		assert.False(t, g.config.Deterministic)
		assert.Equal(t, 2, g.config.DefaultArrayLength)
		assert.NotNil(t, g.formatHandlers)
	})

	t.Run("Deterministic config", func(t *testing.T) {
		g := New(Config{Deterministic: true})
		assert.True(t, g.config.Deterministic)
		assert.NotNil(t, g.rand)
	})

	t.Run("Custom default array length", func(t *testing.T) {
		g := New(Config{DefaultArrayLength: 5})
		assert.Equal(t, 5, g.config.DefaultArrayLength)
	})
}

// TestGenerateDataFromString tests string data generation.
func TestGenerateDataFromString(t *testing.T) {
	g := New(Config{Deterministic: true, UseFieldNameForData: true})

	t.Run("Simple string", func(t *testing.T) {
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
	})

	t.Run("String with format", func(t *testing.T) {
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "email"}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
		assert.Contains(t, data.(string), "@")
	})

	t.Run("String with pattern", func(t *testing.T) {
		pattern := `^[a-z]{5}$`
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}, Pattern: pattern}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
		matched, err := regexp.MatchString(pattern, data.(string))
		assert.NoError(t, err)
		assert.True(t, matched)
	})

	t.Run("String with minLength", func(t *testing.T) {
		minLength := uint64(10)
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}, MinLength: minLength}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
		assert.GreaterOrEqual(t, len(data.(string)), int(minLength))
	})

	t.Run("String with maxLength", func(t *testing.T) {
		maxLength := uint64(5)
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}, MaxLength: &maxLength}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
		assert.LessOrEqual(t, len(data.(string)), int(maxLength))
	})

	t.Run("String from field name", func(t *testing.T) {
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}}
		ctx := GenerationContext{FieldName: "firstName"}
		data := g.GenerateDataWithContext(schema, ctx)
		assert.IsType(t, "", data)
		assert.NotEmpty(t, data.(string))
	})
}

// TestGenerateDataFromNumber tests number data generation.
func TestGenerateDataFromNumber(t *testing.T) {
	g := New(Config{Deterministic: true})

	t.Run("Simple number", func(t *testing.T) {
		schema := &openapi3.Schema{Type: &openapi3.Types{"number"}}
		data := g.GenerateData(schema)
		assert.IsType(t, 0.0, data)
	})

	t.Run("Number with min/max", func(t *testing.T) {
		min := 10.5
		max := 15.5
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"number"},
			Min:  &min,
			Max:  &max,
		}
		data := g.GenerateData(schema)
		val, ok := data.(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, val, min)
		assert.LessOrEqual(t, val, max)
	})

	t.Run("Number with multipleOf", func(t *testing.T) {
		multipleOf := 3.0
		schema := &openapi3.Schema{
			Type:       &openapi3.Types{"number"},
			MultipleOf: &multipleOf,
			Min:        &multipleOf, // Ensure generated value is not 0
		}
		data := g.GenerateData(schema)
		val, ok := data.(float64)
		require.True(t, ok)
		// Check if val is a multiple of multipleOf
		remainder := val / multipleOf
		assert.Equal(t, remainder, float64(int(remainder)))
	})
}

// TestGenerateDataFromInteger tests integer data generation.
func TestGenerateDataFromInteger(t *testing.T) {
	g := New(Config{Deterministic: true})

	t.Run("Simple integer", func(t *testing.T) {
		schema := &openapi3.Schema{Type: &openapi3.Types{"integer"}}
		data := g.GenerateData(schema)
		assert.IsType(t, 0, data)
	})

	t.Run("Integer with min/max", func(t *testing.T) {
		min := float64(10)
		max := float64(15)
		schema := &openapi3.Schema{
			Type: &openapi3.Types{"integer"},
			Min:  &min,
			Max:  &max,
		}
		data := g.GenerateData(schema)
		val, ok := data.(int)
		require.True(t, ok)
		assert.GreaterOrEqual(t, val, int(min))
		assert.LessOrEqual(t, val, int(max))
	})
}

// TestGenerateDataFromBoolean tests boolean data generation.
func TestGenerateDataFromBoolean(t *testing.T) {
	g := New(Config{Deterministic: true})
	schema := &openapi3.Schema{Type: &openapi3.Types{"boolean"}}
	data := g.GenerateData(schema)
	assert.IsType(t, false, data)
}

// TestGenerateDataFromArray tests array data generation.
func TestGenerateDataFromArray(t *testing.T) {
	g := New(Config{Deterministic: true, DefaultArrayLength: 3})

	t.Run("Simple array", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:  &openapi3.Types{"array"},
			Items: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
		}
		data := g.GenerateData(schema)
		arr, ok := data.([]interface{})
		require.True(t, ok)
		assert.Len(t, arr, 3)
		assert.IsType(t, "", arr[0])
	})

	t.Run("Array with minItems", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:     &openapi3.Types{"array"},
			Items:    &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			MinItems: 5,
		}
		data := g.GenerateData(schema)
		arr, ok := data.([]interface{})
		require.True(t, ok)
		assert.Len(t, arr, 5)
	})

	t.Run("Array with maxItems", func(t *testing.T) {
		maxItems := uint64(2)
		schema := &openapi3.Schema{
			Type:     &openapi3.Types{"array"},
			Items:    &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			MaxItems: &maxItems,
			MinItems: 1, // MinItems takes precedence over DefaultArrayLength
		}
		data := g.GenerateData(schema)
		arr, ok := data.([]interface{})
		require.True(t, ok)
		assert.Len(t, arr, 1) // Should be 1 because min is 1 and max is 2, and we are not testing random length here
	})

	t.Run("Array with uniqueItems", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type:        &openapi3.Types{"array"},
			Items:       &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
			UniqueItems: true,
		}
		data := g.GenerateData(schema)
		arr, ok := data.([]interface{})
		require.True(t, ok)
		assert.Len(t, arr, 3) // Default length
		unique := make(map[interface{}]bool)
		for _, item := range arr {
			unique[item] = true
		}
		assert.Len(t, unique, len(arr))
	})
}

// TestGenerateDataFromObject tests object data generation.
func TestGenerateDataFromObject(t *testing.T) {
	g := New(Config{Deterministic: true})
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: map[string]*openapi3.SchemaRef{
			"id":   {Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
			"name": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
		},
	}
	data := g.GenerateData(schema)
	obj, ok := data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, obj, "id")
	assert.Contains(t, obj, "name")
	assert.IsType(t, 0, obj["id"])
	assert.IsType(t, "", obj["name"])
}

// TestGenerateDataWithExample tests generation with an explicit example.
func TestGenerateDataWithExample(t *testing.T) {
	g := New(Config{})
	example := "this is an example"
	schema := &openapi3.Schema{
		Type:    &openapi3.Types{"string"},
		Example: example,
	}
	data := g.GenerateData(schema)
	assert.Equal(t, example, data)
}

// TestGenerateDataWithEnum tests generation with enum values.
func TestGenerateDataWithEnum(t *testing.T) {
	g := New(Config{})
	enum := []interface{}{"one", "two", "three"}
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"string"},
		Enum: enum,
	}
	data := g.GenerateData(schema)
	assert.Equal(t, "one", data) // Should pick the first one
}

// TestGenerateDataWithComposition tests schema composition.
func TestGenerateDataWithComposition(t *testing.T) {
	g := New(Config{Deterministic: true})

	t.Run("allOf", func(t *testing.T) {
		schema := &openapi3.Schema{
			AllOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					Properties: map[string]*openapi3.SchemaRef{
						"id": {Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
					},
				}},
				{Value: &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					Properties: map[string]*openapi3.SchemaRef{
						"name": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
					},
				}},
			},
		}
		data := g.GenerateData(schema)
		obj, ok := data.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, obj, "id")
		assert.Contains(t, obj, "name")
	})

	t.Run("oneOf", func(t *testing.T) {
		schema := &openapi3.Schema{
			OneOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "email"}},
				{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uuid"}},
			},
		}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
		// With deterministic generation, it should always pick the first one.
		assert.True(t, strings.Contains(data.(string), "@"))
	})

	t.Run("anyOf", func(t *testing.T) {
		schema := &openapi3.Schema{
			AnyOf: openapi3.SchemaRefs{
				{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "email"}},
				{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uuid"}},
			},
		}
		data := g.GenerateData(schema)
		assert.IsType(t, "", data)
		// With deterministic generation, it should always pick the first one.
		assert.True(t, strings.Contains(data.(string), "@"))
	})
}

// TestCircularReference tests prevention of infinite recursion.
func TestCircularReference(t *testing.T) {
	g := New(Config{})

	// Define a schema that references itself
	schema := &openapi3.Schema{
		Title: "Recursive",
		Type:  &openapi3.Types{"object"},
		Properties: map[string]*openapi3.SchemaRef{
			"child": {Value: nil}, // Placeholder
		},
	}
	schema.Properties["child"].Value = schema // Create circular reference

	data := g.GenerateData(schema)
	obj, ok := data.(map[string]interface{})
	require.True(t, ok)
	assert.Nil(t, obj["child"])
}

// TestDeterministicGeneration tests that deterministic mode produces consistent results.
func TestDeterministicGeneration(t *testing.T) {
	schema := &openapi3.Schema{
		Type: &openapi3.Types{"object"},
		Properties: map[string]*openapi3.SchemaRef{
			"randomString": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			"randomNumber": {Value: &openapi3.Schema{Type: &openapi3.Types{"number"}}},
			"randomInt":    {Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
			"randomBool":   {Value: &openapi3.Schema{Type: &openapi3.Types{"boolean"}}},
		},
	}

	// Generate first set of data
	g1 := New(Config{Deterministic: true, Seed: 123})
	data1 := g1.GenerateData(schema)

	// Generate second set of data with the same seed
	g2 := New(Config{Deterministic: true, Seed: 123})
	data2 := g2.GenerateData(schema)

	assert.Equal(t, data1, data2)

	// Generate third set of data with a different seed
	g3 := New(Config{Deterministic: true, Seed: 456})
	data3 := g3.GenerateData(schema)

	assert.NotEqual(t, data1, data3)
}

// TestFieldNameIntelligence tests data generation based on field names.
func TestFieldNameIntelligence(t *testing.T) {
	g := New(Config{UseFieldNameForData: true, Deterministic: true})

	testCases := []struct {
		name       string
		fieldName  string
		schemaType string
		validator  func(t *testing.T, data interface{})
	}{
		{
			name:       "First Name",
			fieldName:  "firstName",
			schemaType: "string",
			validator: func(t *testing.T, data interface{}) {
				assert.IsType(t, "", data)
				assert.NotEmpty(t, data.(string))
			},
		},
		{
			name:       "Email",
			fieldName:  "user_email",
			schemaType: "string",
			validator: func(t *testing.T, data interface{}) {
				assert.IsType(t, "", data)
				assert.Contains(t, data.(string), "@")
			},
		},
		{
			name:       "Age",
			fieldName:  "age",
			schemaType: "integer",
			validator: func(t *testing.T, data interface{}) {
				val, ok := data.(int)
				require.True(t, ok)
				assert.GreaterOrEqual(t, val, 18)
				assert.LessOrEqual(t, val, 80)
			},
		},
		{
			name:       "Price",
			fieldName:  "price",
			schemaType: "number",
			validator: func(t *testing.T, data interface{}) {
				val, ok := data.(float64)
				require.True(t, ok)
				assert.GreaterOrEqual(t, val, 10.0)
				assert.LessOrEqual(t, val, 1000.0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schema := &openapi3.Schema{Type: &openapi3.Types{tc.schemaType}}
			ctx := GenerationContext{FieldName: tc.fieldName}
			data := g.GenerateDataWithContext(schema, ctx)
			tc.validator(t, data)
		})
	}
}
