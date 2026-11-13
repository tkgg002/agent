# 09. Tasks Solution — Khắc Phục Tận Gốc Độ Trễ, Đốt Token & Lọt Lỗi Thiết Kế Của Agentic Core (`agent_core_v2`)

## 1. Bảng Đối Chiếu Nguyên Nhân Gốc Rễ (`F1`–`F4`) & 8 Lỗi Biên Phản Biện (`F-01`–`F-08`)

| Mã | Mức độ | Vị trí (`file:line`) | Cơ chế gây lỗi | Giải pháp Triệt để (`P1`–`P5`) |
| :--- | :---: | :--- | :--- | :--- |
| **`F1` / `F-01`** | **CRITICAL** | `.agents/hooks.json:10-48` | Chuỗi `"\\|"` trong JSON giải mã thành `\|` (khớp dấu pipe nghĩa đen thay vì toán tử OR `|`) và thiếu `multi_replace_file_content`, `multi_edit`, `write_file`, `edit_file`. | **`P1a`**: Đổi sang toán tử `|` chuẩn không escape thừa: `"write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file|multi_edit"` và `"invoke_subagent|send_message"`. |
| **`F-02` / `R-03`** | **CRITICAL** | `hooks/agy/agy_guard.py:858-898` | `write_file` ghi đè không cần cờ `Overwrite` và `old_s == ""` đi xuyên qua cả 2 nhánh `if` của `H2`; `not old_s` cần giới hạn cho nhóm `is_partial_edit = tool_name in (WRITE_TOOLS - FULL_WRITE_TOOLS)`. | **`P1b`**: Khai báo `FULL_WRITE_TOOLS = frozenset({"write_to_file", "write_file"})`; trong `H2`, nếu file memory đã tồn tại thì chặn khi `tool_name in FULL_WRITE_TOOLS or is_overwrite_flag or (is_partial_edit and not old_s)`; với `APPEND_ONLY` bắt buộc `bool(old_s) and (old_s in body)`. |
| **`F-03` / `R-02`** | **CRITICAL** | `hooks/agy/agy_guard.py:80-105,206-258,330-338,648,960,1087-1096,1253` | Thêm `args.get("Path")` vào `tgt()` và chưa lọc `nm in WRITE_TOOLS` trong `def touched(...)` làm `read_file(Path=".../05_progress_*.md")` hoặc `read_file(Path=".../09_tasks_solution_*.md")` giả mạo thao tác ghi tiến độ / sửa file giải pháp. | **`P1b`**: Đồng bộ Hunk 1 với `READ_ARG_KEYS` trên đĩa (`agy_guard.py:80-105`), lọc `nm in WRITE_TOOLS` ngay trong `def touched(...)` (kèm ngoại lệ `run_command` cho `build_lessons_index`) và tại mọi vị trí gọi `tgt(...)`, giữ nguyên `prg_ok = touched(st, r"05_progress", u_idx)` tại `H4a`. |
| **`F3` / `F-04` / `F-06` / `N-02`..`N-05` / `R-03`** | **CRITICAL** | `hooks/agy/agy_guard.py:206-275,507-560,921-1036` | (1) `code_tripwire_violations` cần lọc đuôi file, tách rời `forbid` với `req` (`N-02`), kiểm tra `trig`/`forbid` trên `new_code_body` nhưng kiểm tra `req` trên toàn khối `full_code_body` (`R-03`), và chỉ suy đoán `ext` từ `trig` (`N-04`); (2) Hợp nhất `_apply_edit_preview` trước khi chạy `_extract_demo_snippets(merged_sol)` (`N-03`/`N-05`) và bỏ `or body`. | **`P1b` + `P3`**: (1) Thêm `_parse_chunks`, `_chunk_texts`, `_apply_edit_preview` và bỏ `or body`; (2) Bổ sung `full_code_body: Optional[str] = None` vào `code_tripwire_violations`, tách biệt `forbid`/`req`, chỉ suy đoán `ext` từ `trig`; (3) Trích xuất `_extract_demo_snippets(merged_sol)` trừ đi `_extract_demo_snippets(cur_txt)` sau khi hợp nhất preview. |
| **`F2` / `F-05` / `F-07` / `N-01` / `R-01` / `R-03`** | **CRITICAL** | `tooling/laya_gate.py:108-142,188-206,284-415,634-644` | (1) Cấm vượt rào khi `0` control claims (`N-01`), xử lý `List[str]` trong `_normalize_evidence_str` (`R-03`), ánh xạ `kind`/`type`/`control` (`bool`); (2) Cấm nới lỏng `INLINE_REF_RE` trong `cmd_resolve` (`R-01`), cấm `--id ALL`, đồng bộ `dg2` khớp cùng timestamp; (3) Cần `--workers` (mặc định `3`). | **`P2`**: (1) Giữ nguyên bắt buộc `>=2 control_false` & `>=1 control_true`, bảo toàn `List[str]` trong `_normalize_evidence_str`, chạy song song qua `ThreadPoolExecutor` (`--workers 3`); (2) Bắt buộc khớp `INLINE_REF_RE` + `resolve_ref_path` cho mọi `REF:` trong `cmd_resolve`, cấm `--id ALL`, kiểm tra nguyên tử toàn bộ `req_ids` trước khi mutate, đồng bộ ưu tiên file `dg2` cùng timestamp. |
| **`F4` / `F-08`** | **MEDIUM** | `tooling/verify_governance.py:32,358` | Regex `[Agent:[a-zA-Z0-9.\-_ ]+]` vẫn từ chối tên model có ngoặc đơn hoặc dấu `/` như `[Agent:Claude Opus 4.6 (Thinking)]` hoặc `[Agent:google/gemini-3.1-pro]`. | **`P4`**: Dùng hằng số dùng chung `AGENT_TAG_RE = r"\[Agent:[^\s\]][^\r\n\]]*\]"` cho cả `LOG_LINE_RE` và `today_re`. |

