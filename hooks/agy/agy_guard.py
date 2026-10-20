#!/usr/bin/env python3
"""Antigravity Lifecycle Hook Dispatcher (H1-H7).

Usage:
  python3 agy_guard.py pre-invocation|pre-write|pre-command|post-invocation|stop

Hooks implemented:
  H1 boot          (PreInvocation)  : Injects S0 Boot checklist + Laya server status (:8200).
  H2 memory_guard  (PreToolUse)     : Denies Overwrite on existing memory files; denies removing
                                      TargetContent on append-only files (05_progress, 04_decisions, lessons).
  H3 secret_guard  (PreToolUse)     : Denies unmasked secrets/URIs in memory files; force_ask in source files.
  H4 change_gate   (PreToolUse)     : Denies production mock pollution (*Fn, sqlmock, fakemodel);
                                      denies source edits without active workspace, 05_progress log, or DG1 PASS;
                                      force_ask if no User APPROVE after 09_tasks_solution in transcript.
  H5 command_guard (PreToolUse)     : Denies git push (Rule #15), git reset --hard / clean -f (Rule #19),
                                      rm -rf memory/.git (Rule #17); force_ask on direct DB write CLI (Rule #12).
  H6 fast_feedback (PostInvocation) : Runs component checks from nearest .agent-project.json after source edits
                                      AND auto-injects Mandatory Final Adversarial Audit & Anti-Smoke-Speculation protocol.
  H7 done_gate     (Stop)           : Blocks premature completion if DG1/DG2/DG3, 05_progress, 06_validation,
                                      audit_report_<suffix>.md (6 sections), component runtime_check, or
                                      verify_governance.py are missing/failing (max 2 blocks per turn).
"""
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, Iterator, List, Optional, Tuple

AGENT = Path(os.environ.get("AGY_AGENT_ROOT", str(Path(__file__).resolve().parents[2])))
MEM = AGENT / "memory"
WS_ROOT = MEM / "workspaces"
FALLBACK_WS_ROOT = AGENT / ".agents" / "state" / "workspaces"
STATE = Path(os.environ.get("AGY_STATE_DIR", "/tmp/agy"))

SRC_EXT = {".go", ".ts", ".tsx", ".js", ".py", ".sql"}
APPEND_ONLY = re.compile(r"(05_progress[^/]*|04_decisions[^/]*|lessons|governance_metrics)\.md$")
SECRET = re.compile(
    r"(AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|ATATT[\w\-]{20,}|ghp_\w{30,}"
    r"|Bearer\s+[\w.\-]{20,}|(mongodb|postgres(ql)?|mysql|redis)://[^*\s:/]+:[^*\s@]+@"
    r"|password\s*[:=]\s*[^\s*'\"]{6,})",
    re.I,
)
APPROVE = re.compile(
    r"(approve|duyệt|ok\s*(sửa|làm|fix)|làm đi|sửa đi|fix đi|triển khai|thực hiện đi|chặn\s*đứng|thêm.*vào.*cho\s*tao|vòng\s+lặp\s+phản\s+tỉnh|self-improvement\s+loop|update\s+agy_g)",
    re.I,
)
PROD_MOCK_POLLUTION = re.compile(
    r"(\b\w+Fn\s+func\s*\(|\blookupMasterTargetsFn\b|\brunEventsFn\b|\bsqlmock\b|\bfakemodel\b)",
)
CLEAN_ROOM_MODE_RE = re.compile(r"^\s*mode\s*:\s*clean-room\b", re.I | re.M)
CLEAN_ROOM_FORBIDDEN_RE = re.compile(r"^\s*clean_room_forbidden_sources\s*:\s*(.+)$", re.I | re.M)
CLEAN_ROOM_SPEC_RE = re.compile(r"clean-room\s+approved\s+spec", re.I)
MANIFEST = ".agent-project.json"
LAYA_START = "/Users/trainguyen/.venv/bin/python3 /Users/trainguyen/.agent/skills/laya_router/api.py"

