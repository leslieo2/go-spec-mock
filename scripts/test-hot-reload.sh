#!/bin/bash

# Hot Reload Integration Test Script for Go-Spec-Mock
# This script tests the hot reload functionality.

set -e

export no_proxy=localhost,127.0.0.1

# Colors for output
RED='[0;31m'
GREEN='[0;32m'
YELLOW='[1;33m'
BLUE='[0;34m'
NC='[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_PORT="8084" # Use a distinct port for this module
CONFIG_FILE="test-hot-reload-config.yaml"
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
    rm -f "$CONFIG_FILE" "$PROJECT_DIR/temp_petstore.yaml" "server.log" "$BINARY_NAME"
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
    print_status "Starting Hot Reload Integration Tests..."
    
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
    print_status "Creating test configuration for Hot Reload..."
    cat > "$CONFIG_FILE" << EOF
# Hot Reload Test Configuration
server:
  host: "localhost"
  port: "$TEST_PORT"

tls:
  enabled: false

security:
  cors:
    enabled: false

observability:
  logging:
    level: "info"
    format: "console"

hot_reload:
  enabled: true # Enable hot reload
  debounce: "500ms"

proxy:
  enabled: false

spec_file: "./examples/petstore.yaml"
EOF
    print_success "Test configuration created"
    
    # Step 3: Run Hot Reload Test
    print_status "Running Hot Reload test..."
    local original_spec_file="$PROJECT_DIR/examples/petstore.yaml"
    local temp_spec_file="$PROJECT_DIR/temp_petstore.yaml"

    # Copy original spec to a temp file for modification
    cp "$original_spec_file" "$temp_spec_file"

    # Start server with hot reload enabled and using the temp spec file
    print_status "Starting server for hot reload test..."
    "./$BINARY_NAME" --spec-file "$temp_spec_file" --config "$CONFIG_FILE" --port "$TEST_PORT" --hot-reload > server.log 2>&1 &
    SERVER_PID=$!

    if ! wait_for_server "$TEST_PORT" "http"; then
        print_error "Server startup failed for hot reload test"
        cat server.log
        exit 1
    fi

    # Initial request
    local initial_response=$(curl -s "http://localhost:$TEST_PORT/pets" | grep -o "name")
    ((TEST_TOTAL_COUNT++))
    if echo "$initial_response" | grep -q "name"; then
        print_success "Initial request to /pets successful."
        ((TEST_PASSED_COUNT++))
    else
        print_error "Initial request to /pets failed."
        # No need to exit, let the test continue to report all failures
    fi

    # Modify the spec file
    print_status "Modifying spec file: changing 'name: doggie' to 'name: hot-reloaded-doggie'..."
    sed -i '' 's/name: doggie/name: hot-reloaded-doggie/' "$temp_spec_file"

    # Wait for hot reload to take effect (debounce is 500ms in config, wait a bit longer)
    print_status "Waiting for hot reload (2 seconds)..."
    sleep 2

    # Test modified endpoint
    local modified_response=$(curl -s "http://localhost:$TEST_PORT/pets" | grep -o "hot-reloaded-doggie")
    ((TEST_TOTAL_COUNT++))
    if echo "$modified_response" | grep -q "hot-reloaded-doggie"; then
        print_success "Hot reload successful: 'name: hot-reloaded-doggie' is accessible."
        ((TEST_PASSED_COUNT++))
        return 0
    else
        print_error "Hot reload failed: 'name: hot-reloaded-doggie' not accessible."
        return 1
    fi

    # Step 4: Print test results
    echo
    print_status "Test Summary:"
    print_status "============="
    print_status "Total tests: $TEST_TOTAL_COUNT"
    print_status "Passed: $TEST_PASSED_COUNT"
    print_status "Failed: $((TEST_TOTAL_COUNT - TEST_PASSED_COUNT))"
    
    if [ $TEST_PASSED_COUNT -eq $TEST_TOTAL_COUNT ]; then
        echo
        print_success "🎉 All Hot Reload integration tests passed!"
        return 0
    else
        echo
        print_error "❌ Some tests failed. Check the output above for details."
        return 1
    fi
}

# Help function
show_help() {
    echo "Hot Reload Integration Test Script for Go-Spec-Mock"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -p, --port     Set test port (default: 8084)"
    echo "  -v, --verbose  Enable verbose output"
    echo
    echo "This script will:"
    echo "  1. Build the go-spec-mock binary"
    echo "  2. Create a test configuration for Hot Reload"
    echo "  3. Start HTTP server with hot reload enabled"
    echo "  4. Modify the spec file and verify hot reload"
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
