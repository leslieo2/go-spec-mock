package generator

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-faker/faker/v4"
	"github.com/lucasjones/reggen"
)

// Config holds configuration options for the data generator
type Config struct {
	Deterministic       bool  // Fixed seed for consistent testing
	Seed                int64 // Optional seed (default 42 for deterministic)
	UseFieldNameForData bool  // Infer data from field names
	DefaultArrayLength  int   // Default array size
}

// GenerationContext provides context for data generation
type GenerationContext struct {
	FieldName     string   // Current property name for context-aware generation
	ParentSchemas []string // Track schemas to prevent infinite recursion
}

// Generator handles dynamic data generation from OpenAPI schemas
type Generator struct {
	config         Config
	rand           *rand.Rand
	formatHandlers map[string]func() string
}

// New creates a new Generator with the given configuration
func New(config Config) *Generator {
	seed := config.Seed
	if config.Deterministic && seed == 0 {
		seed = 42 // Default deterministic seed
	} else if !config.Deterministic && seed == 0 {
		seed = time.Now().UnixNano()
	}

	if config.DefaultArrayLength == 0 {
		config.DefaultArrayLength = 2 // Default to 2 items in arrays
	}

	g := &Generator{
		config: config,
		rand:   rand.New(rand.NewSource(seed)),
	}
	g.initFormatHandlers()
	return g
}

// GenerateData generates example data from an OpenAPI schema
func (g *Generator) GenerateData(schema *openapi3.Schema) interface{} {
	return g.GenerateDataWithContext(schema, GenerationContext{})
}

// GenerateDataWithContext generates example data with additional context
func (g *Generator) GenerateDataWithContext(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	if schema == nil {
		return nil
	}

	// Check for circular reference prevention
	if schema.Title != "" {
		for _, parent := range ctx.ParentSchemas {
			if parent == schema.Title {
				return nil // Return nil to prevent infinite recursion
			}
		}
	}

	// Priority 1: Explicit example (backward compatibility)
	if schema.Example != nil {
		return schema.Example
	}

	// Priority 2: Enum values
	if len(schema.Enum) > 0 {
		return schema.Enum[0] // For now, take first enum value (can be randomized later)
	}

	// Priority 3: Schema composition support
	if len(schema.AllOf) > 0 {
		mergedSchema := g.mergeSchemas(schema.AllOf)
		if mergedSchema != nil {
			return g.GenerateDataWithContext(mergedSchema, ctx)
		}
	}

	if len(schema.OneOf) > 0 {
		selectedSchema := schema.OneOf[g.rand.Intn(len(schema.OneOf))]
		if selectedSchema.Value != nil {
			return g.GenerateDataWithContext(selectedSchema.Value, ctx)
		}
	}

	if len(schema.AnyOf) > 0 {
		selectedSchema := schema.AnyOf[g.rand.Intn(len(schema.AnyOf))]
		if selectedSchema.Value != nil {
			return g.GenerateDataWithContext(selectedSchema.Value, ctx)
		}
	}

	// Priority 4-7: Type-specific generation
	switch {
	case schema.Type.Is("object"):
		return g.generateObject(schema, ctx)
	case schema.Type.Is("array"):
		return g.generateArray(schema, ctx)
	case schema.Type.Is("string"):
		return g.generateString(schema, ctx)
	case schema.Type.Is("number"):
		return g.generateNumber(schema, ctx)
	case schema.Type.Is("integer"):
		return g.generateInteger(schema, ctx)
	case schema.Type.Is("boolean"):
		return g.generateBoolean(schema, ctx)
	default:
		return nil
	}
}

// generateObject generates a mock object from schema properties
func (g *Generator) generateObject(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	result := make(map[string]interface{}, len(schema.Properties))
	for propName, prop := range schema.Properties {
		if prop.Value != nil {
			childCtx := GenerationContext{
				FieldName:     propName,
				ParentSchemas: append(ctx.ParentSchemas, propName),
			}
			result[propName] = g.GenerateDataWithContext(prop.Value, childCtx)
		}
	}
	return result
}

