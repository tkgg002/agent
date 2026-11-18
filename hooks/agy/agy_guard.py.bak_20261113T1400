#!/usr/bin/env python3
"""Antigravity Lifecycle Hook Dispatcher (H1-H7).

Usage:
  python3 agy_guard.py pre-invocation|pre-write|pre-command|pre-subagent|post-invocation|stop

Hooks implemented:
  H1 boot          (PreInvocation)  : Injects S0 Boot checklist + Laya server status (:8200) + matched Lesson Tripwires.
  H2 memory_guard  (PreToolUse)     : Denies Overwrite on existing memory files; denies removing
                                      TargetContent on append-only files; enforces Global Lesson Hygiene on lessons.md.
  H3 secret_guard  (PreToolUse)     : Denies unmasked secrets/URIs in memory files; force_ask in source files.
  H4 change_gate   (PreToolUse)     : Enforces Lesson Tripwires (H4t on both docs & source code), H4c Claim-Evidence,
                                      H4a S1-before-S3, and H4b Anti-Mock-Pollution + DG0/DG1 + User APPROVE.
  H5 command_guard (PreToolUse)     : Denies git push (Rule #15), git reset --hard / clean -f (Rule #19),
                                      rm -rf memory/.git (Rule #17); force_ask on direct DB write CLI (Rule #12).
  H6 fast_feedback (PostInvocation) : Runs component checks from nearest .agent-project.json after source edits
                                      AND auto-injects Mandatory Final Adversarial Audit & Anti-Smoke-Speculation protocol.
  H7 done_gate     (Stop)           : Blocks premature completion if lessons index sync, DG0/DG1/DG2/DG3, 05_progress,
                                      06_validation, audit_report_<suffix>.md (6 sections), component runtime_check, or
                                      verify_governance.py are missing/failing (max 3 blocks per turn).
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

SRC_EXT = {".go", ".ts", ".tsx", ".js", ".py", ".sql", ".css", ".html"}
APPEND_ONLY = re.compile(r"(05_progress[^/]*|04_decisions[^/]*|lessons|governance_metrics)\.md$")
SECRET = re.compile(
    r"(AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|ATATT[\w\-]{20,}|ghp_\w{30,}"
    r"|Bearer\s+[\w.\-]{20,}|(mongodb|postgres(ql)?|mysql|redis)://[^*\s:/%]+:[^*\s@%]+@"
    r"|password\s*[:=]\s*['\"][^\s*'\"]{6,}['\"])",
    re.I,
)
# G-01: neo đầu dòng — chặn false-positive khi từ khoá xuất hiện giữa câu mô tả ("...đã được Jev duyệt...").
APPROVE = re.compile(
    r"^\s*(?:ok\s+|ừ\s+|được\s+)?(approve|duyệt|bắt\s*đầu|ok\s*(sửa|làm|fix)|làm đi|sửa đi|fix đi|triển khai|thực hiện đi|chặn\s*đứng|thêm.*vào.*cho\s*tao|vòng\s+lặp\s+phản\s+tỉnh|self-improvement\s+loop|update\s+agy_g)",
    re.I | re.M,
)
# L02 & L14: Agnostic Anti-Mock-Pollution (không chứa tên biến riêng của dự án con)
PROD_MOCK_POLLUTION = re.compile(
    r"(\b\w+Fn\s+func\s*\(|\bsqlmock\b|\bfakemodel\b|\bfake\w+Repo\b)",
)
FAKE_REAL_TEST_RE = re.compile(
    r"func\s+Test\w*(RealPostgres|RealDB|RealNATS|RealMongo)\w*[\s\S]{0,1500}(sqlmock\.New|sqlite\.Open\(\"file::memory:|httptest\.NewServer)",
)

# H2b: Global Lesson Hygiene Guard constants (bảo vệ cấu trúc lessons.md ngay tại pre-write)
LESSON_H3_RE = re.compile(r"^###\s+\[(\d{4}-\d{2}-\d{2})\]\s+(.+)$")
LESSON_PAREN_EN_RE = re.compile(r"\(([^()]{10,})\)\s*$")
LESSON_REQUIRED_BULLETS = (
    "- **Global Pattern:**",
    "- **Bối cảnh (Trigger):**",
    "- **Root Cause:**",
    "- **Fix/Correct Flow:**",
    "- **Tags:**",
)
LESSON_LOCAL_PATH_RE = re.compile(r"/Users/\w+/Documents/work/", re.I)
LESSON_MAX_LINES = 12

CLEAN_ROOM_MODE_RE = re.compile(r"^\s*mode\s*:\s*clean-room\b", re.I | re.M)
CLEAN_ROOM_FORBIDDEN_RE = re.compile(r"^\s*clean_room_forbidden_sources\s*:\s*(.+)$", re.I | re.M)
CLEAN_ROOM_SPEC_RE = re.compile(r"clean-room\s+approved\s+spec", re.I)
# G-01: task clean-room chỉ mở S4 khi cả dòng là lệnh khởi động.
CLEAN_ROOM_START = re.compile(r"^\s*(BẮT\s*ĐẦU|APPROVE)\s*[.!]?\s*$", re.I | re.M)
# G-03: mỗi rule trong Clean-Room Approved Spec phải có REF: path:line.
SPEC_RULE_RE = re.compile(r"^[ \t]*(?:[-*][ \t]*|\|[ \t]*)?\**(C?R-?\d+)\b[^:\n|]*\**[ \t]*[:.|]", re.I | re.M)
SPEC_REF_RE = re.compile(r"REF:\s*`?\S+:\d+")
SPEC_REJECT_RE = re.compile(r"(\[(?:REJECTED|NO_EVIDENCE|UNVERIFIED|BLOCKED)\]|trạng\s+thái\s*:\s*(?:REJECTED|BLOCK|NO_EVIDENCE))", re.I)
# G-02: tool-call args có thể chứa nội dung/đường dẫn file đọc được.
READ_ARG_KEYS = {
    "view_file": ("AbsolutePath",),
    "read_file": ("Path",),
    "view_file_outline": ("AbsolutePath", "File"),
    "view_code_item": ("AbsolutePath", "File"),
    "grep_search": ("SearchPath",),
    "find_by_name": ("SearchDirectory",),
    "run_command": ("CommandLine", "Cwd"),
    "read_url_content": ("Url",),
}
WRITE_TOOLS = frozenset(
    {
        "write_to_file",
        "replace_file_content",
        "multi_replace_file_content",
        "write_file",
        "edit_file",
        "multi_edit",
    }
)
FULL_WRITE_TOOLS = frozenset({"write_to_file", "write_file"})
MAX_SOLUTION_LINES = 800
# G-08: lesson tripwires.
TRIPWIRE_LIMIT = 5
TRIPWIRE_GUARDED_DOCS = re.compile(r"/workspaces/[^/]+/(13_analysis|09_tasks_solution|audit_report)_[^/]*\.md$")
# AF-1: bằng chứng Require chỉ tính từ tool đọc/lệnh — tool ghi (kể cả lần ghi bị chặn) không được tự thoả.
TRIPWIRE_EVIDENCE_TOOLS = frozenset(
    {"view_file", "read_file", "grep_search", "find_by_name", "list_dir", "run_command", "read_url_content", "search_web"}
)
# AF-5: ranh giới token trong args JSON (khoảng trắng, quote, toán tử shell, backslash escape, '=').
TRIPWIRE_TOKEN_BOUNDARY = frozenset(" \t\n\"'`;|<>()\\=")
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
    "BOOT S0: (1) Đọc agent/GEMINI.md + memory/global/lessons/INDEX.md (CẤM đọc nguyên file lessons.md gốc). "
    "(2) Phân loại T0/Audit/T1-T3 (kiểm tra lỗi/phân tích/thiết kế/sửa code >= T2, CẤM trả lời chay trên chat). "
    "(3) S1 Intake (T1+): tạo 01_requirements_<suffix>.md + 08_tasks_<suffix>.md + append 05_progress START TRƯỚC khi chạy S2.5 (DG0) và lập 09_tasks_solution. "
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
    "   - Đối chiếu lessons/INDEX.md; nếu phát hiện lỗi/anti-pattern mới, ghi vào lessons.md (chuẩn Global <=12 dòng) + chạy build_lessons_index.py.\n"
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
    "2. Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py run-recursion --use-live-model` để khám phá cải tiến trực giao INV/DOM hoặc tra cứu `memory/global/harness_library/`.\n"
    "3. Nếu giải pháp hiện tại không khả thi, dừng sửa code và báo cáo rõ cho User để Re-plan (S3)."
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
    return str(_unquote(args.get("TargetFile") or args.get("Path") or args.get("file_path") or ""))


def _parse_chunks(raw_args: Dict[str, Any]) -> List[Dict[str, Any]]:
    chunks = raw_args.get("ReplacementChunks") or raw_args.get("Edits") or raw_args.get("edits") or []
    if isinstance(chunks, str):
        try:
            chunks = json.loads(chunks)
        except Exception:
            return []
    return [ch for ch in chunks if isinstance(ch, dict)] if isinstance(chunks, list) else []


def _chunk_texts(raw_args: Dict[str, Any], keys: Tuple[str, ...]) -> str:
    return "\n".join(
        str(_unquote(ch.get(k) or ""))
        for ch in _parse_chunks(raw_args)
        for k in keys
        if ch.get(k)
    )


def old_arg(raw_args: Dict[str, Any]) -> str:
    base = str(_unquote(raw_args.get("TargetContent") or raw_args.get("Target") or raw_args.get("old_string") or ""))
    extra = _chunk_texts(raw_args, ("TargetContent", "Target", "oldText", "old_string"))
    return f"{base}\n{extra}".strip("\n") if extra else base


def new_arg(raw_args: Dict[str, Any]) -> str:
    base = (
        str(_unquote(raw_args.get("CodeContent") or ""))
        + str(_unquote(raw_args.get("ReplacementContent") or ""))
        + str(_unquote(raw_args.get("Content") or ""))
        + str(_unquote(raw_args.get("Replacement") or ""))
        + str(_unquote(raw_args.get("new_string") or ""))
    )
    extra = _chunk_texts(raw_args, ("ReplacementContent", "Replacement", "newText", "new_string"))
    return f"{base}\n{extra}".strip("\n") if extra else base


def _apply_edit_preview(cur_txt: str, raw_args: Dict[str, Any], old_s: str, body: str) -> str:
    chunks = _parse_chunks(raw_args)
    if chunks:
        merged = cur_txt
        for ch in chunks:
            c_old = str(_unquote(ch.get("TargetContent") or ch.get("Target") or ch.get("oldText") or ch.get("old_string") or ""))
            c_new = str(_unquote(ch.get("ReplacementContent") or ch.get("Replacement") or ch.get("newText") or ch.get("new_string") or ""))
            if c_old:
                merged = merged.replace(c_old, c_new, 1)
        return merged
    return cur_txt.replace(old_s, body, 1) if old_s else body


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
        i > after
        and (
            (nm in WRITE_TOOLS and bool(re.search(pattern, tgt(a) or json.dumps(a, ensure_ascii=False))))
            or (nm == "run_command" and bool(re.search(pattern, str(a.get("CommandLine") or ""))))
        )
        for i, nm, a in calls(st)
    )


def src_edits(st: List[Dict[str, Any]], mem_root: Optional[Path] = None) -> List[Tuple[int, str]]:
    return [
        (i, tgt(a))
        for i, n, a in calls(st)
        if n in WRITE_TOOLS and is_src(tgt(a), mem_root)
    ]


def lessons_edits_since(st: List[Dict[str, Any]], after: int = -1) -> List[int]:
    return [
        i
        for i, n, a in calls(st)
        if i > after
        and n in WRITE_TOOLS
        and tgt(a).replace("\\", "/").endswith("/memory/global/lessons.md")
    ]


def validate_lesson_append_hygiene(args: Dict[str, Any]) -> List[str]:
    """H2b: Chặn ngay tại pre-write nếu Agent append bài học mới sai cấu trúc Global vào lessons.md."""
    old_text, new_text = old_arg(args), new_arg(args)
    if old_text and new_text.startswith(old_text):
        added = new_text[len(old_text) :]
    elif old_text and old_text in new_text:
        added = new_text.replace(old_text, "", 1)
    else:
        added = new_text

    if not added.strip():
        return []

    lines = added.splitlines()
    h3_indices = [i for i, ln in enumerate(lines) if ln.startswith("### ")]
    if not h3_indices:
        return [
            "Mọi bài học append vào `lessons.md` bắt buộc phải bắt đầu bằng Header `### [YYYY-MM-DD] <Tiêu đề> (<English Summary>)`."
        ]

    issues: List[str] = []
    for idx, start_i in enumerate(h3_indices):
        end_i = h3_indices[idx + 1] if idx + 1 < len(h3_indices) else len(lines)
        block = [ln for ln in lines[start_i:end_i] if ln.strip() and not ln.startswith("## ")]
        header_line = block[0]
        m = LESSON_H3_RE.match(header_line)
        if not m:
            issues.append(f"Header '{header_line[:60]}' sai cú pháp `### [YYYY-MM-DD] <Tiêu đề> (<English Summary>)`.")
        elif not LESSON_PAREN_EN_RE.search(m.group(2)):
            issues.append(
                f"Header '{header_line[:60]}' thiếu cụm tóm tắt Tiếng Anh trong ngoặc đơn `(...)` (>=10 ký tự) ở cuối dòng."
            )
        if len(block) > LESSON_MAX_LINES:
            issues.append(
                f"Mục bài học '{header_line[:50]}' dài {len(block)} dòng (vượt trần {LESSON_MAX_LINES} dòng). "
                "Bắt buộc tóm tắt gọn đúng 1 dòng cho mỗi trường!"
            )
        raw_block = "\n".join(block)
        for req_b in LESSON_REQUIRED_BULLETS:
            if req_b not in raw_block:
                issues.append(f"Mục '{header_line[:50]}' thiếu trường bắt buộc `{req_b}`.")
        if LESSON_LOCAL_PATH_RE.search(raw_block):
            issues.append(
                f"Mục '{header_line[:50]}' chứa đường dẫn cục bộ `/Users/...`. Bắt buộc trừu tượng hóa 100% thành Global Pattern!"
            )
    return issues


def project_reads_since(
    st: List[Dict[str, Any]], after: int = -1, mem_root: Optional[Path] = None
) -> List[str]:
    mroot = str(mem_root or MEM)
    aroot = str(AGENT)
    reads: List[str] = []
    for i, n, a in calls(st):
        if i <= after:
            continue
        if n in ("view_file", "read_file", "grep_search", "find_by_name"):
            p = str(
                a.get("AbsolutePath")
                or a.get("Path")
                or a.get("SearchPath")
                or a.get("SearchDirectory")
                or ""
            )
            if p and (aroot not in p) and (mroot not in p) and ("/.gemini/" not in p):
                reads.append(p)
    return reads


def latest_solution_suffix(st: List[Dict[str, Any]], after: int = -1) -> Optional[str]:
    for i, nm, a in reversed(list(calls(st))):
        if i <= after or nm not in WRITE_TOOLS:
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
    for s in st:
        if s.get("type") == "SYSTEM_MESSAGE" and "sender=" in str(s.get("content", "")):
            s_blob = str(s.get("content", "")).replace("\\", "/")
            for fb in forbidden:
                fb_norm = fb.replace("\\", "/").rstrip("/")
                if fb_norm and fb_norm in s_blob and fb not in hits:
                    hits.append(fb)
    for _, n, a in calls(st):
        keys = READ_ARG_KEYS.get(n or "")
        if not keys:
            continue
        blob = " ".join(str(a.get(k) or "") for k in keys).replace("\\", "/")
        if not blob.strip():
            continue
        for fb in forbidden:
            fb_norm = fb.replace("\\", "/").rstrip("/")
            if fb_norm and fb_norm in blob and fb not in hits:
                hits.append(fb)
    return hits


def leaked_subagent_prompts(st: List[Dict[str, Any]], forbidden: List[str], min_line: int = 40) -> List[str]:
    """G-04: invoke_subagent/send_message chứa path cấm hoặc dòng code (>= min_line ký tự) của file cấm."""
    if not forbidden:
        return []
    prompts = [
        json.dumps(a.get("Subagents") or a.get("Message") or a, ensure_ascii=False).replace("\\\\", "/").replace("\\", "")
        for _, n, a in calls(st)
        if n in ("invoke_subagent", "send_message")
    ]
    if not prompts:
        return []
    raw_prompts = [
        (
            " ".join(str(sa.get("Prompt") or "") for sa in (a.get("Subagents") or []) if isinstance(sa, dict))
            if n == "invoke_subagent"
            else str(a.get("Message") or "")
        )
        for _, n, a in calls(st)
        if n in ("invoke_subagent", "send_message")
    ]
    hits: List[str] = []
    cwds = {str(a.get("Cwd")) for _, n, a in calls(st) if n == "run_command" and a.get("Cwd")}
    for fb in forbidden:
        fb_norm = fb.replace("\\", "/").rstrip("/")
        if not fb_norm:
            continue
        leaked = any(fb_norm in p for p in prompts)
        if not leaked:
            cands = [Path(fb)] if Path(fb).is_absolute() else [Path(fb)] + [Path(c) / fb for c in sorted(cwds)]
            lines: List[str] = []
            for fpath in cands:
                try:
                    if fpath.is_file():
                        lines = [ln.strip() for ln in fpath.read_text(encoding="utf-8", errors="ignore").splitlines()]
                        break
                except OSError:
                    continue
            long_lines = [ln for ln in lines if len(ln) >= min_line]
            leaked = any(ln in p for ln in long_lines for p in raw_prompts)
        if leaked and fb not in hits:
            hits.append(fb)
    return hits


def user_approved(st: List[Dict[str, Any]], start: int, clean_room: bool = False) -> bool:
    """G-01: clean-room chỉ chấp nhận dòng 'BẮT ĐẦU'/'APPROVE' đứng riêng; task thường dùng APPROVE neo đầu dòng."""
    pat = CLEAN_ROOM_START if clean_room else APPROVE
    return any(
        s.get("type") == "USER_INPUT" and pat.search(str(s.get("content", "")))
        for s in st[max(start, 0) :]
    )


def spec_rule_issues(text: str) -> List[str]:
    """G-03: mục 'Clean-Room Approved Spec' phải có >=1 rule R<n>/CR-<n> và mọi rule/bullet có REF: path:line."""
    m = re.search(r"^(#+)\s*.*clean-room\s+approved\s+spec.*$", text, re.I | re.M)
    if not m:
        return ["thiếu mục 'Clean-Room Approved Spec'"]
    level = len(m.group(1))
    rest = text[m.end() :]
    nxt = re.search(rf"^#{{1,{level}}}\s", rest, re.M)
    section = rest[: nxt.start()] if nxt else rest
    rules = [(rm.group(1).upper(), section[rm.start() :].split("\n", 1)[0]) for rm in SPEC_RULE_RE.finditer(section)]
    if not rules:
        return ["Clean-Room Approved Spec chưa có rule nào dạng `R<n>: <hành vi> — REF: path:line`"]
    issues: List[str] = []
    for rid, line in rules:
        if not SPEC_REF_RE.search(line):
            issues.append(f"{rid} thiếu bằng chứng `REF: path:line`")
        rej = SPEC_REJECT_RE.search(line)
        if rej:
            issues.append(f"{rid} đang ở trạng thái chưa duyệt ({rej.group(0)})")
    for raw_ln in section.splitlines():
        if re.match(r"^[ \t]*[-*][ \t]+\S", raw_ln) and not SPEC_RULE_RE.match(raw_ln):
            if not SPEC_REF_RE.search(raw_ln):
                issues.append(f"Dòng bullet '{raw_ln.strip()[:40]}' thiếu mã R<n> hoặc thiếu bằng chứng `REF: path:line`")
    return issues


def _infer_tripwire_exts(trig: str) -> List[str]:
    has_go = bool(re.search(r"(?:\\\.|\.)go\b|\bfunc\s*\(|goroutine|errgroup|sqlDB\.Close|natsserver\.NewServer", trig))
    has_py = bool(re.search(r"(?:\\\.|\.)py\b|\bdef\s+\w+\s*\(|PYTHONDONTWRITEBYTECODE|ast\.parse", trig))
    has_ts = bool(re.search(r"(?:\\\.|\.)(?:tsx|ts|jsx|js)\b|Form\.useWatch|encodeURIComponent", trig))
    has_css = bool(re.search(r"(?:\\\.|\.)(?:css|scss)\b|#sri-markdown-content|\.sri-pre-block", trig))
    if sum([has_go, has_py, has_ts, has_css]) != 1:
        return []
    if has_go:
        return [".go"]
    if has_py:
        return [".py"]
    if has_ts:
        return [".tsx", ".ts", ".jsx", ".js"]
    return [".css", ".scss"]


def load_tripwires(mem_root: Optional[Path] = None) -> List[Dict[str, Any]]:
    base = (mem_root or MEM) / "global"
    path = base / "lessons" / "tripwires.json"
    if not path.is_file() and (base / "tripwires.json").is_file():
        path = base / "tripwires.json"
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
        if isinstance(data, dict) and "tripwires" in data:
            data = data["tripwires"]
    except (OSError, ValueError):
        return []
    if not isinstance(data, list):
        return []
    out: List[Dict[str, Any]] = []
    for t in data:
        if not isinstance(t, dict) or not t.get("id") or not t.get("check"):
            continue
        trig = (
            t.get("trigger_regex")
            or t.get("trigger")
            or t.get("pattern")
            or t.get("forbid_regex")
            or t.get("forbid")
            or t.get("Forbid")
            or ""
        )
        if not trig:
            continue
        item = dict(t)
        item.setdefault("trigger_regex", str(trig))
        if not any(
            k in item
            for k in ("ext", "extensions", "file_ext", "glob", "file_glob", "files", "path_regex", "file_regex", "file_pattern", "lang", "language")
        ):
            inferred = _infer_tripwire_exts(str(trig))
            if inferred:
                item["ext"] = inferred
        out.append(item)
    return out


def match_tripwires(prompt: str, trips: List[Dict[str, Any]], limit: int = TRIPWIRE_LIMIT) -> List[Dict[str, Any]]:
    hits: List[Dict[str, Any]] = []
    for t in trips:
        trig = t.get("trigger") or t.get("trigger_regex") or ""
        if not trig:
            continue
        try:
            if re.search(trig, prompt or "", re.I):
                hits.append(t)
        except re.error:
            continue
        if len(hits) >= limit:
            break
    return hits


def _tripwire_matches_file(file_path: str, t: Dict[str, Any]) -> bool:
    if not file_path:
        return True
    from fnmatch import fnmatch

    fp = file_path.replace("\\", "/")
    p = Path(fp)
    ext = p.suffix.lower()
    exts = t.get("ext") or t.get("extensions") or t.get("file_ext")
    if exts:
        ext_list = [exts] if isinstance(exts, str) else list(exts)
        norm_exts = {str(e).lower() if str(e).startswith(".") else f".{str(e).lower()}" for e in ext_list if e}
        if norm_exts and ext not in norm_exts:
            return False
    lang = str(t.get("lang") or t.get("language") or "").strip().lower()
    if lang:
        lang_map = {
            "go": {".go"},
            "golang": {".go"},
            "py": {".py"},
            "python": {".py"},
            "ts": {".ts", ".tsx"},
            "typescript": {".ts", ".tsx"},
            "js": {".js", ".jsx"},
            "javascript": {".js", ".jsx"},
            "sql": {".sql"},
            "css": {".css", ".scss"},
        }
        if lang in lang_map and ext not in lang_map[lang]:
            return False
    globs = t.get("glob") or t.get("file_glob") or t.get("files")
    if globs:
        glob_list = [globs] if isinstance(globs, str) else list(globs)
        if glob_list and not any(
            fnmatch(fp.lower(), str(g).lower()) or fnmatch(p.name.lower(), str(g).lower()) for g in glob_list if g
        ):
            return False
    path_re = str(t.get("path_regex") or t.get("file_regex") or t.get("file_pattern") or "").strip()
    if path_re:
        try:
            if not re.search(path_re, fp, re.I):
                return False
        except re.error:
            pass
    return True


def code_tripwire_violations(
    file_path: str,
    new_code_body: str,
    trips: List[Dict[str, Any]],
    full_code_body: Optional[str] = None,
) -> List[Dict[str, Any]]:
    """H4t-Code: Kiểm tra trực tiếp nội dung code mới chuẩn bị ghi đối chiếu với các Tripwires có Require/Forbid."""
    if not new_code_body.strip():
        return []
    req_context = full_code_body if full_code_body is not None else new_code_body
    out: List[Dict[str, Any]] = []
    for t in trips:
        tid = str(t.get("id") or "")
        # Bỏ qua các tripwire chỉ dành cho tra cứu tài liệu/workspace ở pha Brain
        if "prior-workspace" in tid or "propose-only" in tid or "agnostic-core" in tid or "worktree-isolation" in tid:
            continue
        if not _tripwire_matches_file(file_path, t):
            continue
        trig = str(t.get("trigger") or t.get("trigger_regex") or t.get("pattern") or "")
        req = str(t.get("require") or t.get("Require") or t.get("require_regex") or "")
        forbid = str(t.get("forbid") or t.get("Forbid") or t.get("forbid_regex") or "")
        if not trig and not forbid:
            continue
        try:
            if trig and not re.search(trig, new_code_body, re.I):
                continue
            if forbid and re.search(forbid, new_code_body, re.I):
                out.append(t)
                continue
            if req and not re.search(req, req_context, re.I):
                out.append(t)
        except re.error:
            continue
    return out


_DEMO_FENCE_RE = re.compile(r"```([a-zA-Z0-9_\-+]*)\n(.*?)```", re.DOTALL)
_BEFORE_LABEL_RE = re.compile(r"\b(before|hiện tại|current|old|cũ|trước khi sửa)\b", re.IGNORECASE)
_AFTER_LABEL_RE = re.compile(r"\b(after|mới|new|sau|fix|đề xuất|proposed|bản vá)\b", re.IGNORECASE)
_LANG_EXT_MAP = {
    "go": ".go",
    "golang": ".go",
    "python": ".py",
    "py": ".py",
    "ts": ".ts",
    "tsx": ".tsx",
    "typescript": ".ts",
    "js": ".js",
    "jsx": ".jsx",
    "javascript": ".js",
    "sql": ".sql",
    "css": ".css",
    "sh": ".sh",
    "bash": ".sh",
}


def _is_before_only_context(prefix: str) -> bool:
    lines = [ln.strip() for ln in prefix.strip().splitlines() if ln.strip()]
    if not lines:
        return False
    last_ln = lines[-1]
    if not (last_ln.startswith("#") or last_ln.startswith("**") or last_ln.startswith("- **")):
        return False
    return bool(_BEFORE_LABEL_RE.search(last_ln)) and not bool(_AFTER_LABEL_RE.search(last_ln))


def _extract_demo_snippets(md_text: str) -> List[Tuple[str, str]]:
    """F-04/N-03: Tách chỉ các dòng code đề xuất từ 09_tasks_solution (bỏ dòng '-' trong diff và block Before theo L026)."""
    snippets: List[Tuple[str, str]] = []
    for m in _DEMO_FENCE_RE.finditer(md_text):
        lang = (m.group(1) or "").strip().lower()
        block = m.group(2) or ""
        if lang in ("diff", "patch") or block.lstrip().startswith(("--- ", "@@ ", "diff --git")):
            cur_file = "_demo.txt"
            plus_by_file: Dict[str, List[str]] = {}
            for ln in block.splitlines():
                if ln.startswith("diff --git "):
                    parts = ln.split()
                    if len(parts) >= 4 and parts[3].startswith("b/"):
                        cur_file = parts[3][2:]
                elif ln.startswith("--- ") and cur_file == "_demo.txt":
                    raw_a = ln[4:].strip().split("\t")[0]
                    if raw_a.startswith("a/"):
                        cur_file = raw_a[2:]
                elif ln.startswith("+++ "):
                    raw_p = ln[4:].strip().split("\t")[0]
                    if raw_p.startswith("b/"):
                        raw_p = raw_p[2:]
                    if raw_p and raw_p != "/dev/null":
                        cur_file = raw_p
                elif ln.startswith("+") and not ln.startswith("+++"):
                    plus_by_file.setdefault(cur_file, []).append(ln[1:])
            for fp_name, lines in plus_by_file.items():
                if lines:
                    snippets.append((fp_name, "\n".join(lines)))
            continue
        prefix = md_text[max(0, m.start() - 240) : m.start()]
        if _is_before_only_context(prefix):
            continue
        ext = _LANG_EXT_MAP.get(lang, "")
        target_m = re.search(r"(?:#|//|--)\s*Target[^:\n]*:\s*([^\s:`\n]+)", block)
        vpath = target_m.group(1) if target_m else (f"_demo{ext}" if ext else "_demo.txt")
        snippets.append((vpath, block))
    return snippets


def tripwire_violations(
    st: List[Dict[str, Any]], trips: List[Dict[str, str]], exclude_ws: Optional[str] = None
) -> List[Dict[str, str]]:
    """G-08: tripwire có Require mà sau prompt khớp Trigger chưa có tool call nào khớp Require."""

    def _token(blob: str, s: int, e: int) -> str:
        while s > 0 and blob[s - 1] not in TRIPWIRE_TOKEN_BOUNDARY:
            s -= 1
        while e < len(blob) and blob[e] not in TRIPWIRE_TOKEN_BOUNDARY:
            e += 1
        return blob[s:e].rstrip(",.")

    def _proves(req: str, blob: str) -> bool:
        for m in re.finditer(req, blob, re.I):
            hit = m.group(0)
            if "$" in hit or (exclude_ws and f"/{exclude_ws}/" in hit):
                continue
            tok = _token(blob, m.start(), m.end())
            if tok.startswith("/") and not tok.startswith("//"):
                try:
                    if not Path(tok).exists():
                        continue
                except OSError:
                    continue
            return True
        return False

    out: List[Dict[str, str]] = []
    for t in trips:
        req = t.get("require") or t.get("require_regex")
        if not req:
            continue
        trig_idx = [
            i
            for i, s in enumerate(st)
            if s.get("type") == "USER_INPUT" and match_tripwires(str(s.get("content", "")), [t], 1)
        ]
        if not trig_idx:
            continue
        first = trig_idx[0]
        try:
            ok = any(
                i > first
                and n in TRIPWIRE_EVIDENCE_TOOLS
                and _proves(req, json.dumps(a, ensure_ascii=False))
                for i, n, a in calls(st)
            )
        except re.error:
            continue
        if not ok:
            out.append(t)
    return out


CLAIM_RE = re.compile(r"\b(PASS(?:ED)?|OK|KILLED|xanh|chạy thật|runtime)\b|\b\d+\s*/\s*\d+\b", re.I)
EV_RE = re.compile(r"\bEV-\d{8}T\d{6}-[0-9a-f]{4}\b")
RATIO_RE = re.compile(r"\b(\d+)\s*/\s*(\d+)\b")
RUNTIME_CLAIM_RE = re.compile(r"(chạy thật|runtime|real\s+run)", re.I)
FAIL_WORD_RE = re.compile(r"\b(FAIL(?:ED)?|đỏ|red|deny|chặn)\b", re.I)
CLAIM_EXEMPT_HEAD_RE = re.compile(r"^#+.*(chưa kiểm|giới hạn|còn mở|not verified|limitation)", re.I)
CLAIM_DOCS = re.compile(r"/workspaces/[^/]+/(06_validation|11_report|audit_report)_[^/]*\.md$")


def _claim_lines(text: str) -> List[str]:
    out: List[str] = []
    exempt = False
    fence = False
    for line in text.splitlines():
        if line.strip().startswith("```"):
            fence = not fence
            continue
        if fence:
            continue
        if line.lstrip().startswith("#"):
            exempt = bool(CLAIM_EXEMPT_HEAD_RE.match(line.strip()))
            continue
        if not exempt and CLAIM_RE.search(line):
            out.append(line)
    return out


def claim_issues(text: str, ws_dir: Path, st: List[Dict[str, Any]], fingerprint: Optional[str]) -> List[str]:
    """H4c: mỗi dòng khẳng định kết quả phải trích biên lai EV- thật, còn mới, khớp số liệu và loại bằng chứng."""
    issues: List[str] = []
    ran = [str(a.get("CommandLine") or "") for _, n, a in calls(st) if n == "run_command"]
    forged = any(
        n in WRITE_TOOLS and "06_evidence/" in tgt(a).replace("\\", "/")
        for _, n, a in calls(st)
    )
    fp_cache: Dict[str, Optional[str]] = {}
    for line in _claim_lines(text):
        short = line.strip()[:90]
        evs = EV_RE.findall(line)
        if not evs:
            issues.append(f"claim không có biên lai EV-: «{short}»")
            continue
        recs: List[Dict[str, Any]] = []
        for ev in evs:
            try:
                rec = json.loads((ws_dir / "06_evidence" / f"{ev}.json").read_text(encoding="utf-8"))
            except (OSError, ValueError):
                issues.append(f"{ev} không tồn tại trong 06_evidence/")
                continue
            if forged:
                issues.append(f"{ev}: phiên có tool ghi tay vào 06_evidence/ (biên lai không đáng tin)")
            if not any("evidence_run.py" in c and f"--kind {rec.get('kind')}" in c for c in ran):
                issues.append(f"{ev}: không thấy run_command evidence_run.py --kind {rec.get('kind')} trong phiên")
            want = fingerprint if fingerprint is not None else _fp_for(str(rec.get("cwd") or ""), fp_cache)
            if want and rec.get("src_fingerprint") != want:
                issues.append(f"{ev}: code đã thay đổi sau khi chạy (bằng chứng cũ)")
            recs.append(rec)
        if not recs:
            continue
        if not FAIL_WORD_RE.search(line) and any(int(r.get("exit", 1)) != 0 for r in recs):
            issues.append(f"claim PASS nhưng biên lai exit≠0: «{short}»")
        blob = "\n".join(str(r.get("stdout_tail", "")) for r in recs)
        for a, b in RATIO_RE.findall(line):
            if not (re.search(rf"\b{a}\b", blob) and re.search(rf"\b{b}\b", blob)):
                issues.append(f"số {a}/{b} không có trong output biên lai: «{short}»")
        if RUNTIME_CLAIM_RE.search(line) and not any(r.get("kind") in ("runtime", "integration") for r in recs):
            issues.append(f"claim runtime nhưng biên lai chỉ là unit test: «{short}»")
    return issues


def _fp_for(cwd: str, cache: Dict[str, Optional[str]]) -> Optional[str]:
    """Fingerprint source hiện tại của repo mà biên lai đã chạy (theo cwd ghi trong biên lai)."""
    if cwd not in cache:
        try:
            if str(AGENT / "tooling") not in sys.path:
                sys.path.insert(0, str(AGENT / "tooling"))
            from evidence_run import source_fingerprint  # type: ignore

            cache[cwd] = source_fingerprint(Path(cwd)) if cwd and Path(cwd).is_dir() else None
        except ImportError:
            cache[cwd] = None
    return cache[cwd]


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


def handle_pre_invocation(
    payload: Dict[str, Any], state_dir: Optional[Path] = None, mem_root: Optional[Path] = None
) -> Dict[str, Any]:
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
    msg = BOOT_MSG.format(laya=laya)
    try:
        if str(AGENT) not in sys.path:
            sys.path.insert(0, str(AGENT))
        from tooling.orchestrator.harness2_engine import (
            AgentSurfaceSpec,
            DEFAULT_LIBRARY_ROOT,
            load_agent_surface_tree,
            validate_agent_surface,
        )

        surf_errs = validate_agent_surface(AgentSurfaceSpec(), load_agent_surface_tree(AGENT))
        lib_dir = (mem_root or MEM) / "global" / "harness_library" if mem_root else DEFAULT_LIBRARY_ROOT
        dom_indexes = sorted(p.parent.name for p in lib_dir.glob("*/INDEX.md")) if lib_dir.is_dir() else []
        h2_status = "VALIDATED" if not surf_errs else f"ERRORS({len(surf_errs)})"
        msg += f"\n[Harness² S0] Surface: {h2_status} | Domain Library: {dom_indexes or ['empty']}"
    except Exception:
        pass
    prompt = str(st[u_idx].get("content", "")) if u_idx >= 0 else ""
    hits = match_tripwires(prompt, load_tripwires(mem_root))
    if hits:
        msg += "\n\n🧷 LESSON TRIPWIRES khớp yêu cầu này — BẮT BUỘC làm TRƯỚC khi hành động:\n" + "\n".join(
            f"- [{t['id']}] {t['check']}" for t in hits
        )
    return {"injectSteps": [{"ephemeralMessage": msg}]}


def handle_pre_write(
    payload: Dict[str, Any],
    mem_root: Optional[Path] = None,
    ws_root: Optional[Path] = None,
) -> Dict[str, Any]:
    tool_call = payload.get("toolCall") or {}
    tool_name = str(tool_call.get("name") or payload.get("tool_name") or payload.get("name") or "")
    raw_args = tool_call.get("args") or payload.get("tool_input") or payload.get("args") or {}
    args = {k: _unquote(v) for k, v in raw_args.items()}
    f = tgt(args)
    old_s = old_arg(raw_args)
    body = new_arg(raw_args)
    is_overwrite_flag = raw_args.get("Overwrite") in (True, "true", "True", "1", 1)

    # H2: Memory File Protection & Global Lesson Hygiene (Rule #4, #5, #17)
    if is_mem_path(f, mem_root):
        try:
            exists = Path(f).exists()
        except PermissionError:
            exists = True
        is_partial_edit = tool_name in (WRITE_TOOLS - FULL_WRITE_TOOLS)
        if exists and (tool_name in FULL_WRITE_TOOLS or is_overwrite_flag or (is_partial_edit and not old_s)):
            return {
                "decision": "deny",
                "reason": (
                    f"Rule 17 (append-only guard): CẤM Overwrite hoặc bỏ trống TargetContent/Target trên memory file {f}. "
                    "Chỉ sửa/append bằng replace_file_content hoặc edit_file."
                ),
            }
        if APPEND_ONLY.search(f) and exists and (not old_s or old_s not in body):
            return {
                "decision": "deny",
                "reason": f"Rule 4/17: {Path(f).name} là file append-only — bắt buộc truyền và giữ nguyên TargetContent/Target.",
            }
        if f.replace("\\", "/").endswith("/memory/global/lessons.md"):
            hygiene_errs = validate_lesson_append_hygiene(raw_args)
            if hygiene_errs:
                return {
                    "decision": "deny",
                    "reason": (
                        "Rule 5 (Global Lesson Hygiene Guard — Chống phá cấu trúc lessons.md):\n- "
                        + "\n- ".join(hygiene_errs)
                    ),
                }

    # H3: Secret & PII Guard (Rule #20)
    m = SECRET.search(body)
    if m:
        return {
            "decision": "deny" if is_mem_path(f, mem_root) else "force_ask",
            "reason": f"Rule 20: phát hiện chuỗi giống secret ({m.group(0)[:10]}***). Mask trước khi ghi.",
        }

    # H4t: Lesson Tripwire Gate (G-08) — kiểm tra tiền điều kiện trên tài liệu kết luận
    tw_match = TRIPWIRE_GUARDED_DOCS.search(f.replace("\\", "/"))
    if tw_match and is_mem_path(f, mem_root):
        tgt_ws = re.search(r"/workspaces/([^/]+)/", f.replace("\\", "/"))
        viol = tripwire_violations(
            steps(payload), load_tripwires(mem_root), exclude_ws=tgt_ws.group(1) if tgt_ws else None
        )
        if viol:
            return {
                "decision": "deny",
                "reason": "Lesson Tripwire (G-08): chưa làm tiền điều kiện bắt buộc trước khi ghi kết luận:\n"
                + "\n".join(f"- [{t['id']}] {t['check']}" for t in viol),
            }

    # H4c: Claim ↔ Evidence Linter (anti_fake_evidence) — chỉ lint dòng MỚI ghi.
    claim_doc = re.search(r"/workspaces/([^/]+)/", f.replace("\\", "/")) if CLAIM_DOCS.search(f.replace("\\", "/")) else None
    if claim_doc and is_mem_path(f, mem_root):
        old_lines = set(old_s.splitlines())
        new_text = body if not old_s else "\n".join(ln for ln in body.splitlines() if ln not in old_lines)
        st_c = steps(payload)
        c_issues = claim_issues(new_text, (ws_root or WS_ROOT) / claim_doc.group(1), st_c, None)
        if c_issues:
            return {
                "decision": "deny",
                "reason": "Anti-Fake-Evidence (H4c): khẳng định kết quả phải trích biên lai thật từ "
                "`tooling/evidence_run.py` (EV-…):\n- " + "\n- ".join(c_issues[:8]),
            }

    # H4a: S1 Intake Gate, Line-Cap (800) & H4t-Demo before S3 Solution (Rule #4, #9, #16)
    sol_match = re.search(r"/workspaces/([^/]+)/09_tasks_solution_([^/]+)\.md$", f)
    if sol_match and is_mem_path(f, mem_root):
        ws_name, suffix = sol_match.group(1), sol_match.group(2)
        ws_dir = (ws_root or WS_ROOT) / ws_name
        cur_txt = ""
        try:
            sol_path = Path(f)
            if sol_path.is_file():
                cur_txt = sol_path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            cur_txt = ""
        merged_sol = _apply_edit_preview(cur_txt, raw_args, old_s, body) if cur_txt else body
        if len(merged_sol.splitlines()) > MAX_SOLUTION_LINES:
            return {
                "decision": "deny",
                "reason": (
                    f"Lesson L004 (Anti-Solution-File Bloat): {Path(f).name} đạt {len(merged_sol.splitlines())} dòng "
                    f"(vượt trần {MAX_SOLUTION_LINES} dòng). Bắt buộc tách sang suffix mới (01/08/09_<new_suffix>.md)!"
                ),
            }
        old_snippets_by_path: Dict[str, set[str]] = {}
        for old_vp, old_code in _extract_demo_snippets(cur_txt):
            old_snippets_by_path.setdefault(old_vp, set()).update(old_code.splitlines())
        loaded_tws = load_tripwires(mem_root)
        demo_tw_viol: List[Dict[str, Any]] = []
        seen_tw_ids = set()
        for synth_path, snippet_code in _extract_demo_snippets(merged_sol):
            prior_lines = old_snippets_by_path.get(synth_path, set())
            added_snippet = (
                snippet_code
                if not prior_lines
                else "\n".join(ln for ln in snippet_code.splitlines() if ln not in prior_lines)
            )
            for hit in code_tripwire_violations(synth_path, added_snippet, loaded_tws, full_code_body=snippet_code):
                if hit["id"] not in seen_tw_ids:
                    seen_tw_ids.add(hit["id"])
                    demo_tw_viol.append(hit)
        if demo_tw_viol:
            return {
                "decision": "deny",
                "reason": (
                    "Demo Tripwire Gate (H4t-Demo): Code Demo trong 09_tasks_solution vi phạm bài học trong lessons.md:\n"
                    + "\n".join(f"- [{t['id']}] {t['check']}" for t in demo_tw_viol)
                ),
            }
        st = steps(payload)
        u_idx = last_user_idx(st)
        try:
            other_sols = [
                p
                for p in ws_dir.glob("09_tasks_solution_*.md")
                if p.name != f"09_tasks_solution_{suffix}.md"
            ] if ws_dir.exists() else []
            any_req = bool(list(ws_dir.glob("01_requirements*.md")) or list(ws_dir.glob("01_plan*.md"))) if ws_dir.exists() else False
            any_tsk = bool(list(ws_dir.glob("08_tasks*.md"))) if ws_dir.exists() else False
            exact_req = ((ws_dir / f"01_requirements_{suffix}.md").is_file() or (ws_dir / f"01_plan_{suffix}.md").is_file()) if ws_dir.exists() else False
            exact_tsk = (ws_dir / f"08_tasks_{suffix}.md").is_file() if ws_dir.exists() else False
        except OSError:
            other_sols, any_req, any_tsk, exact_req, exact_tsk = [], True, True, True, True

        req_ok = exact_req or touched(st, r"01_(requirements|plan)", u_idx) or (not other_sols and any_req)
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
            rule_issues = spec_rule_issues(merged_sol)
            if rule_issues:
                return {
                    "decision": "deny",
                    "reason": "Rule 13 (Clean-Room Spec, G-03): " + "; ".join(rule_issues),
                }

    # H4b: Source Change Gate, Code-Level Tripwires & Anti-Mock-Pollution (Rule #4, #10, #12, #13, #14, #16)
    if is_src(f, mem_root):
        old_src_lines = set(old_s.splitlines())
        added_src_body = body if not old_s else "\n".join(ln for ln in body.splitlines() if ln not in old_src_lines)
        cur_src = ""
        try:
            src_path = Path(f)
            if src_path.is_file():
                cur_src = src_path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            cur_src = ""
        merged_src_body = _apply_edit_preview(cur_src, raw_args, old_s, body) if cur_src else body

        if not is_test_file(f) and PROD_MOCK_POLLUTION.search(added_src_body):
            return {
                "decision": "deny",
                "reason": (
                    "Rule 12/14 (Anti-Smoke-Speculation, L14): CẤM gắn hàm mock/closure giả (*Fn, sqlmock, fakemodel) "
                    "vào code production để lách tích hợp thật!"
                ),
            }
        if is_test_file(f) and FAKE_REAL_TEST_RE.search(added_src_body):
            return {
                "decision": "deny",
                "reason": (
                    "Rule 14 (Zero-Mockup in Real DB Tests, L14): CẤM dùng sqlmock, SQLite file::memory: hoặc "
                    "httptest.NewServer bên trong hàm test mang tên RealPostgres/RealDB/RealNATS!"
                ),
            }

        code_tw_viol = code_tripwire_violations(f, added_src_body, load_tripwires(mem_root), full_code_body=merged_src_body)
        if code_tw_viol:
            return {
                "decision": "deny",
                "reason": (
                    "Code Tripwire Gate (H4t-Code): Phát hiện mẫu code vi phạm bài học trong lessons.md:\n"
                    + "\n".join(f"- [{t['id']}] {t['check']}" for t in code_tw_viol)
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
        dg0_st = laya_gate_status(ws, "dg0", ws_root)
        if dg0_st is not None and dg0_st != "PASS":
            return {
                "decision": "deny",
                "reason": "DG0: Wave phản biện yêu cầu trước Plan chưa đạt PASS — không được sửa source code.",
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
            try:
                if str(AGENT) not in sys.path:
                    sys.path.insert(0, str(AGENT))
                from tooling.orchestrator.harness2_engine import assert_pre_flight_clean_room

                cr_ctx = (ws_root or WS_ROOT) / ws / "clean_room_context"
                if cr_ctx.is_dir():
                    assert_pre_flight_clean_room(cr_ctx, forbidden)
            except ValueError as exc:
                return {
                    "decision": "deny",
                    "reason": f"Rule 13 (Harness² Pre-Flight Clean-Room Check): {exc}",
                }
        sol = [i for i, nm, x in calls(st) if nm in WRITE_TOOLS and "09_tasks_solution" in tgt(x)]
        aud = [
            i
            for i, nm, x in calls(st)
            if nm in WRITE_TOOLS and "audit_report_" in tgt(x)
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
        ok = user_approved(st, first_sol_idx + 1 if first_sol_idx >= 0 else 0, clean_room=is_cr)
        if not ok:
            return {
                "decision": "force_ask",
                "reason": (
                    "Rule 13 (Clean-Room, G-01): chưa thấy User gửi riêng một dòng `BẮT ĐẦU` (hoặc `APPROVE`) "
                    "sau 09_tasks_solution. Cho phép sửa code?"
                    if is_cr
                    else "Rule 13: chưa thấy User APPROVE sau 09_tasks_solution trong phiên này. Cho phép sửa code?"
                ),
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


def handle_pre_subagent(
    payload: Dict[str, Any],
    mem_root: Optional[Path] = None,
    ws_root: Optional[Path] = None,
) -> Dict[str, Any]:
    """G-04: PreToolUse chặn rò rỉ đường dẫn/code gốc vào invoke_subagent hoặc send_message."""
    st = steps(payload)
    ws = active_ws(st, ws_root)
    if not ws:
        return {"decision": "ask"}
    sfx = latest_solution_suffix(st, -1)
    is_cr, forbidden = clean_room_config(ws, ws_root, sfx)
    if is_cr and forbidden:
        tc = payload.get("toolCall") or {}
        leaked = leaked_subagent_prompts(st + [{"tool_calls": [tc]}], forbidden)
        if leaked:
            return {
                "decision": "deny",
                "reason": (
                    "Rule 13 (Clean-Room Leak Guard, G-04): CẤM truyền đường dẫn hoặc nội dung code gốc "
                    f"({', '.join(leaked)}) vào prompt Sub-agent Builder! Chỉ được truyền Clean-Room Approved Spec."
                ),
            }
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
        if not c.get("h6_auto_checks", True):
            continue
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
    inj_key = f"{conv_id}.{u_idx}.audit_inj" if u_idx >= 0 else f"{conv_id}.audit_inj"
    inj_file = sdir / inj_key

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
        if not inj_file.exists():
            inj_file.write_text("1", encoding="utf-8")
            msg = audit_prompt
        else:
            msg = f"🔍 H6 ({edited_names}): nhớ chạy git diff + evidence_run.py + DG3 + cập nhật audit_report trước khi kết thúc."
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

    # Cổng đồng bộ bắt buộc: Nếu có sửa lessons.md trong lượt thì bắt buộc phải chạy build_lessons_index.py sau đó
    les_edits = lessons_edits_since(st, u_idx)
    if les_edits and not touched(st, r"build_lessons_index\.py", les_edits[-1]):
        miss.append(
            "Rule 5 (Lessons Index Sync): Đã cập nhật `lessons.md` nhưng chưa chạy `python3 tooling/build_lessons_index.py` sau lần sửa cuối!"
        )

    sol_touched = touched(st, r"09_tasks_solution", u_idx)
    if sol_touched:
        dg0_st = laya_gate_status(ws, "dg0", ws_root)
        if dg0_st is not None and dg0_st != "PASS":
            miss.append("DG0: laya_gate chưa PASS cho Wave phản biện trước Plan")
        if laya_gate_status(ws, "dg1", ws_root) != "PASS":
            miss.append("DG1: laya_gate chưa PASS cho giải pháp/code demo")
        dg2_st = laya_gate_status(ws, "dg2", ws_root)
        if dg2_st is not None and dg2_st != "PASS":
            miss.append("DG2: laya_gate chưa PASS cho code demo")
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
        if not touched(st, r"(audit_report_|06_validation)", last):
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

    if ws:
        cr_any, cr_forbidden = clean_room_config(ws, ws_root, sol_sfx or latest_solution_suffix(st, -1))
        if cr_any and cr_forbidden:
            leaked = leaked_subagent_prompts(st, cr_forbidden)
            if leaked:
                miss.append(
                    "Clean-Room Leak (G-04): prompt invoke_subagent chứa đường dẫn/nội dung code gốc "
                    f"({', '.join(leaked)}) — builder không còn 'chưa từng thấy code gốc'; dựng lại subagent chỉ với Spec."
                )

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

    if run_linter and src and n == 0 and miss:
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
                "tự sửa dứt điểm source code, chạy lại test + `DG3` + cập nhật `audit_report_<suffix>.md` + `05_progress` "
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
        "pre-subagent": handle_pre_subagent,
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