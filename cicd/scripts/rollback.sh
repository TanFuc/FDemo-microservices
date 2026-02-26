#!/usr/bin/env bash
# cicd/scripts/rollback.sh
set -euo pipefail

SWARM_SERVICE="${1:?'Swarm service name required (e.g. tafu_auth)'}"
SSH_USER="${2:?'SSH user required'}"
SSH_HOST="${3:?'SSH host required'}"
SSH_PORT="${4:-22}"

echo "Rolling back: ${SWARM_SERVICE} on ${SSH_HOST}"
ssh -i "${SSH_KEY_FILE:-$HOME/.ssh/id_rsa}" \
    -o StrictHostKeyChecking=no \
    -p "$SSH_PORT" \
    "$SSH_USER@$SSH_HOST" \
    "docker service rollback '$SWARM_SERVICE' && echo 'Rollback complete'"