---

## 2. Code Demo 1:1 (Khớp Nguyên Văn Từng Dòng Với HEAD — Không Dùng `...`)

### Demo 1 (`P1a` / `F-01`): Cập nhật `.agents/hooks.json:10-48` (và đồng bộ sang `data-hub/.agents/hooks.json:10-48`, `~/.gemini/config/hooks.json:10-22`)

```json
// Target: /Users/trainguyen/Documents/work/agent/.agents/hooks.json:10-48
  "agy-pre-write": {
    "PreToolUse": [
      {
        "matcher": "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file|multi_edit",
        "hooks": [
          {
            "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-write",
            "timeout": 10
          }
        ]
      }
    ]
  },
  "agy-pre-command": {
    "PreToolUse": [
      {
        "matcher": "run_command",
        "hooks": [
          {
            "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-command",
            "timeout": 5
          }
        ]
      }
    ]
  },
  "agy-pre-subagent": {
    "PreToolUse": [
      {
        "matcher": "invoke_subagent|send_message",
        "hooks": [
          {
            "command": "python3 /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py pre-subagent",
            "timeout": 5
          }
        ]
      }
    ]
  },
```

### Demo 2 (`P1b` & `P3` / `F-02`, `F-03`, `F-04`, `F-06`): Cập nhật `hooks/agy/agy_guard.py`

