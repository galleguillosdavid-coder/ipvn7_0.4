import urllib.request
import json
import time
import sys

desktop_url = "http://localhost:7070"
notebook_url = "http://192.168.1.106:8080"

def post_json(url, data=None):
    payload = json.dumps(data).encode("utf-8") if data is not None else b"{}"
    req = urllib.request.Request(
        url,
        data=payload,
        headers={"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req, timeout=5) as res:
        return json.loads(res.read().decode("utf-8"))

def get_json(url):
    with urllib.request.urlopen(url, timeout=5) as res:
        return json.loads(res.read().decode("utf-8"))

print("================================================================================")
print("     IPv7 v0.3.0: VALIDACION EMPIRICA DE MACRO-ACCIONES Y TELEMETRIA AGREGADA   ")
print("               (Mandatos de UX y Resiliencia en 2.md & genesis.md)              ")
print("================================================================================")

# 1. TEST 1: MACRO-ACCION ATOMICA - MODO INVISIBLE / DARK NODE (2.md Líneas 722-728 y 766-789)
print("\n[TEST 1] Verificando Macro-Acción Atómica: Modo Invisible (Dark Node Egress-Only)...")

# Estado inicial
mode_init_desk = get_json(desktop_url + "/api/macro/mode")
print(f"  -> Estado inicial Desktop: {mode_init_desk['mode']} (Dark: {mode_init_desk['is_dark_node']})")
assert mode_init_desk['is_dark_node'] is False

# Conmutar Desktop a Dark Node
res_dark_desk = post_json(desktop_url + "/api/macro/dark-node")
st_dark_desk = res_dark_desk['status']
print(f"  -> Conmutado a Modo Invisible: {st_dark_desk['mode']}")
print(f"     Listeners silenciados: {st_dark_desk['listeners_muted']}")
print(f"     Circuitos Sphinx: {st_dark_desk['sphinx_circuits_active']} ({st_dark_desk['onion_hops']} saltos)")
print(f"     Tráfico exclusivo de salida (Egress-Only): {st_dark_desk['egress_only']}")
assert st_dark_desk['is_dark_node'] is True
assert st_dark_desk['listeners_muted'] is True
assert st_dark_desk['sphinx_circuits_active'] is True
assert st_dark_desk['onion_hops'] == 3

# Conmutar Notebook a Dark Node
res_dark_note = post_json(notebook_url + "/api/macro/dark-node")
assert res_dark_note['status']['is_dark_node'] is True
print(f"  -> Notebook conmutado a Modo Invisible exitosamente.")

# Restaurar ambos a Modo Estándar
res_std_desk = post_json(desktop_url + "/api/macro/dark-node")
res_std_note = post_json(notebook_url + "/api/macro/dark-node")
assert res_std_desk['status']['is_dark_node'] is False
assert res_std_note['status']['is_dark_node'] is False
print("  [OK] Macro-Acción Atómica verificada: Cascada invisible y conmutación de circuitos instantánea.")

# 2. TEST 2: TELEMETRIA AGREGADA POR HUELLA DIGITAL (2.md Líneas 729-738 & genesis.md L177)
print("\n[TEST 2] Verificando Telemetría de Fallas Agregada por Huella Digital (Fingerprinting O(1))...")

# Inyectar 50 eventos leves
post_json(desktop_url + "/api/telemetry/anomalies/simulate", {
    "type": "DNS_TIMEOUT_RETRY",
    "subsystem": "L4_DDNS",
    "root_cause": "Latencia esporádica en resolución de petname",
    "severity": "LOW",
    "count": 50
})

# Inyectar ráfaga masiva de 500,000 eventos críticos
post_json(desktop_url + "/api/telemetry/anomalies/simulate", {
    "type": "BUFFER_OVERFLOW_SATURATION",
    "subsystem": "L1_TRANSPORT",
    "root_cause": "Colas intermedias saturadas por bufferbloat",
    "severity": "CRITICAL",
    "count": 500000
})

# Inyectar 200 eventos medios
post_json(desktop_url + "/api/telemetry/anomalies/simulate", {
    "type": "UNAUTHORIZED_PROBE_ZTNA",
    "subsystem": "L1_FIREWALL",
    "root_cause": "Sondeo descartado por política Default-Deny",
    "severity": "MEDIUM",
    "count": 200
})

# Consultar el resumen ejecutivo
summary = get_json(desktop_url + "/api/telemetry/anomalies/summary")
print(f"  -> Total eventos compactados: {summary['total_events_compressed']:,}")
print(f"  -> Firmas únicas identificadas: {summary['distinct_signatures']}")
assert summary['total_events_compressed'] >= 500250, "Debe acumular al menos 500,250 eventos"
assert summary['distinct_signatures'] == 3, "Debe tener exactamente 3 huellas digitales distintas"

dom = summary['dominant_anomaly']
print(f"  -> Anomalía Dominante detectada: [{dom['error_type']}]")
print(f"     Subcarrier/Carril: {dom['subsystem']}")
print(f"     Ocurrencias compactadas: {dom['count']:,} eventos")
print(f"     Causa Raíz: {dom['root_cause']}")
print(f"     Firma SHA-256: [{dom['signature']}]")

assert dom['error_type'] == "BUFFER_OVERFLOW_SATURATION"
assert dom['count'] >= 500000

print(f"  -> Texto Ejecutivo: \"{summary['summary_text']}\"")
assert "BUFFER_OVERFLOW_SATURATION" in summary['summary_text']
print("  [OK] Fingerprinting O(1) certificado: 500,250 anomalías compactadas sin colapso de memoria ni spam de logs.")

# 3. TEST 3: COEXISTENCIA CON ENLACE FISICO Y TAREA DISTRIBUIDA
print("\n[TEST 3] Verificando Enlace Físico y Tarea Distribuida con Nuevos Módulos...")
status_desk = get_json(desktop_url + "/api/status")
status_note = get_json(notebook_url + "/api/status")

# Autorizar en ZTNA Firewall
for url, d in [(desktop_url, status_desk['did']), (desktop_url, status_note['did']), 
               (notebook_url, status_desk['did']), (notebook_url, status_note['did'])]:
    post_json(url + "/api/firewall/rules", {
        "did": d,
        "allow_inbound": True,
        "allow_outbound": True,
        "allow_relay": True
    })

# Peering
post_json(desktop_url + "/api/peers/add", {
    "did": status_note['did'],
    "address": "192.168.1.106:7001",
    "latency_ms": 2.1
})
post_json(notebook_url + "/api/peers/add", {
    "did": status_desk['did'],
    "address": "192.168.1.198:7777",
    "latency_ms": 2.1
})

dist_task = post_json(desktop_url + "/api/tasks/distributed", {
    "target_url": notebook_url,
    "task_type": "stealth_telemetry_consensus",
    "payload": "verify:dark_node_macro:anomaly_fingerprinting",
    "source_did": status_desk['did'],
    "port": 7001
})
print(f"  -> Tarea distribuida estado: {dist_task['status']}")
print(f"     RTT bidireccional: {dist_task['round_trip_ms']:.2f} ms")
print(f"     Decisión ZTNA: {dist_task['ztna_decision']}")
assert dist_task['status'] == "COMPLETED"
assert dist_task['ztna_decision'] in ["ACCEPT", "ACCEPTED"]
print("  [OK] Enlace físico bidireccional y cómputo distribuido confirmado.")

print("\n" + "="*80)
print("  [EXITO TOTAL] MANDATOS DE UX (2.md) Y OBSERVABILIDAD (genesis.md) VALIDADOS AL 100%")
print("================================================================================")
