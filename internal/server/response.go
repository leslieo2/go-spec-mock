package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/leslieo2/go-spec-mock/internal/constants"
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
		if key == constants.QueryParamStatusCode || key == "_" {
			continue
		}
		for _, value := range values {
			params = append(params, key+"="+value)
		}
	}

	// Sort parameters using efficient sorting for consistent cache keys
	if len(params) > 1 {
		// Use Go's built-in sort for better performance and reliability
		// This ensures consistent ordering regardless of parameter arrival order
		stringsSlice := sort.StringSlice(params)
		sort.Sort(stringsSlice)
		params = stringsSlice
	}

	// Always include status code as it's a primary cache key component
	statusCode := query.Get(constants.QueryParamStatusCode)
	if statusCode == "" {
		statusCode = strconv.Itoa(constants.StatusOK)
	}

	// Include additional request context to prevent collisions
	// - Authentication context (if available)
	// - Accept header for content negotiation
	// - Content-Type for request body format
	var contextParts []string

	// Add authentication context if available
	if authHeader := r.Header.Get(constants.HeaderAuthorization); authHeader != "" {
		// Use a hash of the auth header to avoid storing sensitive data in cache keys
		h := sha256.New()
		h.Write([]byte(authHeader))
		contextParts = append(contextParts, "auth:"+fmt.Sprintf("%x", h.Sum(nil))[:16])
	}

	// Add content negotiation headers
	if accept := r.Header.Get(constants.HeaderAccept); accept != "" && accept != "*/*" {
		contextParts = append(contextParts, "accept:"+accept)
	}

	if contentType := r.Header.Get(constants.HeaderContentType); contentType != "" {
		contextParts = append(contextParts, "content-type:"+contentType)
	}

	// Build cache key with all components
	cacheKey := method + ":" + path + ":" + statusCode

	// Add sorted query parameters
	if len(params) > 0 {
		cacheKey += ":" + strings.Join(params, "&")
	}

	// Add request context
	if len(contextParts) > 0 {
		cacheKey += ":" + strings.Join(contextParts, ";")
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

// clearCache clears all cached responses
func (s *Server) clearCache() {
	// Create a new empty cache map
	s.cache = &sync.Map{}
}

// getStatusCodeFromRequest extracts the desired status code from the request
func (s *Server) getStatusCodeFromRequest(r *http.Request) string {
	statusCode := strconv.Itoa(constants.StatusOK)
	if override := r.URL.Query().Get(constants.QueryParamStatusCode); override != "" {
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
		return constants.StatusOK
	}
	return statusCode
}

// sendJSONResponse sends a JSON response with the specified status code
func (s *Server) sendJSONResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

// sendErrorResponse sends a JSON error response
func (s *Server) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	response := map[string]string{"error": message}
	_ = json.NewEncoder(w).Encode(response)
}

// sendMethodNotAllowedResponse sends a 405 Method Not Allowed response
func (s *Server) sendMethodNotAllowedResponse(w http.ResponseWriter, methods []string, requestedMethod string) {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(constants.StatusMethodNotAllowed)
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
