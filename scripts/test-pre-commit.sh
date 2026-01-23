#!/bin/bash
# scripts/test-pre-commit.sh
# Fast tests on changed packages for pre-commit hook

set -e

echo "🧪 Running pre-commit tests..."

# Get list of changed Go files
CHANGED_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$CHANGED_FILES" ]; then
    echo "✅ No Go files changed, skipping tests"
    exit 0
fi

# Extract unique packages from changed files
PACKAGES=$(echo "$CHANGED_FILES" | xargs -n1 dirname | sort -u | sed 's/^/.\//' | tr '\n' ' ')

echo "📦 Testing packages: $PACKAGES"

# Run tests on changed packages only (fast, skip long-running tests)
go test -short $PACKAGES

echo "✅ Pre-commit tests passed"
