#!/usr/bin/env bash
# cicd/scripts/build.sh
# Usage: ./cicd/scripts/build.sh <service-dir> <full-image-tag>
# Example: ./cicd/scripts/build.sh auth harbor.tafu.internal/tafu/auth-service:a1b2c3d

set -euo pipefail

SERVICE_DIR="${1:?'ERROR: service directory required'}"
FULL_IMAGE_TAG="${2:?'ERROR: full image tag required (registry/project/image:tag)'}"
GIT_COMMIT="${3:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Build Service : ${SERVICE_DIR}"
echo "  Image Tag     : ${FULL_IMAGE_TAG}"
echo "  Git Commit    : ${GIT_COMMIT}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

docker build \
    --file "${SERVICE_DIR}/Dockerfile" \
    --tag  "${FULL_IMAGE_TAG}" \
    --build-arg "GO_VERSION=1.22" \
    --build-arg "BUILD_DATE=${BUILD_DATE}" \
    --build-arg "GIT_COMMIT=${GIT_COMMIT}" \
    --label "org.opencontainers.image.revision=${GIT_COMMIT}" \
    --label "org.opencontainers.image.created=${BUILD_DATE}" \
    "${SERVICE_DIR}"

echo "Build complete: ${FULL_IMAGE_TAG}"
