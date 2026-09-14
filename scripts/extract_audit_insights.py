#!/usr/bin/env python3
import sys

def dump_section(filepath, start_str, max_lines=60):
    print(f"\n=======================================================")
    print(f"FILE: {filepath} | SEARCH: '{start_str}'")
    print(f"=======================================================")
    try:
        with open(filepath, "r", encoding="utf-8", errors="replace") as f:
            lines = f.readlines()
        
        found = False
        count = 0
        for i, line in enumerate(lines):
            if start_str.lower() in line.lower() or found:
                if not found:
                    found = True
                    print(f"[Line {i+1}]")
                print(line.rstrip())
                count += 1
                if count >= max_lines:
                    break
        if not found:
            print("Not found.")
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    dump_section(r"D:\David\dvd\ipv7c\Docs\AUDITORIA_MASIVA_IPV7C_Y_ANTIGUOS_2026_05_15.md", "Patrones positivos recuperables", 80)
    dump_section(r"D:\David\dvd\ipv7c\Docs\AUDITORIA_MASIVA_IPV7C_Y_ANTIGUOS_2026_05_15.md", "Patrones negativos a evitar", 60)
    dump_section(r"D:\David\dvd\IPv7-HRS_Arquitectura_Completa.md", "1. Visi", 40)
    dump_section(r"D:\David\Ipv7IEU\universality_phases.md", "Fase 1", 50)
