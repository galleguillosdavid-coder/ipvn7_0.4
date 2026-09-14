#!/usr/bin/env python3
import os

FILES_TO_EXAMINE = [
    r"D:\David\dvd\IPv7-HRS_Arquitectura_Completa.md",
    r"D:\David\dvd\gen ip7.md",
    r"D:\David\Ipv7IEU\universality_phases.md",
    r"D:\David\asa-mirror\genesis.md",
    r"D:\David\DOCS\observaciones del propio nexus.txt",
    r"D:\David\dvd\ipv7c\Docs\AUDITORIA_MASIVA_IPV7C_Y_ANTIGUOS_2026_05_15.md",
    r"D:\David\dvd\ipv7-hrs-final\target.md",
]

def examine_files():
    for f in FILES_TO_EXAMINE:
        if not os.path.exists(f):
            print(f"[MISSING] {f}")
            continue
        print(f"\n=======================================================")
        print(f"FILE: {f} ({os.path.getsize(f)/1024:.1f} KB)")
        print(f"=======================================================")
        try:
            with open(f, "r", encoding="utf-8", errors="replace") as fp:
                lines = [line.strip() for line in fp.readlines() if line.strip()]
                # Print headers (starting with #)
                headers = [l for l in lines if l.startswith("#")]
                print(f"Total lines: {len(lines)} | Total headers: {len(headers)}")
                print("First 15 lines/headers:")
                for h in (headers[:15] if headers else lines[:15]):
                    print(f"  {h}")
        except Exception as e:
            print(f"Error reading {f}: {e}")

if __name__ == "__main__":
    examine_files()
