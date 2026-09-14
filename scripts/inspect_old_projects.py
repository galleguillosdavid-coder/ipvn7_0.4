#!/usr/bin/env python3
import os
import sys

BASE_DIR = r"D:\David"

def scan_dir():
    print(f"Scanning {BASE_DIR}...")
    if not os.path.exists(BASE_DIR):
        print(f"Path does not exist: {BASE_DIR}")
        return

    items = os.listdir(BASE_DIR)
    for item in sorted(items):
        full_path = os.path.join(BASE_DIR, item)
        if os.path.isdir(full_path):
            sub_items = []
            try:
                for sub in os.listdir(full_path):
                    if not sub.startswith(".") and sub not in ["node_modules", "vendor", "__pycache__"]:
                        sub_items.append(sub)
            except Exception as e:
                sub_items = [f"Error: {e}"]
            print(f"\n[DIR] {item}/")
            print(f"      Contains: {', '.join(sub_items[:15])}")
            if len(sub_items) > 15:
                print(f"      ... and {len(sub_items) - 15} more items")
        else:
            size_kb = os.path.getsize(full_path) / 1024.0
            print(f"[FILE] {item} ({size_kb:.1f} KB)")

if __name__ == "__main__":
    scan_dir()
