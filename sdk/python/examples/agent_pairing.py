#!/usr/bin/env python3
"""
agent_pairing.py - Demostración de comunicación soberana entre dos Agentes de IA
usando el SDK oficial de ipvn7 con delegación UIN y verificación post-cuántica.
"""

import sys
import os

# Permitir ejecución directa sin instalación previa
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from ipvn7 import SovereignNode, BindingScope, IntentScope

def main():
    print("================================================================")
    print("  ipvn7 Network OS — Demostración de Comunicación entre Agentes ")
    print("================================================================")

    # 1. Agente A se conecta al nodo Desktop WSL2
    agent_a_endpoint = "http://172.22.72.89:7070"
    node_a = SovereignNode(agent_a_endpoint, scope=IntentScope.AI_AGENT)

    print(f"\n[1] Verificando salud del nodo de Agente A en {agent_a_endpoint}...")
    if not node_a.is_healthy():
        print("[-] Nodo de Agente A no responde. Verifique que ipvn7 esté activo.")
        return

    passport_a = node_a.get_passport()
    print(f"[+] Agente A autenticado con éxito:")
    print(f"    DID Raíz:       {passport_a.entity_did}")
    print(f"    Root ID:        {passport_a.root_id_hex[:16]}...{passport_a.root_id_hex[-16:]}")
    print(f"    Modo Cripto:    {passport_a.mode}")

    # 2. Agente A emite una delegación criptográfica UIN para una subtarea
    print("\n[2] Agente A emitiendo credencial BindingRecord (Scope: Agente IA)...")
    binding = node_a.issue_binding(scope=BindingScope.AI_AGENT, duration_days=7)
    print(f"[+] Credencial de Agente emitida:")
    print(f"    Key ID:         {binding.key_id_hex}")
    print(f"    Válido hasta:   {binding.valid_until}")
    print(f"    Firma Raíz L0:  {binding.sig_root_hex[:24]}...")

    # 3. Consultar Árbitro de Memoria Global
    arbiter = node_a.get_memory_arbiter_stats()
    print("\n[3] Verificando cuotas del Global Memory Arbiter (Anti-OOM DoS):")
    print(f"    Límite Total:   {arbiter.total_limit_bytes // (1024 * 1024)} MB")
    print(f"    RAM Asignada:   {arbiter.total_allocated_bytes} bytes")
    print(f"    Rechazos DoS:   {arbiter.total_rejections}")

    # 4. Enviar tarea segura hacia el nodo Notebook
    notebook_target = "http://192.168.1.106:8080"
    print(f"\n[4] Despachando tarea criptográfica determinista hacia {notebook_target}...")
    try:
        task_res = node_a.send_task(
            target_url=notebook_target,
            payload="SYN_AI_AGENT_AUTONOMOUS_EXCHANGE",
            task_type="PQC_AGENT_TASK"
        )
        print(f"[+] Tarea completada con éxito:")
        print(f"    Estado:         {task_res.status}")
        print(f"    Latencia RTT:   {task_res.round_trip_ms:.2f} ms")
        print(f"    ZTNA Firewall:  {task_res.ztna_decision}")
        print(f"    Tier Recíproco: {task_res.peer_tier}")
        print(f"    Hash PFO:       {task_res.task_proof_hash[:24]}...")
    except Exception as e:
        print(f"[!] Despacho remoto opcional omitido (Notebook fuera de rango o ocupado): {e}")

    print("\n================================================================")
    print("  Demostración completada exitosamente!")
    print("================================================================")

if __name__ == "__main__":
    main()
