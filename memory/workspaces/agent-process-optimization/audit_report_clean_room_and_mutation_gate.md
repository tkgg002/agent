# Final Adversarial Audit Report: Tích hợp Clean-Room Rebuild Protocol & 3-Mutant Test Verification Gate (`clean_room_and_mutation_gate`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `clean_room_and_mutation_gate`
- **Tier:** `T2` (`mode: clean-room`)
- **Ngày hoàn tất:** `2026-10-20`

---

## 1. Line-by-Line Diff & Task Critique (Phản biện từng dòng update / `git diff` & từng task)

| Task / File:Line | Thay đổi thực tế | Phản biện (Adversarial Question: Cách phá / Edge case) | Bằng chứng Code / Runtime thật | Kết luận |
| :--- | :--- | :--- | :--- | :---: |
| **`T8.1`**<br>[`GEMINI.md:71, 73, 94-99`](file:///Users/trainguyen/Documents/work/agent/GEMINI.md#L71-L99)<br>`skills/{brain-delegate,muscle-execute,qa-agent,workspace-docs}/SKILL.md` | Bổ sung quy tắc **Clean-Room Rebuild (`mode: clean-room` & `clean_room_forbidden_sources`)** tại S3/S4/Rule #13 và **3-Mutant Test Verification Gate (`tooling/mutation_gate.py`)** tại S4/Rule #14 Gate G3. | *Liệu việc thêm quy định vào `GEMINI.md` có làm vượt ngưỡng dung lượng context hay làm sai lệch số hiệu 22 Rules không?* | `GEMINI.md` tăng từ `129` lên `131` dòng (`16,294` bytes `< 18,000` bytes), giữ nguyên 100% số hiệu Rule #0–#22. | **PASS** |
| **`T8.2`**<br>[`tooling/mutation_gate.py:34-43, 160-169`](file:///Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py#L34-L169) | Hàm `_invalidate_pyc(fpath)`, `resolve_target_file`, `auto_detect_component`, và kiểm tra `ast.parse(mutated_text)` trước khi ghi xuống đĩa. | *Điều gì xảy ra nếu mutant làm gãy cú pháp Python (`SyntaxError`), hoặc nếu `target_content` và `mutant_content` có cùng độ dài byte (`return a + b` vs `return a - b`) chạy trong cùng 1 giây khiến Python nạp `.pyc` cũ từ `__pycache__`?* | 1. Dòng `160-169` gọi `ast.parse(mutated_text)` trong RAM và trả mã `3 INVALID_INPUT` ngay nếu có `SyntaxError` (kiểm chứng bởi `test_03_syntax_error_mutant_rejected_by_ast_parse`).<br>2. Dòng `74-75` đặt `PYTHONDONTWRITEBYTECODE=1` và dòng `230, 234` gọi `_invalidate_pyc(fpath)` xoá `__pycache__/<stem>.*.pyc`, triệt tiêu hoàn toàn lỗi nạp `.pyc` cũ trong cùng 1 giây. | **PASS** |
| **`T8.2`**<br>[`tooling/mutation_gate.py:212-249`](file:///Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py#L212-L249) | Vòng lặp tiêm mutant bọc `signal.signal(signal.SIGTERM, _sigterm_to_exit)` và `try...finally` khôi phục `orig_bytes` + kiểm tra `SHA-256`, phân loại `TIMEOUT` / `KILLED` / `SURVIVED`. | *Nếu mutant gây vòng lặp vô hạn (`timed_out == True`) hoặc tiến trình bị gửi `SIGTERM` giữa chừng thì có làm hỏng file gốc hoặc báo `KILLED` giả không?* | 1. Dòng `212-217` chuyển `SIGTERM` thành `SystemExit(143)` để `finally:` (dòng `232-235`) luôn ghi trả `orig_bytes` và kiểm tra `restored_sha == orig_sha`.<br>2. Dòng `241-243` phân loại `res["timed_out"] == True` thành `"TIMEOUT"` (`timeout_ids.append(mid)`) và đánh trượt cổng (`gate = "FAIL"`, kiểm chứng bởi `test_04_timeout_mutant_classified_as_timeout_and_fails_gate`). | **PASS** |
| **`T8.3`**<br>[`hooks/agy/agy_guard.py:243-327, 507-528`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L507-L528) (`H4a`) | Kiểm tra mục `Clean-Room Approved Spec` khi `01_requirements_<suffix>.md` khai báo `mode: clean-room`. | *Nếu Agent dùng `replace_file_content` để sửa vài dòng ở cuối `09_tasks_solution_<suffix>.md`, `body` không chứa tiêu đề `Clean-Room Approved Spec` thì có bị chặn oan không (`F1`)?* | Dòng `512-518` đọc `existing_sol = sol_path.read_text(...)` và ghép `existing_sol.replace(TargetContent, ReplacementContent, 1)` trước khi kiểm tra `CLEAN_ROOM_SPEC_RE` (kiểm chứng bởi `test_25_h4a_clean_room_requires_approved_spec_and_supports_partial_replace`). | **PASS** |
| **`T8.3`**<br>[`hooks/agy/agy_guard.py:557-570`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L557-L570) (`H4b`) | Cưỡng chế cách ly phòng sạch (`Clean-Room Isolation`) qua `clean_room_forbidden_sources` và `contaminated_clean_room_reads(st, forbidden)`. | *Làm sao ngăn phiên Brain đã đọc code gốc (`view_file`, `grep_search`, `find_by_name`) tự ý viết code mới thay vì gọi `invoke_subagent` (`F2`)?* | Dòng `275-294` quét toàn bộ `calls(st)` trong `transcript.jsonl`; nếu phát hiện đã đọc file thuộc `clean_room_forbidden_sources`, dòng `561-570` trả về `decision: "deny"` (kiểm chứng bởi `test_26_h4b_clean_room_blocks_contaminated_session_from_writing_source` và bị bắt sống bởi Mutant `M2`). | **PASS** |
| **`T8.3`**<br>[`hooks/agy/agy_guard.py:701-707`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L701-L707) (`H7`) & [`tooling/verify_governance.py:197-255`](file:///Users/trainguyen/Documents/work/agent/tooling/verify_governance.py#L197-L255) | Kiểm tra `mutation_gate_status == "PASS"` (`killed_count >= 3`, `all_restored_sha256 == True`) và được chạy sau lần sửa code cuối (`last`). | *Liệu việc bổ sung kiểm tra này có làm gãy 24 unit test cũ không? Và nếu Agent sửa code thêm sau khi chạy `mutation_gate.py` thì H7 có phát hiện được không (`F6`)?* | 1. Giữ nguyên 6 phần tử `AUDIT_REQUIRED_PATTERNS`, chỉ kích hoạt khi `is_cr` hoặc có file `06_mutation_summary_*.json` $\rightarrow$ toàn bộ 24 test cũ + 4 test mới (`28/28`) PASS.<br>2. Dòng `704` kiểm tra `touched(st, r"(mutation_gate\.py|06_mutation_summary_)", last)` sau chỉ số `last = src[-1][0]` (kiểm chứng bởi `test_27` và `test_28`). | **PASS** |

---

## 2. Anti-Smoke-Speculation & Real Runtime Trace Verification (Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật)

- **Kiểm tra Production Mock Pollution:** Quét toàn bộ code mới trong `tooling/mutation_gate.py`, `hooks/agy/agy_guard.py`, `tooling/verify_governance.py` xác nhận **0 hàm mock/closure giả** (`*Fn`, `sqlmock`, `fakemodel`).
- **Bằng chứng Test & Runtime thật (Bao gồm 3-Mutant Verification thật trên codebase `/agent`):**
  1. **Unit & Integration Test Suite (`hooks` + `tooling`):**
     - `python3 -m unittest discover -s hooks/agy/tests -p 'test_*.py' -v` $\rightarrow$ **`Ran 28 tests in 0.072s — OK`**.
     - `python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v` $\rightarrow$ **`Ran 20 tests in 1.409s — OK`**.
  2. **Runtime Execution của `tooling/mutation_gate.py` với 3 Bản Cố tình Làm hỏng Thật (`M1`, `M2`, `M3`):**
     - Lệnh chạy thật:
       `python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py --workspace agent-process-optimization --suffix clean_room_and_mutation_gate --mutations /Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_mutations_clean_room_and_mutation_gate.json`
     - Biên bản vật lý: [`06_mutation_summary_clean_room_and_mutation_gate_20261020T120802.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_mutation_summary_clean_room_and_mutation_gate_20261020T120802.json)
     - Kết quả runtime thực tế:
       - `M1` (Vô hiệu hóa kiểm tra `Clean-Room Approved Spec` tại `H4a` trong `hooks/agy/agy_guard.py`): **`KILLED`** (`returncode = 1`, bị bắt sống bởi `test_25`), `SHA-256` trước và sau khôi phục khớp 100% (`b183347e7a6618ca916bb9bc013e37c0f6fdc98a77835e8f6d83f08a88a747b2`).
       - `M2` (Đảo ngược điều kiện chặn Contaminated Context tại `H4b` trong `hooks/agy/agy_guard.py`): **`KILLED`** (`returncode = 1`, bị bắt sống bởi `test_26`), `SHA-256` trước và sau khôi phục khớp 100% (`b183347e7a6618ca916bb9bc013e37c0f6fdc98a77835e8f6d83f08a88a747b2`).
       - `M3` (Vô hiệu hóa kiểm tra Clean-Room tại S3/S6 trong `tooling/verify_governance.py`): **`KILLED`** (`returncode = 1`, bị bắt sống bởi `test_clean_room_s3_and_s6_enforcement`), `SHA-256` trước và sau khôi phục khớp 100% (`85ccb86b8f4be731966b7d1ed95ef5d4ba34e9ea9cb68ccbe658082d6ffcb06d`).
- **Điểm cần phối hợp User trace runtime value (nếu có):** Toàn bộ công cụ CLI và Lifecycle Hooks của `/agent` chạy trực tiếp trên máy local của User và đã được trace đầy đủ đầu vào/đầu ra thật + mã băm `SHA-256`, không có thành phần khuất tầm nhìn cần giả lập.

---

## 3. Concurrency, Timeout/Heartbeat & Data Continuity Audit (Kiểm tra Đa luồng, Timeout & Liên tục dữ liệu)

- **Concurrency / Multi-worker / Debouncer & Bytecode Isolation:**
  - Các mutant được thực thi tuần tự từng biến thể trên cùng 1 tiến trình quản lý và xóa sạch `.pyc` (`_invalidate_pyc`) kèm `PYTHONDONTWRITEBYTECODE=1`, ngăn chặn xung đột cache `.pyc` khi đột biến xảy ra trong cùng 1 giây với Baseline.
  - Tên file biên bản `06_mutation_summary_<suffix>_<ts>.json` định danh theo cả `<suffix>` và `<ts>` để không ghi đè lẫn nhau khi một workspace thực hiện nhiều task kế tiếp.
- **Timeout / Heartbeat / AckWait & Process Signal Safety:**
  - Mỗi lệnh chạy test trong `mutation_gate.py` đều gắn `timeout=timeout_sec` (mặc định `60s`). Khi xảy ra `subprocess.TimeoutExpired`, hệ thống không tính nhầm là `KILLED` mà phân loại rõ `status = "TIMEOUT"` và đánh trượt cổng (`gate = "FAIL"`).
  - Bọc `signal.signal(signal.SIGTERM, _sigterm_to_exit)` đảm bảo ngay cả khi tiến trình cha timeout/kill `mutation_gate.py`, khối `finally:` vẫn chạy để khôi phục nguyên vẹn file gốc trước khi thoát.
- **Data Continuity / Watermark / State Integrity:**
  - Trước và sau mỗi mutant, mã băm `SHA-256` của file nguồn trên đĩa được đối chiếu (`sha256_before == sha256_after`), đảm bảo tính toàn vẹn dữ liệu 100% (`all_restored_sha256 == True`).

---

## 4. Anti-Speculation & Honesty Check (Kiểm tra Suy diễn & Báo cáo láo)

- Toàn bộ số dòng code, mã băm `SHA-256`, số lượng unit test (`28` tests trong `hooks/agy/tests`, `20` tests trong `tooling/tests`) và các file summary của `/laya_router` / `mutation_gate.py` trong báo cáo này đều được trích xuất 1-1 từ file vật lý trên đĩa sau khi hoàn tất code (tuân thủ nghiêm bài học `L001 #audit-report-line-drift`, không suy diễn, không báo cáo láo).

---

## 5. Self-Improvement Loop (Vòng lặp Phản tỉnh & Khắc phục — Lessons Audit)

- **Đối chiếu `lessons/INDEX.md`:** Đã kiểm tra và tuân thủ nghiêm ngặt `L001` (đối chiếu chính xác dòng code sau khi sửa), `L016` (Propose-Only tới khi có `APPROVE`), `L039` (chống Tautological Validation bằng 3-Mutant Gate), `L067` (bảo toàn tương thích ngược cho 24 unit test cũ).
- **Bài học mới đã ghi nhận vào `memory/global/lessons.md`:**
  - Tiêu đề: `### [2026-10-20] Bỏ sót kịch bản replace_file_content từng phần tại Hook H4a, thiếu kiểm tra ast.parse/Timeout chống False-Kill và xung đột bộ nhớ đệm bytecode .pyc 1 giây khi chạy kiểm thử đột biến`
  - Kiểm chứng `wc -l`: Tăng từ `1944` lên `1951` dòng (`249 -> 250` bài học).
  - Đã chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py` $\rightarrow$ `entries=250/250/250, index_bytes=13988 (<15000)`.

---

## 6. Decision Gate `/laya_router` DG3 Final Verdict

- **Summary File (Sau lần sửa code cuối):** [`06_laya_summary_dg3_20261020T120944.json`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/agent-process-optimization/06_laya_summary_dg3_20261020T120944.json)
- **Trusted Rounds:** `r3`
- **Kết quả kiểm định từng Claim trên code thật sau khi sửa:**
  - `A-CR-H4A` (`hooks/agy/agy_guard.py:507-523` — ghép `existing_sol` với `TargetContent -> ReplacementContent` trước khi kiểm tra `CLEAN_ROOM_SPEC_RE`): **`ACCEPT`** (`0.5737`).
  - `A-CR-H4B` (`hooks/agy/agy_guard.py:558-565` — chặn sửa code trực tiếp khi `contaminated_clean_room_reads` phát hiện đã đọc `clean_room_forbidden_sources`): **`ACCEPT`** (`0.9468`).
  - `A-MUT-RESTORE` (`tooling/mutation_gate.py:228-236` — khôi phục `orig_bytes`, xóa `.pyc` và kiểm tra `restored_sha` trong khối `finally`): **`ACCEPT`** (`0.7855`).
  - `A-GOV-S6` (`tooling/verify_governance.py:244-252` — kiểm tra `gate == "PASS"`, `killed_count >= 3`, `all_restored_sha256 == True` ở chặng `S6`): **`ACCEPT`** (`0.8676`).
  - Đối chứng âm tính (`CF1`, `CF2`) bị bác bỏ chính xác; đối chứng dương tính (`CT1`) được xác nhận chính xác.
- **Final Gate Verdict:** **`PASS` 🟢**
