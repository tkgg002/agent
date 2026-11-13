# 08. Tasks — Khắc Phục Độ Trễ, Đốt Token & Lọt Lỗi Của Agentic Core (`agent_core_v2`)

## Task 1 (`T1` — S1/S2/S3 Brain Investigation & Solution Design) [x]
- **Mô tả & Context `file:line`:**
  - Điều tra định lượng trên 5 phiên gần nhất (`e7fd58b2`, `d21bae7a`, `963b9484`, `84a93058`, `094a4735`) đối chiếu trực tiếp mã nguồn `hooks/agy/agy_guard.py:82-97,206-257,318-337,858-1376`, `.agents/hooks.json:10-22`, `tooling/laya_gate.py:284-415`, `tooling/verify_governance.py:32,224-262,358`, `memory/global/lessons/tripwires.json:1-86`.
  - Thiết kế giải pháp `P1`–`P5` kèm Code Demo 1:1 vào `09_tasks_solution_agent_core_v2.md` và `audit_report_agent_core_v2.md`.
- **Definition of Done:** Qua `DG0` và `DG1/DG2` (`PASS`), trình bày trực tiếp bảng nguyên nhân gốc + giải pháp + Code Demo 1:1 trên chat và dừng chờ `APPROVE`.

## Task 2 (`T2` — S4/S5/S6 Execute Core Fixes & Regression Suite After User `APPROVE`) [x]
- **Mô tả & Context `file:line`:**
  - **`P1a` (`F-01`):** Cập nhật `.agents/hooks.json:10-48` (và đồng bộ `data-hub/.agents/hooks.json:10-48`, `~/.gemini/config/hooks.json:10-22`) dùng toán tử `|` chuẩn không escape `\|`: `"write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file|multi_edit"` và `"invoke_subagent|send_message"`.
  - **`P1b` & `P3` (`F-02`, `F-03`, `F-04`, `F-06`, `N-02`, `N-03`, `N-04`, `N-05`, `R-02`, `R-03`):** Cập nhật `hooks/agy/agy_guard.py:80-105,206-275,309-338,507-560,643-651,858-1015,1087-1096,1253-1268`:
    1. Đồng bộ Hunk 1 với `agy_guard.py:80-105` (`READ_ARG_KEYS`), bổ sung `WRITE_TOOLS` (6 tools), `FULL_WRITE_TOOLS`, `MAX_SOLUTION_LINES = 800`, `"read_file"` vào `READ_ARG_KEYS` & `TRIPWIRE_EVIDENCE_TOOLS`.
    2. Bổ sung `_parse_chunks(raw_args)`, `_chunk_texts`, `old_arg(raw_args)`, `new_arg(raw_args)`, `_apply_edit_preview(cur_txt, raw_args, old_s, body)`, lọc `nm in WRITE_TOOLS` ngay trong hàm `def touched(...)` (cùng ngoại lệ `run_command` cho `build_lessons_index`) và tại mọi vị trí gọi `tgt(...)`, giữ nguyên `prg_ok = touched(st, r"05_progress", u_idx)` tại `H4a`.
    3. Trong `H2`: đặt `is_partial_edit = tool_name in (WRITE_TOOLS - FULL_WRITE_TOOLS)` và chặn `exists and (tool_name in FULL_WRITE_TOOLS or is_overwrite_flag or (is_partial_edit and not old_s))` trên mọi file memory đã tồn tại, kèm chặn `not old_s or old_s not in body` trên `APPEND_ONLY`.
    4. Trong `load_tripwires` & `code_tripwire_violations`: chỉ tự suy đoán `ext` từ `trig` đặc thù ngôn ngữ (`N-04`), tách biệt `forbid_pat` và `req` (`N-02`), bổ sung `full_code_body: Optional[str] = None` để kiểm tra `req` trên toàn khối `full_code_body` (`R-03`), tính `merged_sol = _apply_edit_preview(...)` trước khi chạy `_extract_demo_snippets(merged_sol)` (`N-03`/`N-05`), và bỏ hoàn toàn fallback `or body`.
  - **`P2` (`F-05`, `F-07`, `N-01`, `R-01`, `R-03`):** Cập nhật `tooling/laya_gate.py:108-142,188-206,284-415,634-644`:
    1. Kiểm tra `all(isinstance(c, dict) for c in claims)`, chuẩn hóa `kind` từ `c.get("kind") or c.get("type")` và `"control": true/false`, giữ nguyên yêu cầu bắt buộc `>=2 control_false` & `>=1 control_true`, chuẩn hóa `evidence` dạng `List[str]`, `List[dict]`, hoặc `dict`, bổ sung `--workers` (mặc định `3`) và chạy song song qua `ThreadPoolExecutor`.
    2. Trong `cmd_resolve`: giữ nguyên kiểm tra cứng `INLINE_REF_RE` + `resolve_ref_path` cho mọi `REF:` (`R-01`), khử trùng lặp `req_ids`, cấm `--id ALL`, kiểm tra nguyên tử toàn bộ `req_ids` trước khi mutate (`BLOCK` trả exit `2`, `PASS`/missing trả exit `3`), và khi `--stage dg1` đồng bộ ưu tiên file `dg2` khớp cùng timestamp `latest_path.name.replace("_dg1_", "_dg2_", 1)`.
  - **`P4` (`F-08`):** Cập nhật `tooling/verify_governance.py:32,358` dùng `AGENT_TAG_RE = r"\[Agent:[^\s\]][^\r\n\]]*\]"` cho cả `LOG_LINE_RE` và `today_re`.
  - **`P5` (Test Suite):** Tạo `tests/test_agent_core_f01_f08.py` kiểm chứng thực nghiệm Zero-Mock toàn bộ kịch bản `F-01`..`F-08`, `N-01`..`N-05`, và `R-01`..`R-03` (bao gồm 2 assertion mới: `resolve --reason "Đã xem tại dòng 10"` bị chặn với `rc == 3`, và `read_file(Path=".../05_progress_*.md")` trả `touched(st, r"05_progress") is False`).
- **Logs/Test & Definition of Done:** Chạy `tests/test_agent_core_f01_f08.py` và bộ test `tooling/tests/test_*.py` qua `evidence_run.py`, qua `DG3` (`PASS`) và `verify_governance.py` (`PASSED`).
