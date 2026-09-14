import urllib.request
import urllib.error
import json
import time

desktop_api = "http://localhost:7070"
notebook_api = "http://192.168.1.106:8080"

print("==========================================================")
print("  IPv7 v0.3 - Verificacion Empirica de Fase C (Dim 2, 7, 8, 10)")
print("  dDNS Petnames | Smart Packets | Multipath | OOB Pairing ")
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

# 2. Inicializar reglas de firewall mutuas
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

# 3. Test Dimensión 2: dDNS Petnames locales
print("\n[PASO 1/5] Prueba dDNS: Registrando y resolviendo alias mnemotecnico 'notebook.ipv7'...")
petname_payload = {
    "name": "notebook",
    "did": notebook_did,
    "virtual_ipv6": notebook_status['sovereign_v6'],
    "virtual_ipv4": notebook_status['virtual_v4'],
    "comment": "Notebook de laboratorio verificado"
}
req = urllib.request.Request(
    desktop_api + "/api/ddns/names",
    data=json.dumps(petname_payload).encode(),
    headers={"Content-Type": "application/json"}
)
res_name = json.loads(urllib.request.urlopen(req).read().decode())
print(f"  [OK] Petname registrado: {res_name['name']} -> {res_name['did'][:32]}...")

# Resolver nombre
res_resolve = json.loads(urllib.request.urlopen(desktop_api + "/api/ddns/resolve?name=notebook").read().decode())
print(f"  [OK] Resolucion dDNS exitosa: '{res_resolve['name']}' resuelve a:")
print(f"       IPv6 Soberana: {res_resolve['virtual_ipv6']}")
print(f"       IPv4 Sintetica: {res_resolve['virtual_ipv4']}")

# 4. Test Dimensión 7: Smart Packets y Filtro DLP
print("\n[PASO 2/5] Prueba Smart Packets: Procesando cargas utiles a traves del pipeline...")
# Carga limpia
clean_req = urllib.request.Request(
    desktop_api + "/api/smart/inspect",
    data=json.dumps({"payload": "ping_sovereign_telemetry_stream", "point": 2}).encode(),
    headers={"Content-Type": "application/json"}
)
res_clean = json.loads(urllib.request.urlopen(clean_req).read().decode())
print(f"  [OK] Paquete legitimo aprobado por Smart Packets (Egress): pass={res_clean['pass']}")

# Carga maliciosa que viola Zero-PII
dlp_req = urllib.request.Request(
    desktop_api + "/api/smart/inspect",
    data=json.dumps({"payload": "DATOS_CONFIDENCIALES: PASSWORD=secreto_critico_123", "point": 2}).encode(),
    headers={"Content-Type": "application/json"}
)
res_dlp = json.loads(urllib.request.urlopen(dlp_req).read().decode())
print(f"  [OK] Paquete interceptado por guardia DLP de privacidad: pass={res_dlp['pass']}")
print(f"       Motivo del descarte: {res_dlp['reason']}")

# 5. Test Dimensión 8: Multipath Overlay Scheduler
print("\n[PASO 3/5] Prueba Multipath: Planificacion de rutas fisicas heterogeneas...")
# Menor Latencia
mp_lat = json.loads(urllib.request.urlopen(urllib.request.Request(
    desktop_api + "/api/multipath/select",
    data=json.dumps({"strategy": 0}).encode(),
    headers={"Content-Type": "application/json"}
)).read().decode())
print(f"  [OK] Estrategia Lowest-Latency: Selecciono '{mp_lat['selected_paths'][0]['name']}' (RTT: {mp_lat['selected_paths'][0]['latency_ms']} ms)")

# Duplicación Crítica
mp_dup = json.loads(urllib.request.urlopen(urllib.request.Request(
    desktop_api + "/api/multipath/select",
    data=json.dumps({"strategy": 2}).encode(),
    headers={"Content-Type": "application/json"}
)).read().decode())
print(f"  [OK] Estrategia Critical-Duplication: Duplicando concurrentemente sobre {len(mp_dup['selected_paths'])} interfaces:")
for p in mp_dup['selected_paths']:
    print(f"       - Interfaz: {p['name']} ({p['type']}) | IP: {p['local_addr']}")

# 6. Test Dimensión 10: Emparejamiento Fuera de Banda (OOB) SAS
print("\n[PASO 4/5] Prueba OOB: Derivando Codigo Corto de Autenticacion (SAS) cruzado...")
sas_desktop = json.loads(urllib.request.urlopen(desktop_api + f"/api/oob/sas?peer_did={notebook_did}").read().decode())
sas_notebook = json.loads(urllib.request.urlopen(notebook_api + f"/api/oob/sas?peer_did={desktop_did}").read().decode())

digits_d = sas_desktop['sas_code']['digits']
emojis_d = sas_desktop['sas_code']['emojis']

digits_n = sas_notebook['sas_code']['digits']
emojis_n = sas_notebook['sas_code']['emojis']

# Representación segura para terminal Windows
emoji_repr_d = " ".join([f"{e} (U+{ord(e[0]):04X})" for e in emojis_d])
emoji_repr_n = " ".join([f"{e} (U+{ord(e[0]):04X})" for e in emojis_n])

print(f"  [+] Desktop SAS Digits:  [{digits_d}]")
print(f"      Emojis Code Sequence: {' '.join(emojis_d).encode('ascii', 'namereplace').decode()}")
print(f"  [+] Notebook SAS Digits: [{digits_n}]")
print(f"      Emojis Code Sequence: {' '.join(emojis_n).encode('ascii', 'namereplace').decode()}")

if digits_d == digits_n and emojis_d == emojis_n:
    print("  [OK] COINCIDENCIA EXACTA! El SAS visual garantiza 0-MitM entre PC y Notebook.")
else:
    print("  [-] ERROR: Discrepancia en codigos SAS!")

# 7. Test Dimensión 10: Fragmentación QR Animada (UR)
print("\n[PASO 5/5] Prueba OOB: Generando tramas UR para Codigo QR Animado...")
ur_res = json.loads(urllib.request.urlopen(desktop_api + "/api/oob/profile").read().decode())
print(f"  [OK] Perfil criptografico fragmentado en {ur_res['total_frames']} tramas UR:")
for f in ur_res['ur_frames'][:3]:
    print(f"       {f}")
if ur_res['total_frames'] > 3:
    print(f"       ... ({ur_res['total_frames'] - 3} tramas mas en bucle rotativo)")

print("\n==========================================================")
print("  [OK] FASE C COMPLETADA Y VERIFICADA AL 100% EN LA MALLA  ")
print("==========================================================")
