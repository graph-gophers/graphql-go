#!/bin/bash

# Test script to verify v1.7.2 fixes the checksum issue
# This script should be run after the v1.7.2 tag is pushed to GitHub

set -e

echo "Testing v1.7.2 checksum consistency..."

# Create test directory
TEST_DIR="/tmp/graphql-go-checksum-test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Create a test module
cat > go.mod << EOF
module checksum-test

go 1.16

require github.com/graph-gophers/graphql-go v1.7.2
EOF

echo "Testing with GOPROXY=direct..."
GOPROXY=direct go mod download -json github.com/graph-gophers/graphql-go@v1.7.2 | jq -r '.Error // "SUCCESS"'

echo "Cleaning module cache..."
go clean -modcache

echo "Testing with default proxy (including sum.golang.org)..."
GOPROXY=https://proxy.golang.org,direct go mod download -json github.com/graph-gophers/graphql-go@v1.7.2 | jq -r '.Error // "SUCCESS"'

echo "Testing with both configurations in sequence..."
rm -f go.sum

# Test 1: Direct first
echo "Step 1: Download with direct proxy..."
GOPROXY=direct go mod download github.com/graph-gophers/graphql-go@v1.7.2
CHECKSUM1=$(grep "github.com/graph-gophers/graphql-go v1.7.2" go.sum)

echo "Step 2: Download with standard proxy (should use existing checksum)..."
GOPROXY=https://proxy.golang.org,direct go mod download github.com/graph-gophers/graphql-go@v1.7.2
CHECKSUM2=$(grep "github.com/graph-gophers/graphql-go v1.7.2" go.sum)

if [ "$CHECKSUM1" = "$CHECKSUM2" ]; then
    echo "✓ SUCCESS: Checksums are consistent between proxy configurations"
    echo "Checksum: $CHECKSUM1"
else
    echo "✗ FAILURE: Checksums differ between proxy configurations"
    echo "Direct:   $CHECKSUM1"
    echo "Proxy:    $CHECKSUM2"
    exit 1
fi

echo "✓ All checksum tests passed for v1.7.2!"

# Cleanup
cd /
rm -rf "$TEST_DIR"