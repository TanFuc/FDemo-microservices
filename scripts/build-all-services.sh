#!/usr/bin/env bash
set -e

echo "============================================================"
echo " Checking buildability of all 16 microservices"
echo "============================================================"

SERVICES=(
    "api-gateway:cmd/gateway"
    "auth:cmd/api"
    "profile:cmd/main.go"
    "catalog:cmd/main.go"
    "cart:cmd/server"
    "order:cmd/main.go"
    "inventory:cmd/main.go"
    "payment:cmd/server"
    "logistic:cmd/server"
    "campaign:cmd/main.go"
    "notification:cmd/main.go"
    "media:cmd/server"
    "review:cmd/server"
    "search:cmd/api"
    "analytic:cmd/server"
    "realtime:cmd/server"
)

FAILED=0
PASSED=0

for item in "${SERVICES[@]}"; do
    IFS=":" read -r dir target <<< "$item"
    echo -n "Building $dir ($target)... "
    if (cd "$dir" && go build -o "/tmp/${dir}-bin" "./$target" > /tmp/build.log 2>&1); then
        echo "[OK]"
        PASSED=$((PASSED + 1))
    else
        echo "[FAIL]"
        cat /tmp/build.log
        FAILED=$((FAILED + 1))
    fi
done

echo "============================================================"
echo " Build Summary: $PASSED Succeeded, $FAILED Failed"
echo "============================================================"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
