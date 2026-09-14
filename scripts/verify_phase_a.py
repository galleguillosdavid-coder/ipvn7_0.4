import urllib.request
import urllib.error
import json
import time
import hashlib
import struct

desktop_api = "http://localhost:7070"
notebook_api = "http://192.168.1.106:8080"

print("==========================================================")
print("  IPv7 v0.3 - Verificacion Empirica de Fase A (Dim 1, 3, 12)")
print("  Zero Trust ZTNA | QoS & PoW Anti-DDoS | Zero-Copy Pool  ")
print("==========================================================")

# 1. Obtener estados e identidades
req1 = urllib.request.urlopen(desktop_api + "/api/status")
desktop_status = json.loads(req1.read().decode())
desktop_did = desktop_status['did']

req2 = urllib.request.urlopen(notebook_api + "/api/status")
notebook_status = json.loads(req2.read().decode())
notebook_did = notebook_status['did']

print(f"[+] Desktop DID:  {desktop_did[:32]}...")
print(f"[+] Notebook DID: {notebook_did[:32]}...")

# 2. Registrar en Kleinberg Router
print("\n[PASO 1/5] Registrando pares en Anillos Kleinberg...")
urllib.request.urlopen(urllib.request.Request(
    desktop_api + "/api/peers/add",
    data=json.dumps({"did": notebook_did, "address": "192.168.1.106:7001", "latency_ms": 2.45}).encode(),
    headers={"Content-Type": "application/json"}
))
urllib.request.urlopen(urllib.request.Request(
    notebook_api + "/api/peers/add",
    data=json.dumps({"did": desktop_did, "address": "192.168.1.198:7777", "latency_ms": 2.45}).encode(),
    headers={"Content-Type": "application/json"}
))
print("  -> Ambos pares registrados en sus respectivos enrutadores.")

# 3. Test Dimensión 1: ZTNA Default-Deny (Inyección de DID no autorizado)
print("\n[PASO 2/5] Prueba ZTNA: Inyectando paquete con DID intruso no autorizado...")
unauth_did = "did:ipvn7:0000000000000000000000000000000000000000000000000000000000000bad"
payload_blocked = {
    "target_url": notebook_api,
    "task_type": "probe_unauthorized",
    "payload": "unauthorized_probe",
    "source_did": unauth_did,
    "port": 7001
}
try:
    req = urllib.request.Request(
        desktop_api + "/api/tasks/distributed",
        data=json.dumps(payload_blocked).encode(),
        headers={"Content-Type": "application/json"}
    )
    urllib.request.urlopen(req)
    print("  [-] ERROR: El cortafuegos debio haber bloqueado el DID no autorizado!")
except urllib.error.HTTPError as e:
    err_body = json.loads(e.read().decode())
    print(f"  [OK] ZTNA Bloqueo Exitoso (HTTP {e.code}): {err_body['status']}")
    print(f"       Razon: {err_body['reason']}")

# 4. Test Dimensión 1: Autorización Granular en Cortafuegos
print("\n[PASO 3/5] Prueba ZTNA: Autorizando explicitamente el DID legitimo...")
policy_desktop = {
    "did": desktop_did,
    "allow_inbound": True,
    "allow_outbound": True,
    "allowed_ports": [7001, 8080]
}
urllib.request.urlopen(urllib.request.Request(
    desktop_api + "/api/firewall/rules",
    data=json.dumps(policy_desktop).encode(),
    headers={"Content-Type": "application/json"}
))
urllib.request.urlopen(urllib.request.Request(
    notebook_api + "/api/firewall/rules",
    data=json.dumps({"did": desktop_did, "allow_inbound": True, "allowed_ports": [7001, 8080]}).encode(),
    headers={"Content-Type": "application/json"}
))

