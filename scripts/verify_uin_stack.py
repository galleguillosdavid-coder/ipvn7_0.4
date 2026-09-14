#!/usr/bin/env python3
"""
verify_uin_stack.py - Automated End-to-End Verification of Rescued UIN & Memory Arbiter Systems
Tests:
  1. UIN Passport (/api/uin/passport)
  2. Memory Arbiter Quotas & Telemetry (/api/memory/arbiter)
  3. UIN Binding Creation for AI Agents (/api/uin/binding/create)
  4. Anti-Replay 4-Vector Defensive Verification (/api/antireplay/verify)
  5. Hierarchy & Intent Snapshot (/api/hierarchy/snapshot)
"""

import sys
import json
import time
import urllib.request
import urllib.error

BASE_URL = "http://127.0.0.1:7070"
if len(sys.argv) > 1:
    BASE_URL = sys.argv[1]

print(f"===========================================================")
print(f"  VERIFYING UIN & MEMORY ARBITER STACK @ {BASE_URL}")
print(f"===========================================================")

def get_json(path):
    url = f"{BASE_URL}{path}"
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))

def post_json(path, data):
    url = f"{BASE_URL}{path}"
    body = json.dumps(data).encode("utf-8")
    req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))

passed = 0
failed = 0

def check(name, condition, detail=""):
    global passed, failed
    if condition:
        print(f"  [PASS] {name} {detail}")
        passed += 1
    else:
        print(f"  [FAIL] {name} {detail}")
        failed += 1

# 1. Test UIN Passport
print("\n[1] Testing UIN Digital Passport (/api/uin/passport)...")
try:
    passport = get_json("/api/uin/passport")
    root_id = passport.get("root_id_hex", "")
    crypto_mode = passport.get("mode", "")
    bindings = passport.get("active_bindings", [])
    
    check("UIN Root ID present (256-bit / 64 hex chars)", len(root_id) == 64, f"len={len(root_id)} ({root_id[:12]}...)")
    check("UIN Crypto Mode is hybrid", crypto_mode == "hybrid", f"mode={crypto_mode}")
    check("UIN Active Bindings array accessible", isinstance(bindings, list), f"count={len(bindings)}")
except Exception as e:
    check("UIN Passport Endpoint Reachable", False, str(e))

# 2. Test Memory Arbiter
print("\n[2] Testing Global Memory Arbiter (/api/memory/arbiter)...")
try:
    arbiter = get_json("/api/memory/arbiter")
    total_limit = arbiter.get("total_limit_bytes", 0)
    classes = arbiter.get("classes", {})
    replay_q = classes.get("replay", {}).get("limit_bytes", 0)
    qos_q = classes.get("qos", {}).get("limit_bytes", 0)
    trust_q = classes.get("trust", {}).get("limit_bytes", 0)
    bindings_q = classes.get("bindings", {}).get("limit_bytes", 0)
    other_q = classes.get("other", {}).get("limit_bytes", 0)

    check("Memory Arbiter Total Limit is 64MB", total_limit == 64 * 1024 * 1024, f"{total_limit // (1024*1024)} MB")
    check("Replay Quota is 20%", replay_q == int(total_limit * 0.20), f"{replay_q // 1024} KB")
    check("QoS Quota is 30%", qos_q == int(total_limit * 0.30), f"{qos_q // 1024} KB")
    check("Trust Quota is 20%", trust_q == int(total_limit * 0.20), f"{trust_q // 1024} KB")
    check("Bindings Quota is 20%", bindings_q == int(total_limit * 0.20), f"{bindings_q // 1024} KB")
    check("Other Quota is 10%", other_q == int(total_limit * 0.10), f"{other_q // 1024} KB")
except Exception as e:
    check("Memory Arbiter Endpoint Reachable", False, str(e))

# 3. Test UIN Binding Creation
print("\n[3] Testing UIN Delegation Binding Creation (/api/uin/binding/create)...")
try:
    # Scope 3 = ScopeAIAgent
    req_body = {"scope": 3, "duration_days": 30}
    res = post_json("/api/uin/binding/create", req_body)
    status = res.get("status")
    scope = res.get("scope")
    sig_len = len(res.get("sig_root_hex", ""))

    check("Binding Creation Status BINDING_ISSUED", status == "BINDING_ISSUED", f"status={status}")
    check("Binding Scope is ScopeAIAgent (3)", scope == 3, f"scope={scope}")
    check("Binding Root Signature present (Wire v1 canonical)", sig_len > 0, f"sig_len={sig_len}")

    # Verify passport shows the new binding
    passport2 = get_json("/api/uin/passport")
    check("Active Bindings incremented in Passport", len(passport2.get("active_bindings", [])) >= 1)
except Exception as e:
    check("UIN Binding Endpoint Reachable", False, str(e))

# 4. Test 4-Vector Anti-Replay Verification
print("\n[4] Testing 4-Vector Anti-Replay Defensive Verification (/api/antireplay/verify)...")
try:
    did = "did:ipvn7:uin:test_peer_suite"
    sess1 = int(time.time() * 1000) % 1000000 + 1000
    sess2 = sess1 + 100
    now = int(time.time())


    # Vector 1: Initial legitimate frame (seq 10)
    v1 = post_json("/api/antireplay/verify", {"origin_did": did, "session_id": sess1, "sequence": 10, "timestamp": now})
    # Vector 2: Immediate duplicate replay (seq 10 again)
    v2 = post_json("/api/antireplay/verify", {"origin_did": did, "session_id": sess1, "sequence": 10, "timestamp": now})
    # Vector 3: Cross-session stale sequence spoof (old session ID with earlier timestamp)
    v3 = post_json("/api/antireplay/verify", {"origin_did": did, "session_id": sess2, "sequence": 10, "timestamp": now - 5})
    # Vector 4: Jitter reordered frame within sliding window (seq 8 in active session 1)
    v4 = post_json("/api/antireplay/verify", {"origin_did": did, "session_id": sess1, "sequence": 8, "timestamp": now})

    check("Vector 1 (Initial Legitimate Frame): ACCEPTED", v1.get("accepted") is True)
    check("Vector 2 (Immediate Replay Attack): BLOCKED", v2.get("accepted") is False)
    check("Vector 3 (Cross-Session Stale Injection): BLOCKED", v3.get("accepted") is False)
    check("Vector 4 (Jitter Reordered Frame): ACCEPTED within Window", v4.get("accepted") is True)

except Exception as e:
    check("Anti-Replay Verification Endpoint Reachable", False, str(e))

# 5. Test Node Hierarchy & Snapshot
print("\n[5] Testing Node Hierarchy & Intent Snapshot (/api/hierarchy/snapshot)...")
try:
    snap = get_json("/api/hierarchy/snapshot")
    check("Hierarchy Snapshot returns registered node profiles", isinstance(snap, list) and len(snap) >= 1, f"profiles={len(snap)}")
    local_p = snap[0]
    check("Local node profile has valid class and name", "class" in local_p and "class_name" in local_p, f"class={local_p.get('class_name')}")
    check("Local node profile has allowed intents", len(local_p.get("allowed_intents", [])) > 0, f"intents={local_p.get('allowed_intents')}")
except Exception as e:
    check("Hierarchy Snapshot Endpoint Reachable", False, str(e))


print(f"\n===========================================================")
print(f"  VERIFICATION COMPLETE: {passed} PASSED, {failed} FAILED")
print(f"===========================================================")

if failed > 0:
    sys.exit(1)
sys.exit(0)
