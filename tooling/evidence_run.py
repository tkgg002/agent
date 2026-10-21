#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""evidence_run.py — Biên lai chạy lệnh thật (chống "test láo").

Usage:
  python3 tooling/evidence_run.py --workspace <ws> --kind unit|integration|runtime [--cwd DIR] -- <cmd...>

Ghi memory/workspaces/<ws>/06_evidence/EV-<YYYYmmddTHHMMSS>-<hex4>.json và in "EVIDENCE <id> exit=<n>".
Exit code = exit code của lệnh; 3 = INVALID_INPUT.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import secrets
import shlex
import subprocess
import sys
from pathlib import Path
from typing import List, Optional

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_WS_ROOT = AGENT_ROOT / "memory" / "workspaces"
KINDS = ("unit", "integration", "runtime")
SRC_SUFFIXES = (".py", ".go", ".ts", ".tsx", ".js", ".jsx", ".rs", ".java", ".kt", ".sql", ".sh", ".json", ".toml", ".yaml", ".yml")
# Lệnh test runner không được khai báo là bằng chứng runtime.
TEST_RUNNER_RE = re.compile(
    r"(\bunittest\b|\bpytest\b|\bgo\s+test\b|\bjest\b|\bvitest\b|\bmocha\b|\bnpm\s+(run\s+)?test\b|"
    r"\byarn\s+test\b|\bmvn\s+test\b|\bgradle\w*\s+test\b|\bcargo\s+test\b|mutation_gate\.py)",
    re.I,
)
STDOUT_TAIL = 6000


def _git(repo: Path, *args: str) -> Optional[str]:
    try:
        r = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True, timeout=20)
    except (OSError, subprocess.TimeoutExpired):
        return None
    return r.stdout if r.returncode == 0 else None


def source_fingerprint(cwd: Path) -> str:
    """sha256 nội dung file source của repo chứa cwd (tracked + untracked, bỏ memory/). Commit không đổi giá trị."""
    root_out = _git(cwd, "rev-parse", "--show-toplevel")
    root = Path(root_out.strip()) if root_out else cwd.resolve()
    listing = _git(root, "ls-files", "-co", "--exclude-standard")
    files = sorted(
        f for f in (listing.splitlines() if listing is not None else [])
        if f.endswith(SRC_SUFFIXES) and not f.startswith("memory/") and "/__pycache__/" not in f
    )
    h = hashlib.sha256()
    for rel in files:
        p = root / rel
        try:
            data = p.read_bytes()
        except OSError:
            continue
        h.update(rel.encode("utf-8") + b"\0" + hashlib.sha256(data).digest())
    return h.hexdigest()


def run(ws: str, kind: str, cmd: List[str], cwd: Path, ws_root: Path = DEFAULT_WS_ROOT) -> int:
    cmd_str = " ".join(cmd) if len(cmd) > 1 else (cmd[0] if cmd else "")
    if kind not in KINDS or not cmd_str.strip():
        print("INVALID_INPUT: cần --kind unit|integration|runtime và lệnh sau --", file=sys.stderr)
        return 3
    if kind == "runtime" and TEST_RUNNER_RE.search(cmd_str):
        print("INVALID_INPUT: lệnh test runner không phải bằng chứng runtime — dùng --kind unit/integration", file=sys.stderr)
        return 3
    ws_dir = ws_root / ws
    if not ws_dir.is_dir():
        print(f"INVALID_INPUT: workspace không tồn tại: {ws_dir}", file=sys.stderr)
        return 3
    started = dt.datetime.now().astimezone()
    proc = subprocess.run(cmd_str, shell=True, cwd=str(cwd), capture_output=True, text=True)
    out = (proc.stdout or "") + (proc.stderr or "")
    ev_id = f"EV-{started.strftime('%Y%m%dT%H%M%S')}-{secrets.token_hex(2)}"
    record = {
        "id": ev_id,
        "cmd": cmd_str,
        "cwd": str(cwd.resolve()),
        "kind": kind,
        "exit": proc.returncode,
        "started_at": started.isoformat(timespec="seconds"),
        "stdout_tail": out[-STDOUT_TAIL:],
        "stdout_sha256": hashlib.sha256(out.encode("utf-8")).hexdigest(),
        "src_fingerprint": source_fingerprint(cwd),
    }
    ev_dir = ws_dir / "06_evidence"
    ev_dir.mkdir(parents=True, exist_ok=True)
    (ev_dir / f"{ev_id}.json").write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    sys.stdout.write(out[-2000:])
    print(f"\nEVIDENCE {ev_id} exit={proc.returncode}")
    return proc.returncode


def main(argv: Optional[List[str]] = None) -> int:
    argv = list(sys.argv[1:] if argv is None else argv)
    if "--" not in argv:
        print("INVALID_INPUT: thiếu '--' trước lệnh", file=sys.stderr)
        return 3
    i = argv.index("--")
    ap = argparse.ArgumentParser()
    ap.add_argument("--workspace", required=True)
    ap.add_argument("--kind", required=True, choices=KINDS)
    ap.add_argument("--cwd", default=os.getcwd())
    ap.add_argument("--ws-root", default=str(DEFAULT_WS_ROOT))
    a = ap.parse_args(argv[:i])
    return run(a.workspace, a.kind, argv[i + 1 :], Path(a.cwd), Path(a.ws_root))


if __name__ == "__main__":
    sys.exit(main())