```python
# Target 1: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:82-98
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
WRITE_TOOLS = frozenset({"write_to_file", "replace_file_content", "multi_replace_file_content", "write_file", "edit_file", "multi_edit"})
FULL_WRITE_TOOLS = frozenset({"write_to_file", "write_file"})
MAX_SOLUTION_LINES = 800
# G-08: lesson tripwires.
TRIPWIRE_LIMIT = 5
TRIPWIRE_GUARDED_DOCS = re.compile(r"/workspaces/[^/]+/(13_analysis|09_tasks_solution|audit_report)_[^/]*\.md$")
# AF-1: bằng chứng Require chỉ tính từ tool đọc/lệnh — tool ghi (kể cả lần ghi bị chặn) không được tự thoả.
TRIPWIRE_EVIDENCE_TOOLS = frozenset(
    {"view_file", "read_file", "grep_search", "find_by_name", "list_dir", "run_command", "read_url_content", "search_web"}
)
```
```python
# Target 2: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:206-270
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
    return "\n".join(str(_unquote(ch.get(k) or "")) for ch in _parse_chunks(raw_args) for k in keys if ch.get(k))


def old_arg(raw_args: Dict[str, Any]) -> str:
    base = str(_unquote(raw_args.get("TargetContent") or raw_args.get("Target") or raw_args.get("old_string") or ""))
    extra = _chunk_texts(raw_args, ("TargetContent", "Target", "oldText", "old_string"))
    return f"{base}\n{extra}".strip("\n") if extra else base


def new_arg(raw_args: Dict[str, Any]) -> str:
    base = "".join(str(_unquote(raw_args.get(k) or "")) for k in ("CodeContent", "ReplacementContent", "Content", "Replacement", "new_string"))
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


def src_edits(st: List[Dict[str, Any]], mem_root: Optional[Path] = None) -> List[Tuple[int, str]]:
    return [(i, tgt(a)) for i, n, a in calls(st) if n in WRITE_TOOLS and is_src(tgt(a), mem_root)]


def touched(st: List[Dict[str, Any]], pattern: str, after: int = -1) -> bool:
    return any(
        i > after
        and (
            (nm in WRITE_TOOLS and bool(re.search(pattern, tgt(a) or json.dumps(a, ensure_ascii=False))))
            or (nm == "run_command" and bool(re.search(pattern, str(a.get("CommandLine") or ""))))
        )
        for i, nm, a in calls(st)
    )


def lessons_edits_since(st: List[Dict[str, Any]], after: int = -1) -> List[int]:
    return [i for i, n, a in calls(st) if i > after and n in WRITE_TOOLS and tgt(a).replace("\\", "/").endswith("/memory/global/lessons.md")]


def validate_lesson_append_hygiene(args: Dict[str, Any]) -> List[str]:
    old_text, new_text = old_arg(args), new_arg(args)
    if old_text and new_text.startswith(old_text):
        added = new_text[len(old_text) :]
    elif old_text and old_text in new_text:
        added = new_text.replace(old_text, "", 1)
    else:
        added = new_text
```

```python
# Target 3: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:309-338
def project_reads_since(st: List[Dict[str, Any]], after: int = -1, mem_root: Optional[Path] = None) -> List[str]:
    mroot, aroot = str(mem_root or MEM), str(AGENT)
    reads: List[str] = []
    for i, n, a in calls(st):
        if i <= after:
            continue
        if n in ("view_file", "read_file", "grep_search", "find_by_name"):
            p = str(a.get("AbsolutePath") or a.get("Path") or a.get("SearchPath") or a.get("SearchDirectory") or "")
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
```

```python
# Target 4: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:507-560,643-651
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
        trig = t.get("trigger_regex") or t.get("trigger") or t.get("pattern") or t.get("forbid_regex") or t.get("forbid") or t.get("Forbid") or ""
        if not trig:
            continue
        item = dict(t)
        item.setdefault("trigger_regex", str(trig))
        if not any(k in item for k in ("ext", "extensions", "file_ext", "glob", "file_glob", "files", "path_regex", "file_regex", "file_pattern", "lang", "language")):
            inferred = _infer_tripwire_exts(str(trig))
            if inferred:
                item["ext"] = inferred
        out.append(item)
    return out


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
        lang_map = {"go": {".go"}, "golang": {".go"}, "py": {".py"}, "python": {".py"}, "ts": {".ts", ".tsx"}, "typescript": {".ts", ".tsx"}, "js": {".js", ".jsx"}, "javascript": {".js", ".jsx"}, "sql": {".sql"}, "css": {".css", ".scss"}}
        if lang in lang_map and ext not in lang_map[lang]:
            return False
    globs = t.get("glob") or t.get("file_glob") or t.get("files")
    if globs:
        glob_list = [globs] if isinstance(globs, str) else list(globs)
        if glob_list and not any(fnmatch(fp.lower(), str(g).lower()) or fnmatch(p.name.lower(), str(g).lower()) for g in glob_list if g):
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
    if not new_code_body.strip():
        return []
    req_context = full_code_body if full_code_body is not None else new_code_body
    out: List[Dict[str, Any]] = []
    for t in trips:
        tid = str(t.get("id") or "")
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
_LANG_EXT_MAP = {"go": ".go", "golang": ".go", "python": ".py", "py": ".py", "ts": ".ts", "tsx": ".tsx", "typescript": ".ts", "js": ".js", "jsx": ".jsx", "javascript": ".js", "sql": ".sql", "css": ".css", "sh": ".sh", "bash": ".sh"}


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
```

