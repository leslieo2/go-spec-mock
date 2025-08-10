package parser

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

type Parser struct {
	doc *openapi3.T
}

func New(specPath string) (*Parser, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("OpenAPI spec validation failed: %w", err)
	}

	return &Parser{doc: doc}, nil
}

func (p *Parser) GetRoutes() []Route {
	paths := p.doc.Paths.Map()
	routes := make([]Route, 0, len(paths)*3) // Pre-allocate with estimated capacity

	for path, pathItem := range paths {
		for method := range methodMap {
			if operation := pathItem.GetOperation(method); operation != nil {
				routes = append(routes, Route{
					Path:      path,
					Method:    method,
					Operation: operation,
				})
			}
		}
	}

	return routes
}

func (p *Parser) GetExampleResponse(operation *openapi3.Operation, statusCode string) (interface{}, error) {
	if operation.Responses == nil {
		return nil, fmt.Errorf("no responses defined")
	}

	response, exists := operation.Responses.Map()[statusCode]
	if !exists || response == nil || response.Value == nil {
		return nil, fmt.Errorf("response %s not found", statusCode)
	}

	content := response.Value.Content
	if content == nil {
		return nil, fmt.Errorf("no content defined for response %s", statusCode)
	}

	jsonContent := content.Get("application/json")
	if jsonContent == nil {
		return nil, fmt.Errorf("no application/json content defined")
	}

	if jsonContent.Example != nil {
		return jsonContent.Example, nil
	}

	if schema := jsonContent.Schema; schema != nil && schema.Value != nil {
		return generateExampleFromSchema(schema.Value), nil
	}

	return nil, fmt.Errorf("no example or schema found")
}

func generateExampleFromSchema(schema *openapi3.Schema) interface{} {
	if schema == nil {
		return nil
	}

	switch {
	case schema.Type.Is("object"):
		result := make(map[string]interface{}, len(schema.Properties))
		for propName, prop := range schema.Properties {
			if prop.Value != nil {
				result[propName] = generateExampleFromSchema(prop.Value)
			}
		}
		return result
	case schema.Type.Is("array"):
		if schema.Items != nil && schema.Items.Value != nil {
			return []interface{}{generateExampleFromSchema(schema.Items.Value)}
		}
		return []interface{}{}
	case schema.Type.Is("string"):
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		return "string"
	case schema.Type.Is("number"):
		return 0.0
	case schema.Type.Is("integer"):
		return 0
	case schema.Type.Is("boolean"):
		return true
	default:
		return nil
	}
}

var methodMap = map[string]struct{}{
	"GET":     {},
	"POST":    {},
	"PUT":     {},
	"DELETE":  {},
	"PATCH":   {},
	"HEAD":    {},
	"OPTIONS": {},
}

type Route struct {
	Path      string
	Method    string
	Operation *openapi3.Operation
}
