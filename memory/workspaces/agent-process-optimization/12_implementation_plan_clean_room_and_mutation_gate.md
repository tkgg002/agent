# 12 — Implementation Plan (Line-by-Line Spec): Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate (`clean_room_and_mutation_gate` — Revision R1)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Tier:** `T2` (`mode: clean-room`)
- **Nguyên tắc Thiết kế:** **Simplicity First, Minimal Impact, Zero Speculation, 100% Backward-Compatible** với 24 unit tests của `hooks/agy` và 14 unit tests của `tooling`.

---

## 1. Tổng quan Kiến trúc & Danh sách File Thực thi

| STT | File | Hành động | Dòng code bị ảnh hưởng (`file:line`) | Số dòng thêm/sửa dự kiến |
| :-: | :--- | :---: | :--- | :---: |
| 1 | `GEMINI.md` | Sửa (`replace_file_content`) | `Rule #1 (L18-L21)`, `Rule #9 (L71-L74)`, `Rule #13 (L95-L96)`, `Rule #14 (L98-L99)` | `+8 dòng` (`~16.3 KB < 24 KB`) |
| 2 | `skills/brain-delegate/SKILL.md` | Sửa (`replace_file_content`) | `Bước 3 (L42-L50)` | `+9 dòng` |
| 3 | `skills/muscle-execute/SKILL.md` | Sửa (`replace_file_content`) | `Bước 4 & Bước 5 (L45-L62)` | `+10 dòng` |
| 4 | `skills/qa-agent/SKILL.md` | Sửa (`replace_file_content`) | `Gate G3/G4 (L29-L47)` | `+8 dòng` |
| 5 | `skills/workspace-docs/SKILL.md` | Sửa (`replace_file_content`) | `Mục 1 & Mục 2 (L15-L42)` | `+6 dòng` |
| 6 | `tooling/mutation_gate.py` | **Tạo mới** (`write_to_file`) | File mới (`L1-L245`) | `+245 dòng` |
| 7 | `tooling/tests/test_mutation_gate.py` | **Tạo mới** (`write_to_file`) | File mới (`L1-L195`) | `+195 dòng` |
| 8 | `hooks/agy/agy_guard.py` | Sửa (`replace_file_content`) | `L54-L96`, `L238-L268`, `L379-L456`, `L560-L582` | `+68 dòng` |
| 9 | `hooks/agy/tests/test_agy_guard.py` | Sửa (`replace_file_content`) | Cuối file (`L680-L686`) | `+95 dòng` (thêm 4 test `test_25`..`test_28`) |
| 10 | `tooling/verify_governance.py` | Sửa (`replace_file_content`) | `L21-L24`, `L90-L101`, `L192-L215` | `+42 dòng` |
| 11 | `tooling/tests/test_verify_governance.py` | Sửa (`replace_file_content`) | Cuối file (`L180-L185`) | `+55 dòng` (thêm 2 test mới) |

---

## 2. Chi tiết Từng Dòng Code Cần Thực hiện (Line-by-Line Changes)

### 2.1. File 1: `/Users/trainguyen/Documents/work/agent/GEMINI.md`

