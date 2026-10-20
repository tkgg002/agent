# 10 — Gap Analysis & Adversarial Plan Critique: Rà soát Phản biện Plan & Code Demo `clean_room_and_mutation_gate`

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Ngày thực hiện:** `2026-10-20`
- **Chế độ:** `PROPOSE-ONLY AUDIT (Chỉ đọc, phản biện thiết kế & lập lại plan, tuyệt đối không sửa source code)`

---

## 1. Tóm tắt Kết quả Phản biện (Executive Summary)

Qua rà soát phản biện từng hạng mục trong `08_tasks_clean_room_and_mutation_gate.md`, từng dòng code demo trong `09_tasks_solution_clean_room_and_mutation_gate.md`, và đối chiếu trực tiếp với Architecture/Design Pattern của hệ thống (`hooks/agy/agy_guard.py`, `tooling/laya_gate.py`, `tooling/verify_governance.py`, `.agent-project.json`), bản plan đầu tiên có **6 điểm hở kỹ thuật & thiết kế (`F1` – `F6`, tất cả đều `VERIFIED` kèm `file:line`)** cần được khắc phục ngay trong bản thiết kế R1 trước khi thực thi.

| Mã | Mức độ | Hạng mục bị ảnh hưởng | Bản chất Sai sót / Thiếu sót trong Plan ban đầu | Cách khắc phục (Simplicity First & Minimal Impact) |
| :--- | :---: | :--- | :--- | :--- |
| **F1** | **P0 (Critical)** | `hooks/agy/agy_guard.py:347, 379-416` (`H4a`) | Kiểm tra `re.search(r"clean-room\s+approved\s+spec", body)` trực tiếp trên `body` (`CodeContent + ReplacementContent`) sẽ **chặn oan mọi lệnh `replace_file_content`** khi cập nhật một phần nhỏ của `09_tasks_solution_<suffix>.md`. | Khi file `09_tasks_solution_<suffix>.md` đã tồn tại trên đĩa, đọc nội dung hiện có và ghép với `body` (hoặc mô phỏng kết quả sau replace) trước khi kiểm tra mục `Clean-Room Approved Spec`. |
| **F2** | **P0 (Critical)** | `hooks/agy/agy_guard.py:419-455` (`H4b`) | Quy tắc cốt lõi *"Model thấy code gốc thì không bao giờ được viết code, model viết code thì không bao giờ được thấy code gốc"* mới chỉ dừng ở lời văn trong `GEMINI.md` (**Paper-Only Governance — `L194`**), chưa có rào chắn vật lý trong `H4b`. | Bổ sung hàm `clean_room_contaminated_reads(st, forbidden_sources)` (~18 dòng) trong `agy_guard.py`: nếu `01_requirements_<suffix>.md` khai báo `clean_room_forbidden_sources: <paths>` và transcript hiện tại đã `view_file` các file đó, `H4b` **chặn cứng (`deny`)** việc trực tiếp sửa source code, bắt buộc uỷ quyền qua `invoke_subagent` (Clean-Room Builder session mới). |
| **F3** | **P0 (Critical)** | Code demo `tooling/mutation_gate.py` | Chấm `killed = res["returncode"] != 0` một cách ngây thơ: nếu biến thể lỗi (`mutant_content`) bị **lỗi cú pháp (`SyntaxError` / `IndentationError`)** hoặc bị **treo (`timed_out == True`)**, `returncode` vẫn `!= 0` và được tính là `KILLED` $\rightarrow$ báo `PASS` giả tạo! | (1) Với file `.py`, dùng `ast.parse(mutated_text)` kiểm tra cú pháp ngay trong RAM trước khi chạy test (nếu `SyntaxError` $\rightarrow$ trả về `3 INVALID_INPUT`); (2) Nếu `timed_out == True`, đánh dấu `status = "TIMEOUT"` (không tính là `KILLED`, ép `gate = "FAIL"`). |
| **F4** | **P1 (High)** | Code demo `tooling/mutation_gate.py` vs `.agent-project.json` | Code demo ban đầu bắt khai báo thủ công `cwd` và `test_cmd` cho từng mutant mà không kết nối với chuẩn **ADR-07 (`.agent-project.json`)** của hệ thống (`agy_guard.py:270-302`), đồng thời resolve đường dẫn tương đối chỉ theo `Path.cwd()`. | Tái sử dụng pattern `component_for(file_path)` (đọc `.agent-project.json`) để tự động suy ra `cwd` và lệnh `test` mặc định của component nếu mutant không khai báo riêng; hỗ trợ resolve path theo cả `AGENT_ROOT` và `mutations_file.parent` (đúng pattern `laya_gate.py:97-105`). |
| **F5** | **P1 (High)** | Code demo `tooling/mutation_gate.py` (An toàn `SIGTERM`) | Khối `try...finally` trong Python bắt được `Exception` và `KeyboardInterrupt` (`SIGINT`), nhưng **không tự chạy `finally` nếu tiến trình nhận tín hiệu `SIGTERM`** từ OS/runner timeout $\rightarrow$ nguy cơ để lại code đột biến trên đĩa! | Đăng ký `signal.signal(signal.SIGTERM, _raise_system_exit)` trước khi ghi mutant để biến `SIGTERM` thành `SystemExit(143)`, đảm bảo khối `finally` **luôn luôn chạy** và khôi phục byte-for-byte file gốc (`SHA-256` khớp 100%). |
| **F6** | **P1 (High)** | `hooks/agy/agy_guard.py:560-574` (`H7`) & `tooling/verify_governance.py:59-66` | (1) Thiếu kiểm tra `06_mutation_summary_<suffix>_*.json` phải được tạo **SAU lần sửa code cuối (`last = src[-1][0]`)** (bẫy dùng lại summary cũ — `L006/L007`); (2) Rủi ro làm gãy 24 unit test hiện có nếu thêm mục thứ 7 bắt buộc cứng cho các workspace cũ không chạy `clean-room`. | (1) Giữ nguyên 6 phần của `AUDIT_REQUIRED_PATTERNS` (tương thích ngược 100% với 24 test cũ); (2) Khi task có `mode: clean-room` (hoặc có file `06_mutation_summary_*.json`), `H7` kiểm tra cả `mutation_gate_status == "PASS"` **VÀ** `touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last)`. |

