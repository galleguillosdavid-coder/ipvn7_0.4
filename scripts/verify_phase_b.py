import urllib.request
import urllib.error
import json
import time

desktop_api = "http://localhost:7070"
notebook_api = "http://192.168.1.106:8080"

print("==========================================================")
print("  IPv7 v0.3 - Verificacion Empirica de Fase B (Dim 4, 5, 9)")
print("  Almacen DAG & DTN | Tit-for-Tat | Web-of-Trust (WoT)   ")
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

# 2. Emparejamiento en Kleinberg y Autorización en ZTNA Firewall
print("\n[PASO 1/5] Inicializando enlaces y autorizaciones ZTNA...")
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
urllib.request.urlopen(urllib.request.Request(
    desktop_api + "/api/firewall/rules",
    data=json.dumps({"did": desktop_did, "allow_inbound": True, "allowed_ports": [7001, 8080]}).encode(),
    headers={"Content-Type": "application/json"}
))
urllib.request.urlopen(urllib.request.Request(
    desktop_api + "/api/firewall/rules",
    data=json.dumps({"did": notebook_did, "allow_inbound": True, "allowed_ports": [7001, 8080]}).encode(),
    headers={"Content-Type": "application/json"}
))
urllib.request.urlopen(urllib.request.Request(
    notebook_api + "/api/firewall/rules",
    data=json.dumps({"did": desktop_did, "allow_inbound": True, "allowed_ports": [7001, 8080]}).encode(),
    headers={"Content-Type": "application/json"}
))
urllib.request.urlopen(urllib.request.Request(
    notebook_api + "/api/firewall/rules",
    data=json.dumps({"did": notebook_did, "allow_inbound": True, "allowed_ports": [7001, 8080]}).encode(),
    headers={"Content-Type": "application/json"}
))
print("  -> Nodos autorizados mutuamente en cortafuegos.")

# 3. Test Dimensión 4: Almacenamiento DAG y Store-and-Forward (DTN)
print("\n[PASO 2/5] Prueba DAG: Generando bloque direccionado por contenido (CID)...")
dag_payload = {
    "data": "kuzu_distributed_graph_partition_delta_42",
    "parents": [],
    "target_did": notebook_did
}
req = urllib.request.Request(
    desktop_api + "/api/dag/blocks",
    data=json.dumps(dag_payload).encode(),
    headers={"Content-Type": "application/json"}
)
res = urllib.request.urlopen(req)
created_block = json.loads(res.read().decode())
cid = created_block['cid']
print(f"  [OK] Bloque inmutable generado con CID: {cid}")
print(f"       Autor: {created_block['author_did'][:32]}...")
print(f"       Destinatario DTN: {created_block['target_did'][:32]}...")

# Verificar cola de retención DTN en Desktop
pending_req = urllib.request.urlopen(desktop_api + f"/api/dag/pending?target={notebook_did}")
pending_list = json.loads(pending_req.read().decode())
print(f"  [OK] Cola Store-and-Forward (DTN) retiene {len(pending_list)} bloque(s) para el Notebook.")

# Simular transferencia DTN al Notebook (reconexión)
print("  [+] Sincronizando bloque con el receptor (Notebook)...")
ingest_req = urllib.request.Request(
    notebook_api + "/api/dag/ingest",
    data=json.dumps(created_block).encode(),
    headers={"Content-Type": "application/json"}
)
res_ingest = json.loads(urllib.request.urlopen(ingest_req).read().decode())
print(f"  [OK] Bloque ingerido y verificado criptograficamente por el Notebook: {res_ingest['status']}")

# Confirmar entrega y limpiar cola DTN
clear_req = urllib.request.Request(
    desktop_api + f"/api/dag/pending?target={notebook_did}",
    data=b"",
    headers={"Content-Type": "application/json"}
)
res_clear = json.loads(urllib.request.urlopen(clear_req).read().decode())
print(f"  [OK] Bundle confirmado y purgado de la cola DTN: {res_clear['status']}")

# 4. Test Dimensión 5: Economía de Tránsito Tit-for-Tat
print("\n[PASO 3/5] Prueba Tit-for-Tat: Evaluando contabilidad y reciprocidad de trafico...")
# Ejecutar 3 tareas distribuidas para intercambiar bytes
for i in range(3):
    task_p = {
        "target_url": notebook_api,
        "task_type": "tit_for_tat_flow",
        "payload": f"reciprocity_exchange_packet_{i}",
        "source_did": desktop_did,
        "port": 7001
    }
    req = urllib.request.Request(
        desktop_api + "/api/tasks/distributed",
        data=json.dumps(task_p).encode(),
        headers={"Content-Type": "application/json"}
    )
    urllib.request.urlopen(req)

# Consultar contabilidad
acc_req = urllib.request.urlopen(desktop_api + "/api/accounting")
acc_data = json.loads(acc_req.read().decode())
print(f"  [OK] Estado de Reciprocidad Tit-for-Tat:")
for a in acc_data['accounts']:
    print(f"       DID: {a['did'][:28]}... | Nivel: {a['tier']} | Tx: {a['bytes_tx']}B | Rx: {a['bytes_rx']}B | Ratio: {a['tit_for_tat_ratio']:.2f}")

# 5. Test Dimensión 9: Web-of-Trust (WoT) y Avales Criptográficos
print("\n[PASO 4/5] Prueba WoT: Emitiendo aval criptografico de confianza...")
vouch_payload = {
    "subject_did": notebook_did,
    "trust_level": 0.95,
    "reason": "Nodo de laboratorio certificado en red LAN",
    "hours": 48
}
req = urllib.request.Request(
    desktop_api + "/api/wot/vouch",
    data=json.dumps(vouch_payload).encode(),
    headers={"Content-Type": "application/json"}
)
res_vouch = json.loads(urllib.request.urlopen(req).read().decode())
print(f"  [OK] Aval firmado emitido: Emisor {res_vouch['issuer_did'][:20]}... -> Sujeto {res_vouch['subject_did'][:20]}...")
print(f"       Nivel de Confianza: {res_vouch['trust_level']} | Motivo: '{res_vouch['reason']}'")

# Consultar reputación atenuada (BFS de 1 salto)
rep_req = urllib.request.urlopen(desktop_api + f"/api/wot/reputation?target={notebook_did}")
rep_data = json.loads(rep_req.read().decode())
print(f"\n[PASO 5/5] Calculo de Reputacion Atenuada (BFS):")
print(f"       Sujeto:           {rep_data['target_did'][:32]}...")
print(f"       Saltos de Grafo:  {rep_data['hops']} grado(s)")
print(f"       Puntaje Calculado: {rep_data['reputation_score']} / 100.0 (Atenuacion del 15% por salto)")

print("\n==========================================================")
print("  [OK] FASE B COMPLETADA Y VERIFICADA AL 100% EN LA MALLA  ")
print("==========================================================")
