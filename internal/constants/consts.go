package constants

// Environment variable constants
const (
	EnvHost              = "GO_SPEC_MOCK_HOST"
	EnvPort              = "GO_SPEC_MOCK_PORT"
	EnvMetricsPort       = "GO_SPEC_MOCK_METRICS_PORT"
	EnvReadTimeout       = "GO_SPEC_MOCK_READ_TIMEOUT"
	EnvWriteTimeout      = "GO_SPEC_MOCK_WRITE_TIMEOUT"
	EnvIdleTimeout       = "GO_SPEC_MOCK_IDLE_TIMEOUT"
	EnvMaxRequestSize    = "GO_SPEC_MOCK_MAX_REQUEST_SIZE"
	EnvShutdownTimeout   = "GO_SPEC_MOCK_SHUTDOWN_TIMEOUT"
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
	StatusTooManyRequests     = 429
	StatusInternalServerError = 500
	StatusServiceUnavailable  = 503
)

// HTTP header constants
const (
	HeaderAuthorization  = "Authorization"
	HeaderContentType    = "Content-Type"
	HeaderXAPIKey        = "X-API-Key" // #nosec G101 - This is a header name constant, not a credential
	HeaderAccept         = "Accept"
	HeaderXRequestedWith = "X-Requested-With"
	HeaderOrigin         = "Origin"
	HeaderXForwardedFor  = "X-Forwarded-For"
	HeaderXRealIP        = "X-Real-IP"
)

// Content type constants
const (
	ContentTypeJSON = "application/json"
)

// Security header constants
const (
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
)

// Security header values
const (
	XContentTypeOptionsNoSniff = "nosniff"
	XFrameOptionsDeny          = "DENY"
	XXSSProtectionBlock        = "1; mode=block"
)

// CORS headers
const (
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
)

// Authentication constants
const (
	BearerPrefix = "Bearer "
)

// Rate limiting strategy constants
const (
	RateLimitStrategyIP     = "ip"
	RateLimitStrategyAPIKey = "api_key"
)

// Rate limiting headers
const (
	HeaderXRateLimitLimit     = "X-RateLimit-Limit"
	HeaderXRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderXRateLimitReset     = "X-RateLimit-Reset"
	HeaderRetryAfter          = "Retry-After"
)

// Context key types

// Context key constants
const (
	ContextKeyAPIKeyStr string = "api_key"
)

// Error code constants
const (
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeInvalidAPIKey     = "INVALID_API_KEY" // #nosec G101 - This is an error code constant, not a credential
	ErrorCodeAPIKeyExpired     = "API_KEY_EXPIRED" // #nosec G101 - This is an error code constant, not a credential
	ErrorCodeAPIKeyDisabled    = "API_KEY_DISABLED"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// Path constants for skipped authentication
const (
	PathHealth  = "/health"
	PathReady   = "/ready"
	PathMetrics = "/metrics"
)

// Query parameter constants
const (
	QueryParamStatusCode = "__statusCode"
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