---

## 2. Phân tích Chi tiết Từng Phát hiện Kèm Bằng chứng `file:line`

### 2.1. Phát hiện F1 (`P0`): Lỗ hổng chặn oan `replace_file_content` trong Hook `H4a`
- **Bằng chứng tại `hooks/agy/agy_guard.py:344-347` và `379-381`:**
  ```python
  344:     raw_args = (payload.get("toolCall") or {}).get("args") or {}
  345:     args = {k: _unquote(v) for k, v in raw_args.items()}
  346:     f = tgt(args)
  347:     body = str(args.get("CodeContent") or "") + str(args.get("ReplacementContent") or "")
  ...
  379:     sol_match = re.search(r"/workspaces/([^/]+)/09_tasks_solution_([^/]+)\.md$", f)
  380:     if sol_match and is_mem_path(f, mem_root):
  ```
- **Phản biện:** Khi dùng `replace_file_content` để sửa vài dòng trong `09_tasks_solution_<suffix>.md`, `args["CodeContent"]` rỗng và `args["ReplacementContent"]` chỉ chứa đoạn văn bản mới của khối được thay. Nếu kiểm tra `if not re.search(r"clean-room\s+approved\s+spec", body, re.I)` trực tiếp trên `body`, lệnh `replace_file_content` sẽ bị `deny` ngay cả khi file trên đĩa đã có mục `## Clean-Room Approved Spec`!
- **Giải pháp chuẩn Minimal Impact:**
  ```python
  existing_sol = Path(f).read_text(encoding="utf-8", errors="ignore") if Path(f).is_file() else ""
  combined_sol = (existing_sol + "\n" + body) if args.get("TargetContent") else body
  ```
  Nếu đang xóa mất `Clean-Room Approved Spec` trong `TargetContent` mà không có trong `ReplacementContent`, hoặc tạo mới bằng `write_to_file` mà thiếu `Clean-Room Approved Spec`, mới chặn `deny`.

### 2.2. Phát hiện F2 (`P0`): Cưỡng chế Vật lý Quy tắc Phòng Sạch (`clean_room_forbidden_sources`) trong `H4b`
- **Bằng chứng tại `hooks/agy/agy_guard.py:178-196` & `419-455`:**
  - `agy_guard.py:178-196` đã có hàm `project_reads_since(st, after, mem_root)` quét các lệnh `view_file`, `grep_search`, `find_by_name` trong `transcript.jsonl`.
  - Tuy nhiên, tại `H4b` (`hooks/agy/agy_guard.py:419-455`), khi kiểm tra `if is_src(f, mem_root):`, chưa có bước kiểm tra xem phiên hiện tại có bị **ô nhiễm ngữ cảnh (Contaminated Context)** bởi việc đọc các file code gốc bị cấm (`clean_room_forbidden_sources`) hay không.
- **Giải pháp chuẩn Core Systems:**
  - Trong `01_requirements_<suffix>.md`, hỗ trợ dòng khai báo tùy chọn:
    ```markdown
    mode: clean-room
    clean_room_forbidden_sources: path/to/legacy_module.go, path/to/old_helper.go
    ```
  - Khi `clean_room_forbidden_sources` được khai báo, hàm `contaminated_clean_room_reads(st, forbidden_list)` kiểm tra toàn bộ lịch sử `calls(st)` của phiên hiện tại: nếu phiên này đã gọi `view_file` trên bất kỳ file nào khớp với `clean_room_forbidden_sources`, `H4b` trả về `decision: "deny"`, buộc Agent chính (Brain/Extractor) phải gọi `invoke_subagent` (phiên mới có `transcript.jsonl` riêng sạch 100%, chỉ nhận nội dung `Clean-Room Approved Spec`) để viết code!

