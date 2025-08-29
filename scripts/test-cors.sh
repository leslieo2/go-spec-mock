#!/bin/bash
set -euo pipefail

export no_proxy=localhost,127.0.0.1

# --- Helper Functions for Colored Output ---
info() { echo -e "\033[1;34m[INFO]\033[0m $1"; }
success() { echo -e "\033[1;32m[SUCCESS]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }
test_case() { echo -e "\n\033[1;33m--- TEST CASE: $1 ---\033[0m"; }

# Track results
FAILURES=()

# --- Prerequisites Check ---
for cmd in go curl sleep git; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "[ERROR] $cmd is not installed." >&2
        exit 1
    fi
done

# --- Setup Test Environment ---
info "Setting up CORS integration test environment..."
PROJECT_ROOT=$(git rev-parse --show-toplevel)
TMP_DIR=$(mktemp -d -t go-spec-mock-cors-test-XXXXXXXX)

SERVER_PIDS=()

cleanup() {
    info "Cleaning up..."
    
    # First, try graceful shutdown of tracked PIDs and their process groups
    if [ ${#SERVER_PIDS[@]} -gt 0 ]; then
        for pid in "${SERVER_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                info "Stopping server (PID: $pid)..."
                kill "$pid" 2>/dev/null || true
            fi
        done
        
        # Wait a moment for graceful shutdown
        sleep 2
        
        # Force kill any remaining tracked processes
        for pid in "${SERVER_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                info "Force stopping server (PID: $pid)..."
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    # Kill any remaining go-spec-mock processes by name and args
    pkill -f "go-spec-mock.*test-spec.yaml" 2>/dev/null || true
    
    # More aggressive cleanup - kill any go-spec-mock processes on our test ports
    for port in 8181 8182 8183; do
        local pids
        pids=$(lsof -ti:$port 2>/dev/null || true)
        if [ -n "$pids" ]; then
            info "Killing processes on port $port: $pids"
            echo "$pids" | xargs kill -9 2>/dev/null || true
        fi
    done
    
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$TMP_DIR"
info "Temporary directory created at: $TMP_DIR"

cp -r "$PROJECT_ROOT"/* .

# Build binary
info "Building go-spec-mock binary..."
go build -o go-spec-mock .
info "Build complete."

# Test spec
cat > test-spec.yaml << 'EOF'
openapi: 3.0.0
info:
  title: CORS Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint for CORS
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              example:
                message: "CORS test successful"
EOF

# --- Configurations ---
cat > config-default-cors.yaml << 'EOF'
server:
  host: localhost
  port: 8181
security:
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization"]
    allow_credentials: false
    max_age: 3600
EOF

cat > config-specific-origins.yaml << 'EOF'
server:
  host: localhost
  port: 8182
security:
  cors:
    enabled: true
    allowed_origins: ["http://localhost:3000", "https://example.com"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-Custom-Header"]
    allow_credentials: true
    max_age: 7200
EOF

cat > config-disabled-cors.yaml << 'EOF'
server:
  host: localhost
  port: 8183
security:
  cors:
    enabled: false
EOF

# --- Functions ---
cors_test() {
    local port=$1 origin=$2 expected_origin=$3 description=$4
    test_case "$description"

    local response
    response=$(curl -s -X OPTIONS "http://localhost:$port/test" \
        -H "Origin: $origin" \
        -H "Access-Control-Request-Method: GET" \
        -H "Access-Control-Request-Headers: Content-Type" \
        -i)

    if echo "$response" | grep -iq "access-control-allow-origin: $expected_origin"; then
        success "✓ Allow-Origin header correct"
    else
        FAILURES+=("$description - OPTIONS origin header wrong/missing")
    fi

    for header in Methods Headers Max-Age; do
        if echo "$response" | grep -iq "access-control-allow-$header"; then
            success "✓ Allow-$header header present"
        else
            FAILURES+=("$description - OPTIONS allow-$header missing")
        fi
    done

    local get_response
    get_response=$(curl -s -X GET "http://localhost:$port/test" -H "Origin: $origin" -i)

    if echo "$get_response" | grep -iq "access-control-allow-origin: $expected_origin"; then
        success "✓ GET origin header correct"
    else
        FAILURES+=("$description - GET origin header wrong/missing")
    fi
}

wait_for_server() {
    local port=$1
    for attempt in {1..10}; do
        if curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port/test" | grep -qE "200|404"; then
            return 0
        fi
        sleep 1
    done
    FAILURES+=("Server on port $port failed to start")
}

run_cors_test() {
    local config=$1 port=$2 name=$3
    info "Starting $name server on port $port..."
    
    # Start server in background (setsid not available on macOS)
    ./go-spec-mock --config "$config" --spec-file test-spec.yaml &
    local pid=$!
    SERVER_PIDS+=("$pid")
    
    if ! wait_for_server "$port"; then
        return 1
    fi

    case "$name" in
        default-cors)
            cors_test "$port" "http://localhost:3000" "http://localhost:3000" "Default CORS (wildcard origin)"
            cors_test "$port" "https://example.com" "https://example.com" "Default CORS (different origin)"
            ;;
        specific-origins)
            cors_test "$port" "http://localhost:3000" "http://localhost:3000" "Specific origins (allowed localhost)"
            cors_test "$port" "https://example.com" "https://example.com" "Specific origins (allowed example.com)"
            local resp
            resp=$(curl -s -X OPTIONS "http://localhost:$port/test" \
                -H "Origin: http://disallowed.com" \
                -H "Access-Control-Request-Method: GET" -i)
            if echo "$resp" | grep -iq "access-control-allow-origin:"; then
                FAILURES+=("Specific origins - disallowed origin incorrectly accepted")
            else
                success "✓ Disallowed origin rejected (correct)"
            fi
            ;;
        disabled-cors)
            local resp
            resp=$(curl -s -X OPTIONS "http://localhost:$port/test" \
                -H "Origin: http://localhost:3000" \
                -H "Access-Control-Request-Method: GET" -i)
            if echo "$resp" | grep -iq "access-control-allow-origin:"; then
                FAILURES+=("Disabled CORS - headers present when they shouldn't be")
            else
                success "✓ No CORS headers when disabled"
            fi
            ;;
    esac
}

# --- Run Tests in Parallel ---
run_cors_test "config-default-cors.yaml" 8181 default-cors &
run_cors_test "config-specific-origins.yaml" 8182 specific-origins &
run_cors_test "config-disabled-cors.yaml" 8183 disabled-cors &
wait

# --- Final Report ---
echo
if [ ${#FAILURES[@]} -eq 0 ]; then
    success "🎉 All CORS integration tests passed successfully!"
else
    error "❌ Some tests failed:"
    for f in "${FAILURES[@]}"; do
        echo "   - $f"
    done
    exit 1
fi