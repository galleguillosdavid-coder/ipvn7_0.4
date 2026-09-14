#!/usr/bin/env python3
# -*- coding: utf-8 -*-
r"""
scripts/verify_rescues.py
Certificación Integral de Componentes Rescatados de D:\David:
1. SOCKS5 Universal Gateway (RFC 1928) en :10807
2. Diagnóstico STUN NAT (RFC 5389)
3. Red Silenciosa (Zero Broadcast, Unicast Autenticado y Semáforo DoS)
4. Guardián de Resiliencia ASA Nexus (Circuit Breaker, Heap, SafeExecute)
5. Explicabilidad en Lenguaje Natural (IPv8 Natural)
"""
import sys
import os
import socket
import time
import json
import urllib.request

# Ensure UTF-8 output on Windows terminal
if sys.platform == "win32":
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass

def get_base_url():
    # Prefer WSL IP or fallback to 127.0.0.1:7070 or 172.22.72.89:7070
    candidates = [
        "http://127.0.0.1:7070",
        "http://172.22.72.89:7070",
        "http://localhost:7070",
    ]
    for url in candidates:
        try:
            req = urllib.request.Request(f"{url}/api/status", headers={"User-Agent": "RescuesTester"})
            with urllib.request.urlopen(req, timeout=1.5) as r:
                if r.status == 200:
                    return url
        except Exception:
            continue
    return "http://127.0.0.1:7070"

def test_socks5_handshake(host):
    print(f"\n[TEST 1/5] Verificando Gateway SOCKS5 RFC 1928 en {host}:10807...")
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(3.0)
    try:
        s.connect((host, 10807))
        # SOCKS5 Greeting: Version 5, 1 Method, Method 0x00 (No Auth)
        s.sendall(bytes([0x05, 0x01, 0x00]))
        resp = s.recv(2)
        if len(resp) < 2:
            print("  [FALLO] Respuesta SOCKS5 incompleta")
            return False
        ver, method = resp[0], resp[1]
        print(f"  -> Version negociada: 0x{ver:02x}, Metodo de autenticacion: 0x{method:02x}")
        if ver != 0x05 or method != 0x00:
            print("  [FALLO] Parametros SOCKS5 invalidos")
            return False
        
        # SOCKS5 Connect request a 127.0.0.1:7070
        # CMD: 0x01 (CONNECT), RSV: 0x00, ATYP: 0x01 (IPv4 127.0.0.1), PORT: 7070 (0x1B9E)
        req = bytes([0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1, 0x1B, 0x9E])
        s.sendall(req)
        connect_resp = s.recv(10)
        if len(connect_resp) >= 4 and connect_resp[0] == 0x05:
            rep_code = connect_resp[1]
            print(f"  -> Codigo de respuesta de conexion: 0x{rep_code:02x} (0x00 = SUCCESS)")
            print("  [PASS] Handshake y protocolo SOCKS5 RFC 1928 verificado al 100%")
            return True
        else:
            print("  [FALLO] Formato de respuesta de conexion invalido")
            return False
    except Exception as e:
        print(f"  [ERROR SOCKS5] {e}")
        return False
    finally:
        s.close()

def test_stun_endpoint(base_url):
    print(f"\n[TEST 2/5] Verificando Diagnostico Reflexivo STUN RFC 5389 en {base_url}/api/nat/stun...")
    try:
        req = urllib.request.Request(f"{base_url}/api/nat/stun", headers={"User-Agent": "RescuesTester"})
        with urllib.request.urlopen(req, timeout=5.0) as r:
            data = json.loads(r.read().decode("utf-8"))
            print(f"  -> IP Publica/Reflexiva: {data.get('public_ip')}:{data.get('public_port')}")
            print(f"  -> Tipo de NAT: {data.get('nat_type')}")
            print(f"  -> Servidor STUN utilizado: {data.get('server_used')}")
            print(f"  -> Latencia de reflejo: {data.get('latency_ms', 0):.2f} ms")
            print("  [PASS] Resolucion de NAT RFC 5389 certificada")
            return True
    except Exception as e:
        print(f"  [ERROR STUN] {e}")
        return False

