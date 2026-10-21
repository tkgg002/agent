# 06 — Validation Report: Hợp Nhất Đệ Quy 2 Tầng `Harness²` (`harness2_recursive_integration`)

> **Workspace:** `feature-harness2-recursive-integration-2026-10-20` | **Tier:** `T2` | **Suffix:** `harness2_recursive_integration`

---

## 1. Bảng Truy vết Yêu cầu (Gate G1 — Requirement Traceability)

| Req ID | Mô tả Yêu cầu (`01_requirements_harness2_recursive_integration.md`) | File:Line Đáp ứng | Bằng chứng Test / Runtime | Trạng thái |
| :---: | :--- | :--- | :--- | :---: |
| **`R0`** | Định vị Vật lý & Bức tranh Tổng quan Vận hành (`Macro Runtime & Integration Topology` — `L001`). | `13_analysis_harness2_recursive_integration.md:9-48`, `09_tasks_solution_harness2_recursive_integration.md:8-24` | Đã kiểm định qua `DG1` (`06_laya_summary_dg1_20261020T145108.json`) & `DG3` (`06_laya_summary_dg3_20261020T145126.json`). | **PASS** 🟢 |
| **`R1` (`F1`)** | Động cơ Đệ quy 2 Tầng (`Task-Level Propose–Probe–Compose` `agg`/`seq` + `DomainHarnessLibrary`). | `tooling/orchestrator/harness2_engine.py:441-536, 539-751` | `test_06_f1_two_level_recursion_agg_and_seq_with_domain_library_and_cache` (`PASS`). | **PASS** 🟢 |
| **`R2` (`F2`)** | Phân tách 2 Trục Improver Trực giao (`INV` vs `DOM`) & Cưỡng chế Thứ bậc Context + `KEEP_BASE`. | `tooling/orchestrator/harness2_engine.py:65-71, 188-226` | `test_02_f2_orthogonal_axis_isolation_and_keep_base` (`PASS`) + Mutant `M2` (`KILLED`). | **PASS** 🟢 |
| **`R3` (`F3`)** | Bộ Kiểm định Bề mặt Cấu hình `/agent` (`AgentSurfaceValidator & Auto-Trigger Compiler`). | `tooling/orchestrator/harness2_engine.py:78-181` | CLI `validate-surface` (`26 files VALIDATED`) + `test_01_f3_validate_real_agent_surface_and_detect_surface_violations` (`PASS`) + Mutant `M1` (`KILLED`). | **PASS** 🟢 |
| **`R4` (`F4`)** | Cây Bằng chứng Vật lý (`ContextTree`) & Chốt chặn Tiền kiểm Phòng sạch (`Pre-Flight Leak & Grader Verdict Tripwire`). | `tooling/orchestrator/harness2_engine.py:50-63, 233-306` | `test_03_f4_context_tree_and_pre_flight_clean_room_tripwire` (`PASS`). | **PASS** 🟢 |
| **`R5` (`F5`)** | Tuyển chọn Ẩn danh (`Blind Anonymized Selection`) & Đo Độ Đa dạng (`Pairwise Line-Jaccard Diversity`). | `tooling/orchestrator/harness2_engine.py:313-345` | `test_04_f5_anonymous_ids_and_pairwise_line_jaccard_diversity` (`PASS`). | **PASS** 🟢 |
| **`R6` (`F6`)** | Hằng đẳng thức Ngân sách Tất định (`R=4k, P=3k`), SHA-256 Cache & `fcntl.flock` Resume. | `tooling/orchestrator/harness2_engine.py:372-454, 681-722` | `test_05_f6_ledger_identity_and_concurrent_flock_summary` (`PASS`, 8 concurrent workers) + Mutant `M3` (`KILLED`). | **PASS** 🟢 |
| **`R7` (`AF-01..AF-07`)** | Khắc phục Vòng kín `S5 → S3 Re-Plan`: `three_way_merge_trees`, khóa cache SHA-256 đầy đủ (`task_id` + `base_tree` + `context_dir`), tổng hợp `"base"` + `selector_fn`, đếm byte UTF-8 + YAML frontmatter + ranh giới từ tên skill, quét `workspace_dir` thật + gọi `build_index()`, module `proposer_axes.py`, và `SubprocessHarnessRunner` + CLI `run-recursion`. | `tooling/orchestrator/proposer_axes.py:36-98`, `tooling/orchestrator/harness2_engine.py:66-226, 559-943` | `test_07_af01_to_af07_adversarial_remediation_and_real_subprocess_trace` (`PASS`) + CLI `run-recursion` (`PASS`) + Mutant `M4` (`KILLED`). | **PASS** 🟢 |

---

## 2. Kết quả Test, 4-Mutant Gate & Runtime CLI Thật (Gates G2, G3, G4, G5, G6)

### 2.1. Kiểm định Runtime CLI trên Cây `/Users/trainguyen/Documents/work/agent` Thật (`validate-surface` & `run-recursion`)
- **Lệnh 1 (`validate-surface`):**
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py validate-surface
  ```
- **Output thực tế:**
  ```json
  {
    "status": "VALIDATED",
    "agent_root": "/Users/trainguyen/Documents/work/agent",
    "files_scanned": 28,
    "always_on_chars": 18091,
    "always_on_utf8_bytes": 20485,
    "always_on_max": 24000,
    "errors": []
  }
  ```
- **Lệnh 2 (`run-recursion` — khởi chạy tiến trình con `subprocess` thật trên đĩa & đồng bộ `build_index()`):**
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py run-recursion --task-id live-trace-af07 --domain agent_governance --mode agg --k 2 --sync-lessons
  ```
