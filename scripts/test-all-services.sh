#!/usr/bin/env bash
set -e

echo "============================================================"
echo " Running Full Test Suite Across All NexusCommerce Modules"
echo "============================================================"

FAILED=0
PASSED=0

MODULES=(
    "pkg/authclient"
    "pkg/authorization"
    "pkg/cache"
    "pkg/customfields"
    "pkg/idempotency"
    "pkg/logger"
    "pkg/messaging"
    "pkg/response"
    "pkg/saga"
    "api-gateway"
    "auth"
    "profile"
    "catalog"
    "cart"
    "order"
    "inventory"
    "payment"
    "logistic"
    "campaign"
    "notification"
    "media"
    "review"
    "search"
    "analytic"
)

for mod in "${MODULES[@]}"; do
    if [ -f "$mod/go.mod" ]; then
        echo -n "Testing $mod... "
        if (cd "$mod" && go test ./... -short > /tmp/test.log 2>&1); then
            echo "[PASS]"
            PASSED=$((PASSED + 1))
        else
            echo "[FAIL]"
            cat /tmp/test.log
            FAILED=$((FAILED + 1))
        fi
    fi
done

echo "============================================================"
echo " Test Summary: $PASSED Passed, $FAILED Failed"
echo "============================================================"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
