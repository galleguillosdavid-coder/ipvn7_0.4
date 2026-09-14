#!/usr/bin/env python3
"""
verify_ebpf_fastpath.py - Verification script for Fase 4:
Fast-Path eBPF/XDP Bridge & Live Kernel Sync.
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
    ("Desktop (WSL2 Linux)", get_desktop_url()),
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
    
    # 1. Status Check: eBPF metadata in /api/status
    code, status_resp = make_request(f"{base_url}/api/status")
    if code != 200:
        print(f"[FAIL] /api/status returned HTTP {code}: {status_resp}")
        return False
    
    ebpf_meta = status_resp.get("ebpf", {})
    driver_mode = ebpf_meta.get("driver_mode", "unknown")
    avg_ns = ebpf_meta.get("avg_proc_ns", 999)
    print(f"[OK] Daemon status: DID={status_resp.get('did')[:30]}...")
    print(f"     eBPF Status: driver_mode={driver_mode}, kernel_loaded={ebpf_meta.get('kernel_loaded')}, avg_proc={avg_ns} ns")
    
    if avg_ns > 28:
        print(f"[FAIL] Processing latency exceeds 28ns fast-path limit: {avg_ns} ns")
        return False

    # 2. Detailed eBPF Status Endpoint
    code, ep_resp = make_request(f"{base_url}/api/ebpf/status")
    if code != 200:
        print(f"[FAIL] /api/ebpf/status returned HTTP {code}: {ep_resp}")
        return False
    print(f"[OK] eBPF Hook Details:")
    print(f"     - Interface:       {ep_resp.get('interface')}")
    print(f"     - Driver Mode:     {ep_resp.get('driver_mode')}")
    print(f"     - Ring Buffer Cap: {ep_resp.get('ring_buf_slots')} slots")
    print(f"     - Active Routes:   {ep_resp.get('active_routes')}")

    # 3. Dynamic Route Sync into eBPF Map
    sync_payload = {
        "did": "did:ipvn7:test-remote-ebpf-target",
        "next_ip": "10.7.42.100",
        "degree": 3,
        "latency_ms": 1.45,
        "if_index": 1
    }
    code, sync_resp = make_request(f"{base_url}/api/ebpf/sync", method="POST", data=sync_payload)
    if code != 200:
        print(f"[FAIL] /api/ebpf/sync returned HTTP {code}: {sync_resp}")
        return False
    
    print(f"[OK] Dynamic Route Injected into eBPF Kernel Map:")
    print(f"     - Status:     {sync_resp.get('status')}")
    print(f"     - Target DID: {sync_resp.get('did')}")
    print(f"     - Map Dump:\n{sync_resp.get('map_dump')}")

    # 4. Verify Route is Present in Active Table
    code, check_resp = make_request(f"{base_url}/api/ebpf/status")
    routes = check_resp.get("routes", [])
    found = any(r.get("did") == sync_payload["did"] for r in routes)
    if not found:
        print(f"[FAIL] Injected route not found in active eBPF map!")
        return False
    print(f"[OK] Route confirmed in live eBPF map (total routes: {len(routes)})")

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
    print("FINAL SUMMARY - FASE 4 EBPF/XDP FAST-PATH CERTIFICATION")
    print("=======================================================")
    all_ok = True
    for name, ok in results.items():
        print(f"  {name}: {'PASSED' if ok else 'FAILED'}")
        if not ok:
            all_ok = False
            
    if not all_ok:
        sys.exit(1)
    print("\n>>> ALL FASE 4 EBPF/XDP FAST-PATH PROTOCOLS CERTIFIED ON PHYSICAL MESH <<<")
    sys.exit(0)

if __name__ == "__main__":
    main()