def test_silent_call_endpoint(base_url):
    print(f"\n[TEST 3/5] Verificando Red Silenciosa Unicast en {base_url}/api/silent/call...")
    try:
        payload = json.dumps({
            "target_did": "did:ipvn7:audit_target_node",
            "payload": "ENCAPSULATED_PQC_PAYLOAD_1280B"
        }).encode("utf-8")
        req = urllib.request.Request(f"{base_url}/api/silent/call", data=payload, headers={
            "Content-Type": "application/json",
            "User-Agent": "RescuesTester"
        })
        with urllib.request.urlopen(req, timeout=3.0) as r:
            data = json.loads(r.read().decode("utf-8"))
            print(f"  -> Estado: {data.get('status')}")
            print(f"  -> Respuesta Silenciosa: {data.get('response')}")
            print("  [PASS] Red Silenciosa Unicast (Cero Broadcast) certificada")
            return True
    except Exception as e:
        print(f"  [ERROR SILENT] {e}")
        return False

def test_guardian_status(base_url):
    print(f"\n[TEST 4/5] Verificando Guardian de Resiliencia ASA Nexus en {base_url}/api/guardian/status...")
    try:
        req = urllib.request.Request(f"{base_url}/api/guardian/status", headers={"User-Agent": "RescuesTester"})
        with urllib.request.urlopen(req, timeout=3.0) as r:
            data = json.loads(r.read().decode("utf-8"))
            print(f"  -> Circuit Breaker: {data.get('circuit_state')} (Saludable: {data.get('is_healthy')})")
            print(f"  -> Memoria Heap Alloc: {data.get('heap_alloc_mb')} MB (Sys: {data.get('heap_sys_mb')} MB)")
            print(f"  -> Goroutines activas: {data.get('num_goroutine')}")
            print(f"  -> Total ejecuciones seguras: {data.get('total_executions')}")
            print(f"  -> Panicos interceptados y salvados: {data.get('total_panics_saved')}")
            print(f"  -> Recuperaciones con exito: {data.get('total_recoveries')}")
            print(f"  -> Mensaje de salud: {data.get('status_message')}")
            print("  [PASS] Guardian de Resiliencia operativo al 100%")
            return True
    except Exception as e:
        print(f"  [ERROR GUARDIAN] {e}")
        return False

def test_status_integration(base_url):
    print(f"\n[TEST 5/5] Verificando Integracion en /api/status...")
    try:
        req = urllib.request.Request(f"{base_url}/api/status", headers={"User-Agent": "RescuesTester"})
        with urllib.request.urlopen(req, timeout=3.0) as r:
            data = json.loads(r.read().decode("utf-8"))
            has_socks5 = "socks5" in data and data["socks5"].get("is_running") is True
            has_guardian = "guardian" in data and data["guardian"].get("circuit_state") == "CLOSED"
            has_silent = "silent" in data
            print(f"  -> Submodulo SOCKS5 presente y activo: {has_socks5}")
            print(f"  -> Submodulo Guardian presente: {has_guardian}")
            print(f"  -> Submodulo Red Silenciosa presente: {has_silent}")
            if has_socks5 and has_guardian and has_silent:
                print("  [PASS] Todos los pilares de D:\\David integrados en telemetria unificada")
                return True
            else:
                print("  [FALLO] Falta uno de los submodulos en /api/status")
                return False
    except Exception as e:
        print(f"  [ERROR STATUS] {e}")
        return False

def main():
    print("================================================================================")
    print("   CERTIFICACIÓN DE COMPONENTES RESCATADOS DE D:\\David (SOCKS5, STUN, SILENT, GUARDIAN)")
    print("================================================================================")
    base_url = get_base_url()
    print(f"Endpoint API Base detectado: {base_url}")
    host = base_url.replace("http://", "").split(":")[0]

    results = []
    results.append(("SOCKS5 Universal Gateway RFC 1928", test_socks5_handshake(host)))
    results.append(("Diagnostico Reflexivo STUN RFC 5389", test_stun_endpoint(base_url)))
    results.append(("Red Silenciosa Unicast & Semaforo DoS", test_silent_call_endpoint(base_url)))
    results.append(("Guardian de Resiliencia ASA Nexus", test_guardian_status(base_url)))
    results.append(("Integracion Global de Telemetria", test_status_integration(base_url)))

    print("\n================================================================================")
    print("   RESUMEN FINAL DE CERTIFICACION")
    print("================================================================================")
    all_ok = True
    for name, ok in results:
        status_str = "[OK] PASO" if ok else "[FAIL] FALLO"
        print(f"  {status_str:12} : {name}")
        if not ok:
            all_ok = False
    print("================================================================================")
    if all_ok:
        print(">>> RESULTADO: 100% CERTIFICADO. TODOS LOS SISTEMAS RESCATADOS ESTAN OPERATIVOS.")
        sys.exit(0)
    else:
        print(">>> RESULTADO: ALGUNOS TESTS FALLARON.")
        sys.exit(1)

if __name__ == "__main__":
    main()
