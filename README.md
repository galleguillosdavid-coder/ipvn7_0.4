# ipvn7 Network OS — Sovereign Overlay Mesh (v0.4.0)

```text
██╗██████╗ ██╗   ██╗███╗   ██╗███████╗
██║██╔══██╗██║   ██║████╗  ██║╚════██║
██║██████╔╝██║   ██║██╔██╗ ██║    ██╔╝
██║██╔═══╝ ╚██╗ ██╔╝██║╚██╗██║   ██╔╝ 
██║██║      ╚████╔╝ ██║ ╚████║   ██║  
╚═╝╚═╝       ╚═══╝  ╚═╝  ╚═══╝   ╚═╝  
   Network OS - Autonomous Sovereign Mesh v0.4
```

**ipvn7** es un Sistema Operativo de Red Autónomo, Descentralizado, Resistente a Amenazas Post-Cuánticas y Programable (Network Operating System - NOS) que opera como una malla superpuesta (*overlay mesh*) de escala planetaria.

A diferencia del paradigma tradicional TCP/IP donde la identidad y la localización física están fusionadas en la dirección IP, en **ipvn7** los dispositivos son exclusivamente un **Identificador Descentralizado Soberano (DID)** respaldado por criptografía asimétrica Ed25519. La Internet física subyacente es tratada como un sustrato de transporte bruto, hostil y puramente efímero.

---

## 🏛️ Las 20 Dimensiones de Innovación Certificadas

Todas las dimensiones han sido completamente implementadas en Go y empíricamente verificadas en laboratorio físico sobre dos hosts reales (PC Desktop WSL2 $\leftrightarrow$ Notebook física `192.168.1.106`):

