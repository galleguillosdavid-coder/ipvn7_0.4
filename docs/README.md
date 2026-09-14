# ipvn7 — Sistema Operativo de Red (NOS)
## Documentación Oficial y Arquitectura de Referencia (v0.4.0)

Bienvenido al centro de documentación técnica del proyecto **ipvn7 Network OS**, un Sistema Operativo de Red autónomo, descentralizado, resistente a amenazas post-cuánticas y nativo para agentes de Inteligencia Artificial.

---

* ⚡ **[MAGNA SKILL 7.0: Ecosistema AFE-Kùzu](MAGNA_SKILL_7_AFE_KUZU.md)**
  *Motor Soberano de Ingeniería Axiomática y Ejecución Continua: Infraestructura WSL2, Kùzu DB como única fuente de verdad unificada, pipeline PFO (Principios-Funciones-Observaciones), micro-workers DeepSeek y resolución de contradicciones lógicas.*

* 🧭 **[Plan Maestro y Checklist Integral (SSOT)](PLAN_MAESTRO_NOS_IPVN7.md)**
  *Única Fuente de Verdad técnica que certifica el Estado Base de los Horizontes 1 a 5, el Modelo de Invariantes (Core Freeze L0) y la implementación completa de las 17 Dimensiones de Innovación de Próxima Generación.*

* 📜 **[Tratado Fundacional y Paradigma Ontológico](../genesis.md)**
  *Especificación conceptual sobre la separación entre identidad soberana (DID) y localización física de red, axiomas inquebrantables, diseño de los 12 anillos de Kleinberg, señalización ciega EBRA y VPN corporativa.*

* 📄 **[Whitepaper Técnico y Comparativa de Mercado](WHITEPAPER_COMPETITIVO_IPVN7.md)**
  *Análisis exhaustivo comparando ipvn7 frente a Tailscale, Cloudflare Zero Trust, Tor y VPNs corporativas tradicionales (Cisco AnyConnect, GlobalProtect, Zscaler).*

* 🗃️ **Bitácoras Históricas de Génesis y Diseño:**
  * **[Historial de Diseño - Fase 1](HISTORIAL_DISENO_1.md)** (`docs/1.md`): Registro exhaustivo de la concepción y especificación de las virtudes originales del protocolo.
  * **[Historial de Diseño - Fase 2](HISTORIAL_DISENO_2.md)** (`docs/2.md`): Sesión de síntesis arquitectónica y nacimiento del Ecosistema AFE-Kùzu.

* 💻 **[Código Fuente y Módulos Clave](../pkg/)**
  * `pkg/l0/`: Criptografía Fortress (Noise XX, PQC Kyber, Ed25519, CBOR) **[STRICT CORE FREEZE]**.
  * `pkg/l1/`: Blind Rendezvous (EBRA), Blackout Recovery & Decorrelated Jitter, Corporate Zero-Admin VPN & TLS 1.3 Anti-DPI, Cascade Multicast, ZTNA Firewall, QoS Token Bucket, DAG Store, Buffer Pool, Multipath, Smart Packets.
  * `pkg/l2/`: Ring Buffer lock-free (<28 ns), Contabilidad Tit-for-Tat y Detección de Anomalías.
  * `pkg/l3/`: Servidor MCP JSON-RPC 2.0, Copiloto de IA con Auto-Curación y motor KùzuDB.
  * `pkg/l4/`: Chat Soberano E2EE, Escritorio Remoto P2P, Dashboard Web Server, dDNS Petnames, OOB SAS Pairing.
  * `web/`: Interfaz Adaptativa (Niveles 1 al 7), Canvas Orbital 60 FPS, Modal de VPN Corporativa, Consola openCypher y Estación de Ingeniero HUD.

* 🧪 **[Scripts de Verificación Empírica](../scripts/)**
  * `scripts/verify_phase_a.py`: Verificación de Defensa Profunda, ZTNA y Zero-Copy Pool.
  * `scripts/verify_phase_b.py`: Verificación de Persistencia DAG, Tit-for-Tat y Web-of-Trust.
  * `scripts/verify_phase_c.py`: Verificación de dDNS Petnames, Smart Packets, Multipath y OOB SAS.
  * `scripts/verify_phase_d.py`: Verificación Integral del Plano Dual MCP/CLI y Copiloto Autónomo.
  * `go test ./...`: Suite completa de pruebas unitarias y de integración en Go (100% PASS).

---

### Guía Rápida de Inicio

1. **Compilar Binarios Duales (Windows + Linux):**
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\scripts\build_dual.ps1
   ```

2. **Ejecutar Nodo Local (WSL2 Linux):**
   ```bash
   ./bin/linux_amd64/ipvn7 --port 7777 --web-port 7070
   ```
   *Acceso al Dashboard Web:* [http://localhost:7070](http://localhost:7070)

3. **Ejecutar Nodo Remoto (Windows / Notebook):**
   ```cmd
   bin\windows_amd64\ipvn7.exe -port 7001 -web-port 8080
   ```
   *Acceso al Dashboard Web:* [http://192.168.1.106:8080](http://192.168.1.106:8080)

4. **Operar mediante CLI Unificado:**
   ```bash
   ./bin/linux_amd64/ipvn7-cli status
   ./bin/linux_amd64/ipvn7-cli firewall list
   ./bin/linux_amd64/ipvn7-cli copilot diagnose
   ```
