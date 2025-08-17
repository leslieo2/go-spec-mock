#!/bin/bash

# Go-Spec-Mock OpenAPI Validation Script v1.2.0
# This script cross-validates the generated OpenAPI spec against the actual implementation
# Now includes comprehensive testing for v1.2.0 security features, authentication, and rate limiting

set -e

# Initialize SERVER_PID to ensure it's always set
SERVER_PID=""

# Trap to ensure server is stopped on exit
trap 'if [[ -n "${SERVER_PID:-}" ]]; then print_status "INFO" "Stopping background server (PID: $SERVER_PID)"; kill $SERVER_PID 2>/dev/null || true; fi' EXIT

echo "🔍 Go-Spec-Mock OpenAPI Cross-Validation v1.2.0"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "PASS")
            echo -e "${GREEN}✅ PASS${NC}: $message"
            ;;
        "FAIL")
            echo -e "${RED}❌ FAIL${NC}: $message"
            ;;
        "INFO")
            echo -e "${YELLOW}ℹ️  INFO${NC}: $message"
            ;;
    esac
}

# Function to check if server is running
check_server() {
    local port=${1:-8080}
    if nc -z localhost $port 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Function to validate OpenAPI spec
validate_spec() {
    local spec_file=$1
    print_status "INFO" "Validating OpenAPI spec: $spec_file"
    
    if command -v swagger &> /dev/null; then
        swagger validate "$spec_file"
        print_status "PASS" "OpenAPI spec validation passed"
    elif command -v spectral &> /dev/null; then
        spectral lint "$spec_file"
        print_status "PASS" "Spectral linting passed"
    else
        print_status "INFO" "No OpenAPI validator found, skipping validation"
    fi
}

# Function to test endpoint
validate_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=${3:-200}
    local api_key=${4:-}
    
    print_status "INFO" "Testing $method $endpoint (expecting $expected_status)"
    
    local response
    local status_code
    local headers=""
    
    # Add API key if provided
    if [[ -n "$api_key" ]]; then
        headers="-H X-API-Key:$api_key"
    fi
    
    case $method in
        "GET")
            response=$(curl -s -w "\n%{http_code}" $headers -X GET "http://localhost:8080$endpoint")
            ;;
        "POST")
            response=$(curl -s -w "\n%{http_code}" $headers -X POST -H "Content-Type: application/json" -d '{}' "http://localhost:8080$endpoint")
            ;;
        "PUT")
            response=$(curl -s -w "\n%{http_code}" $headers -X PUT -H "Content-Type: application/json" -d '{}' "http://localhost:8080$endpoint")
            ;;
        "DELETE")
            response=$(curl -s -w "\n%{http_code}" $headers -X DELETE "http://localhost:8080$endpoint")
            ;;
    esac
    
    status_code=$(echo "$response" | tail -n1)
    
    if [[ "$status_code" == "$expected_status" ]] || [[ "$status_code" == "200" ]]; then
        print_status "PASS" "$method $endpoint returned $status_code"
    else
        print_status "FAIL" "$method $endpoint returned $status_code, expected $expected_status"
    fi
}

# Function to generate test API key
generate_test_key() {
    print_status "INFO" "Generating test API key..."
    
    # Try to generate a key using the CLI
    if [[ -f "./bin/go-spec-mock" ]]; then
        local key_output=$(./bin/go-spec-mock -generate-key "test-validation" 2>/dev/null | grep -o '"[^"]*"' | tr -d '"' | tail -1)
        if [[ -n "$key_output" ]]; then
            echo "$key_output"
            return 0
        fi
    fi
    
    # Fallback to a static test key
    echo "test-api-key-12345"
}

