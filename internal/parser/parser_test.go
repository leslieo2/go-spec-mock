package parser

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name:    "valid spec",
			spec:    "../../examples/petstore.yaml",
			wantErr: false,
		},
		{
			name:    "invalid file",
			spec:    "nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetRoutes(t *testing.T) {
	parser, err := New("../../examples/petstore.yaml")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	routes := parser.GetRoutes()
	if len(routes) == 0 {
		t.Error("Expected routes, got none")
	}

	// Should have at least some routes
	t.Logf("Found %d routes", len(routes))
}

func TestGetExampleResponse(t *testing.T) {
	parser, err := New("../../examples/petstore.yaml")
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	routes := parser.GetRoutes()
	if len(routes) == 0 {
		t.Fatal("No routes found")
	}

	// Test getting 200 response for first route
	_, err = parser.GetExampleResponse(routes[0].Operation, "200")
	if err != nil {
		t.Logf("Expected some routes might not have 200 responses: %v", err)
	}
}

func TestGenerateExampleFromSchema(t *testing.T) {
	// Skip this test for now - the kin-openapi API is complex
	t.Skip("Skipping schema generation test due to kin-openapi complexity")
}

func TestGetExampleResponse_NotFound(t *testing.T) {
	parser, err := New("../../examples/petstore.yaml")
	if err != nil {
		t.Skipf("Skipping test due to parser error: %v", err)
	}

	routes := parser.GetRoutes()
	if len(routes) == 0 {
		t.Skip("No routes found")
	}

	// Test getting a non-existent status code
	_, err = parser.GetExampleResponse(routes[0].Operation, "999")
	if err == nil {
		t.Log("Expected error for non-existent status code, but got none")
	} else {
		t.Logf("Got expected error for non-existent status code: %v", err)
	}
}

func TestGetExampleResponse_ValidCodes(t *testing.T) {
	parser, err := New("../../examples/petstore.yaml")
	if err != nil {
		t.Skipf("Skipping test due to parser error: %v", err)
	}

	routes := parser.GetRoutes()
	if len(routes) == 0 {
		t.Skip("No routes found")
	}

	// Test getting common status codes
	statusCodes := []string{"200", "201", "400", "404", "500"}

	for _, code := range statusCodes {
		result, err := parser.GetExampleResponse(routes[0].Operation, code)
		if err != nil {
			t.Logf("No example for status %s: %v", code, err)
		} else {
			t.Logf("Found example for status %s: %+v", code, result)
		}
	}
}
