#!/usr/bin/env python3
"""Smoke-test the local Muscle LLM through Ollama's OpenAI-compatible endpoint.

Usage: python3 local_llm_smoke.py [--model qwen35-muscle] [--base-url http://127.0.0.1:11434]
Exit 0 only if: /v1/models lists the model, a chat completion returns text, and a tool call is emitted.
Project-agnostic: no project names, ports come from arguments (ADR-07).
"""
from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.request


def _get(url: str, timeout: float = 10.0) -> dict:
    with urllib.request.urlopen(url, timeout=timeout) as r:
        return json.loads(r.read().decode("utf-8"))


def _post(url: str, payload: dict, timeout: float = 900.0) -> dict:
    # First call after (re)load pays MLX warm-up (~200s on Apple M4 for 9B); keep timeout generous.
    req = urllib.request.Request(url, data=json.dumps(payload).encode("utf-8"),
                                 headers={"Content-Type": "application/json"}, method="POST")
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode("utf-8"))


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="qwen35-muscle")
    ap.add_argument("--base-url", default="http://127.0.0.1:11434")
    a = ap.parse_args()
    base = a.base_url.rstrip("/") + "/v1"

    models = {m["id"] for m in _get(base + "/models").get("data", [])}
    # Ollama lists untagged models as "<name>:latest"
    if a.model not in models and f"{a.model}:latest" not in models:
        print(f"FAIL: model {a.model!r} not in /v1/models: {sorted(models)}")
        return 2
    print(f"OK models: {a.model} present")

    # Qwen3.5 keeps thinking ON through the OpenAI-compatible /v1 route (content stays empty when
    # max_tokens is small); Ollama's native /api/chat honours "think": false. Use native for chat.
    native = a.base_url.rstrip("/") + "/api/chat"
    t0 = time.perf_counter()
    chat = _post(native, {
        "model": a.model, "stream": False, "think": False,
        "messages": [{"role": "user", "content": "Trả lời đúng 1 từ: thủ đô của Việt Nam?"}],
        "options": {"num_predict": 32},
    })
    text = (chat["message"].get("content") or "").strip()
    print(f"OK chat/api think=false ({time.perf_counter() - t0:.1f}s): {text[:60]!r}")
    if not text:
        print("FAIL: empty completion")
        return 3

    tools = [{"type": "function", "function": {
        "name": "verdict", "description": "Classify a claim against evidence",
        "parameters": {"type": "object", "properties": {
            "verdict": {"type": "string", "enum": ["CORRECT", "INCORRECT", "UNSUPPORTED"]},
            "reason": {"type": "string"}}, "required": ["verdict", "reason"]}}}]
    t1 = time.perf_counter()
    tc = _post(native, {
        "model": a.model, "stream": False, "think": False, "tools": tools,
        "options": {"num_predict": 128},
        "messages": [{"role": "user", "content":
                      "Claim: X defaults to 60.\nEvidence: X = int(os.getenv('X', '6'))\nCall verdict."}],
    })
    calls = tc["message"].get("tool_calls") or []
    print(f"OK tool-call ({time.perf_counter() - t1:.1f}s): {json.dumps(calls)[:160]}")
    if not calls:
        print("FAIL: no tool_calls emitted")
        return 4
    args = calls[0]["function"]["arguments"]
    if isinstance(args, str):
        args = json.loads(args)
    print(f"verdict={args.get('verdict')} expected=INCORRECT -> {'MATCH' if args.get('verdict') == 'INCORRECT' else 'MISMATCH (advisory)'}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