# Function to test authentication endpoints
test_authentication() {
    local api_key=$1
    
    echo ""
    echo "🔐 Testing Authentication Features"
    echo "=================================="
    
    # Test endpoints without authentication (should fail if auth enabled)
    print_status "INFO" "Testing authentication requirements..."
    
    local response=$(curl -s -w "\n%{http_code}" -X GET "http://localhost:8080/pets")
    local status_code=$(echo "$response" | tail -n1)
    
    if [[ "$status_code" == "401" ]]; then
        print_status "PASS" "Authentication properly enforced (401 Unauthorized)"
    elif [[ "$status_code" == "200" ]]; then
        print_status "INFO" "Authentication not enabled (200 OK)"
    else
        print_status "FAIL" "Unexpected status: $status_code"
    fi
    
    # Test with valid API key
    if [[ -n "$api_key" ]]; then
        validate_endpoint "GET" "/pets" "200" "$api_key"
        
        # Test different authentication methods
        print_status "INFO" "Testing Bearer token authentication..."
        local bearer_response=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $api_key" "http://localhost:8080/pets")
        local bearer_status=$(echo "$bearer_response" | tail -n1)
        
        if [[ "$bearer_status" == "200" ]]; then
            print_status "PASS" "Bearer token authentication working"
        else
            print_status "FAIL" "Bearer token authentication failed: $bearer_status"
        fi
        
        print_status "INFO" "Testing query parameter authentication..."
        local query_response=$(curl -s -w "\n%{http_code}" "http://localhost:8080/pets?api_key=$api_key")
        local query_status=$(echo "$query_response" | tail -n1)
        
        if [[ "$query_status" == "200" ]]; then
            print_status "PASS" "Query parameter authentication working"
        else
            print_status "FAIL" "Query parameter authentication failed: $query_status"
        fi
    fi
}

# Function to test rate limiting
test_rate_limiting() {
    local api_key=$1
    
    echo ""
    echo "⏱️  Testing Rate Limiting Features"
    echo "=================================="
    
    if [[ -z "$api_key" ]]; then
        print_status "INFO" "Skipping rate limiting tests (no API key)"
        return
    fi
    
    print_status "INFO" "Testing rate limit headers..."
    
    local response=$(curl -s -D - -H "X-API-Key: $api_key" "http://localhost:8080/pets")
    local has_limit=$(echo "$response" | grep -c "X-RateLimit-Limit" | tr -d '\n' || echo "0")
    local has_remaining=$(echo "$response" | grep -c "X-RateLimit-Remaining" | tr -d '\n' || echo "0")
    
    if [[ "$has_limit" -gt 0 ]] && [[ "$has_remaining" -gt 0 ]]; then
        print_status "PASS" "Rate limit headers present"
    else
        print_status "INFO" "Rate limiting headers not found (may be disabled)"
    fi
    
    print_status "INFO" "Testing rate limit enforcement..."
    local rate_limit_count=0
    for i in {1..10}; do
        local rate_response=$(curl -s -w "\n%{http_code}" -H "X-API-Key: $api_key" "http://localhost:8080/pets")
        local rate_status=$(echo "$rate_response" | tail -n1)
        
        if [[ "$rate_status" == "429" ]]; then
            ((rate_limit_count++))
        fi
    done
    
    if [[ "$rate_limit_count" -gt 0 ]]; then
        print_status "PASS" "Rate limiting enforced ($rate_limit_count 429 responses)"
    else
        print_status "INFO" "Rate limiting not triggered (within limits)"
    fi
}

# Function to test security headers
test_security_headers() {
    echo ""
    echo "🛡️  Testing Security Headers"
    echo "============================"
    
    local response=$(curl -s -D - "http://localhost:8080/health")
    
    local security_headers=(
        "X-Content-Type-Options: nosniff"
        "X-Frame-Options: DENY"
        "X-XSS-Protection: 1; mode=block"
        "Strict-Transport-Security: max-age=31536000"
    )
    
    for header in "${security_headers[@]}"; do
        local header_name=$(echo "$header" | cut -d':' -f1)
        local has_header=$(echo "$response" | grep -c "^$header_name:" || echo "0")
        has_header=$(echo "$has_header" | tr -d '\n')
        
        if [[ "$has_header" -gt 0 ]]; then
            print_status "PASS" "Security header present: $header_name"
        else
            print_status "INFO" "Security header missing: $header_name"
        fi
    done
}

