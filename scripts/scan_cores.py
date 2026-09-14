#!/usr/bin/env python3
import os
import sys

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

TARGETS = [
    r"D:\David\Ipv7-4\core",
    r"D:\David\Ipv7-5\core",
    r"D:\David\Ipv7IEU\core",
    r"D:\David\asa-mirror\core",
    r"D:\David\dvd\ipv7c\core",
]

for t in TARGETS:
    if not os.path.exists(t):
        continue
    print(f"\n=======================================================")
    print(f"DIRECTORY: {t}")
    print(f"=======================================================")
    for root, dirs, files in os.walk(t):
        rel = os.path.relpath(root, t)
        py_go_rs = [f for f in files if f.endswith((".go", ".rs", ".py", ".js", ".ts", ".c", ".h"))]
        if py_go_rs:
            print(f"  [{rel}] {', '.join(py_go_rs[:8])}")
            if len(py_go_rs) > 8:
                print(f"       ... and {len(py_go_rs) - 8} more code files")