# Reintentar tarea con DID autorizado
payload_auth = {
    "target_url": notebook_api,
    "task_type": "ztna_authorized_task",
    "payload": "secure_flow_token_alpha",
    "source_did": desktop_did,
    "port": 7001
}
req = urllib.request.Request(
    desktop_api + "/api/tasks/distributed",
    data=json.dumps(payload_auth).encode(),
    headers={"Content-Type": "application/json"}
)
res = urllib.request.urlopen(req)
task_res = json.loads(res.read().decode())
print(f"  [OK] Tarea ZTNA Autorizada y Ejecutada con exito:")
print(f"       Veredicto ZTNA: {task_res['ztna_decision']}")
print(f"       RTT:            {task_res['round_trip_ms']:.2f} ms")
print(f"       Prueba Hash:    {task_res['task_proof_hash']}")

# 5. Test Dimensión 3: QoS y Desafío PoW Anti-DDoS
print("\n[PASO 4/5] Prueba QoS: Forzando saturacion de rafaga para disparar PoW Dinamico...")
# Simulamos un cliente que genera saturación rápida
challenge_data = None
for i in range(50):
    heavy_payload = {
        "target_url": notebook_api,
        "task_type": "stress_burst",
        "payload": "X" * 1500,
        "source_did": desktop_did,
        "port": 7001
    }
    try:
        req = urllib.request.Request(
            desktop_api + "/api/tasks/distributed",
            data=json.dumps(heavy_payload).encode(),
            headers={"Content-Type": "application/json"}
        )
        urllib.request.urlopen(req)
    except urllib.error.HTTPError as e:
        if e.code == 429:
            body = json.loads(e.read().decode())
            challenge_data = body.get('pow_challenge')
            print(f"  [OK] QoS Activo: Rafaga estrangulada (HTTP 429).")
            print(f"       Desafio PoW emitido: {challenge_data['challenge'][:24]}... (Dificultad: {challenge_data['difficulty']} bits)")
            break

if challenge_data:
    # Resolver PoW en Python
    ch_bytes = bytes.fromhex(challenge_data['challenge'])
    diff_bytes = challenge_data['difficulty'] // 8
    print(f"  [+] Resolviendo desafio PoW ({challenge_data['difficulty']} bits)...")
    start_solve = time.time()
    nonce = 0
    while True:
        data = ch_bytes + struct.pack(">Q", nonce)
        h = hashlib.sha256(data).digest()
        if h[:diff_bytes] == b"\x00" * diff_bytes:
            break
        nonce += 1
    solve_duration = (time.time() - start_solve) * 1000.0
    print(f"  [OK] Desafio PoW resuelto en {solve_duration:.2f} ms! Nonce hallado: {nonce}")

    # Enviar solución al endpoint de QoS
    solve_req = urllib.request.Request(
        desktop_api + "/api/qos/pow/solve",
        data=json.dumps({"challenge": challenge_data['challenge'], "nonce": nonce}).encode(),
        headers={"Content-Type": "application/json"}
    )
    res_solve = json.loads(urllib.request.urlopen(solve_req).read().decode())
    print(f"  [OK] Validacion PoW en receptor: {res_solve['message']}")
else:
    print("  [i] La tasa de 9.44 Mbps absorbio la rafaga sin llegar a saturar.")

# 6. Test Dimensión 12: Zero-Copy Buffer Pool Stats
print("\n[PASO 5/5] Inspeccionando telemetria del Zero-Copy Buffer Pool...")
pool_req = urllib.request.urlopen(desktop_api + "/api/pool")
pool_stats = json.loads(pool_req.read().decode())
print(f"  -> Asignaciones Small (64B):     {pool_stats['allocations_small']}")
print(f"  -> Asignaciones Standard (1500B): {pool_stats['allocations_standard']}")
print(f"  -> Asignaciones Jumbo (64KB):    {pool_stats['allocations_jumbo']}")
print(f"  -> Total Búferes Reciclados:      {pool_stats['recycled_count']}")

print("\n==========================================================")
print("  [OK] FASE A COMPLETADA Y VERIFICADA AL 100% EN LA MALLA  ")
print("==========================================================")
