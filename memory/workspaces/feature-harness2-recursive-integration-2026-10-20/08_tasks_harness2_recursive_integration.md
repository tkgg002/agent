# 08 — Tasks & Execution Checklist: Tích hợp Đệ quy 2 Tầng `Harness²` vào Hệ sinh thái `/agent` + `Modular Orchestrator` (`harness2_recursive_integration`)

> **Workspace:** `feature-harness2-recursive-integration-2026-10-20` | **Tier:** `T2` | **Suffix:** `harness2_recursive_integration`
> **Nguyên tắc tổ chức (Lesson `L002`):** Toàn bộ Task thực thi được ánh xạ **1:1 trực diện theo từng Khoảng trống Kiến trúc `F1 → F6`**, chỉ thực thi sau khi có lệnh `APPROVE` của User.

---

## [x] Task 1 (Xử lý Khoảng trống `F1`): Xây dựng Động cơ Đệ quy 2 Tầng (`Task-Level Propose–Probe–Compose` & `Harnessing-Level Domain Library`)
- **[Mô tả]:**
  - Hiện thực hóa module `tooling/orchestrator/harness2_engine.py` tích hợp vòng lặp **Đệ quy Cấp Task (`Propose → Probe → Compose` với 2 chế độ `agg` song song và `seq` tuần tự)** khi Worker gặp task phức tạp hoặc bị trả về từ trạm QA, cùng **Đệ quy Cấp Hệ thống (`Experience Reflector` + `Per-Domain Harness Library`)** lưu trữ cây `/agent` đã thắng (`harnesses/<task_id>/harness/`) và `EXPERIENCE.md` (`cue`, `lesson`, `evidence`, `confidence`, `edits`) vào `memory/global/harness_library/<domain>/INDEX.md`.
- **[Context & Logs]:**
  - Cơ chế tự học hiện tại của `/agent` (`GEMINI.md:47-55`, `tooling/build_lessons_index.py:14-63`).
  - Đối chiếu chuẩn `Harness²` tại `harness2/core/recursion.py:343-567`, `harness2/core/selection.py:402-505`, và `harness2/core/library.py:58-181`.
- **[Definition of Done]:**
  - Pass unit & integration test cho cả 2 luồng `run_parallel` (`agg`) và `run_sequential` (`seq`), tự động sinh `EXPERIENCE.md` + cập nhật `<domain>/INDEX.md` và đồng bộ bài học sang `memory/global/lessons/INDEX.md` qua `build_lessons_index.py`.

## [x] Task 2 (Xử lý Khoảng trống `F2`): Phân tách 2 Trục Improver Trực giao (`INV` vs `DOM`) & Cưỡng chế Thứ bậc Tiết kiệm Context (`Skill → Script → Subagent → Hook → Always-On`)
- **[Mô tả]:**
  - Thiết lập 2 kênh Proposer độc lập trong `tooling/orchestrator/proposer_axes.py`:
    1. **`INV Proposer` (Task-Invariant Workflow/Tool Lever):** Chỉ được phép sửa `tooling/*.py`, `scripts/*.py` hoặc đề xuất rule hook trong `hooks/agy/agy_guard.py`; cấm chạm vào tài liệu nghiệp vụ dự án.
    2. **`DOM Proposer` (Domain Practitioner Lever):** Chỉ được phép tạo/cập nhật `skills/<domain_skill>/SKILL.md` và `memory/global/project/<project_name>/`; cấm sửa script/hook lõi.
  - Cưỡng chế quy tắc `KEEP_BASE` (nếu bằng chứng trajectory không ủng hộ thay đổi thì giữ nguyên bản gốc) và thứ bậc ưu tiên `SKILL → SCRIPT → SUBAGENT → HOOK → ALWAYS-ON`.
- **[Context & Logs]:**
  - Quy tắc quản trị skill hiện tại (`GEMINI.md:56-57`, `OPERATOR_MAP.md:5-18`).
  - Đối chiếu chuẩn `Harness²` tại `harness2/core/prompts.py:129-146, 333-407`.
- **[Definition of Done]:**
  - Kiểm chứng tự động từ chối mọi bản đề xuất `INV` sửa nhầm file `DOM` (và ngược lại), giữ dung lượng `GEMINI.md` luôn dưới trần `always_on_max`.