DENY_CMD = [
    (r"\bgit\s+push\b", "Rule 15: CẤM git push — User là người duy nhất quyết định push."),
    (r"\bgit\s+(reset\s+--hard|clean\s+-\w*f)", "Rule 19: CẤM chạy lệnh huỷ thay đổi chưa commit."),
    (r"\brm\s+-\w*r\w*\b.*(/memory\b|\.git\b)", "Rule 17/19: CẤM xoá thư mục memory hoặc .git."),
]
ASK_CMD = [
    (
        r"\b(psql|mongosh|mysql|redis-cli)\b.*\b(UPDATE|DELETE|INSERT|DROP|TRUNCATE|ALTER|deleteMany|updateMany|FLUSHALL|FLUSHDB)\b",
        "Rule 12: Phát hiện lệnh ghi trực tiếp vào DB (cheat DB) — cần User xác nhận.",
    )
]
BOOT_MSG = (
    "BOOT S0: (1) Đọc agent/GEMINI.md + memory/global/lessons/INDEX.md. "
    "(2) Phân loại T0/Audit/T1-T3 (kiểm tra lỗi/phân tích/thiết kế/sửa code >= T2, CẤM trả lời chay trên chat). "
    "(3) S1 Intake (T1+): tạo 01_requirements_<suffix>.md + 08_tasks_<suffix>.md + append 05_progress START TRƯỚC khi lập 09_tasks_solution hoặc sửa code. "
    "(4) Giải pháp/code demo/audit → chạy laya_gate.py (Laya: {laya}). "
    "(5) Không sửa source trước khi có User APPROVE; sau khi sửa code BẮT BUỘC lập audit_report_<suffix>.md (6 phần, chống suy diễn smoke test). (6) CẤM git push."
)
VERBATIM_USER_AUDIT_PROMPT = (
    'Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, '
    "các file đã update, từng dòng update.\n"
    "Kiểm tra các xem có suy diễn, báo cáo láo không.\n"
    "Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).\n"
    "Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại "
    "1 lần xem quyết định làm như vậy là ổn chưa cùng\n"
    "Lưu toàn bộ vào file audit_report_*.md trong workspace của task\n"
    "Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value."
)
FINAL_AUDIT_MSG = (
    "🔍 MANDATORY FINAL ADVERSARIAL AUDIT & ANTI-SMOKE-SPECULATION PROTOCOL (Rule #9 S5/S6, #14 G3/G7/G8, #18):\n"
    "Đã phát hiện chỉnh sửa source code ({files}). TRƯỚC KHI báo cáo hoàn tất, BẮT BUỘC tự động thực thi NGUYÊN VĂN mệnh lệnh Audit sau của User:\n"
    "«\n"
    + VERBATIM_USER_AUDIT_PROMPT
    + "\n»\n"
    "Checklist bắt buộc trong `audit_report_<suffix>.md`:\n"
    "1. [Line-by-Line Diff & Task Critique]: Chạy `git diff` thật; rà soát phản biện từng task trong 08_tasks, từng file & từng dòng git diff.\n"
    "2. [Anti-Smoke-Speculation & Runtime Trace (CẤM Mockup/Smoke Test/Hallucination)]:\n"
    "   - CẤM tạo/dùng mock closure (*Fn), struct literal giả lập, hoặc smoke script rời để suy diễn luồng thật đúng.\n"
    "   - Kiểm tra đủ 3 trục lỗi thực tế: (a) Concurrency/Multi-worker (Race condition, HoL blocking, Debouncer), "
    "(b) Timeout/Heartbeat (AckWait vs InProgress), (c) Data Continuity (Watermark/Cursor qua Outage gap).\n"
    "   - Nếu cần trace runtime value trên môi trường thật mà không tự chạy được -> BẮT BUỘC nêu rõ để phối hợp với User trace runtime value!\n"
    "3. [Anti-Speculation & Honesty Check]: Kiểm tra xem có suy diễn hay báo cáo láo không (mọi khẳng định phải có file:line + output lệnh thật).\n"
    "4. [Self-Improvement & Autonomous Self-Healing Loop (S5 -> S3 -> S4 -> S5)]:\n"
    "   - Đối chiếu lessons/INDEX.md; nếu phát hiện lỗi/anti-pattern mới, ghi vào lessons.md + chạy build_lessons_index.py.\n"
    "   - Nếu lúc Audit phát hiện còn bug/lỗ hổng/thiếu sót so với Plan: CẤM dừng lại bắt User APPROVE lần 2! "
    "Bắt buộc tự động quay về S3 cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` + chạy `DG1/DG2`, "
    "sau đó tự sửa dứt điểm source code (S4) + chạy lại test + Audit lại (S5 + DG3) trong cùng lượt cho tới khi sạch bóng bug rồi mới báo OK!\n"
    "5. [QC, 3-Mutant Gate & /laya_router DG3]: Chạy `mutation_gate.py` (nếu task có `mode: clean-room` — kiểm chứng >=3 bản cố tình làm hỏng đều bị test đánh trượt KILLED) + gọi `laya_gate.py run --stage dg3` (và/hoặc `review`) để nhận xét lại quyết định thực thi.\n"
    "6. [Lưu báo cáo vật lý]: Lưu toàn bộ kết quả 6 mục trên vào `audit_report_<suffix>.md` trong workspace của task (H7 sẽ chặn nếu thiếu!)."
)
ESCALATION_MENTOR_MSG = (
    "🚨 H6 EXECUTION MONITOR — PHÁT HIỆN VÒNG LẶP THẤT BẠI (Rule #15 Escalation >3 lần):\n"
    "Đã thất bại liên tiếp {loops} lần khi chạy checks sau khi sửa code!\n"
    "BẮT BUỘC DỪNG NGAY việc sửa mò (blind-fixing):\n"
    "1. Quay lại đọc kỹ `09_tasks_solution_<suffix>.md` và log lỗi thực tế.\n"
    "2. Nếu giải pháp hiện tại không khả thi, dừng sửa code và báo cáo rõ cho User để Re-plan (S3)."
)
AUDIT_REQUIRED_PATTERNS = [
    (r"(line-by-line|từng dòng|git diff)", "Mục 1: Phản biện từng dòng update / git diff & từng task"),
    (r"(mockup|hallucination|runtime|trace|smoke)", "Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật"),
    (r"(concurrency|multi-worker|worker|timeout|heartbeat|ackwait|watermark|continuity|debouncer)", "Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity"),
    (r"(suy diễn|báo cáo láo|speculation|trung thực|honesty)", "Mục 4: Kiểm tra suy diễn & báo cáo láo"),
    (r"(self-improvement|phản tỉnh|lessons)", "Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop)"),
    (r"(laya_router|laya_gate|dg3)", "Mục 6: Kết quả QC & nhận xét từ /laya_router (DG3)"),
]


