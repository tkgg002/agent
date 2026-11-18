#!/usr/bin/env python3
"""Laya Universal Decision Gate (Mode A: Calibrated Claim Gate & Mode B: Agentic Debate Review).

Commands:
  laya_gate.py run      --stage dg0|dg1|dg2|dg12|dg3 --workspace <ws> --claims <claims.json> [--objective "..."]
  laya_gate.py resolve  --workspace <ws> --stage dg0|dg1|dg2|dg3|dg3review --id <ID> --reason "... REF: path:line"
  laya_gate.py review   --workspace <ws> --repo <repo_path> [--max-rounds 3] [--fencing <fencing.json>]
  laya_gate.py backtest --r1 <raw_r1.json> [--r23 <raw_r2_r3.json>]

Exit codes:
  0 : PASS
  2 : BLOCK or NEEDS_RESOLUTION
  3 : INVALID_INPUT
  4 : LAYA_DOWN

Core Invariants (Rule #10, ADR-04, ADR-07, ADR-08):
  - Project-agnostic: zero hardcoded project names, services, or ports.
  - Verdicts come exclusively from HTTP responses (or explicit backtest files) — NEVER hardcoded.
  - Ground truth is source code & runtime: every REF: path:start-end is verified against disk.
  - Every Mode A batch requires >=2 control_false and >=1 control_true claims.
  - Any round that misses a control with confidence >= 0.5 is disqualified.
  - Mode B recursive fencing sends a corrupted-diff control alongside the fencing context to
    detect rubber-stamping.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_WS_ROOT = AGENT_ROOT / "memory" / "workspaces"
FALLBACK_STATE_ROOT = AGENT_ROOT / ".agents" / "state" / "workspaces"
DEFAULT_URL = "http://127.0.0.1:8200/v1/predict"  # Laya — kept as fallback engine
# Primary engine: VegaML sidecar (TTT head, mode=auto). Falls back to Laya when /health is not 200.
VEGA_URL = os.getenv("VEGA_GATE_URL", "http://127.0.0.1:8222/v1/predict")
LAYA_FALLBACK_URL = os.getenv("LAYA_FALLBACK_URL", DEFAULT_URL)
GATE_MODE = os.getenv("VEGA_GATE_MODE", "auto")
# Fail-fast: a Vega predict slower than this (or failing) goes straight to Laya, no retry on Vega
VEGA_TIMEOUT = float(os.getenv("VEGA_GATE_TIMEOUT", "3"))
# After a Vega failure skip Vega for this many seconds so later calls do not each wait VEGA_TIMEOUT again
VEGA_COOLDOWN = float(os.getenv("VEGA_GATE_COOLDOWN", "120"))
_vega_skip_until = 0.0

DECISIVE_CONF = 0.5
BLOCK_CONF = 0.8
DEBATE_APPROVE_CONF = 0.95

CRIT_3WAY = {
    "CORRECT": "The claim is directly and accurately supported by the provided code excerpt and flow.",
    "INCORRECT": "The claim contradicts the provided code excerpt or states something false about it.",
    "UNSUPPORTED": "The provided code excerpt does not contain enough information to verify the claim.",
}

QUESTIONS_MODE_A = {
    "r1": {
        "type": "choice",
        "instructions": (
            "You are a strict code auditor. Compare the claim against the exact code evidence. "
            "Choose CORRECT only if the evidence directly proves the claim, INCORRECT if the "
            "evidence contradicts the claim, or UNSUPPORTED if the evidence is insufficient."
        ),
        "criteria": CRIT_3WAY,
    },
    "r2": {
        "type": "choice",
        "instructions": (
            "Read the feature_objective, function_workflow, and diff_lines carefully. "
            "Evaluate whether the claim is CORRECT, INCORRECT, or UNSUPPORTED."
        ),
        "criteria": CRIT_3WAY,
    },
    "r3": {
        "type": "choice",
        "instructions": "Answer strictly YES or NO based solely on the code_excerpt.",
        "criteria": {
            "YES": "The claim is true for the code_excerpt shown.",
            "NO": "The claim is false or not shown in the code_excerpt.",
        },
    },
}

POS_CHOICE = {"r1": "CORRECT", "r2": "CORRECT", "r3": "YES"}
NEG_CHOICE = {"r1": "INCORRECT", "r2": "INCORRECT", "r3": "NO"}

REF_HEADER_RE = re.compile(r"^REF:\s*(\S+?):(\d+)(?:-(\d+))?\s*$", re.M)
INLINE_REF_RE = re.compile(r"REF:\s*(\S+?):(\d+)(?:-(\d+))?")


def norm_ws(text: str) -> str:
    return re.sub(r"\s+", " ", text).strip()


def now_iso() -> str:
    return dt.datetime.now(dt.timezone(dt.timedelta(hours=7))).isoformat(timespec="seconds")


def resolve_ref_path(raw_path: str, base_dirs: List[Path]) -> Optional[Path]:
    p = Path(raw_path)
    if p.is_absolute():
        return p if p.is_file() else None
    for base in base_dirs:
        cand = (base / p).resolve()
        if cand.is_file():
            return cand
    return None


def _normalize_evidence_str(claim_obj: Dict[str, Any], base_dirs: List[Path]) -> str:
    ev = claim_obj.get("evidence")
    items: List[Dict[str, Any]] = []
    if isinstance(ev, list):
        str_parts = [x.strip() for x in ev if isinstance(x, str) and x.strip()]
        if str_parts and not any(isinstance(x, dict) for x in ev):
            return "\n".join(str_parts)
        items.extend(x for x in ev if isinstance(x, dict))
    elif isinstance(ev, dict):
        items.append(ev)
    elif isinstance(ev, str) and ev.strip():
        return ev
    if not items and claim_obj.get("file") and claim_obj.get("lines"):
        items.append({"file": claim_obj["file"], "lines": claim_obj["lines"]})
    if not items:
        return str(ev or "")
    out_blocks: List[str] = []
    for it in items:
        fpath = str(it.get("file") or it.get("path") or "")
        lrange = str(it.get("lines") or it.get("line") or "1-1")
        quote = str(it.get("quote") or it.get("content") or "")
        if not quote and fpath:
            target = resolve_ref_path(fpath, base_dirs)
            if target:
                m_lr = re.match(r"^(\d+)(?:-(\d+))?$", lrange.strip())
                if m_lr:
                    s_ln = int(m_lr.group(1))
                    e_ln = int(m_lr.group(2) or m_lr.group(1))
                    flines = target.read_text(encoding="utf-8", errors="ignore").splitlines()
                    quote = "\n".join(flines[max(0, s_ln - 1) : e_ln])
        out_blocks.append(f"REF: {fpath}:{lrange}\n{quote}".rstrip())
    return "\n".join(out_blocks)


def verify_evidence(claim_obj: Dict[str, Any], base_dirs: Optional[List[Path]] = None) -> Tuple[bool, str]:
    """Verify that every REF: path:start-end exists and its quoted lines match the actual file."""
    bases = base_dirs or [Path.cwd(), AGENT_ROOT, AGENT_ROOT.parent]
    evidence = _normalize_evidence_str(claim_obj, bases)
    claim_obj["evidence"] = evidence
    blocks = re.split(r"(?m)^(?=REF:|DEMO:)", evidence)
    refs = [b for b in blocks if b.startswith("REF:")]
    if not refs and claim_obj.get("kind") != "demo":
        return False, "thiếu REF: <path>:<start>-<end>"
    for block in refs:
        lines = block.splitlines()
        m = REF_HEADER_RE.match(lines[0])
        if not m:
            return False, f"sai cú pháp header REF: {lines[0][:80]}"
        raw_path, start_s, end_s = m.groups()
        start_line = int(start_s)
        end_line = int(end_s or start_s)
        if start_line < 1 or end_line < start_line:
            return False, f"số dòng không hợp lệ {raw_path}:{start_line}-{end_line}"
        target = resolve_ref_path(raw_path, bases)
        if not target:
            return False, f"không tồn tại file {raw_path}"
        file_lines = target.read_text(encoding="utf-8", errors="ignore").splitlines()
        if end_line > len(file_lines):
            return False, f"vượt quá tổng số dòng ({len(file_lines)}) của {raw_path}:{start_line}-{end_line}"
        actual_slice = file_lines[start_line - 1 : end_line]
        actual_norm = {norm_ws(l) for l in actual_slice if norm_ws(l)}
        actual_joined = norm_ws("\n".join(actual_slice))
        quoted = [norm_ws(l) for l in lines[1:] if norm_ws(l)]
        if not quoted:
            return False, f"REF rỗng (không có dòng trích dẫn) tại {raw_path}:{start_line}-{end_line}"
        bad = [q for q in quoted if q not in actual_norm and q not in actual_joined]
        if bad:
            return False, f"trích dẫn không khớp {raw_path}:{start_line}-{end_line}: {bad[0][:60]}"
    return True, ""


def build_state_mode_a(rnd: str, c: Dict[str, Any], objective: str) -> Dict[str, Any]:
    if rnd == "r1":
        return {"claim": c["claim"], "evidence": c.get("evidence", ""), "flow": c.get("flow", "")}
    if rnd == "r2":
        return {
            "feature_objective": objective,
            "function_workflow": c.get("flow", ""),
            "diff_lines": c.get("evidence", ""),
            "claim": c["claim"],
        }
    return {"code_excerpt": c.get("evidence", ""), "claim": c["claim"]}


def _health_ok(predict_url: str, timeout: float = 3.0) -> bool:
    """Vega exposes /health; Laya does not (404) — only used to probe the primary engine."""
    health = predict_url.rsplit("/v1/", 1)[0] + "/health"
    try:
        with urllib.request.urlopen(health, timeout=timeout) as r:
            return r.status == 200
    except (urllib.error.URLError, TimeoutError, OSError, ValueError):
        return False


def resolve_gate_url(explicit: Optional[str]) -> str:
    """--url wins; else Vega if healthy; else Laya with a loud GATE_FALLBACK warning on stderr."""
    if explicit:
        return explicit
    if _health_ok(VEGA_URL):
        return VEGA_URL
    print(f"GATE_FALLBACK: {VEGA_URL} unhealthy -> using {LAYA_FALLBACK_URL}", file=sys.stderr)
    return LAYA_FALLBACK_URL


def post_laya(
    url: str,
    state: Dict[str, Any],
    questions: Dict[str, Any],
    retries: int = 6,
    retry_wait: float = 5.0,
    timeout: float = 120.0,
    mode: str = GATE_MODE,
) -> Dict[str, Any]:
    """Send POST to the decision engine /v1/predict, retrying on HTTP 503 (model cold start).

    "mode" is honoured by VegaML (auto|engine|ttt); Laya's PredictRequest ignores unknown fields.
    """
    global _vega_skip_until
    payload = {"state": state, "questions": questions, "mode": mode}
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    if url == VEGA_URL and LAYA_FALLBACK_URL != VEGA_URL:
        if time.monotonic() < _vega_skip_until:
            url = LAYA_FALLBACK_URL
        else:
            req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"}, method="POST")
            try:
                with urllib.request.urlopen(req, timeout=VEGA_TIMEOUT) as resp:
                    return json.loads(resp.read().decode("utf-8"))
            except (urllib.error.URLError, TimeoutError, OSError, ValueError) as e:
                _vega_skip_until = time.monotonic() + VEGA_COOLDOWN
                print(f"GATE_FALLBACK: {VEGA_URL} predict failed in <= {VEGA_TIMEOUT}s ({e}) -> using {LAYA_FALLBACK_URL}", file=sys.stderr)
                url = LAYA_FALLBACK_URL
    last_err: Optional[Exception] = None
    for attempt in range(retries + 1):
        req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            code, reason = e.code, e.reason
            e.close()
            last_err = e
            if code == 503 and attempt < retries:
                time.sleep(retry_wait)
                continue
            raise OSError(f"HTTP {code}: {reason}") from e
        except (urllib.error.URLError, TimeoutError, OSError) as e:
            last_err = e
            raise OSError(str(e)) from e
    raise OSError(str(last_err))


def ask_round_mode_a(
    url: str,
    rnd: str,
    claim_obj: Dict[str, Any],
    objective: str,
    retries: int = 6,
    retry_wait: float = 5.0,
) -> Dict[str, Any]:
    st = build_state_mode_a(rnd, claim_obj, objective)
    q = {"verdict": QUESTIONS_MODE_A[rnd]}
    raw = post_laya(url, st, q, retries=retries, retry_wait=retry_wait)
    ans = (raw.get("result") or {}).get("answers", {}).get("verdict") or {}
    choice = ans.get("choice")
    conf = float(ans.get("confidence") or 0.0)
    if not choice and isinstance(raw.get("verdict"), str):
        v_up = raw["verdict"].upper()
        if v_up in ("PASS", "CORRECT", "YES"):
            choice, conf = POS_CHOICE[rnd], float(raw.get("confidence") or 0.9)
        elif v_up in ("FAIL", "BLOCK", "INCORRECT", "NO"):
            choice, conf = NEG_CHOICE[rnd], float(raw.get("confidence") or 0.9)
    return {
        "choice": choice,
        "confidence": conf,
        "raw": raw,
        "request": {"state": st, "questions": q},
    }


def check_controls(
    claims: List[Dict[str, Any]], rows: Optional[List[Dict[str, Any]]] = None
) -> Tuple[bool, List[str], str]:
    f_cnt = sum(1 for c in claims if str(c.get("kind") or c.get("type") or "").startswith("control_false"))
    t_cnt = sum(1 for c in claims if str(c.get("kind") or c.get("type") or "").startswith("control_true"))
    if f_cnt < 2 or t_cnt < 1:
        return False, [], f"Need >=2 control_false (got {f_cnt}) and >=1 control_true (got {t_cnt})"
    if rows is None:
        return True, ["r1", "r2", "r3"], ""
    trusted = [rnd for rnd in ("r1", "r2", "r3") if is_round_trusted(rows, rnd)]
    if not trusted:
        return False, [], "No round passed control calibration"
    return True, trusted, ""


def is_round_trusted(rows: List[Dict[str, Any]], rnd: str) -> bool:
    """A round is trusted iff:
    (a) no control claim is answered with the wrong choice at confidence >= DECISIVE_CONF, AND
    (b) at least one control claim is answered with the expected choice at confidence >= DECISIVE_CONF.
    """
    hit_correct = False
    for r in rows:
        kind = r.get("kind", "")
        if not kind.startswith("control_"):
            continue
        expected = POS_CHOICE[rnd] if kind == "control_true" else NEG_CHOICE[rnd]
        v = r.get(rnd) or {}
        conf = float(v.get("confidence") or 0.0)
        choice = v.get("choice")
        if conf >= DECISIVE_CONF and choice != expected:
            return False
        if conf >= DECISIVE_CONF and choice == expected:
            hit_correct = True
    return hit_correct


def decide_claim(row: Dict[str, Any], trusted_rounds: List[str]) -> str:
    if not row.get("evidence_ok", True):
        return "BLOCK"
    if row.get("basis") == "hypothesis":
        return "HYPOTHESIS"
    if not trusted_rounds:
        return "UNVALIDATED"
    verdicts = [(rnd, row.get(rnd) or {}) for rnd in trusted_rounds]
    if any(v.get("choice") == NEG_CHOICE[rnd] and float(v.get("confidence") or 0.0) >= BLOCK_CONF for rnd, v in verdicts):
        return "BLOCK"
    if any(v.get("choice") == NEG_CHOICE[rnd] and float(v.get("confidence") or 0.0) >= DECISIVE_CONF for rnd, v in verdicts):
        return "FLAG"
    if any(v.get("choice") == POS_CHOICE[rnd] and float(v.get("confidence") or 0.0) >= DECISIVE_CONF for rnd, v in verdicts):
        return "ACCEPT"
    return "FLAG"


def compute_gate(summary: Dict[str, Any]) -> str:
    decisions = [c["decision"] for c in summary.get("claims", [])]
    if "BLOCK" in decisions:
        return "BLOCK"
    if any(c["decision"] == "FLAG" and not c.get("resolved") for c in summary.get("claims", [])):
        return "NEEDS_RESOLUTION"
    return "PASS"


def write_workspace_json(ws_root: Path, workspace: str, filename: str, payload: Any) -> Path:
    """Write JSON to workspace directory; fall back to .agents/state only on sandbox PermissionError."""
    content = json.dumps(payload, ensure_ascii=False, indent=2) + "\n"
    primary_dir = ws_root / workspace
    try:
        primary_dir.mkdir(parents=True, exist_ok=True)
        target = primary_dir / filename
        target.write_text(content, encoding="utf-8")
        return target
    except PermissionError:
        fb_dir = FALLBACK_STATE_ROOT / workspace
        fb_dir.mkdir(parents=True, exist_ok=True)
        target = fb_dir / filename
        target.write_text(content, encoding="utf-8")
        return target


def find_summary_files(ws_root: Path, workspace: str, stage: str) -> List[Path]:
    pattern = f"06_laya_summary_{stage}_*.json"
    found: List[Path] = []
    for root in (ws_root / workspace, FALLBACK_STATE_ROOT / workspace):
        try:
            if root.exists():
                found.extend(root.glob(pattern))
        except OSError:
            continue
    return sorted(found, key=lambda p: p.name)


def cmd_run(args: argparse.Namespace) -> int:
    from concurrent.futures import ThreadPoolExecutor

    claims_path = Path(args.claims)
    if not claims_path.is_file():
        print(f"INVALID_INPUT: không tìm thấy file claims {claims_path}", file=sys.stderr)
        return 3
    try:
        raw_loaded = json.loads(claims_path.read_text(encoding="utf-8"))
    except ValueError as e:
        print(f"INVALID_INPUT: JSON không hợp lệ: {e}", file=sys.stderr)
        return 3
    if isinstance(raw_loaded, dict) and isinstance(raw_loaded.get("claims"), list):
        claims = raw_loaded["claims"]
    elif isinstance(raw_loaded, list):
        claims = raw_loaded
    else:
        print("INVALID_INPUT: file claims phải là JSON array hoặc object chứa mảng 'claims'", file=sys.stderr)
        return 3
    if not claims or not all(isinstance(c, dict) for c in claims):
        print("INVALID_INPUT: mỗi phần tử trong claims phải là JSON object (dict)", file=sys.stderr)
        return 3
    for c in claims:
        raw_kind = c.get("kind") if c.get("kind") not in (None, "") else c.get("type")
        ctrl = c.get("control")
        if raw_kind in (None, ""):
            if isinstance(ctrl, bool):
                raw_kind = "control_true" if ctrl else "control_false"
            elif isinstance(ctrl, str) and ctrl.strip().lower() in ("true", "1", "control_true"):
                raw_kind = "control_true"
            elif isinstance(ctrl, str) and ctrl.strip().lower() in ("false", "0", "control_false"):
                raw_kind = "control_false"
            elif isinstance(ctrl, str) and ctrl.strip():
                raw_kind = ctrl.strip()
            else:
                raw_kind = "claim"
        elif isinstance(raw_kind, bool):
            raw_kind = "control_true" if raw_kind else "control_false"
        c["kind"] = str(raw_kind).strip()
    f_cnt = sum(1 for c in claims if c.get("kind", "").startswith("control_false"))
    t_cnt = sum(1 for c in claims if c.get("kind", "").startswith("control_true"))
    if f_cnt < 2 or t_cnt < 1:
        print(
            f"INVALID_INPUT: cần ≥2 control_false (got {f_cnt}) và ≥1 control_true (got {t_cnt}) trong mỗi batch (calibration_ok=False)",
            file=sys.stderr,
        )
        return 3

    rows: List[Dict[str, Any]] = []
    for c in claims:
        if c.get("kind", "").startswith("control_"):
            ok, why = True, ""
            c["evidence"] = str(c.get("evidence") or "")
        else:
            ok, why = verify_evidence(c)
        rows.append({**c, "evidence_ok": ok, "evidence_error": why})

    args.url = resolve_gate_url(args.url)  # Vega :8222 primary, Laya :8200 fallback, --url override
    workers_n = max(1, int(getattr(args, "workers", 3) or 3))
    tasks = [(idx, rnd) for idx in range(len(rows)) for rnd in ("r1", "r2", "r3")]

    def _run_one(item: Tuple[int, str]) -> Tuple[int, str, Dict[str, Any]]:
        idx, rnd = item
        res = ask_round_mode_a(
            args.url,
            rnd,
            rows[idx],
            args.objective,
            retries=args.retries,
            retry_wait=args.retry_wait,
        )
        return idx, rnd, res

    try:
        with ThreadPoolExecutor(max_workers=workers_n) as pool:
            for idx, rnd, res in pool.map(_run_one, tasks):
                rows[idx][rnd] = res
    except OSError as e:
        print(f"LAYA_DOWN: {e}", file=sys.stderr)
        return 4

    trusted_rounds = [rnd for rnd in ("r1", "r2", "r3") if is_round_trusted(rows, rnd)]
    for r in rows:
        if not r.get("kind", "").startswith("control_"):
            r["decision"] = decide_claim(r, trusted_rounds)

    ts = time.strftime("%Y%m%dT%H%M%S")
    ws_root = Path(args.ws_root)
    stage_norm = "dg1" if args.stage == "dg12" else args.stage
    raw_file = write_workspace_json(ws_root, args.workspace, f"06_laya_raw_{stage_norm}_{ts}.json", rows)

    summary: Dict[str, Any] = {
        "mode": "claim",
        "stage": stage_norm,
        "workspace": args.workspace,
        "ts": ts,
        "engine_url": LAYA_FALLBACK_URL if (args.url == VEGA_URL and time.monotonic() < _vega_skip_until) else args.url,
        "engine_mode": GATE_MODE,
        "trusted_rounds": trusted_rounds,
        "raw_file": str(raw_file),
        "claims": [
            {
                "id": r["id"],
                "kind": r.get("kind", "claim"),
                "basis": r.get("basis", "code"),
                "claim": r["claim"],
                "decision": r["decision"],
                "resolved": None,
                "evidence_error": r.get("evidence_error", ""),
                "rounds": {
                    rnd: {"choice": r[rnd]["choice"], "confidence": round(r[rnd]["confidence"], 4)}
                    for rnd in ("r1", "r2", "r3")
                },
            }
            for r in rows
            if "decision" in r
        ],
    }
    summary["gate"] = compute_gate(summary)
    sum_file = write_workspace_json(ws_root, args.workspace, f"06_laya_summary_{stage_norm}_{ts}.json", summary)
    summary["summary_file"] = str(sum_file)
    has_demo = any(c.get("kind") == "demo" or str(c.get("id", "")).startswith("D-") for c in claims)
    if args.stage == "dg12" or (args.stage == "dg1" and has_demo):
        dg2_summary = {**summary, "stage": "dg2"}
        dg2_file = write_workspace_json(ws_root, args.workspace, f"06_laya_summary_dg2_{ts}.json", dg2_summary)
        summary["dg2_summary_file"] = str(dg2_file)
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary["gate"] == "PASS" else 2


def cmd_resolve(args: argparse.Namespace) -> int:
    req_ids = list(dict.fromkeys(x.strip() for x in str(args.id or "").split(",") if x.strip()))
    if not req_ids or any(rid.upper() == "ALL" for rid in req_ids):
        print("INVALID_INPUT: cấm dùng --id ALL; bắt buộc liệt kê tường minh từng claim ID đang FLAG", file=sys.stderr)
        return 3

    ws_root = Path(args.ws_root)
    summaries = find_summary_files(ws_root, args.workspace, args.stage)
    if not summaries:
        print(f"INVALID_INPUT: không tìm thấy summary cho stage {args.stage} trong {args.workspace}", file=sys.stderr)
        return 3
    latest_path = summaries[-1]
    summary = json.loads(latest_path.read_text(encoding="utf-8"))
    by_id = {c.get("id"): c for c in summary.get("claims", []) if isinstance(c, dict)}
    missing_ids = [rid for rid in req_ids if rid not in by_id]
    if missing_ids:
        print(f"INVALID_INPUT: không tìm thấy claim id={','.join(missing_ids)}", file=sys.stderr)
        return 3
    targets = [by_id[rid] for rid in req_ids]
    if any(tc.get("decision") == "BLOCK" for tc in targets):
        bad_blk = ",".join(tc["id"] for tc in targets if tc.get("decision") == "BLOCK")
        print(f"INVALID_INPUT: claim {bad_blk} đang là BLOCK — không được resolve bằng lời, phải sửa và chạy lại", file=sys.stderr)
        return 2
    non_flag = [tc["id"] for tc in targets if tc.get("decision") != "FLAG"]
    if non_flag:
        print(f"INVALID_INPUT: chỉ được resolve claim đang FLAG (không hợp lệ: {','.join(non_flag)})", file=sys.stderr)
        return 3

    reason_str = str(getattr(args, "reason", None) or getattr(args, "evidence", None) or "").strip()
    refs = list(INLINE_REF_RE.finditer(reason_str))
    if not refs:
        print("INVALID_INPUT: --reason bắt buộc chứa 'REF: <path>:<line>'", file=sys.stderr)
        return 3
    for m in refs:
        ref_path = resolve_ref_path(m.group(1), [Path.cwd(), AGENT_ROOT, AGENT_ROOT.parent])
        if not ref_path:
            print(f"INVALID_INPUT: file trong REF không tồn tại: {m.group(1)}", file=sys.stderr)
            return 3

    for tc in targets:
        tc["resolved"] = reason_str
    summary["gate"] = compute_gate(summary)
    write_workspace_json(ws_root, args.workspace, latest_path.name, summary)

    synced_dg2 = False
    if args.stage == "dg1":
        dg2_cand = latest_path.with_name(latest_path.name.replace("_dg1_", "_dg2_", 1))
        if not dg2_cand.is_file():
            dg2_files = find_summary_files(ws_root, args.workspace, "dg2")
            dg2_cand = dg2_files[-1] if dg2_files else dg2_cand
        if dg2_cand.is_file():
            dg2_data = json.loads(dg2_cand.read_text(encoding="utf-8"))
            req_set = set(req_ids)
            for c2 in dg2_data.get("claims", []):
                if isinstance(c2, dict) and c2.get("id") in req_set and c2.get("decision") == "FLAG":
                    c2["resolved"] = reason_str
                    synced_dg2 = True
            if synced_dg2:
                dg2_data["gate"] = compute_gate(dg2_data)
                write_workspace_json(ws_root, args.workspace, dg2_cand.name, dg2_data)

    # Append to 04_decisions_*.md in workspace
    resolved_label = ",".join(tc["id"] for tc in targets)
    sync_tag = " (DG1+DG2)" if synced_dg2 else ""
    log_line = f"- [{now_iso()}] [Agent:laya_gate] {args.stage.upper()} {resolved_label} FLAG resolved{sync_tag}: {reason_str}\n"
    for root in (ws_root / args.workspace, FALLBACK_STATE_ROOT / args.workspace):
        try:
            root.mkdir(parents=True, exist_ok=True)
            decs = sorted(root.glob("04_decisions*.md"))
            dec_file = decs[-1] if decs else (root / f"04_decisions_{args.workspace}.md")
            existing = dec_file.read_text(encoding="utf-8") if dec_file.is_file() else "# 04 — Decisions (Append ONLY)\n\n"
            if not existing.endswith("\n"):
                existing += "\n"
            dec_file.write_text(existing + log_line, encoding="utf-8")
            break
        except PermissionError:
            continue
        except OSError:
            break

    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary["gate"] == "PASS" else 2


def corrupt_diff_for_control(diff_text: str) -> str:
    """Create a deliberately broken variant of a diff chunk to test if Laya is rubber-stamping."""
    swaps = [
        ("==", "!="),
        ("!=", "=="),
        ("return True", "return False"),
        ("return False", "return True"),
        ("return 0", "return 2"),
        ("if not ", "if "),
    ]
    for old, new in swaps:
        if old in diff_text:
            return diff_text.replace(old, new, 1)
    return diff_text + "\n+    raise RuntimeError('CORRUPTED_CONTROL_BUG_INJECTED')\n"


def ask_debate_review(
    url: str,
    filename: str,
    diff_text: str,
    objective: str,
    workflow: str,
    is_fencing: bool,
    retries: int = 6,
    retry_wait: float = 5.0,
) -> Dict[str, Any]:
    state = {
        "feature_objective": objective,
        "filename": filename,
        "diff_lines": "See full_function_code for diff markers (+/-)",
        "full_function_code": diff_text[:4000],
        "function_workflow": workflow,
    }
    if not is_fencing:
        questions = {
            "review_result": {
                "type": "choice",
                "instructions": (
                    "Đánh giá đoạn code thay đổi này dựa trên toàn bộ context, mục tiêu feature và workflow "
                    "được cung cấp. Đảm bảo logic an toàn và không bị lặp code."
                ),
                "criteria": {
                    "APPROVE": "Code hoàn toàn hợp lý, an toàn, đã cover edge-case và đúng mục tiêu.",
                    "REJECT_REDUNDANT": "Code dư thừa, lặp logic.",
                    "REJECT_LOGIC": "Thiếu an toàn, sai logic, hoặc có rủi ro edge-cases chưa xử lý.",
                },
            }
        }
    else:
        questions = {
            "review_result": {
                "type": "choice",
                "instructions": (
                    "Đọc kĩ ngữ cảnh bào chữa (fencing context) của kĩ sư. Đánh giá xem đoạn code thay đổi "
                    "này có thực sự vi phạm logic hay không?"
                ),
                "criteria": {
                    "APPROVE": "Code hoàn toàn hợp lý, an toàn, đã cover edge-case và giải thích rào logic hợp lý.",
                    "REJECT_LOGIC": "Thiếu an toàn, sai logic, hoặc có rủi ro edge-cases chưa xử lý.",
                },
            }
        }
    raw = post_laya(url, state, questions, retries=retries, retry_wait=retry_wait)
    ans = raw.get("result", {}).get("answers", {}).get("review_result", {})
    return {
        "choice": ans.get("choice"),
        "confidence": float(ans.get("confidence") or 0.0),
        "raw": raw,
    }


def cmd_review(args: argparse.Namespace) -> int:
    """Mode B: Laya Agentic Debate (Recursive Fencing with Corrupted-Diff Control)."""
    if args.diff_file:
        diff_text = Path(args.diff_file).read_text(encoding="utf-8")
    else:
        r = subprocess.run(
            ["git", "diff", "-U30", "HEAD"],
            cwd=args.repo,
            capture_output=True,
            text=True,
        )
        diff_text = r.stdout

    fencing_map: Dict[str, Any] = {}
    if args.fencing and Path(args.fencing).is_file():
        fencing_map = json.loads(Path(args.fencing).read_text(encoding="utf-8"))

    chunks = [c for c in diff_text.split("diff --git") if c.strip()]
    file_records: List[Dict[str, Any]] = []
    args.url = resolve_gate_url(args.url)  # Vega :8222 primary, Laya :8200 fallback, --url override

    try:
        for chunk in chunks:
            first_line = chunk.strip().splitlines()[0]
            filename = first_line.split("b/")[-1].strip() if "b/" in first_line else "unknown"
            f_ctx = fencing_map.get(filename, {})
            base_obj = f_ctx.get("objective") or args.objective
            base_wf = f_ctx.get("workflow") or args.workflow

            r1 = ask_debate_review(
                args.url, filename, chunk, base_obj, base_wf, is_fencing=False,
                retries=args.retries, retry_wait=args.retry_wait,
            )
            rec: Dict[str, Any] = {
                "id": filename,
                "filename": filename,
                "round_1": {"choice": r1["choice"], "confidence": round(r1["confidence"], 4)},
                "fencing_rounds": [],
            }
            if r1["choice"] == "APPROVE":
                rec["decision"] = "ACCEPT"
            else:
                fencing_list = f_ctx.get("rounds") or [
                    {
                        "objective": f_ctx.get("fencing_objective") or base_obj,
                        "workflow": f_ctx.get("fencing_workflow") or base_wf,
                    }
                ]
                resolved_by_debate = False
                rubber_stamped = False
                for idx in range(min(args.max_rounds - 1, len(fencing_list))):
                    fc = fencing_list[idx]
                    real_res = ask_debate_review(
                        args.url, filename, chunk, fc["objective"], fc["workflow"], is_fencing=True,
                        retries=args.retries, retry_wait=args.retry_wait,
                    )
                    bad_chunk = fc.get("corrupted_diff") or corrupt_diff_for_control(chunk)
                    ctrl_res = ask_debate_review(
                        args.url, filename, bad_chunk, fc["objective"], fc["workflow"], is_fencing=True,
                        retries=args.retries, retry_wait=args.retry_wait,
                    )
                    ctrl_approved = ctrl_res["choice"] == "APPROVE" and ctrl_res["confidence"] >= DECISIVE_CONF
                    rec["fencing_rounds"].append(
                        {
                            "round": idx + 2,
                            "real": {"choice": real_res["choice"], "confidence": round(real_res["confidence"], 4)},
                            "control_corrupted": {
                                "choice": ctrl_res["choice"],
                                "confidence": round(ctrl_res["confidence"], 4),
                            },
                            "control_passed": not ctrl_approved,
                        }
                    )
                    if ctrl_approved:
                        rubber_stamped = True
                        break
                    if real_res["choice"] == "APPROVE" and real_res["confidence"] >= DEBATE_APPROVE_CONF:
                        resolved_by_debate = True
                        break
                if resolved_by_debate:
                    rec["decision"] = "ACCEPT"
                elif rubber_stamped:
                    rec["decision"] = "UNVALIDATED"
                else:
                    rec["decision"] = "BLOCK"
            file_records.append(rec)
    except OSError as e:
        print(f"LAYA_DOWN: {e}", file=sys.stderr)
        return 4

    ts = time.strftime("%Y%m%dT%H%M%S")
    ws_root = Path(args.ws_root)
    raw_file = write_workspace_json(ws_root, args.workspace, f"06_laya_raw_dg3review_{ts}.json", file_records)
    summary = {
        "mode": "debate_review",
        "stage": "dg3review",
        "workspace": args.workspace,
        "ts": ts,
        "raw_file": str(raw_file),
        "claims": [
            {"id": r["id"], "decision": r["decision"], "resolved": None, "round_1": r["round_1"], "fencing_rounds": r["fencing_rounds"]}
            for r in file_records
        ],
    }
    summary["gate"] = compute_gate(summary)
    sum_file = write_workspace_json(ws_root, args.workspace, f"06_laya_summary_dg3review_{ts}.json", summary)
    summary["summary_file"] = str(sum_file)
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary["gate"] == "PASS" else 2


def cmd_backtest(args: argparse.Namespace) -> int:
    """Re-evaluate saved raw JSON files to verify calibration & decision logic deterministically."""
    r1_data = json.loads(Path(args.r1).read_text(encoding="utf-8"))
    r23_map: Dict[str, Any] = {}
    if args.r23 and Path(args.r23).is_file():
        for item in json.loads(Path(args.r23).read_text(encoding="utf-8")):
            r23_map[item["id"]] = item

    rows: List[Dict[str, Any]] = []
    for item in r1_data:
        cid = item["id"]
        merged = dict(item)
        merged.setdefault("evidence_ok", True)
        if "r1" not in merged and "choice" in merged:
            merged["r1"] = {"choice": merged["choice"], "confidence": float(merged.get("confidence") or 0.0)}
        if cid in r23_map:
            for rnd in ("r2", "r3"):
                if rnd in r23_map[cid]:
                    merged[rnd] = r23_map[cid][rnd]
        rows.append(merged)

    avail = [rnd for rnd in ("r1", "r2", "r3") if all(rnd in r for r in rows)]
    trusted = [rnd for rnd in avail if is_round_trusted(rows, rnd)]
    out_claims = []
    for r in rows:
        if not r.get("kind", "").startswith("control_"):
            out_claims.append({"id": r["id"], "decision": decide_claim(r, trusted)})
    result = {"available_rounds": avail, "trusted_rounds": trusted, "claims": out_claims, "gate": compute_gate({"claims": out_claims})}
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Laya Decision Gate CLI (Mode A Claim & Mode B Debate Review)")
    sub = p.add_subparsers(dest="subcommand", required=True)

    p_run = sub.add_parser("run", help="Run Mode A Calibrated Claim Gate (dg0, dg1, dg2, dg12, dg3)")
    p_run.add_argument("--stage", required=True, choices=["dg0", "dg1", "dg2", "dg12", "dg3"])
    p_run.add_argument("--workspace", required=True)
    p_run.add_argument("--claims", required=True)
    p_run.add_argument("--objective", default="Kiểm định kỹ thuật theo chuẩn Master Workflow")
    p_run.add_argument("--url", default=None, help="Override engine; default = Vega :8222 if healthy else Laya :8200")
    p_run.add_argument("--ws-root", default=str(DEFAULT_WS_ROOT))
    p_run.add_argument("--retries", type=int, default=6)
    p_run.add_argument("--retry-wait", type=float, default=5.0)
    p_run.add_argument("--workers", type=int, default=3)

    p_res = sub.add_parser("resolve", help="Resolve a FLAG claim with REF: evidence into 04_decisions")
    p_res.add_argument("--workspace", required=True)
    p_res.add_argument("--stage", required=True, choices=["dg0", "dg1", "dg2", "dg3", "dg3review"])
    p_res.add_argument("--id", required=True)
    p_res.add_argument("--reason", "--evidence", dest="reason", required=True)
    p_res.add_argument("--ws-root", default=str(DEFAULT_WS_ROOT))

    p_rev = sub.add_parser("review", help="Run Mode B Agentic Debate Review on git diff -U30")
    p_rev.add_argument("--workspace", required=True)
    p_rev.add_argument("--repo", default=".")
    p_rev.add_argument("--diff-file", default="")
    p_rev.add_argument("--fencing", default="")
    p_rev.add_argument("--objective", default="Đảm bảo thay đổi an toàn, đúng kiến trúc và không lặp logic")
    p_rev.add_argument("--workflow", default="1. Nhận input 2. Xử lý theo thiết kế đã duyệt 3. Trả kết quả an toàn")
    p_rev.add_argument("--max-rounds", type=int, default=3)
    p_rev.add_argument("--url", default=None, help="Override engine; default = Vega :8222 if healthy else Laya :8200")
    p_rev.add_argument("--ws-root", default=str(DEFAULT_WS_ROOT))
    p_rev.add_argument("--retries", type=int, default=6)
    p_rev.add_argument("--retry-wait", type=float, default=5.0)

    p_bt = sub.add_parser("backtest", help="Backtest calibration on saved raw JSON files")
    p_bt.add_argument("--r1", required=True)
    p_bt.add_argument("--r23", default="")

    return p


def main(argv: Optional[List[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.subcommand == "run":
        return cmd_run(args)
    if args.subcommand == "resolve":
        return cmd_resolve(args)
    if args.subcommand == "review":
        return cmd_review(args)
    if args.subcommand == "backtest":
        return cmd_backtest(args)
    return 3


if __name__ == "__main__":
    sys.exit(main())
