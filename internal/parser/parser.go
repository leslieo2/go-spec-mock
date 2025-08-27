package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/leslieo2/go-spec-mock/internal/constants"
)

type Parser struct {
	doc   *openapi3.T
	cache *sync.Map // Cache for pre-generated examples
}

func New(specPath string) (*Parser, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	cleanedPath := filepath.Clean(specPath)

	data, err := os.ReadFile(cleanedPath)
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

	return &Parser{doc: doc, cache: &sync.Map{}}, nil
}

func (p *Parser) GetRoutes() []Route {
	paths := p.doc.Paths.Map()
	routes := make([]Route, 0, len(paths)*3) // Pre-allocate with estimated capacity

	for path, pathItem := range paths {
		for method := range methodMap {
			if operation := pathItem.GetOperation(method); operation != nil {
				// Pre-generate examples for common status codes
				p.preGenerateExamples(operation)
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

func (p *Parser) preGenerateExamples(operation *openapi3.Operation) {
	if operation == nil || operation.Responses == nil {
		return
	}

	// Common status codes as set for O(1) lookup
	commonCodes := map[string]struct{}{
		strconv.Itoa(constants.StatusOK):                  {},
		strconv.Itoa(constants.StatusCreated):             {},
		strconv.Itoa(constants.StatusBadRequest):          {},
		strconv.Itoa(constants.StatusUnauthorized):        {},
		strconv.Itoa(constants.StatusForbidden):           {},
		strconv.Itoa(constants.StatusNotFound):            {},
		strconv.Itoa(constants.StatusInternalServerError): {},
	}

	// Get all response codes and only process common ones
	for code := range operation.Responses.Map() {
		if _, isCommon := commonCodes[code]; isCommon {
			_, _ = p.GetExampleResponse(operation, code)
		}
	}
}

func (p *Parser) GetExampleResponse(operation *openapi3.Operation, statusCode string) (interface{}, error) {
	if operation.Responses == nil {
		return nil, fmt.Errorf("no responses defined")
	}

	// Check cache first
	cacheKey := operation.OperationID + ":" + statusCode
	if cached, ok := p.cache.Load(cacheKey); ok {
		return cached, nil
	}

	response, exists := operation.Responses.Map()[statusCode]
	if !exists || response == nil || response.Value == nil {
		return nil, fmt.Errorf("response %s not found", statusCode)
	}

	content := response.Value.Content
	if content == nil {
		return nil, fmt.Errorf("no content defined for response %s", statusCode)
	}

	jsonContent := content.Get(constants.ContentTypeJSON)
	if jsonContent == nil {
		return nil, fmt.Errorf("no application/json content defined")
	}

	var result interface{}
	if jsonContent.Example != nil {
		result = jsonContent.Example
	} else if schema := jsonContent.Schema; schema != nil && schema.Value != nil {
		result = generateExampleFromSchema(schema.Value)
	} else {
		return nil, fmt.Errorf("no example or schema found")
	}

	// Cache the result
	p.cache.Store(cacheKey, result)
	return result, nil
}

func generateExampleFromSchema(schema *openapi3.Schema) interface{} {
	if schema == nil {
		return nil
	}

	// Check if schema has an example
	if schema.Example != nil {
		return schema.Example
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
	constants.MethodGET:     {},
	constants.MethodPOST:    {},
	constants.MethodPUT:     {},
	constants.MethodDELETE:  {},
	constants.MethodPATCH:   {},
	constants.MethodHEAD:    {},
	constants.MethodOPTIONS: {},
}

type Route struct {
	Path      string
	Method    string
	Operation *openapi3.Operation
}
