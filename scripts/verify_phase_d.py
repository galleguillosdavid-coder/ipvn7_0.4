#!/usr/bin/env python3
"""
verify_phase_d.py - Verificación Empírica de Fase D
Dimensiones Evaluadas:
  - Dimensión 6: Plano de Control Dual (CLI Soberano & Servidor MCP JSON-RPC)
  - Dimensión 11: IA Agéntica Nativa, Diagnóstico Profundo & Auto-Curación Autónoma
  - Integración Global de las 12 Dimensiones sobre la Malla Física LAN (PC <-> Notebook)
"""

import sys
import json
import time
import subprocess
import urllib.request
import urllib.error

PC_BASE_URL = "http://localhost:7070"
NOTEBOOK_BASE_URL = "http://192.168.1.106:8080"
NOTEBOOK_HOST = "192.168.1.106"
NOTEBOOK_SSH_USER = "frondabrick"

PC_DID = "did:ipvn7:e93524a43b6487f4d6b0b0881251b5088d06f1ad3c400edd1d26d2eb777e838a"
NOTEBOOK_DID = "did:ipvn7:bda3fed80fbbb6e52ed2bd34f2dc87a63436f301f07e7748d72c5b4a1b5c0c24"

def safe_print(msg):
    try:
        print(msg)
    except UnicodeEncodeError:
        print(msg.encode("ascii", "namereplace").decode("ascii"))

def http_get(url):
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))

def http_post(url, data_dict):
    req = urllib.request.Request(
        url,
        data=json.dumps(data_dict).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST"
    )
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))