def _unquote(val: Any) -> Any:
    if isinstance(val, str) and len(val) >= 2 and val[0] == '"' and val[-1] == '"':
        try:
            return json.loads(val)
        except Exception:
            return val.strip('"')
    return val


def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:
    tr = payload.get("transcriptPath", "")
    if not tr:
        return []
    try:
        with open(tr, encoding="utf-8") as f:
            return [json.loads(line) for line in f if line.strip()]
    except OSError:
        return []


def calls(st: List[Dict[str, Any]]) -> Iterator[Tuple[int, Optional[str], Dict[str, Any]]]:
    for i, s in enumerate(st):
        for tc in s.get("tool_calls") or []:
            raw_args = tc.get("args") or {}
            clean_args = {k: _unquote(v) for k, v in raw_args.items()}
            yield i, tc.get("name"), clean_args


def last_user_idx(st: List[Dict[str, Any]]) -> int:
    for i in range(len(st) - 1, -1, -1):
        if st[i].get("type") == "USER_INPUT":
            return i
    return -1


def tgt(args: Dict[str, Any]) -> str:
    return str(_unquote(args.get("TargetFile") or ""))


def is_mem_path(path_str: str, mem_root: Optional[Path] = None) -> bool:
    mroot = str(mem_root or MEM)
    return (mroot in path_str) or ("/memory/" in path_str)


def is_src(path_str: str, mem_root: Optional[Path] = None) -> bool:
    clean = str(_unquote(path_str or ""))
    if not clean or is_mem_path(clean, mem_root):
        return False
    return Path(clean).suffix in SRC_EXT


def is_test_file(path_str: str) -> bool:
    p = Path(str(_unquote(path_str or "")))
    name = p.name.lower()
    return (
        name.endswith("_test.go")
        or name.startswith("test_")
        or name.endswith("_test.py")
        or ".test." in name
        or ".spec." in name
        or "/tests/" in p.as_posix().lower()
    )


def touched(st: List[Dict[str, Any]], pattern: str, after: int = -1) -> bool:
    return any(
        i > after and re.search(pattern, json.dumps(a, ensure_ascii=False))
        for i, _, a in calls(st)
    )


def src_edits(st: List[Dict[str, Any]], mem_root: Optional[Path] = None) -> List[Tuple[int, str]]:
    return [
        (i, tgt(a))
        for i, n, a in calls(st)
        if n in ("write_to_file", "replace_file_content") and is_src(tgt(a), mem_root)
    ]


def project_reads_since(
    st: List[Dict[str, Any]], after: int = -1, mem_root: Optional[Path] = None
) -> List[str]:
    mroot = str(mem_root or MEM)
    aroot = str(AGENT)
    reads: List[str] = []
    for i, n, a in calls(st):
        if i <= after:
            continue
        if n in ("view_file", "grep_search", "find_by_name"):
            p = str(
                a.get("AbsolutePath")
                or a.get("SearchPath")
                or a.get("SearchDirectory")
                or ""
            )
            if p and (aroot not in p) and (mroot not in p) and ("/.gemini/" not in p):
                reads.append(p)
    return reads


def latest_solution_suffix(st: List[Dict[str, Any]], after: int = -1) -> Optional[str]:
    for i, _, a in reversed(list(calls(st))):
        if i <= after:
            continue
        m = re.search(r"09_tasks_solution_([^/\"\\]+)\.md$", tgt(a))
        if m:
            return m.group(1)
    return None


def active_ws(st: List[Dict[str, Any]], ws_root: Optional[Path] = None) -> Optional[str]:
    roots = [str(ws_root or WS_ROOT), "/memory/workspaces"]
    for _, _, a in reversed(list(calls(st))):
        dumped = json.dumps(a, ensure_ascii=False)
        for r in roots:
            m = re.search(re.escape(r) + r"/([^/\"\\]+)/", dumped)
            if m:
                return m.group(1)
    return None


def laya_gate_status(ws: Optional[str], stage: str, ws_root: Optional[Path] = None) -> Optional[str]:
    if not ws:
        return None
    primary = (ws_root or WS_ROOT) / ws
    candidates: List[Path] = []
    for folder in (primary, FALLBACK_WS_ROOT / ws):
        try:
            if folder.exists():
                candidates.extend(folder.glob(f"06_laya_summary_{stage}_*.json"))
        except OSError:
            continue
    if not candidates:
        return None
    latest = sorted(candidates, key=lambda p: p.name)[-1]
    try:
        return json.loads(latest.read_text(encoding="utf-8")).get("gate")
    except OSError:
        return None