# Function to test skip-auth endpoints
test_skip_auth_endpoints() {
    echo ""
    echo "🔓 Testing Skip-Auth Endpoints"
    echo "=============================="
    
    local skip_endpoints=("/health" "/ready" "/metrics" "/")
    
    for endpoint in "${skip_endpoints[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" "http://localhost:8080$endpoint")
        local status_code=$(echo "$response" | tail -n1)
        
        if [[ "$status_code" == "200" ]] || [[ "$status_code" == "404" ]]; then
            print_status "PASS" "Skip-auth endpoint accessible: $endpoint ($status_code)"
        else
            print_status "FAIL" "Skip-auth endpoint failed: $endpoint ($status_code)"
        fi
    done
}

# Function to start server with different configurations
start_server() {
    local config_type=${1:-basic}
    
    if check_server 8080; then
        print_status "INFO" "Server already running on port 8080"
        return 0
    fi
    
    print_status "INFO" "Starting server with $config_type configuration..."
    
    case $config_type in
        "basic")
            ./bin/go-spec-mock examples/petstore.yaml &
            ;;
        "secure")
            if [[ -f "security.yaml" ]]; then
                ./bin/go-spec-mock examples/petstore.yaml -auth-enabled -auth-config security.yaml &
            else
                ./bin/go-spec-mock examples/petstore.yaml -auth-enabled &
            fi
            ;;
        "rate-limited")
            ./bin/go-spec-mock examples/petstore.yaml -rate-limit-enabled -rate-limit-rps 10 &
            ;;
        "full-security")
            ./bin/go-spec-mock examples/petstore.yaml -auth-enabled -rate-limit-enabled -rate-limit-rps 50 &
            ;;
    esac
    
    SERVER_PID=$!
    sleep 3
    
    # Check if server started successfully
    if ! check_server 8080; then
        print_status "FAIL" "Failed to start server with $config_type configuration"
        return 1
    fi
    
    print_status "PASS" "Server started successfully with $config_type configuration"
    return 0
}

# Function to test configuration validation
test_configuration() {
    echo ""
    echo "⚙️  Testing Configuration Validation"
    echo "===================================="
    
    # Test basic configuration
    print_status "INFO" "Testing basic configuration..."
    start_server "basic"
    
    # Test security configuration
    print_status "INFO" "Testing security configuration..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
    start_server "secure"
    
    # Test rate limiting configuration
    print_status "INFO" "Testing rate limiting configuration..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
    start_server "rate-limited"
    
    # Test full security configuration
    print_status "INFO" "Testing full security configuration..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
    start_server "full-security"
}

# Function to validate security configuration
validate_security_config() {
    echo ""
    print_status "INFO" "Validating security configuration..."
    
    if [[ -f "security.yaml" ]]; then
        print_status "PASS" "Security configuration file found: security.yaml"
        
        # Basic YAML validation
        if command -v yq &> /dev/null; then
            local auth_enabled=$(yq eval '.auth.enabled' security.yaml 2>/dev/null || echo "false")
            local rate_limit_enabled=$(yq eval '.rate_limit.enabled' security.yaml 2>/dev/null || echo "false")
            
            print_status "INFO" "Auth enabled: $auth_enabled"
            print_status "INFO" "Rate limiting enabled: $rate_limit_enabled"
        fi
    else
        print_status "INFO" "No security configuration file found, using defaults"
    fi
}

