#!/usr/bin/env python3
import sys

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

def summarize_file(filepath, max_lines=45):
    print(f"\n=======================================================")
    print(f"FILE: {filepath}")
    print(f"=======================================================")
    try:
        with open(filepath, "r", encoding="utf-8", errors="replace") as f:
            lines = [f.readline().rstrip() for _ in range(max_lines)]
            for line in lines:
                print(line)
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    summarize_file(r"D:\David\Ipv7-4\core\bridge\socks.go")
    summarize_file(r"D:\David\Ipv7-4\core\overlay\nat_traversal.go")
    summarize_file(r"D:\David\Ipv7-4\core\bridge\mqtt_bridge.go")
    summarize_file(r"D:\David\Ipv7-4\core\bridge\coap_proxy.go")
    summarize_file(r"D:\David\asa-mirror\core\NexusGuardian.js")
    summarize_file(r"D:\David\asa-mirror\core\ErrorRecovery.js")
