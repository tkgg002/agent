#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""scope.py — R0 SCOPE cho skill `diff-review` (0 LLM token, stdlib + git).

Usage:
  python3 tooling/diff_review/scope.py --repo <path> --base <ref> --head <ref> \
      [--manifest <path/.agent-project.json>] [--pack <path.md> ...] [--out scope.json]

Đọc cấu hình từ khoá `review` trong manifest (mọi trường optional). Không có manifest
-> chỉ tính name-status/numstat/loc, các cờ còn lại rỗng. Project-agnostic: không chứa
tên service/path/từ khoá dự án nào (ADR-07); mọi từ khoá đến từ manifest hoặc pack.

Exit: 0 OK | 2 git error | 3 invalid input.
"""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional

DEFAULT_GOD_FILE_LOC = 600
IMPORT_RX = {
    ".go": re.compile(r'^\s*"([^"]+)"\s*$|^\s*import\s+"([^"]+)"'),
    ".ts": re.compile(r'^\s*import\s.*?from\s+["\']([^"\']+)["\']'),
    ".tsx": re.compile(r'^\s*import\s.*?from\s+["\']([^"\']+)["\']'),
    ".js": re.compile(r'^\s*import\s.*?from\s+["\']([^"\']+)["\']|require\(["\']([^"\']+)["\']\)'),
    ".py": re.compile(r'^\s*(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))'),
}


def die(msg: str, code: int) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(code)


def git(repo: str, *args: str) -> str:
    try:
        return subprocess.run(
            ["git", "-C", repo, *args], check=True, capture_output=True, text=True
        ).stdout
    except FileNotFoundError:
        die("git not found", 2)
    except subprocess.CalledProcessError as e:
        die(f"git {' '.join(args)}: {e.stderr.strip()}", 2)
    return ""


def load_review_cfg(manifest: Optional[str]) -> Dict:
    if not manifest:
        return {}
    p = Path(manifest)
    if not p.is_file():
        die(f"manifest not found: {manifest}", 3)
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except json.JSONDecodeError as e:
        die(f"manifest invalid JSON: {e}", 3)
    cfg = data.get("review", {})
    return cfg if isinstance(cfg, dict) else {}


def load_pack_keywords(packs: List[str]) -> List[str]:
    """Lấy dòng keyword trong mục '## 0. Money-path keywords' của pack (code block đầu tiên)."""
    kws: List[str] = []
    for path in packs:
        p = Path(path)
        if not p.is_file():
            die(f"pack not found: {path}", 3)
        text = p.read_text(encoding="utf-8")
        m = re.search(r"Money-path keywords.*?```\s*(.*?)```", text, re.S | re.I)
        if m:
            kws += [k.strip() for k in m.group(1).replace("\n", ",").split(",") if k.strip()]
    return kws


def parse_name_status(out: str) -> List[Dict]:
    files = []
    for line in out.splitlines():
        if not line.strip():
            continue
        parts = line.split("\t")
        status = parts[0][0]
        path = parts[-1]
        old = parts[1] if status in ("R", "C") and len(parts) >= 3 else None
        files.append({"path": path, "status": status, "old_path": old})
    return files


def parse_numstat(out: str) -> Dict[str, Dict[str, int]]:
    res = {}
    for line in out.splitlines():
        parts = line.split("\t")
        if len(parts) < 3:
            continue
        a, d, path = parts[0], parts[1], parts[-1]
        if "=>" in path:
            m = re.match(r".*\{.* => (.*)\}(.*)|.* => (.*)", path)
            path = (m.group(1) or "") + (m.group(2) or "") if m and m.group(1) is not None else (m.group(3) if m else path)
        res[path] = {
            "added": int(a) if a.isdigit() else 0,
            "deleted": int(d) if d.isdigit() else 0,
            "binary": not a.isdigit(),
        }
    return res


def match_layer(path: str, layers: Dict[str, str]) -> Optional[str]:
    best, best_len = None, -1
    for name, prefix in layers.items():
        if path.startswith(prefix) and len(prefix) > best_len:
            best, best_len = name, len(prefix)
    return best


def count_lines(repo: str, ref: str, path: str) -> int:
    try:
        out = subprocess.run(
            ["git", "-C", repo, "show", f"{ref}:{path}"], check=True, capture_output=True, text=True, errors="replace"
        ).stdout
    except subprocess.CalledProcessError:
        return 0
    return out.count("\n") + (1 if out and not out.endswith("\n") else 0)


def added_hunk_text(repo: str, base: str, head: str, path: str) -> str:
    out = subprocess.run(
        ["git", "-C", repo, "diff", "-U0", f"{base}..{head}", "--", path],
        capture_output=True, text=True, errors="replace",
    ).stdout
    return "\n".join(l[1:] for l in out.splitlines() if l.startswith("+") and not l.startswith("+++"))


def layer_violations(repo: str, head: str, files: List[Dict], order: List[str], layers: Dict[str, str]) -> List[Dict]:
    """Import từ tầng trong ra tầng ngoài (index nhỏ hơn trong `order`) = vi phạm."""
    if not order or not layers:
        return []
    rank = {name: i for i, name in enumerate(order)}
    out = []
    for f in files:
        if f["status"] == "D" or not f.get("layer") or f["layer"] not in rank:
            continue
        ext = os.path.splitext(f["path"])[1]
        rx = IMPORT_RX.get(ext)
        if not rx:
            continue
        try:
            src = subprocess.run(
                ["git", "-C", repo, "show", f"{head}:{f['path']}"], check=True, capture_output=True, text=True, errors="replace"
            ).stdout
        except subprocess.CalledProcessError:
            continue
        for ln, line in enumerate(src.splitlines(), 1):
            m = rx.search(line)
            if not m:
                continue
            target = next((g for g in m.groups() if g), "")
            for lname, prefix in layers.items():
                norm = prefix.strip("/").replace("/", ".") if ext == ".py" else prefix.strip("/")
                if norm and norm in target and lname in rank and rank[lname] < rank[f["layer"]]:
                    out.append({"file": f["path"], "line": ln, "from_layer": f["layer"], "imports_layer": lname, "import": target})
    return out


def main() -> None:
    ap = argparse.ArgumentParser(description="diff-review R0 scope (0 LLM token)")
    ap.add_argument("--repo", required=True)
    ap.add_argument("--base", required=True)
    ap.add_argument("--head", required=True)
    ap.add_argument("--manifest")
    ap.add_argument("--pack", action="append", default=[])
    ap.add_argument("--out")
    a = ap.parse_args()

    if not Path(a.repo).is_dir():
        die(f"repo not a directory: {a.repo}", 3)
    git(a.repo, "rev-parse", "--is-inside-work-tree")

    cfg = load_review_cfg(a.manifest)
    layers: Dict[str, str] = cfg.get("layers", {}) or {}
    order: List[str] = cfg.get("layer_order", []) or []
    god_loc = int(cfg.get("god_file_loc", DEFAULT_GOD_FILE_LOC))
    chokepoints: List[Dict] = cfg.get("chokepoints", []) or []
    forbidden: List[str] = cfg.get("forbidden_ifelse_files", []) or []
    keywords: List[str] = list(cfg.get("money_path_keywords", []) or []) + load_pack_keywords(a.pack)
    money_rx = re.compile("|".join(rf"\b{re.escape(k)}\b" for k in keywords), re.I) if keywords else None

    rng = f"{a.base}..{a.head}"
    files = parse_name_status(git(a.repo, "diff", "--name-status", rng))
    stats = parse_numstat(git(a.repo, "diff", "--numstat", rng))

    god_files, money_files, hunks = [], [], {}
    for f in files:
        st = stats.get(f["path"], {"added": 0, "deleted": 0, "binary": False})
        f.update(st)
        f["layer"] = match_layer(f["path"], layers)
        f["loc_after"] = 0 if f["status"] == "D" else count_lines(a.repo, a.head, f["path"])
        f["god_file"] = f["loc_after"] > god_loc and f["added"] > 0
        f["forbidden_ifelse"] = f["path"] in forbidden
        text = "" if (f["status"] == "D" or f["binary"]) else added_hunk_text(a.repo, a.base, a.head, f["path"])
        hunks[f["path"]] = text
        f["money_path"] = bool(money_rx and money_rx.search(text))
        if f["god_file"]:
            god_files.append({"path": f["path"], "loc_after": f["loc_after"], "added": f["added"]})
        if f["money_path"]:
            money_files.append(f["path"])

    changed = {f["path"] for f in files}
    cps_touched = []
    for cp in chokepoints:
        by_path = [p for p in cp.get("paths", []) if p in changed]
        by_sym = [s for s in cp.get("symbols", []) if any(s in t for t in hunks.values())]
        if by_path or by_sym:
            cps_touched.append({"id": cp.get("id"), "paths": by_path, "symbols": by_sym})

    result = {
        "repo": os.path.abspath(a.repo),
        "base": a.base, "head": a.head,
        "base_sha": git(a.repo, "rev-parse", "--short", a.base).strip(),
        "head_sha": git(a.repo, "rev-parse", "--short", a.head).strip(),
        "manifest": os.path.abspath(a.manifest) if a.manifest else None,
        "packs": [os.path.abspath(p) for p in a.pack],
        "config": {"god_file_loc": god_loc, "layer_order": order, "layers": layers, "keywords_count": len(keywords)},
        "summary": {
            "files": len(files),
            "added_files": sum(1 for f in files if f["status"] == "A"),
            "deleted_files": sum(1 for f in files if f["status"] == "D"),
            "loc_added": sum(f["added"] for f in files),
            "loc_deleted": sum(f["deleted"] for f in files),
            "layers": sorted({f["layer"] for f in files if f["layer"]}),
        },
        "files": files,
        "god_files": god_files,
        "money_path": money_files,
        "chokepoints_touched": cps_touched,
        "layer_violations": layer_violations(a.repo, a.head, files, order, layers),
        "forbidden_ifelse_touched": [f["path"] for f in files if f["forbidden_ifelse"]],
    }
    text = json.dumps(result, ensure_ascii=False, indent=2)
    if a.out:
        Path(a.out).parent.mkdir(parents=True, exist_ok=True)
        Path(a.out).write_text(text + "\n", encoding="utf-8")
        print(f"SCOPE written: {a.out} files={len(files)} cps={len(cps_touched)} god={len(god_files)} money={len(money_files)} viol={len(result['layer_violations'])}")
    else:
        print(text)


if __name__ == "__main__":
    main()
