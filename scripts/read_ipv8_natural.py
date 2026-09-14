#!/usr/bin/env python3
import sys

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

def read_file_preview(filepath, max_lines=70):
    print(f"\n=======================================================")
    print(f"FILE: {filepath}")
    print(f"=======================================================")
    try:
        with open(filepath, "r", encoding="utf-8", errors="replace") as f:
            for _ in range(max_lines):
                line = f.readline()
                if not line:
                    break
                print(line.rstrip())
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    read_file_preview(r"D:\David\dvd\Ipv8\ipv8_natural\01_ARQUITECTURA_NATURAL_DE_IPV8.md", 60)
    read_file_preview(r"D:\David\dvd\Ipv8\ipv8_natural\05_INNOVACIONES_ESTRATEGICAS_PARA_IPV8.md", 70)
