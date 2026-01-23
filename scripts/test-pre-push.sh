#!/bin/bash
# scripts/test-pre-push.sh
# Full test suite with coverage check for pre-push hook

set -e

echo "🧪 Running full test suite..."

# Run all tests with race detector
go test -race ./...

# Check coverage threshold
echo "📊 Checking coverage threshold..."
go test -coverprofile=coverage.out -covermode=atomic ./...

# Extract total coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

# Check if coverage meets 80% threshold
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
    echo "❌ Coverage $COVERAGE% is below 80% threshold"
    echo "💡 Tip: Run 'go test -cover ./...' to see package-level coverage"
    rm -f coverage.out
    exit 1
fi

echo "✅ Coverage $COVERAGE% meets 80% threshold"
echo "✅ All tests passed"

# Clean up coverage file
rm -f coverage.out