### 2.3. Phát hiện F3, F4, F5 (`P0/P1`): Hoàn thiện `tooling/mutation_gate.py` theo đúng DNA của `tooling/laya_gate.py`
- **Bằng chứng tại `tooling/laya_gate.py:97-105` & `hooks/agy/agy_guard.py:270-302`:**
  - `laya_gate.py:97-105` dùng `resolve_ref_path(raw_path, base_dirs)` để phân giải đường dẫn tương đối theo `[Path.cwd(), AGENT_ROOT, * extra_dirs]`.
  - `agy_guard.py:270-302` dùng `manifest_for(file_path)` và `component_for(file_path)` để tìm `.agent-project.json` gần nhất.
- **Giải pháp tích hợp trong `tooling/mutation_gate.py`:**
  1. **Tự động đọc `.agent-project.json`:** Nếu một phần tử trong `mutations.json` không khai báo `test_cmd` hoặc `cwd`, tự động lấy từ `component_for(fpath)` (`comp.get("test") or comp.get("checks", [None])[0]`).
  2. **Chống False-Kill do lỗi cú pháp (`.py`) và Timeout:**
     - Nếu `fpath.suffix == ".py"`, gọi `ast.parse(mutated_text, filename=str(fpath))` trước khi ghi file. Nếu ném `SyntaxError` $\rightarrow$ trả về exit code `3` (`INVALID_INPUT: Mutant gây lỗi cú pháp SyntaxError`).
     - Nếu `res["timed_out"] == True` $\rightarrow$ gán `status = "TIMEOUT"` và đưa vào `failed_ids` (không tính là `KILLED`).
  3. **Bắt tín hiệu `SIGTERM` để bảo vệ file gốc 100%:**
     - Trước vòng lặp ghi mutant, gắn `signal.signal(signal.SIGTERM, _handle_sigterm)` (ném `SystemExit(143)`) và trả lại handler cũ trong `finally`, đảm bảo không bao giờ để lại mã đột biến trên đĩa.

### 2.4. Phát hiện F6 (`P1`): Chống dùng lại `06_mutation_summary_*.json` cũ và Giữ tương thích ngược 100%
- **Bằng chứng tại `hooks/agy/agy_guard.py:560-574` & `hooks/agy/tests/test_agy_guard.py:639-682`:**
  - Trong `test_24_h7_passes_with_complete_audit_report_and_post_edit_dg3` (`test_agy_guard.py:639-682`), fixture tạo một workspace `feat-audit-ok` không có `01_requirements` mang `mode: clean-room`.
  - Vì vậy, trong `handle_stop` (`H7`), điều kiện kiểm tra bắt buộc `mutation_gate` chỉ kích hoạt khi:
    - Task hiện tại khai báo `mode: clean-room` trong `01_requirements_<suffix>.md`, **HOẶC**
    - Trong workspace đã có file `06_mutation_summary_<suffix>_*.json` (hoặc Agent đã gọi `mutation_gate.py` trong lượt hiện tại).
  - Khi kích hoạt và có `src` (`last = src[-1][0]`), bắt buộc:
    1. `mutation_gate_status(ws, ws_root, sol_sfx) == "PASS"`
---

## 3. Kết luận Kiểm toán Phản biện & Xác nhận từ `/laya_router` (`DG3`)

- **Summary File:** [`06_laya_summary_dg3_20261020T114944.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_laya_summary_dg3_20261020T114944.json)
- **Trusted Rounds:** `r3` (đã loại bỏ đối chứng âm tính `CF1`, `CF2` và xác nhận đối chứng dương tính `CT1`)
- **Kết quả kiểm định từng phát hiện:**
  - `A-F1` (`hooks/agy/agy_guard.py:344-348` — `body` trong `handle_pre_write`): `FLAG -> RESOLVED` vào [`04_decisions_process_optimization.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/04_decisions_process_optimization.md) kèm `REF: hooks/agy/agy_guard.py:344-348`.
  - `A-F2` (`hooks/agy/agy_guard.py:119-125` — `calls(st)` quét lịch sử tool call trong `transcript.jsonl`): `ACCEPT` (`0.7714`).
  - `A-F4` (`hooks/agy/agy_guard.py:270-283` — `manifest_for` đọc `.agent-project.json`): `ACCEPT` (`0.7889`).
  - `A-F6` (`hooks/agy/agy_guard.py:89-96` — bảo toàn 6 phần tử `AUDIT_REQUIRED_PATTERNS`): `ACCEPT` (`0.5272`).
  - `A-SCOPE` (`tooling/laya_gate.py:97-105` — chuẩn `resolve_ref_path`): `ACCEPT` (`0.6742`).
- **Trạng thái Gate `DG3`:** **`PASS`**. Toàn bộ 6 phát hiện (`F1` – `F6`) đã được kiểm chứng trên mã nguồn thật (`VERIFIED`), cập nhật vào [`09_tasks_solution_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/09_tasks_solution_clean_room_and_mutation_gate.md) (Revision R1) và lập chi tiết từng dòng code cần sửa trong [`12_implementation_plan_clean_room_and_mutation_gate.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/12_implementation_plan_clean_room_and_mutation_gate.md).

