#!/bin/bash

# Proxy Integration Test Script for Go-Spec-Mock
# This script tests the proxy functionality for undefined endpoints.

set -e

export no_proxy=localhost,127.0.0.1

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_PORT="8085" # Use a distinct port for this module
PROXY_TARGET_PORT="8186" # Port for mock backend
CONFIG_FILE="test-proxy-config.yaml"
BINARY_NAME="go-spec-mock"
SERVER_PID=""
MOCK_BACKEND_PID=""

# Global test counters
TEST_PASSED_COUNT=0
TEST_TOTAL_COUNT=0

# Function to print colored output
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Cleanup function
cleanup() {
    print_status "Cleaning up test environment..."
    
    # Kill proxy server if running
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        print_status "Proxy server stopped (PID: $SERVER_PID)"
    fi
    
    # Kill mock backend if running
    if [ ! -z "$MOCK_BACKEND_PID" ]; then
        kill $MOCK_BACKEND_PID 2>/dev/null || true
        wait $MOCK_BACKEND_PID 2>/dev/null || true
        print_status "Mock backend stopped (PID: $MOCK_BACKEND_PID)"
    fi
    
    # Remove test files (keep logs for debugging)
    cd "$PROJECT_DIR"
    rm -f "$CONFIG_FILE" "$BINARY_NAME" "mock_backend.py"
    print_status "Test files cleaned up"
}

# Set trap for cleanup
trap cleanup EXIT

# Function to wait for server startup
wait_for_server() {
    local port=$1
    local protocol=$2 # "http" or "https"
    local max_attempts=30
    local attempt=0
    
    print_status "Waiting for server to start on $protocol://localhost:$port..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -k -s --connect-timeout 1 "$protocol://localhost:$port/health" > /dev/null 2>&1; then
            print_success "Server is ready on $protocol://localhost:$port"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    
    print_error "Server failed to start within $max_attempts seconds"
    return 1
}

# Function to wait for mock backend
wait_for_backend() {
    local port=$1
    local max_attempts=30
    local attempt=0
    
    print_status "Waiting for mock backend to start on port $port..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s --connect-timeout 1 "http://localhost:$port/api/v1/status" > /dev/null 2>&1; then
            print_success "Mock backend is ready on port $port"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    
    print_error "Mock backend failed to start within $max_attempts seconds"
    return 1
}

# Function to test HTTP endpoint
test_api_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=${3:-200}
    local description=${4:-"Testing $method $endpoint"}
    local data=$5 # Optional data for POST/PUT
    local headers=$6 # Optional headers
    local protocol=${7:-"http"}
    
    print_status "$description"
    
    local response
    local status_code
    local curl_cmd="curl -s -o /dev/null -w \"%{http_code}\" -X $method $protocol://localhost:$TEST_PORT$endpoint"
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -H \"Content-Type: application/json\" -d '$data'"
    fi
    if [ -n "$headers" ]; then
        curl_cmd="$curl_cmd $headers"
    fi
    
    ((TEST_TOTAL_COUNT++))
    response=$(eval "$curl_cmd" || echo "000")
    status_code="${response: -3}"
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "$description - Status: $status_code"
        ((TEST_PASSED_COUNT++))
        return 0
    else
        print_error "$description - Expected: $expected_status, Got: $status_code"
        return 1
    fi
}

# Function to test proxy response content
test_proxy_content() {
    local endpoint=$1
    local expected_content=$2
    local description=$3
    
    print_status "$description"
    
    ((TEST_TOTAL_COUNT++))
    local response=$(curl -s "http://localhost:$TEST_PORT$endpoint" || echo "")
    
    if echo "$response" | grep -q "$expected_content"; then
        print_success "$description - Content found: $expected_content"
        ((TEST_PASSED_COUNT++))
        return 0
    else
        print_error "$description - Expected content '$expected_content' not found in response: $response"
        return 1
    fi
}

