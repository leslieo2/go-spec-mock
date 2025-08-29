#!/bin/bash

# Core API Mocking Integration Test Script for Go-Spec-Mock
# This script tests the basic HTTP API mocking functionality.

set -e

# Colors for output
RED='[0;31m'
GREEN='[0;32m'
YELLOW='[1;33m'
BLUE='[0;34m'
NC='[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_PORT="8081" # Use a distinct port for this module
CONFIG_FILE="test-api-mocking-config.yaml"
BINARY_NAME="go-spec-mock"
SERVER_PID=""

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
    
    # Kill server if running
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        print_status "Server stopped (PID: $SERVER_PID)"
    fi
    
    # Remove test files
    cd "$PROJECT_DIR"
    rm -f "$CONFIG_FILE" "server.log" "$BINARY_NAME"
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

# Function to test HTTP/HTTPS endpoint
test_api_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=${3:-200}
    local description=${4:-"Testing $method $endpoint"}
    local data=$5 # Optional data for POST/PUT
    local headers=$6 # Optional headers
    local protocol=$7 # "http" or "https"
    
    print_status "$description"
    
    local response
    local status_code
    local curl_cmd="curl -s -o /dev/null -w "%{http_code}" -X $method $protocol://localhost:$TEST_PORT$endpoint"
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -H "Content-Type: application/json" -d '$data'"
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

# Main test function
run_tests() {
    print_status "Starting Core API Mocking Integration Tests..."
    
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
    
    # Step 2: Create test configuration
    print_status "Creating test configuration for API mocking..."
    cat > "$CONFIG_FILE" << EOF
# API Mocking Test Configuration
server:
  host: "localhost"
  port: "$TEST_PORT"
  metrics_port: "9091" # Distinct metrics port
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"
  max_request_size: 10485760
  shutdown_timeout: "30s"

tls:
  enabled: false

security:
  auth:
    enabled: false
  headers:
    enabled: false
  cors:
    enabled: false
  request_limit:
    enabled: false

observability:
  logging:
    level: "info"
    format: "console"
  metrics:
    enabled: false # Disable metrics for this specific test

hot_reload:
  enabled: false

proxy:
  enabled: false

spec_file: "./examples/petstore.yaml"
EOF
    print_success "Test configuration created"
    
    # Step 3: Start the server
    print_status "Starting HTTP server for API mocking tests..."
    "./$BINARY_NAME" --spec-file ./examples/petstore.yaml --config "$CONFIG_FILE" --port "$TEST_PORT" > server.log 2>&1 &
    SERVER_PID=$!
    
    # Wait for server to be ready
    if ! wait_for_server "$TEST_PORT" "http"; then
        print_error "Server startup failed"
        cat server.log
        exit 1
    fi
    
    # Step 4: Run API mocking tests
    print_status "Running API mocking tests..."
    
    # Test health endpoint (basic server check)
    test_api_endpoint "GET" "/health" 200 "Health endpoint test" "" "" "http"
    
    # API endpoints from petstore.yaml
    test_api_endpoint "GET" "/pets" 200 "GET /pets (list all pets)" "" "" "http"
    test_api_endpoint "POST" "/pets" 200 "POST /pets (create pet)" '{"name":"test-pet","photoUrls":["http://example.com/photo.jpg"]}' "" "http"
    test_api_endpoint "GET" "/pets/1" 200 "GET /pets/{petId} (get pet by ID)" "" "" "http" # Assuming petstore.yaml mocks this
    test_api_endpoint "PUT" "/pets/1" 200 "PUT /pets/{petId} (update pet)" '{"name":"updated-pet","photoUrls":["http://example.com/photo.jpg"]}' "" "http"
    test_api_endpoint "DELETE" "/pets/1" 200 "DELETE /pets/{petId} (delete pet)" "" "" "http"
    
    # Test a non-existent endpoint (should return 404)
    test_api_endpoint "GET" "/non-existent-path" 404 "GET /non-existent-path (should be 404)" "" "" "http"

    # Step 5: Print test results
    echo
    print_status "Test Summary:"
    print_status "============="
    print_status "Total tests: $TEST_TOTAL_COUNT"
    print_status "Passed: $TEST_PASSED_COUNT"
    print_status "Failed: $((TEST_TOTAL_COUNT - TEST_PASSED_COUNT))"
    
    if [ $TEST_PASSED_COUNT -eq $TEST_TOTAL_COUNT ]; then
        echo
        print_success "🎉 All Core API Mocking integration tests passed!"
        return 0
    else
        echo
        print_error "❌ Some tests failed. Check the output above for details."
        return 1
    fi
}

# Help function
show_help() {
    echo "Core API Mocking Integration Test Script for Go-Spec-Mock"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -p, --port     Set test port (default: 8081)"
    echo "  -v, --verbose  Enable verbose output"
    echo
    echo "This script will:"
    echo "  1. Build the go-spec-mock binary"
    echo "  2. Create a test configuration for API mocking"
    echo "  3. Start HTTP server"
    echo "  4. Run comprehensive HTTP API mocking tests"
    echo "  5. Clean up test files"
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
