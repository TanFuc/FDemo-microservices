#!/usr/bin/env bash
# cicd/scripts/deploy.sh
# Deploys a service to Docker Swarm on a remote host via SSH.
# Usage: ./deploy.sh <swarm-service> <image> <ssh-user> <ssh-host> <ssh-port> [delay]

set -euo pipefail

SWARM_SERVICE="${1:?'Swarm service name required (e.g. tafu_auth)'}"
IMAGE="${2:?'Full image path required (e.g. harbor.tafu.internal/tafu/auth-service:latest)'}"
SSH_USER="${3:?'SSH user required'}"
SSH_HOST="${4:?'SSH host required'}"
SSH_PORT="${5:-22}"
UPDATE_DELAY="${6:-15s}"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Deploying : ${SWARM_SERVICE}"
echo "  Image     : ${IMAGE}"
echo "  Target    : ${SSH_USER}@${SSH_HOST}:${SSH_PORT}"
echo "  Delay     : ${UPDATE_DELAY}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# SSH key must be available as $SSH_KEY_FILE (set by Jenkins withCredentials)
SSH_KEY_FILE="${SSH_KEY_FILE:-$HOME/.ssh/id_rsa}"

ssh -i "$SSH_KEY_FILE" \
    -o StrictHostKeyChecking=no \
    -o LogLevel=ERROR \
    -p "$SSH_PORT" \
    "$SSH_USER@$SSH_HOST" \
    'bash -s' << REMOTE
set -euo pipefail
docker pull "$IMAGE"
docker service update \
    --image "$IMAGE" \
    --update-parallelism 1 \
    --update-delay "$UPDATE_DELAY" \
    --update-failure-action rollback \
    --update-monitor 60s \
    --rollback-parallelism 1 \
    "$SWARM_SERVICE"
echo "Done: $SWARM_SERVICE"
REMOTE