# Function to start mock backend server
start_mock_backend() {
    print_status "Starting mock backend server..."
    
    # Check if Python3 is available
    if ! command -v python3 &> /dev/null; then
        print_error "Python3 is required but not installed"
        return 1
    fi
    
    # Create a temporary Python script for the mock backend
    cat > "${PROJECT_DIR}/mock_backend.py" << 'EOF'
#!/usr/bin/env python3
import http.server
import socketserver
import json
import sys
import signal
import time
from urllib.parse import urlparse, parse_qs

class MockHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        try:
            if self.path == '/api/v1/status':
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                response = json.dumps({'status': 'backend_running', 'service': 'mock_backend'})
                self.wfile.write(response.encode())
            elif self.path == '/api/v1/users':
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                response = json.dumps({'users': [{'id': 1, 'name': 'Backend User'}]})
                self.wfile.write(response.encode())
            elif self.path == '/api/v1/slow':
                time.sleep(2)  # Simulate slow response
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                response = json.dumps({'message': 'slow_response'})
                self.wfile.write(response.encode())
            else:
                self.send_response(404)
                self.send_header('Content-type', 'application/json')
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                response = json.dumps({'error': 'Not found in backend'})
                self.wfile.write(response.encode())
        except Exception as e:
            self.send_response(500)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = json.dumps({'error': f'Internal server error: {str(e)}'})
            self.wfile.write(response.encode())
    
    def do_OPTIONS(self):
        # Handle CORS preflight requests
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type, Authorization')
        self.end_headers()
    
    def log_message(self, format, *args):
        pass  # Suppress default logging

def signal_handler(signum, frame):
    print("Mock backend shutting down...")
    sys.exit(0)

if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8086
    
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    
    try:
        with socketserver.TCPServer(('0.0.0.0', port), MockHandler) as httpd:
            print(f"Mock backend server starting on port {port}")
            httpd.serve_forever()
    except OSError as e:
        print(f"Failed to start server on port {port}: {e}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("Mock backend shutting down...")
        sys.exit(0)
EOF
    
    # Make the script executable
    chmod +x "${PROJECT_DIR}/mock_backend.py"
    
    # Start the mock backend
    python3 "${PROJECT_DIR}/mock_backend.py" "$PROXY_TARGET_PORT" > backend.log 2>&1 &
    MOCK_BACKEND_PID=$!
    
    if ! wait_for_backend "$PROXY_TARGET_PORT"; then
        print_error "Failed to start mock backend"
        if [ -f backend.log ]; then
            print_error "Backend log:"
            cat backend.log
        fi
        return 1
    fi
    
    print_success "Mock backend started (PID: $MOCK_BACKEND_PID)"
}

# Main test function
run_tests() {
    print_status "Starting Proxy Integration Tests..."
    
    # Cleanup any previous runs
    cleanup
    
    # Change to project directory
    cd "$PROJECT_DIR"
    
    # Step 1: Build the project
    print_status "Building go-spec-mock..."
    if ! go build -o "$BINARY_NAME" .; then
        print_error "Failed to build project"
        exit 1
    fi
    print_success "Project built successfully"
    
    # Step 2: Start mock backend
    if ! start_mock_backend; then
        exit 1
    fi
    
    # Step 3: Create test configuration with proxy enabled
    print_status "Creating test configuration with proxy enabled..."
    cat > "$CONFIG_FILE" << EOF
# Proxy Test Configuration
server:
  host: "localhost"
  port: "$TEST_PORT"

tls:
  enabled: false

security:
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]

observability:
  logging:
    level: "debug"
    format: "console"

hot_reload:
  enabled: false

proxy:
  enabled: true
  target: "http://localhost:$PROXY_TARGET_PORT"
  timeout: "15s"

spec_file: "./examples/petstore.yaml"
EOF
    print_success "Test configuration created with proxy target: http://localhost:$PROXY_TARGET_PORT"
    
    # Step 4: Start go-spec-mock with proxy enabled
    print_status "Starting go-spec-mock with proxy enabled..."
    "./$BINARY_NAME" --config "$CONFIG_FILE" > server.log 2>&1 &
    SERVER_PID=$!
    
    if ! wait_for_server "$TEST_PORT" "http"; then
        print_error "Server startup failed"
        cat server.log
        exit 1
    fi
    
    # Step 5: Run proxy tests
    print_status "Running proxy functionality tests..."
    
    # Test 1: Verify defined endpoint (from petstore.yaml) works normally
    test_api_endpoint "GET" "/pets" "200" "Testing defined endpoint /pets (should be mocked)"
    
    # Test 2: Verify undefined endpoint is proxied to backend
    test_api_endpoint "GET" "/api/v1/status" "200" "Testing undefined endpoint /api/v1/status (should be proxied)"
    test_proxy_content "/api/v1/status" "backend_running" "Verifying proxy response contains backend data"
    
    # Test 3: Test another undefined endpoint
    test_api_endpoint "GET" "/api/v1/users" "200" "Testing undefined endpoint /api/v1/users (should be proxied)"
    test_proxy_content "/api/v1/users" "Backend User" "Verifying proxy response contains backend user data"
    
    # Test 4: Test undefined endpoint that doesn't exist in backend
    test_api_endpoint "GET" "/api/v1/nonexistent" "404" "Testing undefined endpoint that doesn't exist in backend"
    
    # Test 5: Test health endpoint (should be handled by go-spec-mock, not proxied)
    test_api_endpoint "GET" "/health" "200" "Testing /health endpoint (should not be proxied)"
    
    # Test 6: Test ready endpoint (should be handled by go-spec-mock, not proxied)
    test_api_endpoint "GET" "/ready" "200" "Testing /ready endpoint (should not be proxied)"
    
    # Test 7: Test slow backend response (within timeout)
    print_status "Testing slow backend response (should succeed within timeout)..."
    local start_time=$(date +%s)
    test_api_endpoint "GET" "/api/v1/slow" "200" "Testing slow backend endpoint (2s delay, 15s timeout)"
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    if [ $duration -ge 2 ] && [ $duration -le 5 ]; then
        print_success "Proxy request took expected time: ${duration}s"
    else
        print_warning "Proxy request duration unexpected: ${duration}s"
    fi
    
    # Step 6: Test proxy timeout (create new config with short timeout)
    print_status "Testing proxy timeout functionality..."
    
    # Kill current server
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    
    # Create config with short timeout
    cat > "$CONFIG_FILE" << EOF
# Proxy Test Configuration with Short Timeout
server:
  host: "localhost"
  port: "$TEST_PORT"

proxy:
  enabled: true
  target: "http://localhost:$PROXY_TARGET_PORT"
  timeout: "1s"  # Short timeout to test timeout functionality

spec_file: "./examples/petstore.yaml"
EOF
    
    # Restart server with short timeout
    "./$BINARY_NAME" --config "$CONFIG_FILE" > server.log 2>&1 &
    SERVER_PID=$!
    
    if ! wait_for_server "$TEST_PORT" "http"; then
        print_error "Server restart failed"
        exit 1
    fi
    
    # Test timeout - this should fail due to 1s timeout vs 2s backend delay
    print_status "Testing proxy timeout (1s timeout vs 2s backend delay - should timeout)..."
    ((TEST_TOTAL_COUNT++))
    local timeout_response=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$TEST_PORT/api/v1/slow" || echo "000")
    if [ "$timeout_response" = "504" ] || [ "$timeout_response" = "502" ] || [ "$timeout_response" = "000" ]; then
        print_success "Proxy timeout test - Got timeout/error status: $timeout_response"
        ((TEST_PASSED_COUNT++))
    else
        print_error "Proxy timeout test - Expected timeout, got: $timeout_response"
    fi
    
    # Step 7: Print test results
    echo
    print_status "Test Summary:"
    print_status "============="
    print_status "Total tests: $TEST_TOTAL_COUNT"
    print_status "Passed: $TEST_PASSED_COUNT"
    print_status "Failed: $((TEST_TOTAL_COUNT - TEST_PASSED_COUNT))"
    
    if [ $TEST_PASSED_COUNT -eq $TEST_TOTAL_COUNT ]; then
        echo
        print_success "🎉 All Proxy integration tests passed!"
        return 0
    else
        echo
        print_error "❌ Some tests failed. Check the output above for details."
        return 1
    fi
}

# Help function
show_help() {
    echo "Proxy Integration Test Script for Go-Spec-Mock"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -p, --port     Set test port (default: 8085)"
    echo "  -t, --target   Set proxy target port (default: 8086)"
    echo "  -v, --verbose  Enable verbose output"
    echo
    echo "This script will:"
    echo "  1. Build the go-spec-mock binary"
    echo "  2. Start a mock backend server"
    echo "  3. Create a test configuration with proxy enabled"
    echo "  4. Start go-spec-mock with proxy functionality"
    echo "  5. Test defined endpoints (should be mocked)"
    echo "  6. Test undefined endpoints (should be proxied)"
    echo "  7. Test proxy timeout functionality"
    echo "  8. Clean up test files"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -p|--port)
            TEST_PORT="$2"
            shift 2
            ;;
        -t|--target)
            PROXY_TARGET_PORT="$2"
            shift 2
            ;;
        -v|--verbose)
            set -x
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run the tests
run_tests