- **Output thực tế:**
  - `rollouts_count: 8`, `sessions_count: 6`, `chosen_label: "agg1"`, `s3_docs_checked: 30`.
  - `winning_existing_check_py: "VALUE = 2\n#inv1"` (bảo toàn 100% thay đổi của `INV` trên file có sẵn đồng thời giữ skill `skills/dom-dom1/SKILL.md` của `DOM`).
  - `lessons_index_stats: {"source_lines": 1993, "source_entries": 256, "index_entries": 256, "topic_entries": 256, "index_bytes": 14281}`.

### 2.2. Kết quả Bộ Unit & Integration Test + Regression Suite (Sau `Task 8`: `RF-01 → RF-05`)
- **Lệnh đã chạy:**
  ```bash
  python3 -m unittest discover -s tooling/tests -p "test_*.py" -v && python3 -m unittest discover -s hooks/agy/tests -p "test_*.py" -v
  ```
- **Output thực tế:**
  - `tooling/tests`: **Ran 38 tests in 3.714s — OK** (bao gồm `test_08_rf01_to_rf05_live_model_blind_selector_and_s0_s6_wiring` trong `test_harness2_engine.py`).
  - `hooks/agy/tests`: **Ran 51 tests in 0.283s — OK** (0 regression).
- **Lệnh chạy thật `run-recursion --use-live-model` trên bề mặt `/agent` thật với Model thật (`:8200/v1/predict`):**
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py run-recursion \
    --use-live-model --task-id harness2_s0_s6_live_model_run --domain agent_governance --mode agg --k 1 --sync-lessons
  ```
  - **Output thực tế:** `rollouts_count: 4`, `sessions_count: 3`, `chosen_anonymous_id: "A"`, `chosen_label: "agg1"`, `library_slot: "/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run"`, `lessons_index_stats: {"source_entries": 262, "index_entries": 262, "index_bytes": 14571}`.
  - **File vật lý trên đĩa:** `memory/global/harness_library/agent_governance/INDEX.md`, `harnesses/harness2_s0_s6_live_model_run/EXPERIENCE.md`, `harnesses/harness2_s0_s6_live_model_run/harness.json`, và 22 file trong `harnesses/harness2_s0_s6_live_model_run/harness/`.

### 2.3. Kết quả Cổng Kiểm thử Đột biến 5 Bản Hỏng (`5-Mutant Gate` — `mutation_gate.py`)
- **Lệnh đã chạy:**
  ```bash
  python3 /Users/trainguyen/Documents/work/agent/tooling/mutation_gate.py \
    --workspace feature-harness2-recursive-integration-2026-10-20 \
    --suffix harness2_recursive_integration \
    --mutations /Users/trainguyen/.gemini/antigravity/brain/223ce70e-eef2-4adc-9ea6-9daaf96c357d/scratch/mutants_task8.json
  ```
- **Output thực tế (`06_mutation_summary_harness2_recursive_integration_20261021T101736.json`):**
  - `gate`: **`PASS`**
  - `killed_count`: **`5 / 5`** (`M1_bypass_always_on_budget`: `KILLED`, `M2_bypass_orthogonal_inv_axis_fence`: `KILLED`, `M3_corrupt_ledger_budget_identity`: `KILLED`, `M4_corrupt_three_way_merge_overwrite`: `KILLED`, `M5_noop_proposer_without_template`: `KILLED`)
  - `survived_count`: `0`, `timeout_count`: `0`, `all_restored_sha256`: `true`.

---

## 3. Kết quả Decision Gate (`DG1`, `DG2`, `DG3`), Security Gate & Final Audit (Gate G7 & G8)

- **DG1 Summary (`Task 8 Re-Plan`):** `06_laya_summary_dg1_20261021T095032.json` | Gate: **`PASS`** 🟢
- **DG2 Summary (`Task 8 Re-Plan`):** `06_laya_summary_dg2_20261021T095127.json` | Gate: **`PASS`** 🟢
- **DG3 Summary (Post-Task-8 Final Implementation):** `06_laya_summary_dg3_20261021T101946.json` | Gate: **`PASS`** 🟢
- **Harness² Surface & Clean-Room Check:** `python3 /Users/trainguyen/Documents/work/agent/tooling/orchestrator/harness2_engine.py validate-surface` (`VALIDATED`, `always_on_utf8_bytes: 21578 <= 24000`)
- **Final Adversarial Audit Report:** [`audit_report_harness2_recursive_integration.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-harness2-recursive-integration-2026-10-20/audit_report_harness2_recursive_integration.md)

### Security Gate Report (`/security-agent`)
| Hạng mục | Kết quả | Chi tiết |
| :--- | :---: | :--- |
| Secret & PII Scan (Code + Memory) | **PASS** | Quét regex trên toàn bộ `tooling/orchestrator/`, `test_harness2_engine.py`, `memory/global/harness_library/`, và workspace: `CLEAN_ZERO_SECRETS`. |
| Input & Query Injection Check | **PASS** | Không sử dụng `shell=True`; dùng danh sách token argv tường minh với `_expand_token()` và bọc dữ liệu bằng `EVIDENCE_BOUNDARY_HEADER`. |
| AuthN / AuthZ & Error Exposure | **PASS** | Chốt chặn `assert_pre_flight_clean_room` ngăn rò rỉ file cấm, `GRADER_VERDICT_MARKERS` và `SECRET`. |
| VCS Hygiene (No secret staged, No push) | **PASS** | Chỉ tạo local git restore-point commit, tuyệt đối không chạy `git push` (Rule #15). |
| Dynamic Pentest Gate (DAST / PentAGI) | **N/A** | Module thư viện/CLI nội bộ không mở cổng HTTP/gRPC server. |
- **Verdict:** ✅ **PASS**

