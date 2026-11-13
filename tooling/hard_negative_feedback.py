#!/usr/bin/env python3
"""hard_negative_feedback.py — close the flywheel: a runtime FAIL turns the gate's ACCEPTed claims
into INCORRECT training examples for the VegaML sidecar.

Called by evidence_run.py when the evidenced command exits non-zero. Fail-soft: never raises,
never changes the caller's exit code, never writes the sidecar's jsonl directly (the sidecar owns it).

Project-agnostic (ADR-07): the endpoint comes from VEGA_FEEDBACK_URL.
"""
from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional

STAGE = "HARD_NEGATIVE_RUNTIME_FAIL"
FEEDBACK_URL = os.getenv("VEGA_FEEDBACK_URL", "http://127.0.0.1:8222/v1/systemone/feedback")


def latest_summary(ws_dir: Path) -> Optional[Path]:
    files = sorted(ws_dir.glob("06_laya_summary_dg*_*.json"), key=lambda p: p.stat().st_mtime)
    return files[-1] if files else None


def collect_hard_negatives(ws_dir: Path, ev_id: str) -> List[Dict[str, Any]]:
    """ACCEPTed, non-control claims of the latest gate run become INCORRECT examples."""
    summ = latest_summary(ws_dir)
    if not summ:
        return []
    try:
        data = json.loads(summ.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return []
    evid: Dict[str, Dict[str, Any]] = {}
    raw = Path(str(data.get("raw_file") or ""))
    if data.get("raw_file") and raw.is_file():
        try:
            for r in json.loads(raw.read_text(encoding="utf-8")):
                if isinstance(r, dict):
                    evid[str(r.get("id"))] = r
        except (OSError, json.JSONDecodeError):
            pass
    out: List[Dict[str, Any]] = []
    for c in data.get("claims", []):
        if not isinstance(c, dict):
            continue
        if c.get("decision") != "ACCEPT" or str(c.get("kind", "")).startswith("control_"):
            continue
        r = evid.get(str(c.get("id")), {})
        out.append({
            "task_id": f"{ev_id}:{c.get('id')}",
            "stage": STAGE,
            "state": {"claim": c.get("claim", ""), "evidence": r.get("evidence", ""), "flow": r.get("flow", "")},
            "labels": {"verdict": "INCORRECT"},
            "auto_fit": False,
        })
    return out


def post_hard_negatives(items: List[Dict[str, Any]], url: str = FEEDBACK_URL, timeout: float = 20.0) -> Dict[str, Any]:
    sent, err = 0, ""
    for it in items:
        req = urllib.request.Request(url, data=json.dumps(it, ensure_ascii=False).encode("utf-8"),
                                     headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(req, timeout=timeout):
                sent += 1
        except (urllib.error.URLError, TimeoutError, OSError) as e:
            err = str(e)
            break
    return {"stage": STAGE, "url": url, "candidates": len(items), "sent": sent, "error": err}


def feed_runtime_fail(ws_dir: Path, ev_id: str, url: str = FEEDBACK_URL) -> Dict[str, Any]:
    """One call for evidence_run.py: collect + post; always returns a dict, never raises."""
    try:
        items = collect_hard_negatives(ws_dir, ev_id)
        if not items:
            return {"stage": STAGE, "url": url, "candidates": 0, "sent": 0, "error": ""}
        return post_hard_negatives(items, url)
    except Exception as e:  # fail-soft by contract
        return {"stage": STAGE, "url": url, "candidates": 0, "sent": 0, "error": f"internal: {e}"}
