import urllib.request
import json
import time

desktop_api = "http://localhost:7070"
notebook_api = "http://192.168.1.106:8080"

print("==========================================================")
print("     IPv7 v0.3: Orquestador Mesh y Tareas Distribuidas    ")
print("==========================================================")

# 1. Obtener status de ambos nodos
req1 = urllib.request.urlopen(desktop_api + "/api/status")
desktop_status = json.loads(req1.read().decode())
print(f"[+] Nodo PC Desktop (WSL2):")
print(f"    DID:  {desktop_status['did']}")
print(f"    IPv6: {desktop_status['sovereign_v6']}")
print(f"    IPv4: {desktop_status['virtual_v4']}")

req2 = urllib.request.urlopen(notebook_api + "/api/status")
notebook_status = json.loads(req2.read().decode())
print(f"\n[+] Nodo Notebook (Windows):")
print(f"    DID:  {notebook_status['did']}")
print(f"    IPv6: {notebook_status['sovereign_v6']}")
print(f"    IPv4: {notebook_status['virtual_v4']}")

# 2. Emparejar Nodo Notebook en el Router Kleinberg del Nodo Desktop
payload_to_desktop = {
    "did": notebook_status['did'],
    "address": "192.168.1.106:7001",
    "latency_ms": 2.45
}
req = urllib.request.Request(
    desktop_api + "/api/peers/add",
    data=json.dumps(payload_to_desktop).encode(),
    headers={"Content-Type": "application/json"}
)
res = urllib.request.urlopen(req)
print(f"\n[+] Par Notebook registrado en Kleinberg Router del Desktop: {res.read().decode()}")

# 3. Emparejar Nodo Desktop en el Router Kleinberg del Nodo Notebook
payload_to_notebook = {
    "did": desktop_status['did'],
    "address": "192.168.1.198:7777",
    "latency_ms": 2.45
}
req = urllib.request.Request(
    notebook_api + "/api/peers/add",
    data=json.dumps(payload_to_notebook).encode(),
    headers={"Content-Type": "application/json"}
)
res = urllib.request.urlopen(req)
print(f"[+] Par Desktop registrado en Kleinberg Router del Notebook: {res.read().decode()}")

# 4. Ejecutar Tarea Distribuida: Cómputo Criptográfico y Medición RTT bidireccional
print("\n[+] Ejecutando Tarea Distribuida #1: Sincronización de Estado PFO Kùzu & RTT...")
task_payload = {
    "target_url": notebook_api,
    "task_type": "kuzu_pfo_state_sync",
    "payload": "afe_axiom_integrity_check:5_axioms:14_functions"
}
req = urllib.request.Request(
    desktop_api + "/api/tasks/distributed",
    data=json.dumps(task_payload).encode(),
    headers={"Content-Type": "application/json"}
)
res = urllib.request.urlopen(req)
task_result = json.loads(res.read().decode())
print("\n[OK] Resultado de Tarea Distribuida completada con exito:")
print(f"    Tipo de Tarea:     {task_result['task_type']}")
print(f"    Nodo Origen:       {task_result['local_did'][:32]}...")
print(f"    Nodo Destino:      {task_result['remote_did'][:32]}...")
print(f"    RTT Medido:        {task_result['round_trip_ms']:.2f} ms")
print(f"    Prueba Hash (PFO): {task_result['task_proof_hash']}")
print(f"    Packet Pacing Tasa: {task_result['pacer_effective']} bytes/s (9.44 Mbps)")
print(f"    Timestamp UTC:     {task_result['timestamp']}")

# 5. Consultar estado final de ambos radares
print("\n[+] Consultando estado actualizado de radares:")
req_status = urllib.request.urlopen(desktop_api + "/api/status")
final_desktop = json.loads(req_status.read().decode())
print(f"    Desktop tiene {len(final_desktop['peers'])} par(es) en su radar:")
for p in final_desktop['peers']:
    print(f"      - Anillo Kleinberg [{p['ring']}]: DID {p['did'][:28]}... | Latencia: {p['latency_ms']} ms | Salud: {p['health_score']}")

req_nb_status = urllib.request.urlopen(notebook_api + "/api/status")
final_nb = json.loads(req_nb_status.read().decode())
print(f"    Notebook tiene {len(final_nb['peers'])} par(es) en su radar:")
for p in final_nb['peers']:
    print(f"      - Anillo Kleinberg [{p['ring']}]: DID {p['did'][:28]}... | Latencia: {p['latency_ms']} ms | Salud: {p['health_score']}")

print("\n==========================================================")
print("  [OK] MALLA IPv7 v0.3 OPERACIONAL Y TAREAS CONSOLIDADAS  ")
print("==========================================================")
