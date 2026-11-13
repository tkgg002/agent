# Audit Report — Khắc Phục Độ Trễ, Đốt Token & Lọt Lỗi Của Agentic Core (`agent_core_v2`)

## Bảng Tổng Hợp Issues & Tổng Kết Hành Động (Executive Summary Table)

| Mã | Nhóm | Mức độ | Vị trí (`file:line`) | Mô tả ngắn | Trạng thái & Hành động |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`F1` / `F-01`** | Hook Regex Alternation | **CRITICAL** | `.agents/hooks.json:10-48` | Chuỗi `"\\|"` giải mã thành `\|` (khớp dấu pipe nghĩa đen) và thiếu `multi_replace_file_content`, `multi_edit`, `write_file`, `edit_file`. | **FIXED (`P1a`, `EV-20261110T132144-2270`)** |
| **`F-02` / `R-03`** | H2 Full-Write & Partial-Edit Guard | **CRITICAL** | `hooks/agy/agy_guard.py:858-898` | `write_file` không có cờ `Overwrite` và `old_s == ""` xuyên thủng `H2`; đã đặt `is_partial_edit = tool_name in (WRITE_TOOLS - FULL_WRITE_TOOLS)`, chặn `FULL_WRITE_TOOLS or is_overwrite_flag or (is_partial_edit and not old_s)` trên file memory đã tồn tại và chặn `not old_s` trên `APPEND_ONLY`. | **FIXED (`P1b`, `EV-20261110T132144-2270`)** |
| **`F-03` / `R-02`** | Write-Tool Filtering & `touched()` | **CRITICAL** | `hooks/agy/agy_guard.py:80-105,206-258,330-338,648,960,1087-1096,1253` | Đã đồng bộ Hunk 1 với `READ_ARG_KEYS`, lọc `nm in WRITE_TOOLS` ngay trong hàm `def touched(...)` (cùng `nm == "run_command"`) và tại mọi vị trí gọi `tgt(...)`, giữ nguyên `prg_ok = touched(st, r"05_progress", u_idx)` tại `H4a`. | **FIXED (`P1b`, `EV-20261110T132144-2270`)** |
| **`F3` / `F-04` / `F-06` / `N-02`..`N-05` / `R-03`** | Selective Demo Tripwire & Multi-Chunk | **CRITICAL** | `hooks/agy/agy_guard.py:206-275,507-560,921-1036` | Đã bổ sung `_parse_chunks` (`N-05`), `_apply_edit_preview`, bỏ `or body`, tách rời `forbid` và `req` (`N-02`), thêm `full_code_body` để kiểm tra `req` trên toàn khối (`R-03`), chỉ suy đoán `ext` từ `trig` (`N-04`), và tính `merged_sol` trước khi chạy `_extract_demo_snippets(merged_sol)` (`N-03`). | **FIXED (`P1b` + `P3`, `EV-20261110T132144-2270`)** |
| **`F2` / `F-05` / `F-07` / `N-01` / `R-01` / `R-03`** | Atomic Resolve & Parallel Run | **CRITICAL** | `tooling/laya_gate.py:108-142,188-228,284-415,634-650` | Đã giữ nguyên bắt buộc `>=2 control_false` & `>=1 control_true` (`N-01`), xử lý `List[str]` trong `_normalize_evidence_str` (`R-03`), giữ nguyên bắt buộc khớp `INLINE_REF_RE` + `resolve_ref_path` trong `cmd_resolve` (`R-01`), cấm `--id ALL`, kiểm tra nguyên tử trước khi ghi, đồng bộ ưu tiên file `dg2` cùng timestamp, và thêm `--workers` (`3`). | **FIXED (`P2`, `EV-20261110T132144-2270`)** |
| **`F4` / `F-08`** | Governance Agent Tag Regex | **MEDIUM** | `tooling/verify_governance.py:32,358` | Đã thay bằng hằng số dùng chung `AGENT_TAG_RE = r"\[Agent:[^\s\]][^\r\n\]]*\]"` hỗ trợ dấu cách, ngoặc đơn và dấu `/`. | **FIXED (`P4`, `EV-20261110T132144-2270`)** |

