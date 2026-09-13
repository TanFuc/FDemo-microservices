#!/usr/bin/env bash
# ==============================================================================
# NexusCommerce Protobuf Compilation Pipeline
# Compiles all .proto files across all microservices into Go code stubs.
# ==============================================================================
set -e

echo "============================================================"
echo " NexusCommerce - Automated Protobuf Compiler Pipeline"
echo "============================================================"

# 1. Verify protoc compiler is installed
if ! command -v protoc &> /dev/null; then
    echo "[!] protoc binary not found. Attempting automated installation..."
    if command -v curl &> /dev/null && command -v python3 &> /dev/null; then
        curl -fsSL -o /tmp/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v25.3/protoc-25.3-linux-x86_64.zip
        python3 -c "import zipfile; zipfile.ZipFile('/tmp/protoc.zip').extractall('/usr/local')"
        chmod 755 /usr/local/bin/protoc || true
    else
        echo "[ERROR] protoc is required but could not be automatically installed."
        exit 1
    fi
fi

echo "[+] protoc version: $(protoc --version)"

# 2. Verify protoc-gen-go & protoc-gen-go-grpc plugins
export PATH="$PATH:/usr/local/bin:$HOME/go/bin"

if ! command -v protoc-gen-go &> /dev/null || ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "[!] Installing protoc-gen-go and protoc-gen-go-grpc..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "------------------------------------------------------------"
echo " Compiling Protocol Buffers across services..."
echo "------------------------------------------------------------"

# 1. auth service
echo -n "[*] Compiling auth/api/proto/v1/auth.proto... "
(cd "$ROOT_DIR/auth" && protoc -I . --go_out=. --go_opt=module=microservices/auth --go-grpc_out=. --go-grpc_opt=module=microservices/auth api/proto/v1/auth.proto)
echo "[DONE]"

# 2. campaign service
echo -n "[*] Compiling campaign/api/proto/v1/campaign.proto... "
(cd "$ROOT_DIR/campaign" && protoc -I . --go_out=. --go_opt=module=microservices/campaign --go-grpc_out=. --go-grpc_opt=module=microservices/campaign api/proto/v1/campaign.proto)
echo "[DONE]"

# 3. cart service
echo -n "[*] Compiling cart/api/proto/v1/cart.proto... "
(cd "$ROOT_DIR/cart" && protoc -I . --go_out=. --go_opt=module=microservices/cart --go-grpc_out=. --go-grpc_opt=module=microservices/cart api/proto/v1/cart.proto)
echo "[DONE]"

# 4. catalog service
echo -n "[*] Compiling catalog/api/proto/catalog.proto... "
(cd "$ROOT_DIR/catalog" && protoc -I . --go_out=. --go_opt=module=microservices/catalog --go-grpc_out=. --go-grpc_opt=module=microservices/catalog api/proto/catalog.proto)
echo "[DONE]"

# 5. inventory service
echo -n "[*] Compiling inventory/api/proto/inventory.proto... "
(cd "$ROOT_DIR/inventory" && protoc -I . --go_out=. --go_opt=module=microservices/inventory --go-grpc_out=. --go-grpc_opt=module=microservices/inventory api/proto/inventory.proto)
echo "[DONE]"

# 6. order service (inventory client stubs)
echo -n "[*] Compiling order/proto/inventory/inventory.proto... "
(cd "$ROOT_DIR/order" && protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/inventory/inventory.proto)
echo "[DONE]"

# 7. review service (order client stubs)
echo -n "[*] Compiling review/proto/order/order.proto... "
(cd "$ROOT_DIR/review" && protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/order/order.proto)
echo "[DONE]"

echo "============================================================"
echo " All 7 Protobuf definitions compiled successfully!"
echo "============================================================"