def clean_room_config(
    ws: Optional[str], ws_root: Optional[Path] = None, suffix: Optional[str] = None
) -> Tuple[bool, List[str]]:
    if not ws:
        return False, []
    ws_dir = (ws_root or WS_ROOT) / ws
    req_files: List[Path] = []
    try:
        if ws_dir.exists():
            if suffix and (ws_dir / f"01_requirements_{suffix}.md").is_file():
                req_files = [ws_dir / f"01_requirements_{suffix}.md"]
            else:
                req_files = sorted(ws_dir.glob("01_requirements*.md"), key=lambda p: p.stat().st_mtime)
    except OSError:
        return False, []
    if not req_files:
        return False, []
    try:
        text = req_files[-1].read_text(encoding="utf-8", errors="ignore")
    except OSError:
        return False, []
    is_cr = bool(CLEAN_ROOM_MODE_RE.search(text))
    forbidden: List[str] = []
    m = CLEAN_ROOM_FORBIDDEN_RE.search(text)
    if m:
        for item in m.group(1).split(","):
            tok = item.strip().strip("`'\"")
            if tok and tok.lower() not in ("none", "n/a", "-"):
                forbidden.append(tok)
    return is_cr, forbidden


def contaminated_clean_room_reads(st: List[Dict[str, Any]], forbidden: List[str]) -> List[str]:
    if not forbidden:
        return []
    hits: List[str] = []
    for _, n, a in calls(st):
        if n in ("view_file", "grep_search", "find_by_name"):
            p = str(
                a.get("AbsolutePath")
                or a.get("SearchPath")
                or a.get("SearchDirectory")
                or ""
            ).replace("\\", "/")
            if not p:
                continue
            for fb in forbidden:
                fb_norm = fb.replace("\\", "/").rstrip("/")
                if fb_norm and (p.endswith(fb_norm) or f"/{fb_norm}" in p or p == fb_norm):
                    if fb not in hits:
                        hits.append(fb)
    return hits


def mutation_gate_status(
    ws: Optional[str], ws_root: Optional[Path] = None, suffix: Optional[str] = None
) -> Optional[str]:
    if not ws:
        return None
    ws_dir = (ws_root or WS_ROOT) / ws
    candidates: List[Path] = []
    try:
        if ws_dir.exists():
            if suffix:
                candidates = sorted(
                    ws_dir.glob(f"06_mutation_summary_{suffix}_*.json"), key=lambda p: p.name
                )
            if not candidates:
                candidates = sorted(ws_dir.glob("06_mutation_summary_*.json"), key=lambda p: p.name)
    except OSError:
        return None
    if not candidates:
        return None
    latest = candidates[-1]
    try:
        data = json.loads(latest.read_text(encoding="utf-8"))
        if (
            data.get("gate") == "PASS"
            and int(data.get("killed_count", 0)) >= 3
            and bool(data.get("all_restored_sha256", False))
        ):
            return "PASS"
        return str(data.get("gate") or "FAIL")
    except Exception:
        return "FAIL"


def audit_report_issues(
    ws: Optional[str],
    ws_root: Optional[Path] = None,
    suffix: Optional[str] = None,
) -> List[str]:
    if not ws:
        return []
    ws_dir = (ws_root or WS_ROOT) / ws
    candidates: List[Path] = []
    try:
        if ws_dir.exists():
            if suffix and (ws_dir / f"audit_report_{suffix}.md").is_file():
                candidates = [ws_dir / f"audit_report_{suffix}.md"]
            else:
                candidates = sorted(ws_dir.glob("audit_report*.md"), key=lambda p: p.stat().st_mtime)
    except OSError:
        return []
    if not candidates:
        return ["audit_report: chưa tồn tại file vật lý audit_report_<suffix>.md trong workspace"]
    latest = candidates[-1]
    try:
        text = latest.read_text(encoding="utf-8")
    except OSError:
        return [f"audit_report: không đọc được {latest.name}"]
    missing = [label for pat, label in AUDIT_REQUIRED_PATTERNS if not re.search(pat, text, re.I)]
    if missing:
        return [f"audit_report ({latest.name}) thiếu nội dung bắt buộc: " + "; ".join(missing)]
    return []


def manifest_for(file_path: str) -> Tuple[Optional[Path], Optional[Dict[str, Any]]]:
    """Find nearest .agent-project.json walking up from file_path."""
    try:
        resolved = Path(_unquote(file_path)).resolve()
    except OSError:
        return None, None
    for d in [resolved.parent, *resolved.parents]:
        mf = d / MANIFEST
        try:
            if mf.is_file():
                return d, json.loads(mf.read_text(encoding="utf-8"))
        except OSError:
            continue
    return None, None