// generateArray generates a mock array from schema items
func (g *Generator) generateArray(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	if schema.Items == nil || schema.Items.Value == nil {
		return []interface{}{}
	}

	// Determine array length
	length := g.config.DefaultArrayLength
	if schema.MinItems > 0 {
		length = int(schema.MinItems)
	}
	if schema.MaxItems != nil && int(*schema.MaxItems) < length {
		length = int(*schema.MaxItems)
	}
	if schema.MinItems == 0 && schema.MaxItems != nil {
		// Generate random length between 1 and maxItems
		length = 1 + g.rand.Intn(int(*schema.MaxItems))
	}

	// Generate items
	result := make([]interface{}, 0, length)
	if schema.UniqueItems {
		return g.generateUniqueItems(schema, ctx, length)
	}

	for i := 0; i < length; i++ {
		item := g.GenerateDataWithContext(schema.Items.Value, ctx)
		result = append(result, item)
	}

	return result
}

// generateString generates a mock string value
func (g *Generator) generateString(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	// Priority 3: Format-specific generation
	if schema.Format != "" {
		if handler, exists := g.formatHandlers[schema.Format]; exists {
			result := handler()
			return g.applyStringConstraints(result, schema)
		}
	}

	// Priority 4: Pattern-based generation
	if schema.Pattern != "" {
		if result, err := reggen.Generate(schema.Pattern, 10); err == nil {
			return g.applyStringConstraints(result, schema)
		}
	}

	// Priority 5: Field name intelligence
	if g.config.UseFieldNameForData && ctx.FieldName != "" {
		if result := g.generateByFieldName(ctx.FieldName); result != "" {
			return g.applyStringConstraints(result, schema)
		}
	}

	// Priority 6: Default realistic string
	result := faker.Word()
	return g.applyStringConstraints(result, schema)
}

// generateNumber generates a mock number value
func (g *Generator) generateNumber(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	// Field name intelligence for realistic ranges
	if g.config.UseFieldNameForData && ctx.FieldName != "" {
		if result := g.generateNumberByFieldName(ctx.FieldName); result != 0 {
			return g.applyNumberConstraints(result, schema)
		}
	}

	// Default range: 1.0 to 100.0
	min := 1.0
	max := 100.0

	// Apply schema constraints
	if schema.Min != nil {
		min = *schema.Min
	}
	if schema.Max != nil {
		max = *schema.Max
	}

	// Generate random value in range
	val := min + g.rand.Float64()*(max-min)

	// Apply multipleOf constraint
	if schema.MultipleOf != nil {
		val = math.Round(val / *schema.MultipleOf) * *schema.MultipleOf
	}

	return val
}

// generateInteger generates a mock integer value
func (g *Generator) generateInteger(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	// Field name intelligence for realistic ranges
	if g.config.UseFieldNameForData && ctx.FieldName != "" {
		if result := g.generateIntegerByFieldName(ctx.FieldName); result != 0 {
			return g.applyIntegerConstraints(result, schema)
		}
	}

	// Default range: 1 to 100
	min := 1
	max := 100

	// Apply schema constraints
	if schema.Min != nil {
		min = int(*schema.Min)
	}
	if schema.Max != nil {
		max = int(*schema.Max)
	}

	// Generate random value in range
	val := min + g.rand.Intn(max-min+1)

	// Apply multipleOf constraint
	if schema.MultipleOf != nil {
		multiple := int(*schema.MultipleOf)
		if multiple > 0 {
			val = (val / multiple) * multiple
		}
	}

	return val
}

// generateBoolean generates a mock boolean value
func (g *Generator) generateBoolean(schema *openapi3.Schema, ctx GenerationContext) interface{} {
	return g.rand.Intn(2) == 1
}

// initFormatHandlers initializes the format handler registry
func (g *Generator) initFormatHandlers() {
	g.formatHandlers = map[string]func() string{
		"email":     func() string { return faker.Email() },
		"uuid":      func() string { return faker.UUIDHyphenated() },
		"uri":       func() string { return faker.URL() },
		"url":       func() string { return faker.URL() },
		"hostname":  func() string { return faker.DomainName() },
		"ipv4":      func() string { return faker.IPv4() },
		"ipv6":      func() string { return faker.IPv6() },
		"date":      func() string { return time.Now().AddDate(0, 0, g.rand.Intn(365)-182).Format("2006-01-02") },
		"date-time": func() string { return time.Now().AddDate(0, 0, g.rand.Intn(365)-182).Format(time.RFC3339) },
	}
}