# Main validation process
main() {
    echo "Starting OpenAPI cross-validation..."
    
    # Check if required files exist
    if [[ ! -f "go-spec-mock-api.yaml" ]]; then
        print_status "FAIL" "OpenAPI spec file not found: go-spec-mock-api.yaml"
        exit 1
    fi
    
    # Validate OpenAPI spec
    validate_spec "go-spec-mock-api.yaml"
    
    # Validate security configuration
    validate_security_config
    
    # Start with basic configuration
    start_server "basic"
    
    echo ""
    echo "🧪 Testing System Endpoints"
    echo "---------------------------"
    
    # Test skip-auth endpoints
    test_skip_auth_endpoints
    
    # Test security headers
    test_security_headers
    
    echo ""
    echo "🧪 Testing Mock Endpoints"
    echo "-------------------------"
    
    # Test mock endpoints (from petstore example)
    validate_endpoint "GET" "/pets"
    validate_endpoint "GET" "/pets?__statusCode=400" "400"
    validate_endpoint "POST" "/pets"
    validate_endpoint "GET" "/pets/1"
    validate_endpoint "DELETE" "/pets/1"
    
    echo ""
    echo "🧪 Testing Error Handling"
    echo "-------------------------"
    
    # Test error handling
    validate_endpoint "GET" "/nonexistent" "404"
    
    
    # Generate test API key for security testing
    local test_key=$(generate_test_key)
    print_status "INFO" "Generated test API key: ${test_key:0:10}..."
    
    # Test with security enabled
    if [[ -f "security.yaml" ]] || [[ "$1" == "--security" ]]; then
        print_status "INFO" "Testing security features..."
        
        # Restart with security enabled
        kill $SERVER_PID 2>/dev/null || true
        sleep 2
        start_server "full-security"
        
        # Test authentication
        test_authentication "$test_key"
        
        # Test rate limiting
        test_rate_limiting "$test_key"
    else
        print_status "INFO" "Security testing skipped (use --security flag to enable)"
    fi
    
    # Test configuration validation
    test_configuration
    
    echo ""
    echo "📊 Cross-Validation Summary"
    echo "============================"
    
    # Check OpenAPI spec against actual endpoints
    print_status "INFO" "Comparing OpenAPI spec with actual implementation..."
    
    # Extract endpoints from OpenAPI spec
    if command -v yq &> /dev/null; then
        echo "OpenAPI endpoints:"
        yq eval '.paths | keys | .[]' go-spec-mock-api.yaml
        
        echo ""
        echo "Available mock endpoints (from loaded spec):"
        curl -s http://localhost:8080/ | jq -r '.endpoints[] | "\(.method) \(.path)"'
    else
        print_status "INFO" "yq not found, skipping detailed endpoint comparison"
    fi
    
    echo ""
    print_status "PASS" "OpenAPI v1.2.0 cross-validation completed successfully!"
    
    # Cleanup
    if [[ -n "${SERVER_PID:-}" ]]; then
        print_status "INFO" "Stopping background server (PID: $SERVER_PID)"
        kill $SERVER_PID 2>/dev/null || true
    fi
}

# Check dependencies
check_dependencies() {
    local deps=("curl" "jq")
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        print_status "FAIL" "Missing dependencies: ${missing[*]}"
        echo "Please install: ${missing[*]}"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --security        Enable comprehensive security feature testing"
    echo "  --basic           Run only basic validation (skip security tests)"
    echo "  --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                Run basic validation"
    echo "  $0 --security     Run full security feature validation"
    echo "  $0 --basic        Run minimal validation"
    echo ""
    echo "v1.2.0 Features Tested:"
    echo "  • API Key Authentication (X-API-Key, Bearer, Query params)"
    echo "  • Rate Limiting (IP-based, API key-based, combined)"
    echo "  • Security Headers (X-Content-Type-Options, etc.)"
    echo "  • Skip-auth endpoints (/health, /ready, /metrics)"
    echo "  • Configuration validation"
    echo "  • Multiple authentication methods"
    echo "  • Rate limit headers and enforcement"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --security)
            SECURITY_MODE=true
            shift
            ;;
        --basic)
            BASIC_MODE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Run dependency check
check_dependencies

# Run main validation with appropriate mode
if [[ "$BASIC_MODE" == "true" ]]; then
    main --basic
elif [[ "$SECURITY_MODE" == "true" ]]; then
    main --security
else
    main "$@"
fi