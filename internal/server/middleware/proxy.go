package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

// Proxy represents a reverse proxy handler
type Proxy struct {
	target  *url.URL
	timeout time.Duration
	proxy   *httputil.ReverseProxy
}

// NewProxy creates a new proxy instance
func NewProxy(cfg config.ProxyConfig) (*Proxy, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("proxy is not enabled")
	}

	targetURL, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy target URL: %w", err)
	}

	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Preserve the original request path and append to target
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			// Keep the original path but ensure it's properly joined
			if targetURL.Path != "/" && targetURL.Path != "" {
				req.URL.Path = joinPaths(targetURL.Path, req.URL.Path)
			}
			req.Host = targetURL.Host

			// Remove hop-by-hop headers
			removeHopByHopHeaders(req.Header)
		},
		ModifyResponse: func(resp *http.Response) error {
			// Remove hop-by-hop headers from response
			removeHopByHopHeaders(resp.Header)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Proxy error: "+err.Error(), http.StatusBadGateway)
		},
	}

	return &Proxy{
		target:  targetURL,
		timeout: cfg.Timeout,
		proxy:   reverseProxy,
	}, nil
}

// ServeHTTP handles the HTTP request by forwarding it to the target server
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), p.timeout)
	defer cancel()

	p.proxy.ServeHTTP(w, r.WithContext(ctx))
}

// removeHopByHopHeaders removes hop-by-hop headers that should not be forwarded
func removeHopByHopHeaders(headers http.Header) {
	hopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, h := range hopHeaders {
		headers.Del(h)
	}
}

// joinPaths properly joins two URL paths, ensuring exactly one slash between them
func joinPaths(basePath, additionalPath string) string {
	if basePath == "" {
		return additionalPath
	}
	if additionalPath == "" {
		return basePath
	}

	// Remove trailing slash from basePath and leading slash from additionalPath
	basePath = strings.TrimSuffix(basePath, "/")
	additionalPath = strings.TrimPrefix(additionalPath, "/")

	// Join with a single slash
	if additionalPath == "" {
		return basePath
	}
	return basePath + "/" + additionalPath
}