1. **Cortafuegos ZTNA Default-Deny (`pkg/l1/firewall.go`):** Micro-segmentación nativa por DID soberano sin confianza implícita.
2. **dDNS Petnames Contextuales (`pkg/l4/ddns_petnames.go`):** Sistema de nombres local mnemotécnico (`notebook.ipv7`) con soporte para ámbitos relativos y sincronización con `/etc/hosts`.
3. **Calidad de Servicio (QoS) & PoW Anti-DDoS (`pkg/l1/qos.go`):** Token Bucket de 3 clases y desafíos criptográficos SHA-256 de 16 bits emitidos en caliente ante saturación.
4. **Almacén DAG & DTN Store-and-Forward (`pkg/l1/dag_store.go`):** Bloques inmutables direccionados por contenido (`cid:ipvn7:<sha256>`), firmas Ed25519 y colas asíncronas tolerantes a desconexiones.
5. **Economía de Tránsito Tit-for-Tat (`pkg/l2/accounting.go`):** Algoritmo de reciprocidad estricta en 4 tiers (`PRIORITY`, `NORMAL`, `BEST_EFFORT`, `THROTTLED`) con 1 MB de cortesía inicial.
6. **Plano de Control Dual Humano / IA (`cmd/ipvn7-cli/`, `pkg/l3/mcp_server.go`):** Consola CLI interactiva multiplataforma y servidor JSON-RPC 2.0 (Model Context Protocol) para agentes de IA con 9 herramientas nativas.
7. **Smart Packets con Timeout Estricto & DLP (`pkg/l1/smart_packets.go`):** Ganchos de inspección en vuelo con salvaguarda de 10 ms por filtro y descarte inmediato de datos que contengan secretos o tokens PII.
8. **Multipath Overlay Scheduler (`pkg/l1/multipath.go`):** Enrutamiento coordinado sobre enlaces heterogéneos (WLAN + Ethernet + 5G) con modos `LowestLatency`, `Balanced` y `CriticalDuplication` (cero pérdida).
9. **Web-of-Trust (WoT) & Reputación Social (`pkg/l1/wot.go`):** Avales firmados entre pares y cálculo de reputación atenuada mediante BFS (-15% por grado social, max 3 saltos).
10. **Autenticación Fuera-de-Banda (OOB SAS + UR QR) (`pkg/l4/oob_pairing.go`):** Derivación canónica de código SAS (6 dígitos numéricos y 4 emojis: `🎷 🎨 🔮 🎁`) con tramas UR para códigos QR animados.
11. **IA Agéntica Nativa & Auto-Curación (`pkg/l3/ai_copilot.go`):** Copiloto autónomo integrado con DeepSeek micro-worker para deducción de causas raíz y auto-reparación dinámica de la malla.
12. **Arquitectura Zero-Copy Buffer Pool (`pkg/l1/buffer_pool.go`):** Preasignación en 3 niveles (64B / 1500B / 64KB) con conteo de referencias atómico y reciclaje `sync.Pool`.
13. **Señalización Ciega Efímera Zero-Knowledge (EBRA) (`pkg/l1/blind_rendezvous.go`):** Bootstrap de arranque en frío sin fuga de metadatos mediante tópicos deterministas por época horaria, auto-destrucción (*Consume-and-Burn*) y Circuit Breaker atómico P2P puro.
14. **Resiliencia Post-Apagón & Jitter Descorrelacionado (`pkg/l1/blackout_recovery.go`):** Cascada estocástica de 5 fases (KùzuDB $\to$ Offgrid BLE/LoRa $\to$ WAN STUN $\to$ EBRA $\to$ Malla P2P) con temporizador $T_{i+1} = \min(T_{\text{max}}, \, \text{Uniforme}(T_{\text{base}}, \, T_{i} \times 3))$ inmune a estampidas (*Thundering Herd*).
15. **VPN Corporativa Fricción Cero para Multinacionales (`pkg/l1/corporate_vpn.go`):** Despliegue Zero-Admin sin permisos root (conmutación automática a proxy local SOCKS5 `:10807` + HTTP CONNECT `:10808`), camuflaje Anti-DPI RFC 8446 (TLS 1.3 / :443) sobre datagramas 1280B y pasarelas de salida multijurisdicción (Frankfurt, Zúrich, Tokio, NY, Singapur).
16. **Ecosistema de Aplicaciones Soberanas (`pkg/l4/`, `pkg/l1/`):** Chat Soberano E2EE con secreto perfecto hacia adelante (`chat_manager.go`), Escritorio Remoto P2P estilo RustDesk (`remote_desktop.go`), Nube Personal DAG Store y Transmisión Multicast en Cascada $O(\log N)$ (`cascade_multicast.go`).
17. **Arquitectura de Interfaz Adaptativa (Niveles 1 al 7) (`web/`):** Espectro de complejidad que transita fluidamente desde el Nivel 1 (Modo Consumidor: física orbital en canvas 60 FPS y apps de un solo clic) hasta el Nivel 7 (Modo Ingeniero Soberano: consola openCypher KùzuDB, bypass eBPF/XDP de kernel, 12 anillos Kleinberg y modo Override).
18. **Streaming Real en Vivo P2P (WHIP/WHEP) (`pkg/l4/live_stream.go`, `web/js/live-stream.js`):** Transmisión de pantalla o cámara web con captura en navegador y difusión SSE de baja latencia.
19. **Música y Radio Soberana Hi-Fi sin Video (`pkg/l4/audio_radio.go`, `web/js/audio-player.js`):** Transmisión de audio ligero (Opus / Web Audio API) a 96 kbps, Mini-Player Dock flotante y visualizador espectral a 60 FPS.
20. **Sonda Centinela de Red & Detección Anti-Censura (`pkg/l1/network_probe.go`, `web/js/network-probe.js`):** Auditoría voluntaria de latencia y detección proactiva de manipulación perimetral (DNS Poisoning, inyección TCP RST).

---

## ⚡ Principios Inviolables

* **Strict Core Freeze en L0 (`pkg/l0/`):** El núcleo criptográfico, el formato determinista CBOR y el handshake Noise XX permanecen inmutables byte por byte.
* **Axioma de Zero-PII:** Prohibición estricta de registrar o persistir direcciones IP reales, geolocalizaciones o metadatos privados en disco.
* **Flujo Sostenible & Packet Pacing:** Subordinación estricta de la inyección de tráfico a la capacidad útil del cuello de botella (9.44 Mbps / 1085 µs anti-bufferbloat).
* **Zero-Admin & Anti-DPI:** Inmunidad contra bloqueos corporativos mediante proxy userspace local y tramas indistinguibles de HTTPS estándar.

