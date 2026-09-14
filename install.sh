#!/usr/bin/env bash
# ==============================================================================
# ipvn7 Network OS — Instalador Universal Soberano de 1 Clic (Linux & macOS)
# Repositorio: https://github.com/galleguillosdavid-coder/ipvn7_0.4
# ==============================================================================

set -e

COLOR_CYAN='\033[0;36m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[1;33m'
COLOR_RED='\033[0;31m'
COLOR_RESET='\033[0m'

echo -e "${COLOR_CYAN}"
cat << "EOF"
  _                   _____ 
 (_)_ ____   ___ __  |___  |
 | | '_ \ \ / / '_ \    / / 
 | | |_) \ V /| | | |  / /  
 |_| .__/ \_/ |_| |_| /_/   
   |_| Network OS Sovereign Mesh
EOF
echo -e "${COLOR_RESET}"

INSTALL_DIR="/opt/ipvn7"
BIN_DIR="/usr/local/bin"
REPO="galleguillosdavid-coder/ipvn7_0.4"

# 1. Detección de Arquitectura y Sistema Operativo
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo -e "${COLOR_RED}[-] Arquitectura no soportada: $ARCH${COLOR_RESET}"
    exit 1
    ;;
esac

if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
  echo -e "${COLOR_RED}[-] Sistema operativo no soportado por este script: $OS${COLOR_RESET}"
  exit 1
fi

echo -e "${COLOR_YELLOW}[1/4] Detectado: $OS ($ARCH)${COLOR_RESET}"

# 2. Preparar directorios de instalación
echo -e "${COLOR_YELLOW}[2/4] Configurando entorno en $INSTALL_DIR...${COLOR_RESET}"
sudo mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/keystore" "$INSTALL_DIR/logs"

# Si existe código local compilado en el repo clonado, copiarlo; de lo contrario descargar release
if [ -f "./bin/linux_amd64/ipvn7" ] && [ "$OS" = "linux" ] && [ "$ARCH" = "amd64" ]; then
  echo -e "${COLOR_GREEN}[+] Instalando desde compilación local verificada...${COLOR_RESET}"
  sudo cp ./bin/linux_amd64/ipvn7 "$INSTALL_DIR/bin/ipvn7"
  sudo cp ./bin/linux_amd64/ipvn7-cli "$INSTALL_DIR/bin/ipvn7-cli"
  sudo cp -r ./web "$INSTALL_DIR/web"
else
  RELEASE_URL="https://github.com/$REPO/releases/latest/download/ipvn7-v0.4.0-${OS}-${ARCH}.tar.gz"
  echo -e "${COLOR_YELLOW}[+] Descargando release soberano desde GitHub...${COLOR_RESET}"
  TMP_DIR="$(mktemp -d)"
  if curl -sSL -f "$RELEASE_URL" -o "$TMP_DIR/ipvn7.tar.gz" 2>/dev/null; then
    tar -xzf "$TMP_DIR/ipvn7.tar.gz" -C "$INSTALL_DIR/"
  else
    echo -e "${COLOR_YELLOW}[!] Release binario aún no publicado en GitHub. Construyendo localmente con Go...${COLOR_RESET}"
    go build -o "$INSTALL_DIR/bin/ipvn7" ./cmd/ipvn7
    go build -o "$INSTALL_DIR/bin/ipvn7-cli" ./cmd/ipvn7-cli
    sudo cp -r ./web "$INSTALL_DIR/web"
  fi
  rm -rf "$TMP_DIR"
fi

sudo chmod +x "$INSTALL_DIR/bin/ipvn7" "$INSTALL_DIR/bin/ipvn7-cli"
sudo ln -sf "$INSTALL_DIR/bin/ipvn7" "$BIN_DIR/ipvn7"
sudo ln -sf "$INSTALL_DIR/bin/ipvn7-cli" "$BIN_DIR/ipvn7-cli"

# 3. Configuración de Servicio systemd (en Linux)
if [ "$OS" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
  echo -e "${COLOR_YELLOW}[3/4] Creando servicio systemd persistente (/etc/systemd/system/ipvn7.service)...${COLOR_RESET}"
  cat << EOF | sudo tee /etc/systemd/system/ipvn7.service > /dev/null
[Unit]
Description=ipvn7 Network OS — Sovereign Overlay Mesh Node
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/bin/ipvn7 --port 7777 --web-port 7070 --keystore $INSTALL_DIR/keystore/node_identity.key
Restart=always
RestartSec=5s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

  sudo systemctl daemon-reload
  sudo systemctl enable --now ipvn7
  echo -e "${COLOR_GREEN}[+] Servicio ipvn7 iniciado y habilitado en el arranque del sistema.${COLOR_RESET}"
else
  echo -e "${COLOR_YELLOW}[3/4] macOS / Entorno sin systemd: Se puede iniciar manualmente con 'ipvn7'.${COLOR_RESET}"
fi

# 4. Verificación de Estado
echo -e "${COLOR_YELLOW}[4/4] Verificando conectividad local...${COLOR_RESET}"
sleep 2
if curl -s http://localhost:7070/api/status >/dev/null 2>&1; then
  echo -e "${COLOR_GREEN}==================================================================${COLOR_RESET}"
  echo -e "${COLOR_GREEN}  ¡ipvn7 Network OS instalado y operando con éxito!${COLOR_RESET}"
  echo -e "${COLOR_CYAN}  Panel de Control Web: http://localhost:7070${COLOR_RESET}"
  echo -e "${COLOR_CYAN}  Gateway Universal SOCKS5: 127.0.0.1:10807${COLOR_RESET}"
  echo -e "${COLOR_CYAN}  Comando de diagnóstico: ipvn7-cli status${COLOR_RESET}"
  echo -e "${COLOR_GREEN}==================================================================${COLOR_RESET}"
else
  echo -e "${COLOR_YELLOW}[+] Binarios instalados en $BIN_DIR. Ejecute 'ipvn7 --port 7777 --web-port 7070' para iniciar.${COLOR_RESET}"
fi
