#!/usr/bin/env python3
"""
kuzu_pfo_manager.py - Gestor del Grafo PFO (Principio - Función - Observación)
Ecosistema AFE-Kùzu (SKILL 7.0) para ipvn7 Network OS
"""

import os
import sys
import argparse
from pathlib import Path

try:
    import kuzu
except ImportError:
    print("[ERROR] Kùzu DB no está instalado. Ejecute: pip install kuzu", file=sys.stderr)
    sys.exit(1)

BASE_DIR = Path(__file__).resolve().parent.parent
DEFAULT_DB_PATH = str(BASE_DIR / ".kuzu_db" / "pfo_db")

PRINCIPLES = [
    {
        "name": "Zero-PII",
        "axiom_rule": "Axiom 1",
        "description": "Estrictamente prohibido registrar o persistir IPs reales, logs de usuario o metadatos de identidad."
    },
    {
        "name": "Sustainable-Flow",
        "axiom_rule": "Axiom 2",
        "description": "Tasa cooperativa adaptada a la capacidad real sostenida de la ruta, eliminando latencia por colas saturadas."
    },
    {
        "name": "Strict-Core-Freeze",
        "axiom_rule": "Axiom 3",
        "description": "El núcleo fundacional (crypto, wire CBOR, Noise XX) es inmutable; expansiones nacen como adaptadores periféricos."
    },
    {
        "name": "Fast-Path-First",
        "axiom_rule": "Axiom 4",
        "description": "Prioridad al descarte y filtrado en eBPF/XDP y estructuras zero-copy; nunca en el camino crítico del espacio de usuario."
    },
    {
        "name": "Living-Lab",
        "axiom_rule": "Axiom 5",
        "description": "Cultura de evidencia empírica rigurosa: Hecho, Tecnología, Inferencia, Hipótesis y Experimento."
    }
]

INITIAL_FUNCTIONS = [
    ("L0_Identity_Ed25519", "L0", "ACTIVE", "pkg/l0/identity.go", "Strict-Core-Freeze"),
    ("L0_Wire_CBOR", "L0", "ACTIVE", "pkg/l0/wire.go", "Strict-Core-Freeze"),
    ("L0_Crypto_NoiseXX", "L0", "ACTIVE", "pkg/l0/crypto.go", "Strict-Core-Freeze"),
    ("L1_Routing_Kleinberg", "L1", "ACTIVE", "pkg/l1/routing.go", "Sustainable-Flow"),
    ("L1_Tun_Adapter", "L1", "ACTIVE", "pkg/l1/tun_adapter.go", "Fast-Path-First"),
    ("L2_Telemetry_LockFree", "L2", "ACTIVE", "pkg/l2/telemetry.go", "Living-Lab"),
    ("L3_MCP_Server", "L3", "ACTIVE", "pkg/l3/mcp_server.go", "Living-Lab"),
    ("L3_Kuzu_Bridge", "L3", "ACTIVE", "pkg/l3/kuzu_bridge.go", "Living-Lab"),
    ("L4_CLI_Operator", "L4", "ACTIVE", "cmd/ipvn7-cli/main.go", "Zero-PII"),
    ("L4_Daemon_Supervisor", "L4", "ACTIVE", "cmd/ipvn7/main.go", "Sustainable-Flow")
]


def get_conn(db_path=None):
    if db_path is None:
        db_path = DEFAULT_DB_PATH
    parent_dir = os.path.dirname(db_path)
    if parent_dir:
        os.makedirs(parent_dir, exist_ok=True)
    db = kuzu.Database(db_path)
    return kuzu.Connection(db)


def init_db(conn):
    print("[*] Inicializando esquema de grafo PFO en Kùzu...")

    tables = [
        "CREATE NODE TABLE Principle(name STRING, axiom_rule STRING, description STRING, PRIMARY KEY (name))",
        "CREATE NODE TABLE Function(name STRING, layer STRING, status STRING, filepath STRING, PRIMARY KEY (name))",
        "CREATE NODE TABLE Observation(id STRING, kind STRING, details STRING, PRIMARY KEY (id))",
        "CREATE NODE TABLE Decision(id STRING, title STRING, justification STRING, timestamp STRING, PRIMARY KEY (id))",
        "CREATE REL TABLE IMPLEMENTS(FROM Function TO Principle)",
        "CREATE REL TABLE OBSERVES(FROM Observation TO Function)",
        "CREATE REL TABLE CONSTRAINS(FROM Principle TO Decision)"
    ]

    for stmt in tables:
        try:
            conn.execute(stmt)
        except Exception:
            pass

    # Insert principles
    for p in PRINCIPLES:
        try:
            conn.execute(
                f"CREATE (p:Principle {{name: '{p['name']}', axiom_rule: '{p['axiom_rule']}', description: '{p['description']}'}})"
            )
        except Exception:
            pass

    # Insert initial functions and link to principles
    for name, layer, status, filepath, principle in INITIAL_FUNCTIONS:
        try:
            conn.execute(
                f"CREATE (f:Function {{name: '{name}', layer: '{layer}', status: '{status}', filepath: '{filepath}'}})"
            )
        except Exception:
            pass

        try:
            conn.execute(
                f"MATCH (f:Function), (p:Principle) "
                f"WHERE f.name = '{name}' AND p.name = '{principle}' "
                f"CREATE (f)-[:IMPLEMENTS]->(p)"
            )
        except Exception:
            pass

    print("[+] Esquema inicializado y principios fundacionales cargados exitosamente.")


def verify_db(conn):
    print("=== ESTADO DEL GRAFO PFO KÙZU ===")
    try:
        res_p = conn.execute("MATCH (p:Principle) RETURN count(p) AS count").get_next()
        print(f"[*] Principios Axiomáticos registrados: {res_p[0]}")
    except Exception as e:
        print(f"[!] Error consultando Principios: {e}")
        return

    res_f = conn.execute("MATCH (f:Function) RETURN count(f) AS count").get_next()
    print(f"[*] Funciones registradas: {res_f[0]}")

    res_r = conn.execute("MATCH (f:Function)-[:IMPLEMENTS]->(p:Principle) RETURN f.name, f.layer, p.name")
    print("\n--- Mapeo de Funciones a Principios (PFO) ---")
    while res_r.has_next():
        row = res_r.get_next()
        print(f"  [{row[1]}] {row[0]} -> Axioma: {row[2]}")


def run_cypher(conn, query):
    print(f"[*] Ejecutando Cypher: {query}")
    res = conn.execute(query)
    results = res if isinstance(res, list) else [res]
    for r in results:
        while hasattr(r, 'has_next') and r.has_next():
            print(r.get_next())


def main():
    parser = argparse.ArgumentParser(description="AFE-Kùzu PFO Manager")
    parser.add_argument("--init", action="store_true", help="Inicializa el grafo y tablas PFO")
    parser.add_argument("--verify", action="store_true", help="Verifica el estado del grafo")
    parser.add_argument("--cypher", type=str, help="Ejecuta una consulta openCypher")
    parser.add_argument("--db-path", type=str, default=DEFAULT_DB_PATH, help="Ruta de persistencia Kùzu")

    args = parser.parse_args()
    conn = get_conn(args.db_path)

    if args.init:
        init_db(conn)

    if args.verify or not args.init:
        verify_db(conn)

    if args.cypher:
        run_cypher(conn, args.cypher)


if __name__ == "__main__":
    main()