## [x] Task 3 (Xử lý Khoảng trống `F3`): Xây dựng Bộ Kiểm định Bề mặt Cấu hình `/agent` (`AgentSurfaceValidator & Auto-Trigger Compiler`)
- **[Mô tả]:**
  - Xây dựng module kiểm định bề mặt cấu hình `/agent` (`SurfaceSpec` cho `/agent`: `GEMINI.md`, `OPERATOR_MAP.md`, `skills/*/SKILL.md`, `tooling/*.py`, `hooks/agy/agy_guard.py`):
    1. Kiểm tra tổng số ký tự `always_on` (`GEMINI.md`) không vượt ngưỡng `always_on_max`.
    2. Kiểm tra mọi `skills/<name>/SKILL.md` có đầy đủ frontmatter `name: <name>` khớp tên thư mục và `description:` hợp lệ.
    3. Kiểm tra mọi `skills/<name>/SKILL.md` đều có dòng Trigger nhắc tên trong `GEMINI.md` hoặc `OPERATOR_MAP.md` (hỗ trợ `autotrigger()`).
    4. Kiểm tra mọi lệnh `python3 .../tooling/<script>.py` được gọi trong `GEMINI.md` và `skills/*/SKILL.md` đều tồn tại thật trên đĩa và vượt qua `compile(src, rel, "exec")`.
- **[Context & Logs]:**
  - Bài học `L009` (`memory/global/lessons/topics/01_governance_and_workflow.md:45-50`) và `tooling/verify_governance.py:25-69`.
  - Đối chiếu chuẩn `Harness²` tại `harness2/core/tree.py:40-62, 192-390`.
- **[Definition of Done]:**
  - Chạy kiểm định trên chính cây `/Users/trainguyen/Documents/work/agent` đạt `100% VALIDATED`, phát hiện chính xác skill thiếu trigger hoặc script lỗi cú pháp trong bộ test đột biến (`mutation_gate.py`).

## [x] Task 4 (Xử lý Khoảng trống `F4`): Chuẩn hóa Cây Bằng chứng Vật lý (`ContextTree`) & Chốt chặn Tiền kiểm Phòng sạch (`Pre-Flight Leak & Grader Verdict Tripwire`)
- **[Mô tả]:**
  - Xây dựng bộ đóng gói bằng chứng trên đĩa (`context/task.md`, `context/rollouts/<label>.md`, `context/deliverables/<label>/`, `context/harness/`, `context/pool/<label>/harness.diff`, `context/library/INDEX.md`) kèm hàm sinh prompt con trỏ ngắn `mission()`.
  - Nâng cấp cơ chế `Clean-Room Isolation` (`hooks/agy/agy_guard.py:259-301`): bổ sung hàm tiền kiểm `assert_no_leak_paths(root, clean_room_forbidden_sources)` và `assert_no_grader_verdicts(root)` quét sạch thư mục làm việc **trước** khi khởi chạy Sub-agent/Worker, kết hợp với hậu kiểm `contaminated_clean_room_reads` hiện có.
- **[Context & Logs]:**
  - Hậu kiểm phòng sạch hiện tại tại `hooks/agy/agy_guard.py:259-301` & `hooks/agy/agy_guard.py:528-535`.
  - Đối chiếu chuẩn `Harness²` tại `harness2/core/context.py:42-188, 399-546`.
- **[Definition of Done]:**
  - Chặn đứng 100% file thuộc `clean_room_forbidden_sources` và các chuỗi `GRADER_VERDICT_MARKERS` ngay tại bước staging `context/` trước khi tiến trình CLI Worker kịp đọc.

## [x] Task 5 (Xử lý Khoảng trống `F5`): Tuyển chọn Ẩn danh Chống Thiên kiến (`Blind Anonymized Selection`) & Đo Độ Đa dạng (`Pairwise Line-Jaccard Diversity`)
- **[Mô tả]:**
  - Khi đánh giá nhiều nhánh giải pháp/code (`agg1..aggk` hoặc `base` vs `candidate`), áp dụng hàm ẩn danh hóa có seed `assign_ids()` (`A, B, C...`) kết hợp tính toán độ đa dạng `diversity()` (`mean_line_jaccard`) và cổng thẩm định toán học `/laya_router` (`DG3` tại `tooling/laya_gate.py:44-46`).
- **[Context & Logs]:**
  - Cổng quyết định `/laya_router` hiện tại (`tooling/laya_gate.py:44-46`).
  - Đối chiếu chuẩn `Harness²` tại `harness2/core/selection.py:71-86, 268-357` và `harness2/core/tree.py:407-428`.
- **[Definition of Done]:**
  - Đảm bảo tính tất định của bảng ánh xạ ẩn danh `id_map` theo `SHA-1(task_id|k|mode)` và tính chính xác chỉ số `mean_line_jaccard`.

