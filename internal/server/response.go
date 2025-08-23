package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/leslieo2/go-spec-mock/internal/parser"
)

// ResponseGenerator handles response generation and caching

// cachedResponse represents a cached HTTP response
type cachedResponse struct {
	StatusCode int
	Body       []byte
}

// generateCacheKey creates a cache key from request parameters
func (s *Server) generateCacheKey(method, path string, r *http.Request) string {
	// Create a consistent cache key including all relevant request parameters
	query := r.URL.Query()

	// Build parameter string from all query parameters (sorted for consistency)
	var params []string
	for key, values := range query {
		// Skip internal parameters that don't affect response content
		if key == "__statusCode" || key == "_" {
			continue
		}
		for _, value := range values {
			params = append(params, key+"="+value)
		}
	}

	// Sort parameters for consistent cache keys
	for i := 0; i < len(params); i++ {
		for j := i + 1; j < len(params); j++ {
			if params[i] > params[j] {
				params[i], params[j] = params[j], params[i]
			}
		}
	}

	// Always include status code as it's a primary cache key component
	statusCode := query.Get("__statusCode")
	if statusCode == "" {
		statusCode = "200"
	}

	cacheKey := method + ":" + path + ":" + statusCode
	if len(params) > 0 {
		cacheKey += ":" + strings.Join(params, "&")
	}

	return cacheKey
}

// getCachedResponse retrieves a cached response if available
func (s *Server) getCachedResponse(cacheKey string) (*cachedResponse, bool) {
	if cached, ok := s.cache.Load(cacheKey); ok {
		if response, ok := cached.(cachedResponse); ok {
			return &response, true
		}
	}
	return nil, false
}

// cacheResponse stores a response in the cache
func (s *Server) cacheResponse(cacheKey string, statusCode int, body []byte) {
	s.cache.Store(cacheKey, cachedResponse{
		StatusCode: statusCode,
		Body:       body,
	})
}

// getStatusCodeFromRequest extracts the desired status code from the request
func (s *Server) getStatusCodeFromRequest(r *http.Request) string {
	statusCode := "200"
	if override := r.URL.Query().Get("__statusCode"); override != "" {
		statusCode = override
	}
	return statusCode
}

// generateResponse generates a response for the given route and status code
func (s *Server) generateResponse(route *parser.Route, statusCode string) ([]byte, int, error) {
	example, err := s.parser.GetExampleResponse(route.Operation, statusCode)
	if err == nil {
		buf, err := json.Marshal(example)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to serialize response: %w", err)
		}
		return buf, parseStatusCode(statusCode), nil
	}

	// Try to find any 2xx response if requested status not found
	for code := range route.Operation.Responses.Map() {
		if strings.HasPrefix(code, "2") {
			if example, err = s.parser.GetExampleResponse(route.Operation, code); err == nil {
				buf, err := json.Marshal(example)
				if err != nil {
					return nil, 0, fmt.Errorf("failed to serialize response: %w", err)
				}
				return buf, parseStatusCode(code), nil
			}
		}
	}

	return nil, 0, fmt.Errorf("no example found for status code %s", statusCode)
}

// parseStatusCode converts string status code to int with fallback
func parseStatusCode(code string) int {
	var statusCode int
	_, err := fmt.Sscanf(code, "%d", &statusCode)
	if err != nil {
		return 200
	}
	return statusCode
}

// sendJSONResponse sends a JSON response with the specified status code
func (s *Server) sendJSONResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

// sendErrorResponse sends a JSON error response
func (s *Server) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]string{"error": message}
	_ = json.NewEncoder(w).Encode(response)
}

// sendMethodNotAllowedResponse sends a 405 Method Not Allowed response
func (s *Server) sendMethodNotAllowedResponse(w http.ResponseWriter, methods []string, requestedMethod string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	response := map[string]interface{}{
		"error":   fmt.Sprintf("Method %s not allowed", requestedMethod),
		"methods": methods,
	}
	_ = json.NewEncoder(w).Encode(response)
}

// ResponseWriter wraps http.ResponseWriter to capture status code for logging
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