// applyStringConstraints applies minLength and maxLength constraints to a string
func (g *Generator) applyStringConstraints(str string, schema *openapi3.Schema) string {
	// Apply maxLength constraint
	if schema.MaxLength != nil && uint64(len(str)) > *schema.MaxLength {
		maxLen := int(*schema.MaxLength)
		if maxLen > 0 {
			str = str[:maxLen]
		}
	}

	// Apply minLength constraint
	if schema.MinLength > 0 && uint64(len(str)) < schema.MinLength {
		minLen := int(schema.MinLength)
		for len(str) < minLen {
			str += faker.Word()
		}
		// Trim to exact length if we overshot
		if uint64(len(str)) > schema.MinLength {
			str = str[:minLen]
		}
	}

	return str
}

// generateByFieldName generates realistic data based on field names
func (g *Generator) generateByFieldName(fieldName string) string {
	lowerField := strings.ToLower(fieldName)

	switch {
	case strings.Contains(lowerField, "firstname") || strings.Contains(lowerField, "first_name"):
		return faker.FirstName()
	case strings.Contains(lowerField, "lastname") || strings.Contains(lowerField, "last_name"):
		return faker.LastName()
	case strings.Contains(lowerField, "name") && !strings.Contains(lowerField, "user"):
		return faker.Name()
	case strings.Contains(lowerField, "email"):
		return faker.Email()
	case strings.Contains(lowerField, "phone"):
		return faker.Phonenumber()
	case strings.Contains(lowerField, "address"):
		return faker.Sentence()
	case strings.Contains(lowerField, "company"):
		return faker.Word()
	case strings.Contains(lowerField, "username"):
		return faker.Username()
	}

	return ""
}

// generateNumberByFieldName generates realistic numbers based on field names
func (g *Generator) generateNumberByFieldName(fieldName string) float64 {
	lowerField := strings.ToLower(fieldName)

	switch {
	case strings.Contains(lowerField, "price") || strings.Contains(lowerField, "cost"):
		return 10.0 + g.rand.Float64()*989.99 // 10.00-999.99
	case strings.Contains(lowerField, "latitude"):
		return -90.0 + g.rand.Float64()*180.0 // -90 to 90
	case strings.Contains(lowerField, "longitude"):
		return -180.0 + g.rand.Float64()*360.0 // -180 to 180
	case strings.Contains(lowerField, "rating"):
		return 1.0 + g.rand.Float64()*4.0 // 1.0-5.0
	}

	return 0
}

// generateIntegerByFieldName generates realistic integers based on field names
func (g *Generator) generateIntegerByFieldName(fieldName string) int {
	lowerField := strings.ToLower(fieldName)

	switch {
	case strings.Contains(lowerField, "age"):
		return 18 + g.rand.Intn(63) // 18-80
	case strings.Contains(lowerField, "quantity") || strings.Contains(lowerField, "count"):
		return 1 + g.rand.Intn(100) // 1-100
	case strings.Contains(lowerField, "id"):
		return 1 + g.rand.Intn(10000) // 1-10000
	case strings.Contains(lowerField, "year"):
		return 2000 + g.rand.Intn(25) // 2000-2024
	case strings.Contains(lowerField, "month"):
		return 1 + g.rand.Intn(12) // 1-12
	case strings.Contains(lowerField, "day"):
		return 1 + g.rand.Intn(28) // 1-28 (safe for all months)
	}

	return 0
}

// applyNumberConstraints applies min/max constraints to a number
func (g *Generator) applyNumberConstraints(val float64, schema *openapi3.Schema) float64 {
	if schema.Min != nil && val < *schema.Min {
		val = *schema.Min
	}
	if schema.Max != nil && val > *schema.Max {
		val = *schema.Max
	}
	return val
}