```python
# Target 5: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:643-651,858-1015,1087-1096,1253
    # Tại claim_issues (dòng 647-650):
    forged = any(
        n in WRITE_TOOLS and "06_evidence/" in tgt(a).replace("\\", "/")
        for _, n, a in calls(st)
    )

    # Tại handle_pre_write (dòng 858-1015):
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
            other_sols = [p for p in ws_dir.glob("09_tasks_solution_*.md") if p.name != f"09_tasks_solution_{suffix}.md"] if ws_dir.exists() else []
            any_req = bool(list(ws_dir.glob("01_requirements*.md")) or list(ws_dir.glob("01_plan*.md"))) if ws_dir.exists() else False
            any_tsk = bool(list(ws_dir.glob("08_tasks*.md"))) if ws_dir.exists() else False
            any_prg = bool(list(ws_dir.glob("05_progress*.md"))) if ws_dir.exists() else False
            exact_req = ((ws_dir / f"01_requirements_{suffix}.md").is_file() or (ws_dir / f"01_plan_{suffix}.md").is_file()) if ws_dir.exists() else False
            exact_tsk = (ws_dir / f"08_tasks_{suffix}.md").is_file() if ws_dir.exists() else False
        except OSError:
            other_sols, any_req, any_tsk, any_prg, exact_req, exact_tsk = [], True, True, True, True, True

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
            return {"decision": "deny", "reason": f"Rule 9 (S1 Intake trước S3 Solution): CẤM tạo 09_tasks_solution_{suffix}.md khi chưa hoàn thành S1: {', '.join(missing_s1)}."}

    # H4b: Source Change Gate, Code-Level Tripwires & Anti-Mock-Pollution (Rule #4, #10, #12, #13, #14, #16)
    if is_src(f, mem_root):
        old_src_lines = set(old_s.splitlines())
        added_src_body = body if not old_s else "\n".join(ln for ln in body.splitlines() if ln not in old_src_lines)
        cur_src = Path(f).read_text(encoding="utf-8", errors="ignore") if Path(f).is_file() else ""
        merged_src_body = _apply_edit_preview(cur_src, raw_args, old_s, body) if cur_src else body
        code_tw_viol = code_tripwire_violations(f, added_src_body, load_tripwires(mem_root), full_code_body=merged_src_body)

    # Tại H4b sol/aud (dòng 1087-1096) và handle_stop sol_touched (dòng 1253):
        sol = [i for i, nm, x in calls(st) if nm in WRITE_TOOLS and "09_tasks_solution" in tgt(x)]
        aud = [i for i, nm, x in calls(st) if nm in WRITE_TOOLS and "audit_report_" in tgt(x)]
    # Tại handle_stop (dòng 1253):
    sol_touched = touched(st, r"09_tasks_solution", u_idx)
```

### Demo 3 (`P2` / `F-05`, `F-07`, `N-01`): Cập nhật `tooling/laya_gate.py:108-142,188-228,284-415,634-650`

```python
# Target 1: /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:108-142,188-228
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
```

```python
# Target 2: /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py:284-415,634-650
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

    # Tại build_parser (dòng 643, 648):
    p_run.add_argument("--workers", type=int, default=3)
    p_res.add_argument("--reason", "--evidence", dest="reason", required=True)
```

### Demo 4 (`P4` / `F-08`): Cập nhật `tooling/verify_governance.py:32,358`

```python
# Target 1: /Users/trainguyen/Documents/work/agent/tooling/verify_governance.py:32
AGENT_TAG_RE = r"\[Agent:[^\s\]][^\r\n\]]*\]"
LOG_LINE_RE = re.compile(rf"^-\s*\[\d{{4}}-\d{{2}}-\d{{2}}[^\]]*\]\s*{AGENT_TAG_RE}\s+.+")

# Target 2: /Users/trainguyen/Documents/work/agent/tooling/verify_governance.py:358
    today_re = re.compile(rf"^-\s*\[{re.escape(today_str)}[^\]]*\]\s*{AGENT_TAG_RE}")
```

