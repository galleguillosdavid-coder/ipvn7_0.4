import os
import sys

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

def search_keywords_in_file(filepath, keywords, max_matches=10):
    print(f"\n=======================================================")
    print(f"FILE: {filepath}")
    print(f"=======================================================")
    if not os.path.exists(filepath):
        print("File does not exist.")
        return
    
    with open(filepath, "r", encoding="utf-8", errors="replace") as f:
        lines = f.readlines()
    
    matches = 0
    for i, line in enumerate(lines):
        for kw in keywords:
            if kw.lower() in line.lower():
                print(f"[L{i+1}] {line.strip()[:140]}")
                matches += 1
                if matches >= max_matches:
                    return

if __name__ == "__main__":
    search_keywords_in_file(r"D:\David\dvd\ipv7-hrs-final\target.md", ["call-and-response", "silenciosa", "rendimiento", "wintun", "handshake", "discovery"])
    search_keywords_in_file(r"D:\David\asa-mirror\genesis.md", ["nexus", "herramientas", "conversacion", "cerebro"])
    search_keywords_in_file(r"D:\David\comparativa-ipv4-ipv6-ipv7.html", ["ventajas", "innovacion", "caracteristicas", "tabla"])
