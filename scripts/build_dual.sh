#!/usr/bin/env bash
# build_dual.sh - Compilación cruzada dual simultánea desde Linux/WSL2
# Ecosistema AFE-Kùzu (SKILL 7.0)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$ROOT_DIR/bin"
WIN_DIR="$BIN_DIR/windows_amd64"
LIN_DIR="$BIN_DIR/linux_amd64"

mkdir -p "$WIN_DIR" "$LIN_DIR"

echo "=== INICIANDO COMPILACIÓN CRUZADA DUAL (ipvn7) ==="

# 1. Compilación Windows (amd64)
echo "[1/2] Compilando binarios para Windows (.exe)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$WIN_DIR/ipvn7.exe" "$ROOT_DIR/cmd/ipvn7"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$WIN_DIR/ipvn7-cli.exe" "$ROOT_DIR/cmd/ipvn7-cli"
cp -f "$WIN_DIR/ipvn7.exe" "$BIN_DIR/ipvn7.exe"
cp -f "$WIN_DIR/ipvn7-cli.exe" "$BIN_DIR/ipvn7-cli.exe"
echo "  -> Generados: bin/windows_amd64/ipvn7.exe y bin/ipvn7.exe"

# 2. Compilación Linux (amd64)
echo "[2/2] Compilando binarios para Linux ELF..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$LIN_DIR/ipvn7" "$ROOT_DIR/cmd/ipvn7"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$LIN_DIR/ipvn7-cli" "$ROOT_DIR/cmd/ipvn7-cli"
cp -f "$LIN_DIR/ipvn7" "$BIN_DIR/ipvn7"
cp -f "$LIN_DIR/ipvn7-cli" "$BIN_DIR/ipvn7-cli"
chmod +x "$LIN_DIR/ipvn7" "$LIN_DIR/ipvn7-cli" "$BIN_DIR/ipvn7" "$BIN_DIR/ipvn7-cli"
echo "  -> Generados: bin/linux_amd64/ipvn7 y bin/ipvn7"

echo ""
echo "[+] Compilación cruzada dual completada exitosamente!"
ls -lh "$BIN_DIR"