def component_for(file_path: str) -> Tuple[Optional[Path], Optional[Dict[str, Any]]]:
    """Return (project_root, component_dict) with the longest matching path prefix."""
    clean_path = str(_unquote(file_path))
    root, mf = manifest_for(clean_path)
    if not root or not mf:
        return None, None
    try:
        rel = Path(clean_path).resolve().relative_to(root).as_posix()
    except ValueError:
        return None, None
    hits: List[Dict[str, Any]] = []
    for c in mf.get("components", []):
        cpath = c.get("path", "").rstrip("/")
        if cpath in ("", ".") or rel == cpath or rel.startswith(cpath + "/"):
            hits.append(c)
    best = max(hits, key=lambda c: len(c.get("path", "").rstrip("/")), default=None)
    return root, best


def handle_pre_invocation(payload: Dict[str, Any], state_dir: Optional[Path] = None) -> Dict[str, Any]:
    sdir = state_dir or STATE
    conv_id = payload.get("conversationId", "default")
    st = steps(payload)
    u_idx = last_user_idx(st)
    flag = sdir / f"{conv_id}.boot"
    if flag.exists():
        try:
            prev = flag.read_text(encoding="utf-8").strip()
        except OSError:
            prev = ""
        if prev == str(u_idx) or (prev == "" and u_idx < 0):
            return {}
    sdir.mkdir(parents=True, exist_ok=True)
    flag.write_text(str(u_idx), encoding="utf-8")
    try:
        up = (
            subprocess.run(
                ["lsof", "-nP", "-iTCP:8200", "-sTCP:LISTEN"],
                capture_output=True,
                timeout=3,
            ).returncode
            == 0
        )
    except OSError:
        up = False
    laya = (
        "UP"
        if up
        else f"DOWN — nhờ User chạy: {LAYA_START} (cold start ~30s, trả 503 khi đang nạp)"
    )
    return {"injectSteps": [{"ephemeralMessage": BOOT_MSG.format(laya=laya)}]}


