#!/bin/bash
# Run all PAN integration tests
# Usage: ./run_tests.sh
# Requires: PAN server running on localhost:7337

set -e

echo "📟 PAN Integration Tests"
echo "========================"
echo ""
echo "⚠️  Make sure the server is running first:"
echo "   cd ../server && go run main.go"
echo ""

go test ./... -v -timeout 30s 2>&1

echo ""
echo "✅ All tests complete"
