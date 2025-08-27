#!/bin/bash

# TLS Integration Test Script for Go-Spec-Mock
# This script tests the HTTPS/TLS functionality from user perspective

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_PORT="8443"
METRICS_PORT="9090"
CERT_FILE="test-cert.pem"
KEY_FILE="test-key.pem"
CONFIG_FILE="test-tls-config.yaml"
BINARY_NAME="go-spec-mock"
SERVER_PID=""

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
    
    # Kill server if running
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        print_status "Server stopped (PID: $SERVER_PID)"
    fi
    
    # Remove test files
    cd "$PROJECT_DIR"
    rm -f "$CERT_FILE" "$KEY_FILE" "$CONFIG_FILE"
    print_status "Test files cleaned up"
}

# Set trap for cleanup
trap cleanup EXIT

# Function to wait for server startup
wait_for_server() {
    local port=$1
    local max_attempts=30
    local attempt=0
    
    print_status "Waiting for server to start on port $port..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -k -s --connect-timeout 1 "https://localhost:$port/health" > /dev/null 2>&1; then
            print_success "Server is ready on port $port"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    
    print_error "Server failed to start within $max_attempts seconds"
    return 1
}

# Function to test HTTP endpoint
test_https_endpoint() {
    local endpoint=$1
    local expected_status=${2:-200}
    local description=${3:-"Testing $endpoint"}
    
    print_status "$description"
    
    local response
    local status_code
    
    response=$(curl -k -s -w "%{http_code}" "https://localhost:$TEST_PORT$endpoint" || echo "000")
    status_code="${response: -3}"
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "$description - Status: $status_code"
        return 0
    else
        print_error "$description - Expected: $expected_status, Got: $status_code"
        return 1
    fi
}

# Function to test TLS connection details
test_tls_connection() {
    print_status "Testing TLS connection details..."
    
    local tls_info
    tls_info=$(echo | openssl s_client -connect localhost:$TEST_PORT -servername localhost 2>/dev/null | grep -E "(Protocol|Cipher)")
    
    if echo "$tls_info" | grep -q "TLSv1.3"; then
        print_success "TLS 1.3 protocol confirmed"
    else
        print_warning "TLS 1.3 not detected, got: $tls_info"
    fi
    
    if echo "$tls_info" | grep -q "TLS_AES_128_GCM_SHA256"; then
        print_success "Secure cipher suite confirmed"
    else
        print_warning "Expected cipher suite not found, got: $tls_info"
    fi
}

# Function to test certificate details
test_certificate() {
    print_status "Testing certificate details..."
    
    local cert_info
    cert_info=$(echo | openssl s_client -connect localhost:$TEST_PORT -servername localhost 2>/dev/null | openssl x509 -noout -subject -issuer 2>/dev/null)
    
    if echo "$cert_info" | grep -q "CN=localhost"; then
        print_success "Certificate subject contains localhost"
    else
        print_error "Certificate subject validation failed: $cert_info"
        return 1
    fi
}

# Function to test HTTP to HTTPS rejection
test_http_rejection() {
    print_status "Testing HTTP request to HTTPS port (should fail)..."
    
    # Use timeout to prevent hanging
    local response
    response=$(timeout 5s curl -s -w "%{http_code}" "http://localhost:$TEST_PORT/health" 2>&1 || echo "failed")
    
    if echo "$response" | grep -q -E "(failed|Empty reply|SSL_ERROR|Connection reset)"; then
        print_success "HTTP requests correctly rejected on HTTPS port"
        return 0
    else
        print_error "HTTP request should have failed but got: $response"
        return 1
    fi
}

