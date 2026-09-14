#!/usr/bin/env python3
"""
deepseek_worker.py - Micro-Worker para tareas atómicas y acotadas
Ecosistema AFE-Kùzu (SKILL 7.0)
"""

import os
import sys
import json
import urllib.request
import urllib.error
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent
ENV_FILE = BASE_DIR / ".env"


def load_env():
    env_vars = {}
    if ENV_FILE.exists():
        with open(ENV_FILE, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    k, v = line.split("=", 1)
                    env_vars[k.strip()] = v.strip()
    return env_vars


def call_deepseek(prompt: str, snippet: str = "") -> str:
    env = load_env()
    api_key = os.environ.get("DEEPSEEK_API_KEY") or env.get("DEEPSEEK_API_KEY", "")
    base_url = os.environ.get("DEEPSEEK_BASE_URL") or env.get("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
    model = os.environ.get("DEEPSEEK_MODEL") or env.get("DEEPSEEK_MODEL", "deepseek-chat")

    if not api_key or "your_" in api_key:
        return "[LOCAL-FALLBACK] No API key configurada. Microtarea procesada localmente."

    system_prompt = (
        "Eres un micro-worker del sistema AFE-Kùzu (ipvn7). "
        "Tu objetivo es resolver tareas atómicas, de micro-alcance, con código limpio y sin contexto macro."
    )

    user_content = prompt
    if snippet:
        user_content += f"\n\n```snippet\n{snippet}\n```"

    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_content}
        ],
        "temperature": 0.2,
        "max_tokens": 1024
    }

    req = urllib.request.Request(
        f"{base_url}/chat/completions",
        data=json.dumps(payload).encode("utf-8"),
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {api_key}"
        },
        method="POST"
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return data["choices"][0]["message"]["content"]
    except urllib.error.HTTPError as e:
        err_msg = e.read().decode("utf-8")
        return f"[ERROR HTTP {e.code}] {err_msg}"
    except Exception as e:
        return f"[ERROR DE CONEXIÓN] {e}"


def main():
    if len(sys.argv) < 2:
        print("Uso: python3 deepseek_worker.py '<instrucción_atómica>' ['<snippet_código>']")
        sys.exit(1)

    prompt = ""
    snippet = ""

    if "--prompt" in sys.argv:
        idx = sys.argv.index("--prompt")
        if idx + 1 < len(sys.argv):
            prompt = sys.argv[idx + 1]
    else:
        prompt = sys.argv[1]
        if len(sys.argv) > 2:
            snippet = sys.argv[2]

    if "--snippet" in sys.argv:
        idx = sys.argv.index("--snippet")
        if idx + 1 < len(sys.argv):
            snippet = sys.argv[idx + 1]

    result = call_deepseek(prompt, snippet)
    print(result)


if __name__ == "__main__":
    main()