def handle_pre_write(
    payload: Dict[str, Any],
    mem_root: Optional[Path] = None,
    ws_root: Optional[Path] = None,
) -> Dict[str, Any]:
    raw_args = (payload.get("toolCall") or {}).get("args") or {}
    args = {k: _unquote(v) for k, v in raw_args.items()}
    f = tgt(args)
    body = str(args.get("CodeContent") or "") + str(args.get("ReplacementContent") or "")

    # H2: Memory File Protection (Rule #4, #17)
    if is_mem_path(f, mem_root):
        try:
            exists = Path(f).exists()
        except PermissionError:
            exists = True
        if args.get("Overwrite") and exists:
            return {
                "decision": "deny",
                "reason": f"Rule 17: CẤM Overwrite memory file {f}. Chỉ APPEND bằng replace_file_content.",
            }
        if (
            APPEND_ONLY.search(f)
            and args.get("TargetContent")
            and str(args["TargetContent"]) not in str(args.get("ReplacementContent") or "")
        ):
            return {
                "decision": "deny",
                "reason": f"Rule 4/17: {Path(f).name} là file append-only — bắt buộc giữ nguyên TargetContent.",
            }

    # H3: Secret & PII Guard (Rule #20)
    m = SECRET.search(body)
    if m:
        return {
            "decision": "deny" if is_mem_path(f, mem_root) else "force_ask",
            "reason": f"Rule 20: phát hiện chuỗi giống secret ({m.group(0)[:10]}***). Mask trước khi ghi.",
        }

    # H4a: S1 Intake Gate before S3 Solution (Rule #4, #9, #16)
    sol_match = re.search(r"/workspaces/([^/]+)/09_tasks_solution_([^/]+)\.md$", f)
    if sol_match and is_mem_path(f, mem_root):
        ws_name, suffix = sol_match.group(1), sol_match.group(2)
        ws_dir = (ws_root or WS_ROOT) / ws_name
        st = steps(payload)
        u_idx = last_user_idx(st)
        try:
            other_sols = [
                p
                for p in ws_dir.glob("09_tasks_solution_*.md")
                if p.name != f"09_tasks_solution_{suffix}.md"
            ] if ws_dir.exists() else []
            any_req = bool(list(ws_dir.glob("01_requirements*.md"))) if ws_dir.exists() else False
            any_tsk = bool(list(ws_dir.glob("08_tasks*.md"))) if ws_dir.exists() else False
            exact_req = (ws_dir / f"01_requirements_{suffix}.md").is_file() if ws_dir.exists() else False
            exact_tsk = (ws_dir / f"08_tasks_{suffix}.md").is_file() if ws_dir.exists() else False
        except OSError:
            other_sols, any_req, any_tsk, exact_req, exact_tsk = [], True, True, True, True

        req_ok = exact_req or touched(st, r"01_requirements", u_idx) or (not other_sols and any_req)
        tsk_ok = exact_tsk or touched(st, r"08_tasks", u_idx) or (not other_sols and any_tsk)
        prg_ok = touched(st, r"05_progress", u_idx)

        missing_s1: List[str] = []
        if not req_ok:
            missing_s1.append(f"01_requirements_{suffix}.md")
        if not tsk_ok:
            missing_s1.append(f"08_tasks_{suffix}.md")
        if not prg_ok:
            missing_s1.append("append START vào 05_progress_*.md trong lượt này")
        if missing_s1:
            return {
                "decision": "deny",
                "reason": (
                    f"Rule 9 (S1 Intake trước S3 Solution): CẤM tạo 09_tasks_solution_{suffix}.md "
                    f"khi chưa hoàn thành S1: {', '.join(missing_s1)}."
                ),
            }
        is_cr, _ = clean_room_config(ws_name, ws_root, suffix)
        if is_cr:
            merged_sol = body
            try:
                sol_path = Path(f)
                if sol_path.is_file() and args.get("TargetContent") is not None:
                    existing_sol = sol_path.read_text(encoding="utf-8", errors="ignore")
                    merged_sol = existing_sol.replace(
                        str(args.get("TargetContent") or ""),
                        str(args.get("ReplacementContent") or ""),
                        1,
                    )
            except OSError:
                pass
            if not CLEAN_ROOM_SPEC_RE.search(merged_sol):
                return {
                    "decision": "deny",
                    "reason": (
                        f"Rule 9/13 (Clean-Room Mode): 01_requirements_{suffix}.md khai báo mode: clean-room, "
                        f"bắt buộc 09_tasks_solution_{suffix}.md phải có mục 'Clean-Room Approved Spec'."
                    ),
                }

    # H4b: Source Change Gate & Anti-Mock-Pollution (Rule #4, #10, #12, #13, #14, #16)
    if is_src(f, mem_root):
        if not is_test_file(f) and PROD_MOCK_POLLUTION.search(body):
            return {
                "decision": "deny",
                "reason": (
                    "Rule 12/14 (Anti-Smoke-Speculation): CẤM gắn hàm mock/closure giả (*Fn, sqlmock, fakemodel) "
                    "vào code production để lách tích hợp thật (L001/L056)!"
                ),
            }
        st = steps(payload)
        ws = active_ws(st, ws_root)
        if not ws:
            return {
                "decision": "deny",
                "reason": "Rule 4/16: chưa có workspace (01/05/08) cho thay đổi source code.",
            }
        if not touched(st, r"05_progress"):
            return {
                "decision": "deny",
                "reason": f"Rule 4: phải append START vào {ws}/05_progress_*.md trước khi sửa code.",
            }
        if laya_gate_status(ws, "dg1", ws_root) != "PASS":
            return {
                "decision": "deny",
                "reason": "DG1: giải pháp chưa qua laya_gate (PASS) — không được sửa source code.",
            }
        sfx = latest_solution_suffix(st, -1)
        is_cr, forbidden = clean_room_config(ws, ws_root, sfx)
        if is_cr and forbidden:
            tainted = contaminated_clean_room_reads(st, forbidden)
            if tainted:
                return {
                    "decision": "deny",
                    "reason": (
                        f"Rule 13 (Clean-Room Isolation): Phiên hiện tại đã đọc code gốc ({', '.join(tainted)}) "
                        "nên CẤM trực tiếp viết code mới. Hãy dùng invoke_subagent uỷ quyền cho Sub-agent "
                        "mới tinh chỉ nhận Clean-Room Approved Spec!"
                    ),
                }
        sol = [i for i, _, x in calls(st) if "09_tasks_solution" in tgt(x)]
        aud = [
            i
            for i, nm, x in calls(st)
            if nm in ("write_to_file", "replace_file_content") and "audit_report_" in tgt(x)
        ]
        first_sol_idx = sol[0] if sol else -1
        last_sol_idx = sol[-1] if sol else -1
        last_aud_idx = aud[-1] if aud else -1
        if last_aud_idx > last_sol_idx:
            return {
                "decision": "deny",
                "reason": (
                    "Rule 9/13 (S5 -> S3 -> S4 Self-Healing Loop): Đã bước vào chặng S5 Audit (audit_report_). "
                    "Nếu phát hiện bug/gap mới, CẤM sửa mò trực tiếp! Bắt buộc cập nhật bổ sung vào "
                    "09_tasks_solution_<suffix>.md + 08_tasks_<suffix>.md (kèm DG1/DG2) trước, sau đó hệ thống "
                    "sẽ tự động mở khoá cho sửa code dứt điểm ngay trong cùng lượt (không cần chờ User APPROVE lần 2)."
                ),
            }
        ok = any(
            s.get("type") == "USER_INPUT" and APPROVE.search(s.get("content", ""))
            for s in st[(first_sol_idx + 1 if first_sol_idx >= 0 else 0) :]
        )
        if not ok:
            return {
                "decision": "force_ask",
                "reason": "Rule 13: chưa thấy User APPROVE sau 09_tasks_solution trong phiên này. Cho phép sửa code?",
            }

    return {"decision": "ask"}


def handle_pre_command(payload: Dict[str, Any]) -> Dict[str, Any]:
    raw_cmd = ((payload.get("toolCall") or {}).get("args") or {}).get("CommandLine", "") or ""
    cmd = str(_unquote(raw_cmd))
    for pat, why in DENY_CMD:
        if re.search(pat, cmd):
            return {"decision": "deny", "reason": why}
    for pat, why in ASK_CMD:
        if re.search(pat, cmd, re.I):
            return {"decision": "force_ask", "reason": why}
    return {"decision": "ask"}


