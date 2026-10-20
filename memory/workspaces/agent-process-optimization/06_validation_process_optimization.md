# 06 — Validation & Verification Evidence (DoD G1–G8)

## 1. Đối chiếu Yêu cầu (G1 — Requirement Traceability)
| ID | Hạng mục yêu cầu | Trạng thái | Bằng chứng vật lý |
| :--- | :--- | :---: | :--- |
| **U1** | Rút gọn `GEMINI.md` `<= 12,000 bytes`, giữ đủ 23 rule headers (`0`–`22`), tích hợp Master Workflow `S0–S6` & Decision Gate `/laya_router` | PASS | `GEMINI.md` (`11,928 bytes`, `23` headers `### 0.`–`### 22.`); bản gốc lưu tại `rules/GEMINI_FULL_ARCHIVE_2026-10.md` (`30,949 bytes`). |
| **U2** | Chuẩn hóa 10 Core Skills tại `agent/skills/<name>/SKILL.md`, lưu trữ 96 workflow phẳng cũ vào `workflows/_archive/`, đăng ký `.agents/skills.json` | PASS | 10 file `SKILL.md` (`0` lệnh `git push`, `0` lệnh hardcode dự án); `workflows/_archive/` (96 files). |
| **U3** | Công cụ CLI `tooling/laya_gate.py` (Mode A Calibrated Claim Gate + Mode B Agentic Debate Review) | PASS | `tooling/laya_gate.py` + `tooling/tests/test_laya_gate.py` (`5/5 PASS`). |
| **U4** | Dispatcher `hooks/agy/agy_guard.py` (`H1–H7`), `.agents/hooks.json`, manifest `.agent-project.json` (100% Agnostic Core) | PASS | `hooks/agy/agy_guard.py` + `hooks/agy/tests/test_agy_guard.py` (`16/16 PASS`). |
| **U5** | Chỉ mục 2 tầng `memory/global/lessons/INDEX.md` (`< 15 KB`) + `topics/*.md` từ `lessons.md` (Read-Only) | PASS | `tooling/build_lessons_index.py` (`241/241` entries, `INDEX.md` = `14,576 bytes < 15,000 bytes`, `lessons.md` nguyên vẹn `1,888` dòng). |
| **U6** | Nâng cấp `tooling/verify_governance.py` hỗ trợ Tier `T0–T3`, bắt buộc `--workspace`, đường dẫn động | PASS | `tooling/verify_governance.py` + `tooling/tests/test_verify_governance.py` (`5/5 PASS`). |

## 2. Kết quả Chạy Test Tự động (G3, G4, G5 — Unit & Fixture Suites)

### 2.1. Bộ Test Tooling (`laya_gate`, `build_lessons_index`, `verify_governance`)
- **Lệnh thực thi:**
  ```bash
  python3 -m unittest discover -s /Users/trainguyen/Documents/work/agent/tooling/tests -p 'test_*.py' -v
  ```
- **Kết quả:** `Ran 11 tests in 0.021s — OK`
  - `test_build_index_conserves_entries_and_respects_size` ... `ok`
  - `test_backtest_reproduces_2026_10_16_batch` ... `ok`
  - `test_mode_a_retries_503_and_calibrates_rounds_and_resolves_flag` ... `ok`
  - `test_mode_b_debate_review_catches_rubber_stamp_and_clears_valid_fencing` ... `ok`
  - `test_run_requires_minimum_controls` ... `ok`
  - `test_verify_evidence_checks_file_and_lines` ... `ok`
  - `test_malformed_progress_log_fails` ... `ok`
  - `test_missing_workspace_arg_fails` ... `ok`
  - `test_tier_t0_passes_without_workspace` ... `ok`
  - `test_tier_t1_passes_with_minimal_3_files` ... `ok`
  - `test_tier_t2_fails_when_missing_09_06_11_then_passes` ... `ok`

### 2.2. Bộ Test Lifecycle Hooks (`agy_guard.py` — 16 Kịch bản `H1–H7`)
- **Lệnh thực thi:**
  ```bash
  python3 -m unittest discover -s /Users/trainguyen/Documents/work/agent/hooks/agy/tests -p 'test_*.py' -v
  ```