---

## 🚀 Despliegue y Ejecución

### 1. Compilación Cruzada Dual (Linux + Windows)
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_dual.ps1
```

### 2. Puesta en Marcha del Nodo Maestro (PC WSL2)
```bash
./bin/linux_amd64/ipvn7 --port 7777 --web-port 7070
```
Dashboard interactivo disponible en: **[http://localhost:7070](http://localhost:7070)**

### 3. Puesta en Marcha del Nodo Remoto (Notebook Física 192.168.1.106)
```cmd
bin\windows_amd64\ipvn7.exe -port 7001 -web-port 8080
```
Dashboard interactivo disponible en: **[http://192.168.1.106:8080](http://192.168.1.106:8080)**

### 4. Operaciones con el CLI Soberano
```bash
# Estado general del nodo
ipvn7-cli status

# Gestión de Cortafuegos ZTNA
ipvn7-cli firewall list
ipvn7-cli firewall test did:ipvn7:...

# Almacén Inmutable DAG
ipvn7-cli dag put "Contenido inmutable firmado"

# Diagnóstico de Copiloto de IA
ipvn7-cli copilot diagnose

# Servidor MCP para Agentes de IA
ipvn7-cli mcp
```

---

## 🧪 Verificación Automatizada

Para reproducir la suite empírica completa contra la red física viva:

```bash
# Ejecutar verificación de la Fase D (Plano Dual + Copiloto IA + Tarea Distribuida)
python scripts/verify_phase_d.py

# Ejecutar verificación de la suite UIN, Memory Arbiter y Anti-Replay
python scripts/verify_uin_stack.py http://localhost:7070

# Ejecutar suite de pruebas unitarias completa en Go
go test -v ./...

# Ejecutar pruebas específicas de Resiliencia, Señalización Ciega (EBRA) y VPN Corporativa
go test ./pkg/l1 -v -run "TestBlindBeacon|TestRecovery|TestDecorrelated|TestCorporateVPN"

# Ejecutar suite de pruebas del SDK de Python
python sdk/python/tests/test_sdk.py
```

---

## ⚡ Instalación en 1 Línea (Zero-Friction)

### En Linux / macOS
```bash
curl -fsSL https://raw.githubusercontent.com/galleguillosdavid-coder/ipvn7_0.4/main/install.sh | bash
```

### En Windows (PowerShell Administrador)
```powershell
iwr -useb https://raw.githubusercontent.com/galleguillosdavid-coder/ipvn7_0.4/main/install.ps1 | iex
```

---

## 🤖 Python SDK para Agentes de IA (`sdk/python`)

Conecta frameworks agénticos (LangChain, CrewAI, AutoGPT) a la malla soberana:

```python
from ipvn7 import SovereignNode, BindingScope

node = SovereignNode("http://localhost:7070")
passport = node.get_passport()
print(f"UIN Root ID: {passport.root_id_hex}")

# Emitir credencial delegada de agente IA con firma raíz L0
binding = node.issue_binding(scope=BindingScope.AI_AGENT, duration_days=30)
print(f"Binding Key ID: {binding.key_id_hex}")
```

---

## 📚 Documentación Adicional
* ⚡ **[MAGNA SKILL 7.0: Ecosistema AFE-Kùzu (Ingeniería Axiomática)](docs/MAGNA_SKILL_7_AFE_KUZU.md)**
* 📄 **[Whitepaper Técnico y Comparativa de Mercado (ipvn7 vs Tailscale/Tor/WireGuard)](docs/WHITEPAPER_COMPETITIVO_IPVN7.md)**
* 🧭 [Plan Maestro y Checklist Integral (SSOT)](docs/PLAN_MAESTRO_NOS_IPVN7.md)
* 📜 [Tratado Ontológico y Fundacional (genesis.md)](genesis.md)
* 📖 [Portal de Documentación](docs/README.md)

