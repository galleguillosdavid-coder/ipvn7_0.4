#!/usr/bin/env python3
"""
verify_kuzu_pfo.py - Verification script for Fase 1:
Unified KùzuDB Graph Engine & PFO Axiomatic Pipeline.
Tests both Desktop (WSL2) and Notebook (192.168.1.106:8080).
"""

import sys
import json
import urllib.request
import urllib.error
import subprocess

def get_desktop_url():
    # Try localhost first
    try:
        with urllib.request.urlopen("http://127.0.0.1:7070/api/status", timeout=1) as resp:
            if resp.status == 200:
                return "http://127.0.0.1:7070"
    except Exception:
        pass
    
    # Try WSL2 IP
    try:
        wsl_ip = subprocess.check_output(["wsl", "-d", "Ubuntu", "-e", "hostname", "-I"], text=True).strip().split()[0]
        return f"http://{wsl_ip}:7070"
    except Exception:
        return "http://127.0.0.1:7070"

HOSTS = [
    ("Desktop (WSL2)", get_desktop_url()),
    ("Notebook Physical", "http://192.168.1.106:8080"),
]

def make_request(url, method="GET", data=None):
    headers = {"Content-Type": "application/json"}
    payload = json.dumps(data).encode("utf-8") if data is not None else None
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")
        try:
            return e.code, json.loads(body)
        except Exception:
            return e.code, {"raw": body}
    except Exception as e:
        return 0, {"error": str(e)}