## [x] Task 6 (Xử lý Khoảng trống `F6`): Hằng đẳng thức Ngân sách Tất định (`Ledger.assert_identity: R=4k, P=3k`) & Content-Addressed SHA-256 Cache + `flock` Resume
- **[Mô tả]:**
  - Tích hợp bộ kiểm toán ngân sách `Ledger.assert_identity()` (`R = 4k` rollouts, `P = 3k` sessions, cấm trùng lặp nhãn `label`) vào sổ cái `autopilot_rounds` của Orchestrator.
  - Bổ sung cơ chế định danh nội dung `_call_fingerprint` bằng `SHA-256` trên toàn bộ `(label, mission, system, context_dir)` và ghi file tổng kết nguyên tử bằng `fcntl.flock(LOCK_EX)` + `os.replace` để hỗ trợ resume đa tiến trình an toàn tuyệt đối.
- **[Context & Logs]:**
  - Nhật ký tiến độ hiện tại (`tooling/verify_governance.py:213-241`).
  - Đối chiếu chuẩn `Harness²` tại `harness2/core/ledger.py:26-114`, `harness2/core/proposer.py:507-534`, và `harness2/core/schedule.py:289-329`.
- **[Definition of Done]:**
  - Ném lỗi `BudgetError` ngay lập tức nếu có nhãn trùng lặp hoặc lệch số lượt chạy `R != 4k`, `P != 3k`; resume từ cache SHA-256 tốn `0.0 USD` và an toàn khi chạy đa tiến trình với `flock`.

---

## [x] Task 7 (`S5 → S3 Re-Plan` — Khắc phục 7 Lỗi Phản biện `AF-01 → AF-07` sau khi có User `APPROVE` mới)
- **[Mô tả]:**
  - **`AF-01` (3-Way Tree Merge):** Thay thế `merged_seed.update(inv{i})` + `merged_seed.update(dom{i})` tuần tự tại `tooling/orchestrator/harness2_engine.py:642-644, 675-677` bằng hàm `three_way_merge_trees(base_tree, inv_tree, dom_tree)` bảo toàn cả các file có sẵn được sửa bởi `INV` lẫn các file bị xóa chủ đích.
  - **`AF-02` (Full SHA-256 Cache Fingerprint):** Sửa `_run_cached_session` (`harness2_engine.py:559-569`) đưa `task_id`, mã băm `SHA-256` của toàn bộ `base_tree` (`json.dumps(base_tree, sort_keys=True)`) và `context_dir` vào `call_fingerprint_sha256`, chống ngộ độc cache liên task và liên vòng.
  - **`AF-03` (`"base"` Rollout Aggregation & Grader-Free `selector_fn`):** Ghi nhận `rollout_outputs["base"]` (tổng hợp từ `base1..basek` ở chế độ `agg` và `b1` ở chế độ `seq`) và bổ sung tham số `selector_fn(id_map, candidate_trees, rollout_outputs)` hoạt động trên mã ẩn danh `A, B, C...` mà không cần trường `score` có sẵn (`harness2_engine.py:616, 660, 700-706`).
  - **`AF-04` (UTF-8 Byte Cap, YAML Frontmatter Delimiter & Word-Boundary Trigger):** Sửa `validate_agent_surface` & `autotrigger_agent_skills` (`harness2_engine.py:116, 130, 136, 168, 204`) kiểm tra `len(body.encode("utf-8")) > surface.always_on_max`, tách khối YAML frontmatter giữa `--- ... ---`, và dùng regex ranh giới từ `(?<![\w\-]){re.escape(sname)}(?![\w\-])`.
  - **`AF-05` (Real Workspace `s3_docs` & `build_lessons_index` Sync):** Xóa mảng giả `s3_docs = [f"{pfx}_ok.md" ...]` tại `harness2_engine.py:711`, đọc danh sách file thật từ `workspace_dir`, và gọi hàm đồng bộ `build_lessons_index.py` khi ghi nhận bài học mới.
  - **`AF-06` (`proposer_axes.py` Module & Strict Axis Whitelist):** Tạo `tooling/orchestrator/proposer_axes.py` đúng cam kết tại `08_tasks:19` và chặn `INV` sửa bất kỳ file nào trong `skills/` (`harness2_engine.py:210`).
  - **`AF-07` (Subprocess CLI Runner & Live Runtime Trace):** Bổ sung `SubprocessProposerRunner` / `SubprocessRolloutRunner` kèm `timeout` và lệnh CLI `run-recursion` để trace runtime value thật trên đĩa cùng User.
- **[Context & Logs]:**
  - Biên bản phản biện `audit_report_harness2_recursive_integration.md` và kết quả `/laya_router` `DG3` (`06_laya_summary_dg3_20261020T160852.json`).
- **[Definition of Done]:**
  - Thực thi sau khi User `APPROVE`; bổ sung unit/integration test phá hủy cho từng lỗi `AF-01..AF-07`, chạy lại `mutation_gate.py` (`>= 3` mutants `KILLED`), chạy trace CLI thật và vượt qua `DG3` post-edit.