- **Kết quả:** `Ran 16 tests in 0.013s — OK`
  - `test_01_h2_overwrite_existing_memory_denied` ... `ok`
  - `test_02_h2_append_only_strip_target_denied` ... `ok`
  - `test_03_h2_append_only_preserving_target_allowed` ... `ok`
  - `test_04_h3_unmasked_secret_in_memory_denied` ... `ok`
  - `test_05_h3_masked_secret_in_memory_allowed` ... `ok`
  - `test_06_h4_source_edit_without_workspace_denied` ... `ok`
  - `test_07_h4_source_edit_dg1_block_denied` ... `ok`
  - `test_08_h4_source_edit_without_approve_force_ask` ... `ok`
  - `test_09_h4_source_edit_with_approve_allowed` ... `ok`
  - `test_10_h5_git_push_denied` ... `ok`
  - `test_11_h5_direct_db_delete_force_ask` ... `ok`
  - `test_12_h7_stop_missing_runtime_check_continues` ... `ok`
  - `test_13_h7_stop_third_time_loop_breaker` ... `ok`
  - `test_14_h7_stop_tier_t0_passes` ... `ok`
  - `test_15_repo_without_manifest_graceful` ... `ok`
  - `test_16_nested_components_longest_prefix_match` ... `ok`

## 3. Kiểm định Decision Gate `/laya_router` (`DG1`, `DG2`, `DG3`)
- **DG1 (`06_laya_summary_dg1_20261016T100944.json`):** `gate = PASS` (6 claims kiến trúc được đối chiếu trực tiếp với tài liệu Antigravity & source code).
- **DG2 (`06_laya_summary_dg2_20261016T134438.json`):** `gate = PASS` (chạy trực tiếp với Laya server `127.0.0.1:8200` trên code thật của `agy_guard.py`).
- **DG3 (`06_laya_summary_dg3_20261016T134800.json`):** `gate = PASS` (kiểm định `verify_governance.py` và `build_lessons_index.py`).

## 4. Kiểm định Bảo toàn Dữ liệu (`G6 — Data Integrity`)
- `memory/global/lessons.md`: `1,888` dòng, `241` bài học (`### [YYYY-MM-DD]`), không mất bất kỳ dòng nào.
- `memory/global/lessons/INDEX.md`: `14,576 bytes` (`< 15,000 bytes`), `241/241` bài học được đánh chỉ mục đầy đủ sang 5 file `topics/01..05_*.md` (T1: `72`, T2: `23`, T3: `56`, T4: `80`, T5: `10`).

## 5. Kiểm định Đợt Rà soát Bổ sung (Mục 1 → 5)
- `tooling/build_lessons_index.py`: Phân loại trọng số `title + tags (x4) + body (x1)` chính xác 100% (`241/241` bài học).
- `scripts/_archive_legacy/` & `rules/_archive_templates/`: Lưu trữ gọn toàn bộ script/rule mẫu cũ.
- `~/.gemini/GEMINI.md` (`471B`), `~/.gemini/config/{hooks,skills}.json`, `~/.agent/skills/laya_router/SKILL.md`: Đã đồng bộ hoàn chỉnh.
- `verify_governance.py --workspace agent-process-optimization`: `PASSED 🟢 (Tier: T2)`.

## 6. Kiểm định Phase 6 — Khắc phục 5 Lỗ hổng Bỏ qua Workflow & Hook (`workflow_enforcement_fix`)
1. **Giải phóng Context Budget `<skills>` (`~/.gemini/config/skills/`):**
   - Lưu trữ `438` skill ngoại lai sang `~/.gemini/config/skills_archive/`.
   - Thư mục `~/.gemini/config/skills/` hiện chỉ chứa đúng **10 Core Skills** (`audit-propose-only`, `brain-delegate`, `context-manager`, `debug-agent`, `laya-router`, `muscle-execute`, `qa-agent`, `security-agent`, `session-wrapup`, `workspace-docs`).
   - Đồng bộ `/Users/trainguyen/Documents/work/data-hub/.agents/skills.json` và `/Users/trainguyen/Documents/work/data-hub/.agents/hooks.json`.
   - Cập nhật `GEMINI.md` Rule #9 (`12,684 bytes < 24,000 bytes`) inline đầy đủ checklist `S0 → S6`.
2. **Unit & Fixture Tests (`33/33 PASS`):**
   - `python3 -m unittest discover -s hooks/agy/tests -p 'test_*.py' -v` $\rightarrow$ **`Ran 20 tests in 0.015s — OK`** (bổ sung `test_17` unquote `TargetFile`, `test_18` H1 per-turn re-injection, `test_19` H4a S1-before-S3 gate, `test_20` H7 No Shadow Files check).
   - `python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v` $\rightarrow$ **`Ran 13 tests in 0.018s — OK`** (bổ sung `test_stage_s3_and_explicit_suffix_enforcement` và `test_reused_workspace_piggyback_fails_without_new_01_08`).
3. **Bằng chứng Runtime Thật (Live Hook Verification trong chính phiên hiện tại):**
   - Ngay sau khi cập nhật `agy_guard.py` và `verify_governance.py`, Hook `H1` đã tự động tiêm `BOOT S0` ở lượt kế tiếp, và Hook `H7 (Stop)` đã đọc chính xác `TargetFile` bọc dấu `"` trong `transcript.jsonl` thật để chặn dừng khi chưa cập nhật `01_requirements`, `05_progress` và `06_validation`!
