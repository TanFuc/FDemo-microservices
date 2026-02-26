#!/usr/bin/env bash
# cicd/scripts/detect-changes.sh
# Detects which service directories have changed in the last commit.
# Output: space-separated list of changed service names
# Usage: changed=$(./detect-changes.sh)

set -euo pipefail

SERVICES="api-gateway auth profile catalog cart order inventory payment logistic notification media search campaign review analytic"

# Get list of changed files from last commit
CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD 2>/dev/null || git diff --name-only HEAD 2>/dev/null || echo "")

if [ -z "$CHANGED_FILES" ]; then
    echo "No changed files detected" >&2
    exit 0
fi

echo "Changed files:" >&2
echo "$CHANGED_FILES" >&2

# Check if pkg/ changed — affects all services
if echo "$CHANGED_FILES" | grep -q '^pkg/'; then
    echo "pkg_changed=true"
    echo "$SERVICES"  # Return all services
    exit 0
fi

# Check which specific services changed
CHANGED_SERVICES=""
for svc in $SERVICES; do
    if echo "$CHANGED_FILES" | grep -q "^${svc}/"; then
        CHANGED_SERVICES="$CHANGED_SERVICES $svc"
    fi
done

echo "${CHANGED_SERVICES# }"  # trim leading space
