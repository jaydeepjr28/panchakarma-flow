#!/usr/bin/env bash
# Runs all 7 PanchakarmaFlow Go backend microservices with a single command.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -d "$SCRIPT_DIR/backend" ]; then
    ROOT_DIR="$SCRIPT_DIR"
    BACKEND_DIR="$SCRIPT_DIR/backend"
else
    ROOT_DIR="$(dirname "$SCRIPT_DIR")"
    BACKEND_DIR="$SCRIPT_DIR"
fi

BIN_DIR="$BACKEND_DIR/bin"
LOGS_DIR="$BACKEND_DIR/logs"
PID_FILE="$BACKEND_DIR/.backend-pids"

mkdir -p "$BIN_DIR" "$LOGS_DIR"

echo "=========================================================================="
echo "   PANCHAKARMA FLOW — ALL BACKEND SERVICES ORCHESTRATOR"
echo "=========================================================================="

# Check infrastructure
if command -v docker &> /dev/null; then
    echo "[1/3] Ensuring Docker infrastructure is running..."
    docker compose -f "$ROOT_DIR/docker-compose.yml" up -d 2>/dev/null || true
fi

# Services
SERVICES=(
    "identity-service:8081:9081:./cmd/identity-service"
    "clinic-service:8082:9082:./cmd/clinic-service"
    "appointment-service:8083:9083:./cmd/appointment-service"
    "therapy-service:8084:9084:./cmd/therapy-service"
    "notification-service:8085:9085:./cmd/notification-service"
    "feedback-analytics-service:8086:9086:./cmd/feedback-analytics-service"
    "api-gateway:8080:0:./cmd/api-gateway"
)

# Build
echo "[2/3] Compiling microservice binaries..."
cd "$BACKEND_DIR"
for item in "${SERVICES[@]}"; do
    IFS=":" read -r name http grpc cmd <<< "$item"
    if [ ! -f "$BIN_DIR/$name" ]; then
        echo "      Building $name..."
        go build -o "$BIN_DIR/$name" "$cmd"
    fi
done

# Cleanup on exit
PIDS=()
cleanup() {
    echo ""
    echo "==> Stopping all backend services..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    rm -f "$PID_FILE"
    echo "==> All backend services stopped."
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

# Launch
echo "[3/3] Launching all 7 services..."
> "$PID_FILE"
for item in "${SERVICES[@]}"; do
    IFS=":" read -r name http grpc cmd <<< "$item"
    log_file="$LOGS_DIR/$name.log"
    "$BIN_DIR/$name" > "$log_file" 2>&1 &
    pid=$!
    PIDS+=("$pid")
    echo "$pid" >> "$PID_FILE"
    echo "      [STARTED] $name -> HTTP :$http (PID: $pid)"
    sleep 0.3
done

echo ""
echo "=========================================================================="
echo "   ALL 7 BACKEND SERVICES ARE LIVE AND OPERATIONAL!"
echo "   API Gateway Entry Point : http://localhost:8080/api/v1"
echo "   Press Ctrl+C to stop all services."
echo "=========================================================================="

wait
