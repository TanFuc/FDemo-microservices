#!/usr/bin/env bash
# cicd/scripts/health-check.sh
set -euo pipefail

HOST="${1:?'Host required'}"
PORT="${2:-3000}"
PATH_ENDPOINT="${3:-/health}"
MAX_RETRIES="${4:-12}"
INTERVAL=5

echo "Health check: http://${HOST}:${PORT}${PATH_ENDPOINT}"
for i in $(seq 1 "$MAX_RETRIES"); do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
        "http://${HOST}:${PORT}${PATH_ENDPOINT}" 2>/dev/null || echo "000")
    [ "$STATUS" = "200" ] && { echo "Healthy (attempt ${i})"; exit 0; }
    echo "Attempt ${i}/${MAX_RETRIES} — HTTP ${STATUS} — retrying in ${INTERVAL}s..."
    sleep "$INTERVAL"
done
echo "Health check failed after ${MAX_RETRIES} attempts"
exit 1
