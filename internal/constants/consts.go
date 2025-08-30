package constants

import "time"

// Environment variable constants
const (
	EnvHost              = "GO_SPEC_MOCK_HOST"
	EnvPort              = "GO_SPEC_MOCK_PORT"
	EnvSpecFile          = "GO_SPEC_MOCK_SPEC_FILE"
	EnvHotReload         = "GO_SPEC_MOCK_HOT_RELOAD"
	EnvHotReloadDebounce = "GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE"
	EnvProxyEnabled      = "GO_SPEC_MOCK_PROXY_ENABLED"
	EnvProxyTarget       = "GO_SPEC_MOCK_PROXY_TARGET"
	EnvProxyTimeout      = "GO_SPEC_MOCK_PROXY_TIMEOUT"
	EnvTLSEnabled        = "GO_SPEC_MOCK_TLS_ENABLED"
	EnvTLSCertFile       = "GO_SPEC_MOCK_TLS_CERT_FILE"
	EnvTLSKeyFile        = "GO_SPEC_MOCK_TLS_KEY_FILE"
)

// HTTP method constants
const (
	MethodGET     = "GET"
	MethodPOST    = "POST"
	MethodPUT     = "PUT"
	MethodDELETE  = "DELETE"
	MethodPATCH   = "PATCH"
	MethodOPTIONS = "OPTIONS"
	MethodHEAD    = "HEAD"
)

// HTTP status code constants
const (
	StatusOK                  = 200
	StatusCreated             = 201
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusInternalServerError = 500
	StatusServiceUnavailable  = 503
)

// HTTP header constants
const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderOrigin        = "Origin"
)

// Content type constants
const (
	ContentTypeJSON = "application/json"
)

// CORS headers
const (
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
)

// Server timeout constants (internal use only - not user configurable)
const (
	// ServerReadTimeout is the read timeout for the HTTP server
	ServerReadTimeout = 15 * time.Second
	// ServerWriteTimeout is the write timeout for the HTTP server
	ServerWriteTimeout = 15 * time.Second
	// ServerIdleTimeout is the idle timeout for the HTTP server
	ServerIdleTimeout = 60 * time.Second
	// ServerMaxRequestSize is the maximum request body size (10MB)
	ServerMaxRequestSize = 10 * 1024 * 1024
	// ServerShutdownTimeout is the graceful shutdown timeout
	ServerShutdownTimeout = 30 * time.Second
)

// Path constants for skipped authentication
const (
	PathHealth        = "/health"
	PathReady         = "/ready"
	PathDocumentation = "/docs"
)

// Query parameter constants
const (
	QueryParamStatusCode = "__statusCode"
	QueryParamDelay      = "__delay"
	QueryParamExample    = "__example"
)

// Context key type for avoiding collisions
type contextKey string

// Context key constants
const (
	ContextKeyStatusCode = contextKey("statusCode")
)

// Delay simulation constants
const (
	MaxDelayDuration = 30 * time.Second
)

// Hop-by-hop headers that should not be forwarded
var HopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}