1. **Tại `GEMINI.md:71-74` (Rule #9 — S3, DG1+DG2, S4):**
   - **Dòng gốc (`L71-L73`):**
     ```markdown
     4. **S3 SOLUTION:** Đọc `agent/skills/brain-delegate/SKILL.md`. Trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** (cấm Option 1/2/3) kèm code demo vào `09_tasks_solution_<suffix>.md` (hoặc `13_analysis_<suffix>.md` nếu chỉ Audit).
     5. **DG1 + DG2:** Đọc `agent/skills/laya-router/SKILL.md`. Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run` (`--stage dg1`, `--stage dg2`). Append kết quả vào `05_progress_*.md`. Đạt `PASS` → trình User, **DỪNG CHỜ `APPROVE`**.
     6. **S4 EXECUTE (sau `APPROVE`):** Đọc `agent/skills/muscle-execute/SKILL.md`. Tạo git restore-point local → sửa code → chạy `checks`, `test`, `runtime_check` theo `.agent-project.json` + gọi request thật. **CẤM gắn hàm mock/closure giả (`*Fn`) vào struct production!**
     ```
   - **Dòng thay thế:**
     ```markdown
     4. **S3 SOLUTION:** Đọc `agent/skills/brain-delegate/SKILL.md`. Trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** (cấm Option 1/2/3) kèm code demo vào `09_tasks_solution_<suffix>.md` (hoặc `13_analysis_<suffix>.md` nếu chỉ Audit). Với task `mode: clean-room` (Rebuild/Refactor module cũ), bắt buộc lập mục `## Clean-Room Approved Spec` chỉ chứa các quy tắc hành vi có bằng chứng đã qua `DG1 = PASS`.
     5. **DG1 + DG2:** Đọc `agent/skills/laya-router/SKILL.md`. Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run` (`--stage dg1`, `--stage dg2`). Append kết quả vào `05_progress_*.md`. Đạt `PASS` → trình User, **DỪNG CHỜ `APPROVE`**.
     6. **S4 EXECUTE (sau `APPROVE`):** Đọc `agent/skills/muscle-execute/SKILL.md`. Tạo git restore-point local → sửa code (nếu có `clean_room_forbidden_sources`, phiên đã đọc code gốc CẤM tự viết code mà phải gọi `invoke_subagent` chỉ truyền `Clean-Room Approved Spec`) → chạy `checks`, `test`, `runtime_check` theo `.agent-project.json` + kiểm chứng độ mạnh của test với **≥3 bản cố tình làm hỏng (`tooling/mutation_gate.py`)** + gọi request thật. **CẤM gắn hàm mock/closure giả (`*Fn`) vào struct production!**
     ```

2. **Tại `GEMINI.md:95-99` (Rule #13 & Rule #14):**
   - **Dòng gốc (`L95-L99`):**
     ```markdown
     ### 13. Quy tắc Phân cách Brain/Muscle trên Source Code
     - Khi ở vai trò Brain, đang Audit, hoặc chưa có `APPROVE`: **CẤM** `write_to_file`/`replace_file_content` trên source code (`.go`, `.ts`, `.tsx`, `.js`, `.py`, `.sql`, v.v.). Phải lập `09_tasks_solution_*.md` → qua `DG1/DG2` → chờ `APPROVE`.

     ### 14. Quy tắc Đảm bảo Chất lượng Đầu ra (Definition of Done — Gates G1–G8)
     BẮT BUỘC pass đủ 8 Gate trước khi báo Done (xem `/qa-agent`): **(G1) Truy vết:** Đối chiếu `01_requirements_*`. **(G2) Reproduce (Red→Green):** Tái hiện bug trước khi sửa; sau sửa PASS. **(G3) Test & Runtime thật (Zero Smoke-Speculation):** Integration/e2e PASS trên DB/Broker thật + `runtime_check` (process mới listen, request thật, log sạch); nếu thiếu môi trường phải phối hợp với User trace runtime value thật. Build-OK / Smoke-Test-OK ≠ Done. **(G4) Edge-case & Concurrency:** Null/empty, trùng, lỗi mạng/DB, multi-worker batch/timeout vs heartbeat, watermark continuity sau outage; cấm nuốt lỗi. **(G5) Regression:** Test suite cũ PASS; kiểm caller/consumer. **(G6) Output Correctness:** Đối soát giá trị & số lượng (`count nguồn == count đích`). **(G7) Adversarial Line-by-Line Audit & DG3:** Phản biện từng dòng `git diff` + `DG3` sau lần sửa cuối + `/security-agent` + xuất `audit_report_<suffix>.md`. **(G8) Bằng chứng vật lý:** Lưu lệnh + output vào `06_validation_*`, `11_report_*`, `audit_report_*`, `05_progress_*`.
     ```
   - **Dòng thay thế:**
     ```markdown
     ### 13. Quy tắc Phân cách Brain/Muscle & Luật Phòng Sạch (Clean-Room Rule)
     - Khi ở vai trò Brain, đang Audit, hoặc chưa có `APPROVE`: **CẤM** `write_to_file`/`replace_file_content` trên source code (`.go`, `.ts`, `.tsx`, `.js`, `.py`, `.sql`, v.v.). Phải lập `09_tasks_solution_*.md` → qua `DG1/DG2` → chờ `APPROVE`.
     - **Clean-Room Rule (`mode: clean-room`):** *"Model thấy code gốc thì không bao giờ được viết code, và model viết code thì không bao giờ được thấy code gốc!"* Phiên đã đọc `clean_room_forbidden_sources` bị Hook `H4` chặn trực tiếp viết code; bắt buộc uỷ quyền cho Sub-agent mới (`invoke_subagent`) chỉ nhận `Clean-Room Approved Spec` đã được `/laya_router` (`DG1`) duyệt.

     ### 14. Quy tắc Đảm bảo Chất lượng Đầu ra (Definition of Done — Gates G1–G8)
     BẮT BUỘC pass đủ 8 Gate trước khi báo Done (xem `/qa-agent`): **(G1) Truy vết:** Đối chiếu `01_requirements_*`. **(G2) Reproduce (Red→Green):** Tái hiện bug trước khi sửa; sau sửa PASS. **(G3) Test, 3-Mutant Gate & Runtime thật (Zero Smoke-Speculation):** Integration/e2e PASS + chạy `tooling/mutation_gate.py` với **≥3 bản cố tình làm hỏng** (code chuẩn PASS test, cả 3 bản hỏng đều bị test đánh trượt `KILLED`, khôi phục `SHA-256` 100%) + `runtime_check` (process mới listen, request thật, log sạch); nếu thiếu môi trường phải phối hợp với User trace runtime value thật. Build-OK / Smoke-Test-OK ≠ Done. **(G4) Edge-case & Concurrency:** Null/empty, trùng, lỗi mạng/DB, multi-worker batch/timeout vs heartbeat, watermark continuity sau outage; cấm nuốt lỗi. **(G5) Regression:** Test suite cũ PASS; kiểm caller/consumer. **(G6) Output Correctness:** Đối soát giá trị & số lượng (`count nguồn == count đích`). **(G7) Adversarial Line-by-Line Audit & DG3:** Phản biện từng dòng `git diff` + `DG3` sau lần sửa cuối + `/security-agent` + xuất `audit_report_<suffix>.md`. **(G8) Bằng chứng vật lý:** Lưu lệnh + output vào `06_validation_*`, `11_report_*`, `audit_report_*`, `05_progress_*`.
     ```

---

### 2.2. File 2–5: Cập nhật 4 Core Skills (`brain-delegate`, `muscle-execute`, `qa-agent`, `workspace-docs`)

1. **`skills/brain-delegate/SKILL.md:42-50` (Bước 3):**
   - Bổ sung mục **6. `Clean-Room Approved Spec` (bắt buộc khi `01_requirements_<suffix>.md` có `mode: clean-room`):** Bảng các quy tắc hành vi (`CR-01..CR-n`) kèm `REF:` đã được `/laya_router` (`DG1`) duyệt `PASS`, hoàn toàn không chứa code rườm rà của file gốc cũ để bàn giao cho Clean-Room Builder Sub-agent ở chặng S4.
2. **`skills/muscle-execute/SKILL.md:45-62` (Bước 4 & Bước 5):**
   - Tại Bước 4 (`L45-L52`): Bổ sung quy tắc **Clean-Room Sub-Agent Isolation**: nếu `01_requirements_<suffix>.md` khai báo `clean_room_forbidden_sources`, phiên đã đọc các file đó không được tự sửa code mà dùng `invoke_subagent` chỉ truyền `Clean-Room Approved Spec`.
   - Tại Bước 5 (`L53-L62`): Bổ sung mục **Kiểm thử Đột biến 3 Bản Làm Hỏng (`tooling/mutation_gate.py`)** để xác nhận bộ test đánh trượt cả 3 bản lỗi (`KILLED`) và khôi phục `SHA-256` 100%.
3. **`skills/qa-agent/SKILL.md:29-40` (Gate G3):**
   - Bổ sung mục **5. Cổng Kiểm thử Đột biến 3 Bản Làm Hỏng (`3-Mutant Verification Gate` — `tooling/mutation_gate.py`):** Chống bệnh viết test hình thức luôn pass (`L039`): chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py --workspace <ws> --suffix <suffix> --mutations <mutations.json>` để chứng minh `baseline == PASS` và `>= 3 mutants == KILLED`.
4. **`skills/workspace-docs/SKILL.md:15-42`:**
   - Ghi chú rõ siêu dữ liệu tùy chọn `mode: clean-room` và `clean_room_forbidden_sources: <csv_paths>` trong `01_requirements_<suffix>.md`, cùng file bằng chứng `06_mutation_summary_<suffix>_<ts>.json`.

---

### 2.3. File 6: Tạo mới `/Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py`

Thiết kế tuân thủ 100% pattern của `tooling/laya_gate.py` và `hooks/agy/agy_guard.py` (đã khắc phục triệt để các phát hiện **F3, F4, F5**):
- **Giải quyết F3 (Chống False-Kill do SyntaxError / Timeout):**
  - Với file `.py`, gọi `ast.parse(mutated_text, filename=str(fpath))` ngay trong RAM trước khi ghi ra đĩa. Nếu ném `SyntaxError` $\rightarrow$ trả về ngay mã `3` (`INVALID_INPUT: Mutant <id> gây lỗi cú pháp SyntaxError, không phải đột biến logic`).
  - Khi chạy `run_test_cmd`, nếu `timed_out == True` $\rightarrow$ trạng thái là `"TIMEOUT"` (đưa vào danh sách thất bại, không tính là `"KILLED"`).
- **Giải quyết F4 (Kế thừa `.agent-project.json` & chuẩn resolve path):**
  - Có hàm `resolve_target_path(raw_path, base_dirs)` tìm file theo `[Path.cwd(), AGENT_ROOT, mutations_file.parent]` giống `laya_gate.py:97-105`.
  - Có hàm `component_for(file_path)` đọc `.agent-project.json` gần nhất giống `agy_guard.py:270-302` để tự lấy `cwd` và lệnh `test` mặc định nếu mutant không truyền `test_cmd`.
- **Giải quyết F5 (An toàn tuyệt đối trước `SIGTERM` & Kiểm chứng `SHA-256`):**
  - Đăng ký `signal.signal(signal.SIGTERM, _sigterm_to_exit)` trong lúc áp dụng mutant để mọi tín hiệu dừng đều đi qua khối `finally: fpath.write_bytes(orig_bytes)` và đối chiếu `sha256_bytes(fpath.read_bytes()) == orig_sha`.

---

### 2.4. File 8: Sửa `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py`

1. **Khối 1 — Bổ sung Regex & Helper đọc cấu hình Clean-Room / Mutation Gate (`sau dòng 54` và `sau dòng 238`):**
   - **Tại `agy_guard.py:51-55`:**
     ```python
     PROD_MOCK_POLLUTION = re.compile(
         r"(\b\w+Fn\s+func\s*\(|\blookupMasterTargetsFn\b|\brunEventsFn\b|\bsqlmock\b|\bfakemodel\b)",
     )
     CLEAN_ROOM_MODE_RE = re.compile(r"^\s*mode\s*:\s*clean-room\b", re.I | re.M)
     CLEAN_ROOM_FORBIDDEN_RE = re.compile(r"^\s*clean_room_forbidden_sources\s*:\s*(.+)$", re.I | re.M)
     CLEAN_ROOM_SPEC_RE = re.compile(r"clean-room\s+approved\s+spec", re.I)
     MANIFEST = ".agent-project.json"
     ```
   - **Tại `agy_guard.py:238` (ngay sau `laya_gate_status`):**
     ```python
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
                     candidates = sorted(ws_dir.glob(f"06_mutation_summary_{suffix}_*.json"), key=lambda p: p.name)
                 if not candidates:
                     candidates = sorted(ws_dir.glob("06_mutation_summary_*.json"), key=lambda p: p.name)
         except OSError:
             return None
         if not candidates:
             return None
         try:
             data = json.loads(candidates[-1].read_text(encoding="utf-8"))
             if data.get("killed_count", 0) < 3 or not data.get("all_restored_sha256", False):
                 return "FAIL"
             return data.get("gate")
         except OSError:
             return None
     ```

2. **Khối 2 — Tại `handle_pre_write` (`H4a` & `H4b`, dòng `379-456`):**
   - **Trong `H4a` (`agy_guard.py:409-416` — Giải quyết F1):**
     Sau khi kiểm tra `missing_s1`, bổ sung kiểm tra `mode: clean-room` an toàn với cả `write_to_file` và `replace_file_content`:
     ```python
         is_cr, _ = clean_room_config(ws_name, ws_root, suffix)
         if is_cr:
             sol_path = ws_dir / f"09_tasks_solution_{suffix}.md"
             existing_sol = ""
             try:
                 if sol_path.is_file() and not args.get("Overwrite"):
                     existing_sol = sol_path.read_text(encoding="utf-8", errors="ignore")
             except OSError:
                 existing_sol = ""
             effective_sol = (
                 existing_sol.replace(str(args["TargetContent"]), str(args.get("ReplacementContent") or ""), 1)
                 if (existing_sol and args.get("TargetContent"))
                 else (existing_sol + "\n" + body)
             )
             if not CLEAN_ROOM_SPEC_RE.search(effective_sol):
                 return {
                     "decision": "deny",
                     "reason": (
                         f"Rule 9/13 (Clean-Room Mode): 01_requirements_{suffix}.md khai báo mode: clean-room — "
                         f"09_tasks_solution_{suffix}.md bắt buộc phải có mục 'Clean-Room Approved Spec'."
                     ),
                 }
     ```
   - **Trong `H4b` (`agy_guard.py:430-455` — Giải quyết F2):**
     Ngay sau khi lấy `ws = active_ws(st, ws_root)`, kiểm tra ô nhiễm nguồn `clean_room_forbidden_sources`:
     ```python
         sol_sfx = latest_solution_suffix(st)
         is_cr, forbidden_srcs = clean_room_config(ws, ws_root, sol_sfx)
         if is_cr and forbidden_srcs:
             contaminated = contaminated_clean_room_reads(st, forbidden_srcs)
             if contaminated:
                 return {
                     "decision": "deny",
                     "reason": (
                         "Rule 13 (Clean-Room Isolation): Phiên hiện tại đã đọc code gốc trong "
                         f"clean_room_forbidden_sources ({', '.join(contaminated)}) nên CẤM trực tiếp viết code! "
                         "Bắt buộc dùng invoke_subagent (Clean-Room Builder) chỉ nhận Clean-Room Approved Spec."
                     ),
                 }
     ```

3. **Khối 3 — Tại `handle_stop` (`H7`, dòng `560-575` — Giải quyết F6):**
   - Trong khối `if src:`, bổ sung kiểm tra `mutation_gate`:
     ```python
         is_cr, _ = clean_room_config(ws, ws_root, sol_sfx)
         mut_status = mutation_gate_status(ws, ws_root, sol_sfx)
         if is_cr or mut_status is not None or touched(st, r"mutation_gate\.py", u_idx):
             if mut_status != "PASS" or not touched(
                 st, r"(mutation_gate\.py|06_mutation_summary_)", last
             ):
                 miss.append(
                     "MutationGate (3-Mutant Verification): chưa chạy hoặc chưa PASS `mutation_gate.py` "
                     "(>=3 bản cố tình làm hỏng phải bị test đánh trượt KILLED) sau lần sửa code cuối"
                 )
     ```

---

### 2.5. File 10: Sửa `/Users/trainguyen/Documents/work/agent/tooling/verify_governance.py`

1. **Tại `verify_governance.py:21-23`:**
   ```python
   TIER_RE = re.compile(r"^\s*tier\s*:\s*(T[0-3])\b", re.I | re.M)
   CLEAN_ROOM_MODE_RE = re.compile(r"^\s*mode\s*:\s*clean-room\b", re.I | re.M)
   CLEAN_ROOM_SPEC_RE = re.compile(r"clean-room\s+approved\s+spec", re.I)
   LOG_LINE_RE = re.compile(r"^-\s*\[\d{4}-\d{2}-\d{2}[^\]]*\]\s*\[Agent:[a-zA-Z0-9.\-_]+\]\s+.+")
   ```
2. **Tại `verify_governance.py:193-212` (trong `verify_workspace_docs`):**
   - Kiểm tra nếu file `01_requirements` của task hiện tại (`01_requirements_<suffix>.md` hoặc file mới nhất) có `CLEAN_ROOM_MODE_RE`:
     - **Ở cả `S3` và `S6` (Tier `T2`/`T3`):** Kiểm tra `09_tasks_solution_<suffix>.md` có chứa `CLEAN_ROOM_SPEC_RE` (`Clean-Room Approved Spec`). Nếu thiếu $\rightarrow$ `log_error` và trả về `False`.
     - **Ở `S6` (Tier `T2`/`T3`):** Kiểm tra tồn tại file `06_mutation_summary_<suffix>_*.json` (hoặc `06_mutation_summary_*.json`) có `"gate": "PASS"`, `"killed_count" >= 3`, và `"all_restored_sha256": true`. Nếu thiếu hoặc `!= "PASS"` $\rightarrow$ `log_error` và trả về `False`.