# Main test function
run_tests() {
    print_status "Starting TLS Integration Tests..."
    
    # Change to project directory
    cd "$PROJECT_DIR"
    
    # Step 1: Build the project
    print_status "Building go-spec-mock..."
    if ! go build -o "$BINARY_NAME" .; then
        print_error "Failed to build project"
        exit 1
    fi
    print_success "Project built successfully"
    
    # Step 2: Generate test certificates
    print_status "Generating test TLS certificates..."
    if ! openssl req -x509 -newkey rsa:2048 -keyout "$KEY_FILE" -out "$CERT_FILE" -days 365 -nodes \
        -subj "/C=US/ST=Test/L=Test/O=Test/CN=localhost" > /dev/null 2>&1; then
        print_error "Failed to generate TLS certificates"
        exit 1
    fi
    print_success "TLS certificates generated"
    
    # Step 3: Create test configuration
    print_status "Creating test configuration..."
    cat > "$CONFIG_FILE" << EOF
# TLS Integration Test Configuration
server:
  host: "localhost"
  port: "$TEST_PORT"
  metrics_port: "$METRICS_PORT"
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"
  max_request_size: 10485760
  shutdown_timeout: "30s"

tls:
  enabled: true
  cert_file: "$CERT_FILE"
  key_file: "$KEY_FILE"

security:
  auth:
    enabled: false
  rate_limit:
    enabled: false
  headers:
    enabled: true
    content_security_policy: "default-src 'self'"
    hsts_max_age: 31536000
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"]
    allowed_headers: ["Content-Type", "Authorization", "Accept", "X-Requested-With"]
    allow_credentials: false
    max_age: 86400

observability:
  logging:
    level: "info"
    format: "console"
  metrics:
    enabled: true
    path: "/metrics"
  tracing:
    enabled: false

hot_reload:
  enabled: true
  debounce: "500ms"

proxy:
  enabled: false

spec_file: "./examples/petstore.yaml"
EOF
    print_success "Test configuration created"
    
    # Step 4: Start the server
    print_status "Starting HTTPS server..."
    "./$BINARY_NAME" --spec-file ./examples/petstore.yaml --config "$CONFIG_FILE" --port "$TEST_PORT" --tls-enabled --tls-cert-file "$CERT_FILE" --tls-key-file "$KEY_FILE" > server.log 2>&1 &
    SERVER_PID=$!
    
    # Wait for server to be ready
    if ! wait_for_server "$TEST_PORT"; then
        print_error "Server startup failed"
        cat server.log
        exit 1
    fi
    
    # Step 5: Run HTTPS endpoint tests
    local test_passed=0
    local test_total=0
    
    # Test health endpoint
    ((test_total++))
    if test_https_endpoint "/health" 200 "Health endpoint test"; then
        ((test_passed++))
    fi
    
    # Test API endpoint
    ((test_total++))
    if test_https_endpoint "/pets" 200 "API endpoint test"; then
        ((test_passed++))
    fi
    
    # Test metrics endpoint
    ((test_total++))
    if test_https_endpoint "/metrics" 200 "Metrics endpoint test"; then
        ((test_passed++))
    fi
    
    # Test POST endpoint
    ((test_total++))
    print_status "Testing HTTPS POST request..."
    local post_response
    post_response=$(curl -k -s -X POST "https://localhost:$TEST_PORT/pets" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-pet","photoUrls":["http://example.com/photo.jpg"]}' \
        -w "%{http_code}" || echo "000")
    local post_status="${post_response: -3}"
    
    if [ "$post_status" = "200" ]; then
        print_success "HTTPS POST request test - Status: $post_status"
        ((test_passed++))
    else
        print_error "HTTPS POST request test - Expected: 200, Got: $post_status"
    fi
    
    # Test TLS connection details
    ((test_total++))
    if test_tls_connection; then
        ((test_passed++))
    fi
    
    # Test certificate
    ((test_total++))
    if test_certificate; then
        ((test_passed++))
    fi
    
    # Test HTTP rejection
    ((test_total++))
    if test_http_rejection; then
        ((test_passed++))
    fi
    
    # Step 6: Print test results
    echo
    print_status "Test Summary:"
    print_status "============="
    print_status "Total tests: $test_total"
    print_status "Passed: $test_passed"
    print_status "Failed: $((test_total - test_passed))"
    
    if [ $test_passed -eq $test_total ]; then
        echo
        print_success "🎉 All TLS integration tests passed!"
        return 0
    else
        echo
        print_error "❌ Some tests failed. Check the output above for details."
        return 1
    fi
}

# Help function
show_help() {
    echo "TLS Integration Test Script for Go-Spec-Mock"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -p, --port     Set test port (default: 8443)"
    echo "  -v, --verbose  Enable verbose output"
    echo
    echo "This script will:"
    echo "  1. Build the go-spec-mock binary"
    echo "  2. Generate test TLS certificates"
    echo "  3. Create a test configuration"
    echo "  4. Start HTTPS server"
    echo "  5. Run comprehensive TLS tests"
    echo "  6. Clean up test files"
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