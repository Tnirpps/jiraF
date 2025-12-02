#!/bin/bash

# Script to run Go tests

set -e

# Default values
VERBOSE=false
PACKAGE_PATH="./..."
RACE=false
COVER=false
TEST_TIMEOUT="30s"
CLEAN_CACHE=false
UPDATE_GOLDEN=false

# Print help message
function show_help {
    echo "Usage: ./run_tests.sh [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help           Show this help message"
    echo "  -v, --verbose        Run tests in verbose mode"
    echo "  -p, --package        Specify package path to test (default: ./...)"
    echo "  -r, --race           Run tests with race detector"
    echo "  -c, --cover          Run tests with coverage"
    echo "  -t, --timeout        Set test timeout (default: 5m)"
    echo "  --clean-cache        Clear test cache before running"
    echo "  --update-golden      Update golden files"
    echo ""
    echo "Examples:"
    echo "  ./run_tests.sh                         # Run all tests"
    echo "  ./run_tests.sh -v                      # Run all tests with verbose output"
    echo "  ./run_tests.sh -p ./internal/todoist   # Run only todoist tests"
    echo "  ./run_tests.sh -v -r -c                # Run all tests with verbose output, race detection, and coverage"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -p|--package)
            PACKAGE_PATH="$2"
            shift 2
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -c|--cover)
            COVER=true
            shift
            ;;
        -t|--timeout)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --clean-cache)
            CLEAN_CACHE=true
            shift
            ;;
        --update-golden)
            UPDATE_GOLDEN=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Clean test cache if requested
if [ "$CLEAN_CACHE" = true ]; then
    echo "Cleaning test cache..."
    go clean -testcache
fi

# Build test command
TEST_CMD="go test"

# Add timeout
TEST_CMD="$TEST_CMD -timeout $TEST_TIMEOUT"

# Add verbosity if requested
if [ "$VERBOSE" = true ]; then
    TEST_CMD="$TEST_CMD -v"
fi

# Add race detection if requested
if [ "$RACE" = true ]; then
    TEST_CMD="$TEST_CMD -race"
fi

# Add coverage if requested
if [ "$COVER" = true ]; then
    TEST_CMD="$TEST_CMD -coverprofile=coverage.out"
fi

# Add update golden files if requested
if [ "$UPDATE_GOLDEN" = true ]; then
    TEST_CMD="$TEST_CMD -update"
fi

# Add package path
TEST_CMD="$TEST_CMD $PACKAGE_PATH"

# Run tests
echo "Running tests: $TEST_CMD"
eval $TEST_CMD

# Show coverage report if requested
if [ "$COVER" = true ]; then
    echo "Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report generated: coverage.html"
fi

echo "Tests completed successfully!"
