#!/bin/bash

# Smart Config Demo Script
# This script provides a comprehensive demonstration of vcfg's smart configuration features

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Available commands:"
    echo "  api        - Test basic API functionality"
    echo "  hotreload  - Test hot reload functionality (interactive)"
    echo "  isolation  - Test configuration isolation"
    echo "  live       - Test live configuration changes"
    echo "  all        - Run all tests sequentially"
    echo "  help       - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 api        # Test basic API"
    echo "  $0 hotreload  # Start hot reload demo"
    echo "  $0 all        # Run all demos"
}

# Function to check if Go files exist
check_files() {
    local files=("main.go" "plugin_kafka.go" "plugin_redis.go" "config.yaml")
    for file in "${files[@]}"; do
        if [[ ! -f "$file" ]]; then
            print_error "Required file $file not found!"
            exit 1
        fi
    done
    print_success "All required files found"
}

# Function to backup and restore config
backup_config() {
    cp config.yaml config.yaml.backup
    print_success "Configuration backed up"
}

restore_config() {
    if [[ -f "config.yaml.backup" ]]; then
        mv config.yaml.backup config.yaml
        print_success "Configuration restored"
    fi
}

# Function to test basic API
test_api() {
    print_header "Testing Basic API Functionality"
    echo "This test will verify plugin registration, startup, and basic operations."
    echo ""
    
    print_warning "Running API test..."
    go run main.go plugin_kafka.go plugin_redis.go api
    
    if [[ $? -eq 0 ]]; then
        print_success "API test completed successfully"
    else
        print_error "API test failed"
        return 1
    fi
}

# Function to test hot reload
test_hotreload() {
    print_header "Testing Hot Reload Functionality"
    echo "This test will start the configuration watcher."
    echo "You can manually modify config.yaml to see hot reload in action."
    echo ""
    
    print_warning "Starting hot reload demo..."
    echo "Press Ctrl+C to stop the demo."
    echo ""
    
    go run main.go plugin_kafka.go plugin_redis.go hotreload
}

# Function to test configuration isolation
test_isolation() {
    print_header "Testing Configuration Isolation"
    echo "This test verifies that only specific plugins are reloaded when their config changes."
    echo ""
    
    print_warning "Running isolation test..."
    go run main.go plugin_kafka.go plugin_redis.go isolation
    
    if [[ $? -eq 0 ]]; then
        print_success "Isolation test completed successfully"
    else
        print_error "Isolation test failed"
        return 1
    fi
}

# Function to test live configuration changes
test_live() {
    print_header "Testing Live Configuration Changes"
    echo "This test demonstrates configuration isolation by simulating config changes."
    echo "Note: This is the same as isolation test but with automatic config backup/restore."
    echo ""
    
    # Backup original config
    backup_config
    
    print_warning "Running live configuration test..."
    go run main.go plugin_kafka.go plugin_redis.go isolation
    
    if [[ $? -eq 0 ]]; then
        print_success "Live configuration test completed successfully"
    else
        print_error "Live configuration test failed"
        restore_config
        return 1
    fi
    
    # Restore original config
    restore_config
}

# Function to run all tests
run_all_tests() {
    print_header "Running All Smart Config Demos"
    echo "This will run all available tests sequentially."
    echo ""
    
    # Test API
    test_api
    echo ""
    
    # Test isolation
    test_isolation
    echo ""
    
    # Test live changes
    test_live
    echo ""
    
    print_success "All tests completed!"
    echo ""
    print_warning "To test hot reload interactively, run: $0 hotreload"
}

# Function to show current configuration
show_config() {
    print_header "Current Configuration"
    if [[ -f "config.yaml" ]]; then
        cat config.yaml
    else
        print_error "config.yaml not found"
    fi
}

# Main script logic
main() {
    # Check if we're in the right directory
    if [[ ! -f "main.go" ]]; then
        print_error "Please run this script from the smart_config_demo directory"
        exit 1
    fi
    
    # Check required files
    check_files
    echo ""
    
    # Parse command line arguments
    case "${1:-help}" in
        "api")
            test_api
            ;;
        "hotreload")
            test_hotreload
            ;;
        "isolation")
            test_isolation
            ;;
        "live")
            test_live
            ;;
        "all")
            run_all_tests
            ;;
        "config")
            show_config
            ;;
        "help"|"--help"|"-h")
            show_usage
            ;;
        *)
            print_error "Unknown command: $1"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Trap to ensure config is restored on script exit
trap restore_config EXIT

# Run main function
main "$@"