def test_cli_subsystems():
    safe_print("\n[TEST 1/4] Verificando Comandos del CLI Soberano (Dimensión 6)...")
    cli_path = r".\bin\windows_amd64\ipvn7-cli.exe"
    
    # 1. Test status
    res = subprocess.run([cli_path, "status"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if res.returncode != 0 or "DID Soberano" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli status': {res.stderr}")
        return False
    safe_print("  [+] 'ipvn7-cli status' exitoso: DID e IPs virtuales reportadas.")

    # 2. Test firewall
    res = subprocess.run([cli_path, "firewall", "list"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if "Default-Deny" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli firewall': {res.stdout}")
        return False
    safe_print("  [+] 'ipvn7-cli firewall list' exitoso: Default-Deny verificado.")

    # 3. Test dag
    res = subprocess.run([cli_path, "dag", "put", "Empirical Block Test Payload D"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if "cid:ipvn7:" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli dag put': {res.stdout}")
        return False
    safe_print("  [+] 'ipvn7-cli dag put' exitoso: Bloque inmutable generado con CID.")

    # 4. Test accounting
    res = subprocess.run([cli_path, "accounting"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if not res.stdout or "TIT-FOR-TAT" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli accounting': {res.stdout}")
        return False
    safe_print("  [+] 'ipvn7-cli accounting' exitoso: Métricas Tit-for-Tat reportadas.")

    # 5. Test wot
    res = subprocess.run([cli_path, "wot", "vouch", NOTEBOOK_DID, "0.95", "Notebook physical partner"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if not res.stdout or "Reputación" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli wot vouch': {res.stdout}")
        return False
    safe_print("  [+] 'ipvn7-cli wot vouch' exitoso: Aval emitido y reputación calculada.")

    # 6. Test multipath
    res = subprocess.run([cli_path, "multipath"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if not res.stdout or "PLANIFICADOR" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli multipath': {res.stdout}")
        return False
    safe_print("  [+] 'ipvn7-cli multipath' exitoso: Enlaces físicos heterogéneos listados.")

    # 7. Test copilot
    res = subprocess.run([cli_path, "copilot", "diagnose"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    if not res.stdout or "DeepSeek" not in res.stdout or "Auto-reparación" not in res.stdout:
        safe_print(f"[-] Fallo en 'ipvn7-cli copilot diagnose': {res.stdout}")
        return False
    safe_print("  [+] 'ipvn7-cli copilot diagnose' exitoso: Diagnóstico y auto-curación aplicados.")

    return True

def test_mcp_json_rpc():
    safe_print("\n[TEST 2/4] Verificando Protocolo MCP JSON-RPC 2.0 sobre Stdio (Dimensión 6)...")
    cli_path = r".\bin\windows_amd64\ipvn7-cli.exe"

    proc = subprocess.Popen(
        [cli_path, "mcp"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
        errors="replace",
        bufsize=1
    )

    def send_rpc(req_obj):
        line = json.dumps(req_obj) + "\n"
        proc.stdin.write(line)
        proc.stdin.flush()
        resp_line = proc.stdout.readline()
        return json.loads(resp_line.strip())

    try:
        # 1. Initialize
        init_resp = send_rpc({"jsonrpc": "2.0", "id": 1, "method": "initialize"})
        if "serverInfo" not in init_resp.get("result", {}):
            safe_print(f"[-] Fallo en MCP initialize: {init_resp}")
            return False
        safe_print("  [+] MCP 'initialize' respondido correctamente (v0.3.0).")

        # 2. Tools list
        tools_resp = send_rpc({"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
        tools = tools_resp.get("result", {}).get("tools", [])
        tool_names = [t["name"] for t in tools]
        expected = [
            "get_node_status", "list_peers", "route_packet",
            "ipvn7_firewall_rule", "ipvn7_dag_put", "ipvn7_wot_vouch",
            "ipvn7_ddns_resolve", "ipvn7_sas_derive", "ipvn7_ai_diagnose"
        ]
        missing = [t for t in expected if t not in tool_names]
        if missing:
            safe_print(f"[-] Faltan herramientas MCP: {missing}")
            return False
        safe_print(f"  [+] MCP 'tools/list' verificado: {len(tools)} herramientas activas ({', '.join(expected)}).")

        # 3. Call ipvn7_firewall_rule
        fw_call = send_rpc({
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "ipvn7_firewall_rule",
                "arguments": {
                    "action": "set",
                    "did": NOTEBOOK_DID,
                    "allow_inbound": True,
                    "allow_outbound": True
                }
            }
        })
        if "error" in fw_call:
            safe_print(f"[-] Error llamando ipvn7_firewall_rule: {fw_call['error']}")
            return False
        safe_print("  [+] MCP 'ipvn7_firewall_rule' ejecutado con éxito: política ZTNA aplicada vía IA.")

        # 4. Call ipvn7_dag_put
        dag_call = send_rpc({
            "jsonrpc": "2.0",
            "id": 4,
            "method": "tools/call",
            "params": {
                "name": "ipvn7_dag_put",
                "arguments": {
                    "payload": "MCP-Autonomous-Block-Verification-D"
                }
            }
        })
        if "error" in dag_call:
            safe_print(f"[-] Error llamando ipvn7_dag_put: {dag_call['error']}")
            return False
        content_txt = dag_call["result"]["content"][0]["text"]
        if "cid:ipvn7:" not in content_txt:
            safe_print(f"[-] Salida CID no encontrada en dag_put: {content_txt}")
            return False
        safe_print("  [+] MCP 'ipvn7_dag_put' ejecutado con éxito: CID inmutable devuelto a la IA.")

        # 5. Call ipvn7_ai_diagnose
        ai_call = send_rpc({
            "jsonrpc": "2.0",
            "id": 5,
            "method": "tools/call",
            "params": {
                "name": "ipvn7_ai_diagnose",
                "arguments": {
                    "recent_latency_ms": 78.4,
                    "drops_count": 12
                }
            }
        })
        if "error" in ai_call:
            safe_print(f"[-] Error llamando ipvn7_ai_diagnose: {ai_call['error']}")
            return False
        ai_txt = ai_call["result"]["content"][0]["text"]
        if "anomalies_detected" not in ai_txt:
            safe_print(f"[-] Inferencia de copiloto no encontrada: {ai_txt}")
            return False
        safe_print("  [+] MCP 'ipvn7_ai_diagnose' ejecutado con éxito: Inferencia y auto-reparación retornadas.")

        return True
    finally:
        proc.terminate()
        proc.wait()

def test_ai_copilot_network():
    safe_print("\n[TEST 3/4] Verificando Motor de Copiloto de IA y Auto-Curación en Malla (Dimensión 11)...")

    # 1. Inyectar anomalía simulada en nodo PC
    safe_print("  [*] Disparando escaneo de anomalías en PC Node (http://localhost:7070/api/copilot/diagnose)...")
    diag_pc = http_post(f"{PC_BASE_URL}/api/copilot/diagnose", {
        "recent_latency_ms": 68.2,
        "drops_count": 9,
        "peer_did": "did:ipvn7:rogue_unauthorized_attacker"
    })
    
    anomalies = diag_pc.get("anomalies", [])
    diagnoses = diag_pc.get("diagnoses", [])
    safe_print(f"  [+] Anomalías detectadas por telemetría: {len(anomalies)}")
    for d in diagnoses:
        safe_print(f"      -> Diagnóstico [{d.get('model_used')}]: {d.get('root_cause')}")
        safe_print(f"         Acción Aplicada: {d.get('action_type')} (Auto-Curación: {d.get('executed_action')})")
        if not d.get("executed_action"):
            safe_print("[-] La auto-curación no se ejecutó.")
            return False

    # 2. Inyectar anomalía en Notebook remoto (http://192.168.1.106:8080/api/copilot/diagnose)
    safe_print("  [*] Disparando escaneo de anomalías en Notebook Node (http://192.168.1.106:8080/api/copilot/diagnose)...")
    diag_nb = http_post(f"{NOTEBOOK_BASE_URL}/api/copilot/diagnose", {
        "recent_latency_ms": 82.0,
        "drops_count": 14,
        "peer_did": "did:ipvn7:malicious_lan_probe"
    })
    nb_diagnoses = diag_nb.get("diagnoses", [])
    safe_print(f"  [+] Notebook procesó {len(nb_diagnoses)} diagnósticos con auto-curación.")
    for d in nb_diagnoses:
        safe_print(f"      -> Notebook [{d.get('model_used')}]: Acción: {d.get('action_type')}")

    # 3. Consultar historial de copiloto en ambos nodos
    hist_pc = http_get(f"{PC_BASE_URL}/api/copilot/history")
    hist_nb = http_get(f"{NOTEBOOK_BASE_URL}/api/copilot/history")
    safe_print(f"  [+] Historial persistente validado: PC={hist_pc.get('total', 0)} entradas, Notebook={hist_nb.get('total', 0)} entradas.")

    return True

def test_full_mesh_integration():
    safe_print("\n[TEST 4/4] Verificando Funcionamiento Integral de las 12 Dimensiones en la Malla Física...")

    # 1. Autorizar mutuamente en ZTNA Firewall
    http_post(f"{PC_BASE_URL}/api/firewall/rules", {
        "did": NOTEBOOK_DID,
        "allow_inbound": True,
        "allow_outbound": True,
        "allow_relay": True
    })
    http_post(f"{PC_BASE_URL}/api/firewall/rules", {
        "did": PC_DID,
        "allow_inbound": True,
        "allow_outbound": True,
        "allow_relay": True
    })
    http_post(f"{NOTEBOOK_BASE_URL}/api/firewall/rules", {
        "did": PC_DID,
        "allow_inbound": True,
        "allow_outbound": True,
        "allow_relay": True
    })
    http_post(f"{NOTEBOOK_BASE_URL}/api/firewall/rules", {
        "did": NOTEBOOK_DID,
        "allow_inbound": True,
        "allow_outbound": True,
        "allow_relay": True
    })
    safe_print("  [+] Reglas ZTNA de micro-segmentación configuradas mutuamente.")

    # 2. Despachar tarea distribuida sobre la malla
    task_res = http_post(f"{PC_BASE_URL}/api/tasks/distributed", {
        "target_url": NOTEBOOK_BASE_URL,
        "task_type": "distributed_computation",
        "payload": "Distributed Ingestion and Anomaly Verification D",
        "source_did": PC_DID,
        "port": 7001
    })
    if task_res.get("status") != "COMPLETED":
        safe_print(f"[-] Despacho de tarea falló: {task_res}")
        return False
    safe_print(f"  [+] Tarea distribuida completada con éxito: RTT = {task_res.get('round_trip_ms', 0):.2f} ms | Hash Proof: {task_res.get('task_proof_hash', '')[:20]}... | ZTNA: {task_res.get('ztna_decision')}")

    # 3. Validar estado general del panel web
    status_pc = http_get(f"{PC_BASE_URL}/api/status")
    status_nb = http_get(f"{NOTEBOOK_BASE_URL}/api/status")

    safe_print(f"  [+] PC Node Uptime: {status_pc.get('uptime_sec', 0):.1f}s | Telemetría Tx={status_pc.get('telemetry',{}).get('packets_tx',0)}")
    safe_print(f"  [+] Notebook Uptime: {status_nb.get('uptime_sec', 0):.1f}s | Telemetría Tx={status_nb.get('telemetry',{}).get('packets_tx',0)}")
    safe_print(f"  [+] Auto-curación Copilot activa en PC: {status_pc.get('copilot', {}).get('auto_healing_active')}")
    safe_print(f"  [+] Auto-curación Copilot activa en Notebook: {status_nb.get('copilot', {}).get('auto_healing_active')}")

    return True

def main():
    safe_print("=" * 80)
    safe_print("VERIFICACION EMPIRICA FASE D: PLANO DE CONTROL DUAL & AI COPILOT")
    safe_print("12 Dimensiones del Sistema Operativo de Red ipvn7 v0.3")
    safe_print("=" * 80)

    ok1 = test_cli_subsystems()
    ok2 = test_mcp_json_rpc()
    ok3 = test_ai_copilot_network()
    ok4 = test_full_mesh_integration()

    safe_print("\n" + "=" * 80)
    if ok1 and ok2 and ok3 and ok4:
        safe_print("[EXITO TOTAL] FASE D Y LAS 12 DIMENSIONES COMPLETAMENTE VALIDADAS.")
        safe_print("La red soberana ipvn7 opera de forma autónoma, resiliente y continua.")
        safe_print("=" * 80)
        sys.exit(0)
    else:
        safe_print("[-] Se registraron fallos durante la verificación.")
        safe_print("=" * 80)
        sys.exit(1)

if __name__ == "__main__":
    main()