def test_host(name, base_url):
    print(f"\n=======================================================")
    print(f"[*] Testing {name} at {base_url}")
    print(f"=======================================================")
    
    # 1. Status & Kuzu status
    code, status_resp = make_request(f"{base_url}/api/status")
    if code != 200:
        print(f"[FAIL] /api/status returned HTTP {code}: {status_resp}")
        return False
    kuzu_meta = status_resp.get("kuzu", {})
    sovereign_v6 = status_resp.get("sovereign_v6", "unknown")
    did = status_resp.get("did", "unknown")
    total_principles = kuzu_meta.get("total_principles", 0)
    total_functions = kuzu_meta.get("total_functions", 0)
    print(f"[OK] Daemon status: DID={did[:25]}... v6={sovereign_v6}")
    print(f"     Kuzu DB Status: principles={total_principles}, functions={total_functions}, peers={kuzu_meta.get('total_peers')}")
    if total_principles < 5 or total_functions < 5:
        print(f"[FAIL] Kuzu PFO hierarchy not seeded properly on {name}")
        return False

    # 2. PFO Tree
    code, pfo_resp = make_request(f"{base_url}/api/kuzu/pfo")
    if code != 200:
        print(f"[FAIL] /api/kuzu/pfo returned HTTP {code}")
        return False
    pfo_tree = pfo_resp.get("pfo_tree", [])
    print(f"[OK] PFO Axiomatic Hierarchy retrieved:")
    print(f"     - Total Principles: {pfo_resp.get('total_principles')}")
    print(f"     - Total Functions: {pfo_resp.get('total_functions')}")
    for p in pfo_tree:
        p_id = p.get("principle_id")
        fn_count = len(p.get("functions", []))
        print(f"       -> {p_id}: {fn_count} linked operational functions")

    # 3. OpenCypher Queries
    # 3a. Axiom query
    cypher_query1 = {"query": "MATCH (a:AxiomPrinciple) RETURN a"}
    code, query_resp1 = make_request(f"{base_url}/api/kuzu/query", method="POST", data=cypher_query1)
    if code != 200:
        print(f"[FAIL] /api/kuzu/query failed: {query_resp1}")
        return False
    print(f"[OK] Cypher Query 1 (Axioms):")
    print(f"     Query: {cypher_query1['query']}")
    print(f"     Rows returned: {query_resp1.get('row_count')}, exec: {query_resp1.get('exec_ms'):.3f} ms")

    # 3b. Peer query
    cypher_query2 = {"query": "MATCH (p:Peer) RETURN p"}
    code, query_resp2 = make_request(f"{base_url}/api/kuzu/query", method="POST", data=cypher_query2)
    if code != 200:
        print(f"[FAIL] /api/kuzu/query failed: {query_resp2}")
        return False
    print(f"[OK] Cypher Query 2 (Peers):")
    print(f"     Query: {cypher_query2['query']}")
    print(f"     Rows returned: {query_resp2.get('row_count')}, exec: {query_resp2.get('exec_ms'):.3f} ms")

    # 4. Anti-Contradiction Audit (Valid Directive)
    valid_directive = {"directive": "Adicionar nuevo colector de metricas lock-free y telemetria comprimida en pkg/l2"}
    code, audit_resp1 = make_request(f"{base_url}/api/kuzu/audit", method="POST", data=valid_directive)
    if code != 200:
        print(f"[FAIL] /api/kuzu/audit failed for valid directive: {audit_resp1}")
        return False
    collides = audit_resp1.get("collides", True)
    verdict = audit_resp1.get("verdict", "")
    print(f"[OK] Valid Directive Audit:")
    print(f"     Collides: {collides} | Verdict: {verdict}")
    if collides:
        print(f"[FAIL] Valid directive was erroneously rejected!")
        return False

    # 5. Anti-Contradiction Audit (Core Freeze Contradiction)
    bad_core = {"directive": "Modificar core/ y alter wire format para insertar campo opcional"}
    code, audit_resp2 = make_request(f"{base_url}/api/kuzu/audit", method="POST", data=bad_core)
    collides2 = audit_resp2.get("collides", False)
    verdict2 = audit_resp2.get("verdict", "")
    print(f"[OK] Core Freeze Contradiction Audit:")
    print(f"     Collides: {collides2} | Verdict: {verdict2}")
    if not collides2:
        print(f"[FAIL] Core Freeze violation was NOT detected!")
        return False

    # 6. Anti-Contradiction Audit (Zero-PII Contradiction)
    bad_pii = {"directive": "Guardar IP real y persist PII de nodos clientes en disco"}
    code, audit_resp3 = make_request(f"{base_url}/api/kuzu/audit", method="POST", data=bad_pii)
    collides3 = audit_resp3.get("collides", False)
    verdict3 = audit_resp3.get("verdict", "")
    print(f"[OK] Zero-PII Contradiction Audit:")
    print(f"     Collides: {collides3} | Verdict: {verdict3}")
    if not collides3:
        print(f"[FAIL] Zero-PII violation was NOT detected!")
        return False

    # 7. Anti-Contradiction Audit (Sustainable Flow Contradiction)
    bad_flow = {"directive": "Desactivar marcapasos y habilitar unlimited burst sin control de ritmo"}
    code, audit_resp4 = make_request(f"{base_url}/api/kuzu/audit", method="POST", data=bad_flow)
    collides4 = audit_resp4.get("collides", False)
    verdict4 = audit_resp4.get("verdict", "")
    print(f"[OK] Sustainable Flow Contradiction Audit:")
    print(f"     Collides: {collides4} | Verdict: {verdict4}")
    if not collides4:
        print(f"[FAIL] Sustainable flow violation was NOT detected!")
        return False

    return True

def main():
    results = {}
    for name, url in HOSTS:
        try:
            results[name] = test_host(name, url)
        except Exception as e:
            print(f"[ERR] Exception testing {name}: {e}")
            results[name] = False

    print("\n=======================================================")
    print("FINAL SUMMARY - FASE 1 KUZU GRAPH & PFO CERTIFICATION")
    print("=======================================================")
    all_ok = True
    for name, ok in results.items():
        print(f"  {name}: {'PASSED' if ok else 'FAILED'}")
        if not ok:
            all_ok = False
            
    if not all_ok:
        sys.exit(1)
    print("\n>>> ALL FASE 1 AUDITS AND CYPHER QUERIES CERTIFIED ON PHYSICAL MESH <<<")
    sys.exit(0)

if __name__ == "__main__":
    main()
