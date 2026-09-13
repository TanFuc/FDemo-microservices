#!/usr/bin/env bash
# ==============================================================================
# NexusCommerce - Multi-Microservice Background Daemon Launcher
# Starts all 16 Go microservices with their respective .env configurations.
# ==============================================================================
set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_FILE="/tmp/nexus_services.pid"
LOG_DIR="/tmp/nexus_logs"

mkdir -p "$LOG_DIR"
rm -f "$PID_FILE"
touch "$PID_FILE"

echo "============================================================"
echo " NexusCommerce - Launching All 16 Microservices"
echo "============================================================"

# Ensure binaries are compiled
bash "$ROOT_DIR/scripts/build-all-services.sh"

SERVICES=(
    "auth:3001"
    "profile:3002"
    "catalog:3003"
    "cart:3004"
    "order:3005"
    "inventory:3006"
    "payment:3007"
    "logistic:3008"
    "campaign:3009"
    "notification:3010"
    "media:3011"
    "review:3012"
    "search:3013"
    "analytic:3014"
    "realtime:3015"
    "api-gateway:8080"
)

for item in "${SERVICES[@]}"; do
    IFS=":" read -r name port <<< "$item"
    echo -n "[*] Starting $name on port $port... "
    log_file="$LOG_DIR/${name}.log"
    bin_file="/tmp/${name}-bin"
    
    (
        cd "$ROOT_DIR/$name"
        if [ -f .env ]; then
            set -a
            . ./.env
            set +a
        fi
        exec nohup "$bin_file" </dev/null > "$log_file" 2>&1
    ) &
    pid=$!
    disown "$pid" 2>/dev/null || true
    echo "$name:$pid:$port" >> "$PID_FILE"
    echo "[PID $pid, LOG $log_file]"
done

echo "------------------------------------------------------------"
echo " Waiting for services to initialize (5s)..."
echo "------------------------------------------------------------"
sleep 5

echo "Probing service health ports..."
READY=0
TOTAL=${#SERVICES[@]}

for item in "${SERVICES[@]}"; do
    IFS=":" read -r name port <<< "$item"
    if curl -s -m 2 "http://localhost:$port/health" > /dev/null 2>&1 || curl -s -m 2 "http://localhost:$port/api/v1/health" > /dev/null 2>&1 || nc -z localhost "$port" 2>/dev/null; then
        echo "  [+] $name on port $port is ONLINE"
        READY=$((READY + 1))
    else
        echo "  [-] $name on port $port is not responding (check $LOG_DIR/${name}.log)"
    fi
done

echo "============================================================"
echo " Service Launch Complete: $READY/$TOTAL ports active"
echo " PIDs recorded in $PID_FILE"
echo "============================================================"
