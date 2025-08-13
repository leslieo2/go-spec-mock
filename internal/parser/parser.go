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
	var routes []Route

	for path, pathItem := range p.doc.Paths.Map() {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		for _, method := range methods {
			operation := pathItem.GetOperation(method)
			if operation != nil {
				route := Route{
					Path:      path,
					Method:    method,
					Operation: operation,
				}
				routes = append(routes, route)
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

	if jsonContent.Schema != nil && jsonContent.Schema.Value != nil {
		return generateExampleFromSchema(jsonContent.Schema.Value), nil
	}

	return nil, fmt.Errorf("no example or schema found")
}

func generateExampleFromSchema(schema *openapi3.Schema) interface{} {
	if schema == nil {
		return nil
	}

	if schema.Type.Is("object") {
		result := make(map[string]interface{})
		for propName, prop := range schema.Properties {
			if prop.Value != nil {
				result[propName] = generateExampleFromSchema(prop.Value)
			}
		}
		return result
	} else if schema.Type.Is("array") {
		if schema.Items != nil && schema.Items.Value != nil {
			return []interface{}{generateExampleFromSchema(schema.Items.Value)}
		}
		return []interface{}{}
	} else if schema.Type.Is("string") {
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		return "string"
	} else if schema.Type.Is("number") {
		return 0.0
	} else if schema.Type.Is("integer") {
		return 0
	} else if schema.Type.Is("boolean") {
		return true
	}
	return nil
}

type Route struct {
	Path      string
	Method    string
	Operation *openapi3.Operation
}