def handle_post_invocation(
    payload: Dict[str, Any],
    mem_root: Optional[Path] = None,
    state_dir: Optional[Path] = None,
) -> Dict[str, Any]:
    sdir = state_dir or STATE
    st = steps(payload)
    conv_id = payload.get("conversationId", "default")
    mark = sdir / f"{conv_id}.fb"
    last = int(mark.read_text(encoding="utf-8")) if mark.exists() else -1

    new_edits = [(i, f) for i, f in src_edits(st, mem_root) if i > last]
    if not new_edits:
        return {}

    comps: Dict[Tuple[str, str], Tuple[Path, Dict[str, Any]]] = {}
    for _, f in new_edits:
        root, c = component_for(f)
        if root and c:
            comps[(str(root), c["name"])] = (root / c.get("path", "."), c)

    sdir.mkdir(parents=True, exist_ok=True)
    mark.write_text(str(len(st) - 1), encoding="utf-8")

    errs: List[str] = []
    for (_, name), (cwd, c) in sorted(comps.items()):
        for cmd in c.get("checks", []):
            try:
                r = subprocess.run(cmd, shell=True, cwd=cwd, capture_output=True, text=True, timeout=40)
                if r.returncode != 0:
                    errs.append(f"[{name}] $ {cmd}\n{(r.stdout + r.stderr)[-1200:]}")
            except Exception as e:
                errs.append(f"[{name}] $ {cmd}\nError: {e}")

    u_idx = last_user_idx(st)
    loop_key = f"{conv_id}.{u_idx}.loop" if u_idx >= 0 else f"{conv_id}.loop"
    loop_file = sdir / loop_key

    edited_names = ", ".join(sorted({Path(f).name for _, f in new_edits}))
    audit_prompt = FINAL_AUDIT_MSG.format(files=edited_names)
    if errs:
        prev_loops = int(loop_file.read_text(encoding="utf-8")) if loop_file.exists() else 0
        cur_loops = prev_loops + 1
        loop_file.write_text(str(cur_loops), encoding="utf-8")
        escalation = ESCALATION_MENTOR_MSG.format(loops=cur_loops) + "\n\n" if cur_loops >= 3 else ""
        msg = escalation + "⚠️ H6 fast-feedback:\n" + "\n".join(errs) + "\n\n" + audit_prompt
    else:
        loop_file.write_text("0", encoding="utf-8")
        msg = audit_prompt
    return {"injectSteps": [{"ephemeralMessage": msg}]}


