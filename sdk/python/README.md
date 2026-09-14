# ipvn7 Python SDK (v0.4.0)

El SDK oficial en Python para conectar **Agentes de Inteligencia Artificial (CrewAI, LangChain, AutoGPT)** al **Sistema Operativo de Red Soberana ipvn7**.

---

## Características Principales

- **Identidad Criptográfica UIN:** Desacople total de direcciones IP físicas mediante `root_id` inmutable de 256 bits (`did:ipvn7:uin:<hex>`).
- **Delegaciones con Alcance (Intent Scopes):** Emisión instantánea de `BindingRecord` criptográficos firmados por la raíz con ámbito restringido para agentes autónomos (`ScopeAIAgent = 3`).
- **Resistencia Post-Cuántica Nativa (NIST Level 3):** Firma dual (Ed25519 + ML-DSA-65) y encapsulación reticular (X25519 + ML-KEM-768).
- **Protección Anti-OOM DoS:** Telemetría y validación en tiempo real del Árbitro de Memoria Global (`GlobalMemoryArbiter`).
- **Cero Metadatos Expuestos:** Wire determinista de 1280 bytes y enrutamiento cebolla Sphinx de 3 saltos.

---

## Instalación

```bash
pip install ipvn7
```

O desde el código fuente:

```bash
cd sdk/python
pip install -e .
```

---

## Inicio Rápido

```python
from ipvn7 import SovereignNode, BindingScope, IntentScope

# 1. Conectar con el nodo local ipvn7
node = SovereignNode("http://localhost:7070")

if node.is_healthy():
    print("[+] Conectado al nodo soberano ipvn7")

# 2. Consultar el Pasaporte Digital UIN
passport = node.get_passport()
print(f"DID de Entidad: {passport.entity_did}")
print(f"Root ID (256-bit): {passport.root_id_hex}")
print(f"Modo Criptográfico: {passport.mode}")

# 3. Emitir una credencial delegada de Agente de IA
binding = node.issue_binding(scope=BindingScope.AI_AGENT, duration_days=30)
print(f"[+] Binding emitido para Agente IA:")
print(f"    Key ID: {binding.key_id_hex}")
print(f"    Firma Raíz: {binding.sig_root_hex[:32]}...")

# 4. Despachar una tarea con prueba de cómputo determinista PFO
result = node.send_task(
    target_url="http://192.168.1.106:8080",
    payload="SOLICITUD_INFERENCIA_AGENTE_001"
)
print(f"Resultado RTT: {result.round_trip_ms:.2f} ms | Hash PFO: {result.task_proof_hash[:16]}")
```
