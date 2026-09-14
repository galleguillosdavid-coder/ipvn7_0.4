#!/usr/bin/env python3
"""
verify_pqc_sphinx.py - Verification script for Fase 2:
Hybrid PQC Cryptography (ML-DSA / ML-KEM) & Sphinx 3-hop Onion Routing (1280B Fixed Wire).
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
    
    # 1. Status Check: PQC & Sphinx metadata
    code, status_resp = make_request(f"{base_url}/api/status")
    if code != 200:
        print(f"[FAIL] /api/status returned HTTP {code}: {status_resp}")
        return False
    
    pqc_meta = status_resp.get("pqc", {})
    sphinx_meta = status_resp.get("sphinx", {})
    print(f"[OK] Daemon status: DID={status_resp.get('did')[:30]}...")
    print(f"     PQC: algorithm_sig={pqc_meta.get('algorithm_sig')}, quantum_ready={pqc_meta.get('quantum_ready')}")
    print(f"     Sphinx: frame_bytes={sphinx_meta.get('fixed_frame_bytes')}, mtu={sphinx_meta.get('deterministic_mtu')}")

    if not pqc_meta.get("quantum_ready"):
        print(f"[FAIL] PQC quantum readiness flag not true on {name}")
        return False
    if sphinx_meta.get("fixed_frame_bytes") != 1280:
        print(f"[FAIL] Sphinx fixed frame bytes not 1280 on {name}")
        return False

    # 2. Hybrid Crypto Info
    code, info_resp = make_request(f"{base_url}/api/crypto/hybrid/info")
    if code != 200:
        print(f"[FAIL] /api/crypto/hybrid/info returned HTTP {code}")
        return False
    print(f"[OK] Hybrid Keypair certified:")
    print(f"     - Ed25519 Pub: {info_resp.get('ed25519_pub_hex')[:24]}...")
    print(f"     - X25519 Pub:  {info_resp.get('x25519_pub_hex')[:24]}...")
    print(f"     - ML-DSA Pub:  {info_resp.get('ml_dsa_pub_hex')[:24]}...")
    print(f"     - ML-KEM Pub:  {info_resp.get('ml_kem_pub_hex')[:24]}...")
    print(f"     - Security:    {info_resp.get('quantum_security')}")

    # 3. Hybrid Sign and Dual Verification Benchmark
    test_msg = {"message": "Datagrama canónico ipvn7 autenticado por firma dual cuántica ML-DSA"}
    code, sign_resp = make_request(f"{base_url}/api/crypto/hybrid/sign-verify", method="POST", data=test_msg)
    if code != 200:
        print(f"[FAIL] /api/crypto/hybrid/sign-verify failed: {sign_resp}")
        return False
    verified = sign_resp.get("verified", False)
    print(f"[OK] Dual Hybrid Signature (Ed25519 + ML-DSA):")
    print(f"     - Verified: {verified}")
    print(f"     - Sign Latency:   {sign_resp.get('sign_latency_us')} µs")
    print(f"     - Verify Latency: {sign_resp.get('verify_latency_us')} µs")
    print(f"     - Ed25519 Sig Len: {sign_resp.get('classical_sig_len')} bytes")
    print(f"     - ML-DSA Sig Len:  {sign_resp.get('pqc_sig_len')} bytes")
    if not verified:
        print(f"[FAIL] Dual signature verification failed!")
        return False

    # 4. Sphinx 3-Hop Onion Circuit Construction (1280B Wire)
    code, build_resp = make_request(f"{base_url}/api/onion/circuit/build", method="POST", data={})
    if code != 200:
        print(f"[FAIL] /api/onion/circuit/build failed: {build_resp}")
        return False
    wire_bytes = build_resp.get("wire_bytes", 0)
    print(f"[OK] Sphinx Circuit Construction (3 Hops):")
    print(f"     - Circuit ID: {build_resp.get('circuit_id')}")
    print(f"     - Wire Size:  {wire_bytes} bytes (Exact Deterministic MTU: {build_resp.get('deterministic_mtu')})")
    print(f"     - Build Time: {build_resp.get('build_latency_us')} µs")
    print(f"     - Zero-Leak Padded: {build_resp.get('zero_leak_padded')}")
    for h in build_resp.get("hops", []):
        print(f"       -> Hop {h.get('hop')} [{h.get('role')}]: DID={h.get('did')[:24]}... @ {h.get('address')}")
    if wire_bytes != 1280:
        print(f"[FAIL] Circuit wire size is not exactly 1280 bytes!")
        return False

    # 5. Sphinx 3-Hop Step-by-Step Peeling Execution
    code, peel_resp = make_request(f"{base_url}/api/onion/circuit/peel", method="POST", data={})
    if code != 200:
        print(f"[FAIL] /api/onion/circuit/peel failed: {peel_resp}")
        return False
    delivered = peel_resp.get("delivered", False)
    success = peel_resp.get("success", False)
    print(f"[OK] Sphinx Iterative Peeling Through 3 Hops:")
    print(f"     - Delivered: {delivered}")
    print(f"     - Success:   {success}")
    for step in peel_resp.get("peeling_steps", []):
        print(f"       -> Hop {step.get('hop')} [{step.get('node_role')}]: Action={step.get('action')} | Next={step.get('next_hop')} | Wire={step.get('wire_bytes', 1280)}B | Peel={step.get('peel_time_us')} µs | Tag={step.get('tag_hash')}")
    print(f"     - Payload Received: \"{peel_resp.get('payload_received')}\"")
    if not success or not delivered:
        print(f"[FAIL] Onion peeling delivery did not succeed completely!")
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
    print("FINAL SUMMARY - FASE 2 PQC & SPHINX ONION CERTIFICATION")
    print("=======================================================")
    all_ok = True
    for name, ok in results.items():
        print(f"  {name}: {'PASSED' if ok else 'FAILED'}")
        if not ok:
            all_ok = False
            
    if not all_ok:
        sys.exit(1)
    print("\n>>> ALL FASE 2 PQC & SPHINX ONION PROTOCOLS CERTIFIED ON PHYSICAL MESH <<<")
    sys.exit(0)

if __name__ == "__main__":
    main()
