#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""mutation_gate.py — 3-Mutant Test Verification Gate (Chống 'Test Phế' / Tautological Tests).

Usage:
  python3 tooling/mutation_gate.py --workspace <ws> --suffix <suffix> --mutations <mutations.json>

Exit codes:
  0 : PASS (Baseline PASS + All >=3 Mutants KILLED + 100% SHA-256 Restored)
  2 : FAIL (Baseline FAIL or >=1 Mutant SURVIVED/TIMEOUT or Restore Mismatch)
  3 : INVALID_INPUT (<3 mutants, invalid target_content, or SyntaxError mutant)
"""
from __future__ import annotations

import argparse
import ast
import datetime as dt
import hashlib
import json
import os
import signal
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_WS_ROOT = AGENT_ROOT / "memory" / "workspaces"
MIN_MUTANTS = 3


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _invalidate_pyc(fpath: Path) -> None:
    if fpath.suffix != ".py":
        return
    pycache = fpath.parent / "__pycache__"
    if pycache.is_dir():
        for pyc in pycache.glob(f"{fpath.stem}.*.pyc"):
            try:
                pyc.unlink()
            except OSError:
                pass


def resolve_target_file(raw_path: str, mutations_file: Path) -> Optional[Path]:
    p = Path(raw_path.strip())
    if p.is_absolute() and p.is_file():
        return p.resolve()
    for base in (Path.cwd(), AGENT_ROOT, mutations_file.resolve().parent):
        cand = (base / p).resolve()
        if cand.is_file():
            return cand
    return None


def auto_detect_component(fpath: Path) -> Tuple[Path, Optional[str]]:
    cur = fpath.resolve().parent
    while True:
        manifest = cur / ".agent-project.json"
        if manifest.is_file():
            try:
                cfg = json.loads(manifest.read_text(encoding="utf-8"))
                rel = str(fpath.resolve().relative_to(cur)).replace("\\", "/")
                hits: List[Dict[str, Any]] = []
                for comp in cfg.get("components", []):
                    cpath = comp.get("path", "").strip("/")
                    if cpath in ("", ".") or rel == cpath or rel.startswith(cpath + "/"):
                        hits.append(comp)
                if hits:
                    best = max(hits, key=lambda c: len(c.get("path", "").strip("/")))
                    comp_dir = (cur / best.get("path", "")).resolve()
                    test_cmd = best.get("test") or (best.get("checks") or [None])[0]
                    return comp_dir, test_cmd
                if cfg.get("test"):
                    return cur, cfg.get("test")
            except Exception:
                pass
            return cur, None
        if cur.parent == cur:
            break
        cur = cur.parent
    return fpath.resolve().parent, None


def run_test_cmd(cmd: str, cwd: Path, timeout_sec: int = 60) -> Dict[str, Any]:
    env = dict(os.environ)
    env["PYTHONDONTWRITEBYTECODE"] = "1"
    try:
        r = subprocess.run(
            cmd,
            shell=True,
            cwd=cwd,
            env=env,
            capture_output=True,
            text=True,
            timeout=timeout_sec,
        )
        return {
            "returncode": r.returncode,
            "output_tail": (r.stdout + r.stderr)[-800:],
            "timed_out": False,
        }
    except subprocess.TimeoutExpired as e:
        return {
            "returncode": -1,
            "output_tail": f"TimeoutExpired after {timeout_sec}s: {e}",
            "timed_out": True,
        }


def execute_mutation_gate(
    ws_name: str,
    suffix: str,
    mutations_file: Path,
    workspaces_dir: Path = DEFAULT_WS_ROOT,
    timeout_sec: int = 60,
) -> Tuple[int, Dict[str, Any]]:
    try:
        raw = json.loads(mutations_file.read_text(encoding="utf-8"))
    except Exception as e:
        return 3, {"gate": "INVALID_INPUT", "error": f"Không đọc được file JSON mutations: {e}"}

    mutants: List[Dict[str, Any]] = raw if isinstance(raw, list) else raw.get("mutants", [])
    if len(mutants) < MIN_MUTANTS:
        return 3, {
            "gate": "INVALID_INPUT",
            "error": f"Bắt buộc tối thiểu {MIN_MUTANTS} biến thể cố tình làm hỏng (nhận được {len(mutants)}).",
        }

    prepared: List[Dict[str, Any]] = []
    for idx, m in enumerate(mutants, start=1):
        mid = str(m.get("id") or f"M{idx}")
        raw_file = str(m.get("file") or "")
        fpath = resolve_target_file(raw_file, mutations_file) if raw_file else None
        t_content = str(m.get("target_content") or "")
        m_content = str(m.get("mutant_content") or "")
        if not fpath or not t_content or not m_content:
            return 3, {
                "gate": "INVALID_INPUT",
                "error": f"Mutant {mid}: thiếu hoặc không tìm thấy file/target_content/mutant_content.",
            }
        if t_content == m_content:
            return 3, {
                "gate": "INVALID_INPUT",
                "error": f"Mutant {mid}: mutant_content trùng hệt target_content.",
            }

        comp_cwd, comp_test = auto_detect_component(fpath)
        t_cmd = str(m.get("test_cmd") or comp_test or "").strip()
        cwd = Path(m["cwd"]).resolve() if m.get("cwd") else comp_cwd
        if not t_cmd:
            return 3, {
                "gate": "INVALID_INPUT",
                "error": f"Mutant {mid}: không có test_cmd và không tìm thấy 'test' trong .agent-project.json.",
            }

        text = fpath.read_text(encoding="utf-8")
        if text.count(t_content) != 1:
            return 3, {
                "gate": "INVALID_INPUT",
                "error": f"Mutant {mid}: target_content phải xuất hiện đúng 1 lần trong {fpath.name} (hiện tại: {text.count(t_content)}).",
            }

        mutated_text = text.replace(t_content, m_content, 1)
        if fpath.suffix == ".py":
            try:
                ast.parse(mutated_text, filename=str(fpath))
            except SyntaxError as se:
                return 3, {
                    "gate": "INVALID_INPUT",
                    "error": (
                        f"Mutant {mid}: biến thể làm hỏng gây lỗi cú pháp Python "
                        f"(SyntaxError: {se.msg} tại line {se.lineno}). Mutant phải là lỗi logic hợp lệ!"
                    ),
                }

        prepared.append(
            {
                "id": mid,
                "description": str(m.get("description") or ""),
                "fpath": fpath,
                "cwd": cwd,
                "test_cmd": t_cmd,
                "target_content": t_content,
                "mutant_content": m_content,
            }
        )

    # Bước 1: Chạy Baseline trên các lệnh test duy nhất
    baseline_results: Dict[str, Dict[str, Any]] = {}
    for item in prepared:
        key = f"{item['cwd']}::{item['test_cmd']}"
        if key not in baseline_results:
            res = run_test_cmd(item["test_cmd"], item["cwd"], timeout_sec)
            baseline_results[key] = res
            if res["returncode"] != 0 or res["timed_out"]:
                return 2, {
                    "gate": "FAIL",
                    "reason": f"Baseline test FAILED trước khi tiêm lỗi: {item['test_cmd']}",
                    "baseline": res,
                }

    # Bước 2: Chạy từng Mutant, bảo vệ SIGTERM và luôn khôi phục nguyên trạng trong khối finally
    results: List[Dict[str, Any]] = []
    survived_ids: List[str] = []
    timeout_ids: List[str] = []
    all_restored = True

    prev_sigterm = signal.getsignal(signal.SIGTERM)

    def _sigterm_to_exit(signum: int, frame: Any) -> None:
        raise SystemExit(128 + signum)

    signal.signal(signal.SIGTERM, _sigterm_to_exit)
    try:
        for item in prepared:
            mid = item["id"]
            fpath = item["fpath"]
            orig_bytes = fpath.read_bytes()
            orig_sha = sha256_bytes(orig_bytes)
            mutated_text = orig_bytes.decode("utf-8").replace(
                item["target_content"], item["mutant_content"], 1
            )

            try:
                fpath.write_bytes(mutated_text.encode("utf-8"))
                _invalidate_pyc(fpath)
                res = run_test_cmd(item["test_cmd"], item["cwd"], timeout_sec)
            finally:
                fpath.write_bytes(orig_bytes)
                _invalidate_pyc(fpath)
                restored_sha = sha256_bytes(fpath.read_bytes())

            sha_ok = restored_sha == orig_sha
            if not sha_ok:
                all_restored = False

            if res["timed_out"]:
                status = "TIMEOUT"
                timeout_ids.append(mid)
            elif res["returncode"] != 0:
                status = "KILLED"
            else:
                status = "SURVIVED"
                survived_ids.append(mid)

            results.append(
                {
                    "id": mid,
                    "description": item["description"],
                    "file": str(fpath),
                    "status": status,
                    "returncode": res["returncode"],
                    "timed_out": res["timed_out"],
                    "sha256_before": orig_sha,
                    "sha256_after": restored_sha,
                    "sha256_restored": sha_ok,
                    "output_tail": res["output_tail"],
                }
            )
    finally:
        signal.signal(signal.SIGTERM, prev_sigterm)

    killed_count = sum(1 for r in results if r["status"] == "KILLED")
    gate = (
        "PASS"
        if (
            killed_count == len(prepared)
            and not survived_ids
            and not timeout_ids
            and all_restored
        )
        else "FAIL"
    )
    ts = dt.datetime.now(dt.timezone(dt.timedelta(hours=7))).strftime("%Y%m%dT%H%M%S")
    ws_dir = workspaces_dir / ws_name
    ws_dir.mkdir(parents=True, exist_ok=True)
    summary_path = ws_dir / f"06_mutation_summary_{suffix}_{ts}.json"
    summary = {
        "workspace": ws_name,
        "suffix": suffix,
        "ts": ts,
        "total_mutants": len(prepared),
        "killed_count": killed_count,
        "survived_count": len(survived_ids),
        "survived_ids": survived_ids,
        "timeout_count": len(timeout_ids),
        "timeout_ids": timeout_ids,
        "all_restored_sha256": all_restored,
        "gate": gate,
        "mutants": results,
        "summary_file": str(summary_path),
    }
    summary_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    return (0 if gate == "PASS" else 2), summary


def parse_args(argv: Optional[Sequence[str]] = None) -> argparse.Namespace:
    ap = argparse.ArgumentParser(description="3-Mutant Test Verification Gate (mutation_gate.py)")
    ap.add_argument("--workspace", "-w", required=True, help="Tên thư mục workspace trong memory/workspaces/")
    ap.add_argument("--suffix", "-s", required=True, help="Suffix định danh của task hiện tại")
    ap.add_argument("--mutations", "-m", required=True, help="Đường dẫn tới file JSON chứa >=3 mutants")
    ap.add_argument("--workspaces-dir", default=str(DEFAULT_WS_ROOT), help="Thư mục gốc chứa workspaces")
    ap.add_argument("--timeout", type=int, default=60, help="Timeout (giây) cho mỗi lần chạy test")
    return ap.parse_args(argv)


def main(argv: Optional[Sequence[str]] = None) -> int:
    args = parse_args(argv)
    code, summary = execute_mutation_gate(
        ws_name=args.workspace,
        suffix=args.suffix,
        mutations_file=Path(args.mutations).resolve(),
        workspaces_dir=Path(args.workspaces_dir).resolve(),
        timeout_sec=args.timeout,
    )
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return code


if __name__ == "__main__":
    sys.exit(main())
