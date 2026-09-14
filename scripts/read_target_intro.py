#!/usr/bin/env python3
import sys

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

with open(r"D:\David\dvd\ipv7-hrs-final\target.md", "r", encoding="utf-8", errors="replace") as f:
    for i in range(120):
        print(f.readline().rstrip())
