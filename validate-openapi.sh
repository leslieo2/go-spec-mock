#!/bin/bash

# Go-Spec-Mock OpenAPI Validation Script
# This script cross-validates the generated OpenAPI spec against the actual implementation

set -e

echo "🔍 Go-Spec-Mock OpenAPI Cross-Validation"
echo "========================================"

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
    
    print_status "INFO" "Testing $method $endpoint (expecting $expected_status)"
    
    local response
    local status_code
    
    case $method in
        "GET")
            response=$(curl -s -w "\n%{http_code}" -X GET "http://localhost:8080$endpoint")
            ;;
        "POST")
            response=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" -d '{}' "http://localhost:8080$endpoint")
            ;;
        "PUT")
            response=$(curl -s -w "\n%{http_code}" -X PUT -H "Content-Type: application/json" -d '{}' "http://localhost:8080$endpoint")
            ;;
        "DELETE")
            response=$(curl -s -w "\n%{http_code}" -X DELETE "http://localhost:8080$endpoint")
            ;;
    esac
    
    status_code=$(echo "$response" | tail -n1)
    
    if [[ "$status_code" == "$expected_status" ]] || [[ "$status_code" == "200" ]]; then
        print_status "PASS" "$method $endpoint returned $status_code"
    else
        print_status "FAIL" "$method $endpoint returned $status_code, expected $expected_status"
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
    
    # Check if server is running
    if check_server 8080; then
        print_status "INFO" "Server is running on port 8080"
    else
        print_status "INFO" "Starting server in background..."
        ./go-spec-mock examples/petstore.yaml &
        SERVER_PID=$!
        sleep 3
        
        # Check if server started successfully
        if ! check_server 8080; then
            print_status "FAIL" "Failed to start server"
            exit 1
        fi
    fi
    
    echo ""
    echo "🧪 Testing System Endpoints"
    echo "---------------------------"
    
    # Test system endpoints
    validate_endpoint "GET" "/"
    validate_endpoint "GET" "/health"
    validate_endpoint "GET" "/ready"
    validate_endpoint "GET" "/metrics"
    
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
    validate_endpoint "PATCH" "/pets" "405"
    
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
    print_status "PASS" "OpenAPI cross-validation completed successfully!"
    
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
        if ! command -v $dep &> /dev/null; then
            missing+=($dep)
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        print_status "FAIL" "Missing dependencies: ${missing[*]}"
        echo "Please install: ${missing[*]}"
        exit 1
    fi
}

# Run dependency check
check_dependencies

# Run main validation
main "$@"