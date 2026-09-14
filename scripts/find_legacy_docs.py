#!/usr/bin/env python3
import os
import glob

TARGET_DIRS = [
    r"D:\David\Ipv7-4",
    r"D:\David\Ipv7-5",
    r"D:\David\Ipv7IEU",
    r"D:\David\asa-mirror",
    r"D:\David\DOCS",
    r"D:\David\dvd",
]

def find_docs():
    print("=== SEARCHING DOCUMENTATION AND KEY ARCHITECTURE FILES ===")
    found_files = []
    for root_dir in TARGET_DIRS:
        if not os.path.exists(root_dir):
            continue
        for root, dirs, files in os.walk(root_dir):
            # Skip noise dirs
            dirs[:] = [d for d in dirs if d not in [".git", ".gocache", "node_modules", "dist", "tmp", "logs"]]
            for file in files:
                if file.lower().endswith((".md", ".txt", ".html", ".rst", ".json")) and not file.startswith("."):
                    full_path = os.path.join(root, file)
                    rel_path = os.path.relpath(full_path, r"D:\David")
                    size = os.path.getsize(full_path)
                    found_files.append((rel_path, full_path, size))
    
    # Sort by size descending
    found_files.sort(key=lambda x: x[2], reverse=True)
    print(f"Total documentation files found: {len(found_files)}\n")
    for rel, full, size in found_files[:40]:
        print(f"[{size/1024:.1f} KB] {rel}")

if __name__ == "__main__":
    find_docs()