// applyIntegerConstraints applies min/max constraints to an integer
func (g *Generator) applyIntegerConstraints(val int, schema *openapi3.Schema) int {
	if schema.Min != nil && float64(val) < *schema.Min {
		val = int(*schema.Min)
	}
	if schema.Max != nil && float64(val) > *schema.Max {
		val = int(*schema.Max)
	}
	return val
}

// generateUniqueItems generates an array with unique items
func (g *Generator) generateUniqueItems(schema *openapi3.Schema, ctx GenerationContext, length int) []interface{} {
	result := make([]interface{}, 0, length)
	seen := make(map[string]bool)
	maxAttempts := length * 10 // Prevent infinite loops

	for len(result) < length && maxAttempts > 0 {
		item := g.GenerateDataWithContext(schema.Items.Value, ctx)

		// Create a key for uniqueness checking
		key := g.getItemKey(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}

		maxAttempts--
	}

	return result
}

// getItemKey creates a string key for uniqueness checking
func (g *Generator) getItemKey(item interface{}) string {
	switch v := item.(type) {
	case string:
		return "s:" + v
	case int:
		return fmt.Sprintf("i:%d", v)
	case float64:
		return fmt.Sprintf("f:%f", v)
	case bool:
		if v {
			return "b:true"
		}
		return "b:false"
	default:
		return fmt.Sprintf("o:%d", g.rand.Int())
	}
}

// mergeSchemas combines multiple schemas for allOf composition support
func (g *Generator) mergeSchemas(schemas openapi3.SchemaRefs) *openapi3.Schema {
	if len(schemas) == 0 {
		return nil
	}
	if len(schemas) == 1 && schemas[0] != nil && schemas[0].Value != nil {
		return schemas[0].Value
	}

	merged := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(map[string]*openapi3.SchemaRef),
	}

	for _, schemaRef := range schemas {
		if schemaRef == nil || schemaRef.Value == nil {
			continue
		}
		schema := schemaRef.Value

		// Merge properties
		if schema.Properties != nil {
			for propName, propRef := range schema.Properties {
				if propRef != nil && propRef.Value != nil {
					merged.Properties[propName] = propRef
				}
			}
		}

		// Merge constraints (take the most restrictive)
		if schema.Min != nil && (merged.Min == nil || *schema.Min > *merged.Min) {
			merged.Min = schema.Min
		}
		if schema.Max != nil && (merged.Max == nil || *schema.Max < *merged.Max) {
			merged.Max = schema.Max
		}
		if schema.MultipleOf != nil && (merged.MultipleOf == nil || *schema.MultipleOf > *merged.MultipleOf) {
			merged.MultipleOf = schema.MultipleOf
		}

		// Merge string constraints
		if schema.MinLength > merged.MinLength {
			merged.MinLength = schema.MinLength
		}
		if schema.MaxLength != nil && (merged.MaxLength == nil || *schema.MaxLength < *merged.MaxLength) {
			merged.MaxLength = schema.MaxLength
		}

		// Merge array constraints
		if schema.MinItems > merged.MinItems {
			merged.MinItems = schema.MinItems
		}
		if schema.MaxItems != nil && (merged.MaxItems == nil || *schema.MaxItems < *merged.MaxItems) {
			merged.MaxItems = schema.MaxItems
		}
		if schema.UniqueItems {
			merged.UniqueItems = true
		}

		// Merge format and pattern (take first non-empty)
		if schema.Format != "" && merged.Format == "" {
			merged.Format = schema.Format
		}
		if schema.Pattern != "" && merged.Pattern == "" {
			merged.Pattern = schema.Pattern
		}

		// Merge enums (combine unique values)
		if len(schema.Enum) > 0 {
			enumSet := make(map[interface{}]bool)
			for _, e := range merged.Enum {
				enumSet[e] = true
			}
			for _, e := range schema.Enum {
				enumSet[e] = true
			}
			merged.Enum = make([]interface{}, 0, len(enumSet))
			for e := range enumSet {
				merged.Enum = append(merged.Enum, e)
			}
		}
	}

	return merged
}