def handle_stop(
    payload: Dict[str, Any],
    mem_root: Optional[Path] = None,
    ws_root: Optional[Path] = None,
    state_dir: Optional[Path] = None,
    run_linter: bool = True,
) -> Dict[str, Any]:
    sdir = state_dir or STATE
    conv_id = payload.get("conversationId", "default")
    st = steps(payload)
    u_idx = last_user_idx(st)
    stop_key = f"{conv_id}.{u_idx}.stop" if u_idx >= 0 else f"{conv_id}.stop"
    cnt_file = sdir / stop_key
    legacy_cnt = sdir / f"{conv_id}.stop"
    if cnt_file.exists():
        n = int(cnt_file.read_text(encoding="utf-8"))
    elif u_idx < 0 and legacy_cnt.exists():
        n = int(legacy_cnt.read_text(encoding="utf-8"))
    else:
        n = 0
    if n >= 3:
        return {}

    ws = active_ws(st, ws_root)
    src = [(i, f) for i, f in src_edits(st, mem_root) if i > u_idx]
    miss: List[str] = []

    sol_touched = touched(st, r"09_tasks_solution", u_idx)
    if sol_touched:
        for g in ("dg1", "dg2"):
            if laya_gate_status(ws, g, ws_root) != "PASS":
                miss.append(f"{g.upper()}: laya_gate chưa PASS cho giải pháp/code demo")
        if not touched(st, r"05_progress", u_idx):
            miss.append("05_progress: chưa append tiến độ trong lượt lập 09_tasks_solution")

    if touched(st, r"(audit_report|13_analysis)", u_idx) and laya_gate_status(ws, "dg3", ws_root) != "PASS":
        miss.append("DG3: laya_gate chưa PASS cho báo cáo phân tích/audit")

    # Chặn Shadow Analysis / Investigation trả lời chay trên chat (Rule #4, #9)
    proj_reads = project_reads_since(st, u_idx, mem_root)
    if len(proj_reads) >= 4 and not touched(st, r"workspaces/", u_idx):
        miss.append(
            f"Rule 4/9 (No Shadow Files): Đã đọc/tra cứu {len(proj_reads)} file dự án nhưng chưa ghi hồ sơ "
            "vào memory/workspaces/<ws>/ (01_requirements, 05_progress, 08_tasks, 09_tasks_solution/13_analysis)."
        )

    sol_sfx = latest_solution_suffix(st, u_idx)
    if src:
        last = src[-1][0]
        task_sfx = sol_sfx or latest_solution_suffix(st, -1)
        if run_linter and not touched(st, r"\bgit\b.*\b(diff|show)\b", last):
            miss.append("Line-by-Line Diff: chưa chạy `git diff` hoặc `git show` thật sau lần sửa code cuối")
        if not touched(st, r"05_progress", last):
            miss.append("05_progress: chưa append kết quả sau lần sửa code cuối")
        if not touched(st, r"06_validation", last):
            miss.append("06_validation: chưa ghi bằng chứng test/runtime sau lần sửa code cuối")
        if not touched(st, r"audit_report_", last):
            miss.append("audit_report: chưa tạo/cập nhật audit_report_<suffix>.md sau lần sửa code cuối")
        miss.extend(audit_report_issues(ws, ws_root, task_sfx))
        if laya_gate_status(ws, "dg3", ws_root) != "PASS" or not touched(
            st, r"(laya_gate\.py.*(dg3|review)|06_laya_.*dg3)", last
        ):
            miss.append("DG3: chưa chạy lại /laya_router (dg3/review PASS) sau lần sửa code cuối")
        is_cr, _ = clean_room_config(ws, ws_root, task_sfx)
        mut_st = mutation_gate_status(ws, ws_root, task_sfx)
        if is_cr or mut_st is not None:
            if mut_st != "PASS" or not touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last):
                miss.append(
                    "Mutation Gate: chưa chạy mutation_gate.py (PASS với >=3 bản cố tình làm hỏng bị KILLED) sau lần sửa code cuối"
                )

        seen_comps = set()
        for _, f in src:
            _, c = component_for(f)
            rc = c.get("runtime_check") if c else None
            if rc and c["name"] not in seen_comps and not touched(st, re.escape(rc), last):
                seen_comps.add(c["name"])
                miss.append(f"Runtime: sửa component `{c['name']}` nhưng chưa chạy `{rc}` sau lần sửa cuối")

    if run_linter and ws and (src or touched(st, r"workspaces/", u_idx)):
        linter = AGENT / "tooling" / "verify_governance.py"
        if linter.is_file():
            cmd_args = [sys.executable, str(linter), "--workspace", ws]
            if sol_touched and not src:
                cmd_args.extend(["--stage", "S3"])
            if sol_sfx:
                ws_dir = (ws_root or WS_ROOT) / ws
                if ws_dir.exists() and (ws_dir / f"01_requirements_{sol_sfx}.md").is_file():
                    cmd_args.extend(["--suffix", sol_sfx])
            r = subprocess.run(
                cmd_args,
                capture_output=True,
                text=True,
                timeout=30,
            )
            if r.returncode != 0:
                miss.append("verify_governance.py FAILED:\n" + (r.stdout + r.stderr)[-600:])

    if run_linter and src and n == 0:
        sdir.mkdir(parents=True, exist_ok=True)
        cnt_file.write_text("1", encoding="utf-8")
        extra_miss = ("\n\n⚠️ Các mục Done-gate còn thiếu:\n- " + "\n- ".join(miss)) if miss else ""
        return {
            "decision": "continue",
            "reason": (
                "⛔ MANDATORY ADVERSARIAL STOP-AUDIT (H7 Pass #1 — Chống báo cáo OK sớm khi còn bug/lỗ hổng):\n"
                + VERBATIM_USER_AUDIT_PROMPT
                + "\n\n👉 BẮT BUỘC: Thực hiện rà soát phản biện thực sự từng dòng `git diff` vừa sửa. "
                "Nếu phát hiện bất kỳ bug, lỗ hổng hay thiếu sót nào: CẤM báo cáo đẩy trách nhiệm cho User và CẤM bắt User APPROVE lại! "
                "Hãy cập nhật ngay `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` + chạy `DG1/DG2`, "
                "tự sửa dứt điểm source code, chạy lại test + `DG3` + cập nhật `audit_report_<suffix>.md` + `06_validation_<suffix>.md` + `05_progress` "
                "rồi mới kết thúc lượt!"
                + extra_miss
            ),
        }

    if not miss:
        return {}

    sdir.mkdir(parents=True, exist_ok=True)
    cnt_file.write_text(str(n + 1), encoding="utf-8")
    reason_msg = "⛔ Done-gate (H7) chưa đạt:\n- " + "\n- ".join(miss)
    if src:
        reason_msg += "\n\n🔍 MỆNH LỆNH AUDIT BẮT BUỘC CỦA USER:\n" + VERBATIM_USER_AUDIT_PROMPT
    return {"decision": "continue", "reason": reason_msg}


def dispatch(event_name: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    handlers = {
        "pre-invocation": handle_pre_invocation,
        "pre-write": handle_pre_write,
        "pre-command": handle_pre_command,
        "post-invocation": handle_post_invocation,
        "stop": handle_stop,
    }
    if event_name not in handlers:
        return {}
    return handlers[event_name](payload)


def main() -> int:
    if len(sys.argv) < 2:
        print("{}")
        return 0
    raw_in = sys.stdin.read().strip()
    payload = json.loads(raw_in) if raw_in else {}
    res = dispatch(sys.argv[1], payload)
    print(json.dumps(res, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
