#!/usr/bin/env bash
# ==============================================================================
# NexusCommerce - Multi-Microservice Background Daemon Stopper
# Stops all services recorded in /tmp/nexus_services.pid
# ==============================================================================
PID_FILE="/tmp/nexus_services.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "No PID file found at $PID_FILE. No services to stop."
    exit 0
fi

echo "============================================================"
echo " Stopping all NexusCommerce background services"
echo "============================================================"

while IFS=":" read -r name pid port; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        echo -n "Stopping $name (PID $pid)... "
        kill "$pid" 2>/dev/null || true
        # Also kill children if go run spawned a subprocess
        pkill -P "$pid" 2>/dev/null || true
        echo "[STOPPED]"
    fi
done < "$PID_FILE"

rm -f "$PID_FILE"
echo "============================================================"
echo " All services stopped cleanly."
echo "============================================================"