---

## 1. Line-by-Line Diff & Task Critique
- Đã chạy `git diff --stat -- hooks tooling .agents` (`4 files changed, 524 insertions(+), 97 deletions(-)`) và đo `wc -l`: `hooks/agy/agy_guard.py` (`1682` dòng), `tooling/laya_gate.py` (`823` dòng), `tooling/verify_governance.py` (`456` dòng), `tests/test_agent_core_f01_f08.py` (`394` dòng), `09_tasks_solution_agent_core_v2.md` (`786` dòng <= `800`).

## 2. Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật
- Toàn bộ 94 bài kiểm thử đơn vị và hồi quy (`test_agent_core_f01_f08`, `test_tripwire_cleanroom`, `test_agy_guard`, `test_claim_linter`, `test_laya_gate`, `test_verify_governance`, `test_anti_fake_evidence`) đạt `94/94` `OK` (`EV-20261110T132142-1b17`).
- Bộ kiểm thử tích hợp `tests/test_agent_core_f01_f08.py` đạt `7/7` `OK` (`EV-20261110T132144-2270`).
- Kiểm chứng `runtime` thực tế trên `laya_gate.py run --stage dg3 --workers 3` đạt `exit=0` trong `16.0s` wall-clock (`EV-20261110T132222-622b`), `python3 orchestrator/harness2_engine.py validate-surface` đạt `VALIDATED` (`EV-20261110T132653-fbea`), và subprocess `agy_guard.py pre-write` chặn `write_file` trên file memory hiện hữu (`decision: deny`, `EV-20261110T132144-2270`).

## 3. Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity
- `cmd_run` trong `laya_gate.py` dùng `ThreadPoolExecutor(max_workers=workers_n)` ở mức `(idx, rnd)` (mặc định `3` workers), giữ nguyên thứ tự mảng `rows` và tránh nghẽn hàng đợi single-GPU (`EV-20261110T132222-622b`).
- `cmd_resolve` kiểm tra nguyên tử toàn bộ `targets` trước khi ghi xuống đĩa và đồng bộ ưu tiên file `dg2` khớp cùng timestamp (`EV-20261110T132144-2270`).

## 4. Kiểm tra suy diễn & báo cáo láo (Honesty Check)
- Không dùng mockup giả lập kết quả runtime; mọi biên lai `EV-20261110T132142-1b17`, `EV-20261110T132144-2270`, `EV-20261110T132222-622b` đều khớp `src_fingerprint: 1daaf4237d75a34c20b213738c9b7f470ea765d7cf9421a6acd6538b989a3c62`.

## 5. Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop)
- Trong quá trình chạy regression suite `S5`, phát hiện `SPEC_REJECT_RE` tại `agy_guard.py:80` bị dính chuỗi `\vert{}` và `touched()` cần mở cho `nm == "run_command"`; đã thực hiện đúng vòng lặp `S5 -> S3 -> S4 -> S5` cập nhật `09_tasks_solution_agent_core_v2.md` + `agy_guard.py` và chạy lại toàn bộ `94/94` test đạt `OK` (`EV-20261110T132142-1b17`).

## 6. Kết quả QC & nhận xét từ `/laya_router` (DG3)
- **DG0**: `06_laya_summary_dg0_20261109T195328.json`.
- **DG1 + DG2**: `06_laya_summary_dg1_20261110T105631.json` & `06_laya_summary_dg2_20261110T105631.json` (`S-RC` & `D-BEHAV` resolved).
- **DG3 (sau lần sửa code cuối)**: `06_laya_summary_dg3_20261110T132238.json` (`EV-20261110T132222-622b`).
