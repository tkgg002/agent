---
task_key: TASK-9
suffix: task_9
title: Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5
target_repo: /Users/trainguyen/Documents/work/agent
affected_repos: [/Users/trainguyen/Documents/work/agent]
branch: task/TASK-9
worktree_path: /Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-9
cli_runner: antigravity
model: opus
stage: DONE
tier: T2
mode: standard
laya_health_s0: false
harness_surface_s0: c1..c8 PASS (always-on=22661B <= 24576B)
harness_libraries: [agent_gov, agent_governance]
routed_topics: [01_governance_and_workflow.md]
matched_tripwires: []
kernel_constitution: /Users/trainguyen/Documents/work/agent/GEMINI.md
total_lessons_indexed: 350
---
# 📋 [S1 · UDF-v1] Đặc Tả Yêu Cầu & Phân Rã Sub-Tasks — TASK-9: Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5
> 📌 **Task:** `TASK-9` · **Stage:** `DONE` · **Tier/Mode:** `T2 / standard` · **Repos (1):** `agent` · **Model:** `opus` · **Gate S0:** `PASS 🟢`

## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)
- **Mục tiêu công việc:** Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5
- **Mô tả chi tiết:** Chốt chặn 2 Pha S1..S3 -> DỪNG CHỜ APPROVE -> S4..S6, Chọn Repo Đích & Setup CLI/Model, Cơ chế Chia nhỏ Sub-Tasks theo Wave, Khung Đọc Report/Giải Pháp/Code Demo Trực Tiếp, Đồng bộ 100% Cửa Sổ Phải theo Task & Bảng Hiển thị /agent Kernel + Bài học FTS5
- **Phạm vi Repository (1):** `agent` (Primary: `/Users/trainguyen/Documents/work/agent`)
- **Cô lập Git Worktree (Layer 1):** `/Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-9` (Branch: `task/TASK-9`)

## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`4 Sub-Tasks` · `file:line` & `DoD`)
| Mã Sub-Task | Wave | Repo | Điểm chạm (`file:line`) | Giải pháp & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-9-M01`** | `Wave 0` | `agent` | `agent/memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py:1 (VALUE = 2)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`existing_check.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_gov/harnesses/cli_subproc_test/harness/tooling/existing_check.py` (`VALUE = 2`) trong worktree cô lập của `agent`. (`DoD:` go test -race ./...) |
| **`TASK-9-M02`** | `Wave 0` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:154 (def _unquote(val: Any) -> Any:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`agy_guard.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def _unquote(val: Any) -> Any:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` go test -race ./...) |
| **`TASK-9-M03`** | `Wave 1` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `agent` (`agy_guard.py`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (`def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:`) trong worktree cô lập của `agent`. (`DoD:` go test -race ./...) |
| **`TASK-9-M04`** | `Wave 1` | `agent` | `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `agent` (`build_lessons_index.py`)** — Kết nối dây dẫn nghiệp vụ tại `memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (`class LessonEntry:`) và viết integration test xác minh đầu-cuối trên `agent`. (`DoD:` go test -race ./...) |

## 3. 💻 Code Demo / Cấu Hình Ràng Buộc Khởi Tạo (`S0 Boot`)
```yaml
task_key: TASK-9
affected_repos: [agent]
worktree: /Users/trainguyen/Documents/work/agent/.orchestrator/worktrees/TASK-9
branch: task/TASK-9
runner_model: antigravity / opus
```

## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)
- **Hiến pháp & Laya (`:8200`):** `/Users/trainguyen/Documents/work/agent/GEMINI.md` · Laya Health: `false` · Harness²: `c1..c8 PASS (always-on=22661B <= 24576B)`
- **Topic Files định tuyến (`lessons/topics/`):** `01_governance_and_workflow.md` (Tổng: `350` bài học)
- **Lesson Tripwires (`G-08` & `H4t`):**
- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).
- **Bài học khớp trực tiếp (FTS5 BM25):**
- **`L041` [2026-10-26 · `T1:L285`] Tự động lướt qua cổng `APPROVE` chạy một mạch tới `DONE` kiểu smoke-test (`plan -> approve -> implement -> qa` trong 0.2s), thiếu chọn Repo/CLI/Model & chia Sub-task khi tạo việc, không hiển thị nội dung Report/Giải pháp/Code Demo ở từng bước workflow, trộn lẫn dữ liệu cửa sổ bên phải khi đổi Task, và làm tàng hình bộ quy tắc `/agent` (Smoke-Speed Autopilot Bypassing S3 Human Gate, Missing Repo/CLI/Subtask Breakdown, Invisible Step Reports, Unscoped Right-Pane State & Hidden `/agent` Kernel Binding):** (1) Dừng lại ngay (Rule #5), ghi bài học vào `lessons.md` và chạy `build_lessons_index.py`; (2) Mở vòng Re-Plan `Task 9` (`S1 -> S2 -> S3`) trong workspace: thiết kế lại luồng **2-...
- **`L053` [2026-10-23 · `T1:L369`] Viết…:** Khi thiết kế và trình bày một hệ thống nền tảng điều phối đa tác nhân chạy ngầm theo hàng đợi (`Event/Queue-Driven Multi-Agent Orchestrator` [A]) có tích hợp bộ quy tắc quản trị hi...
- **`L051` [2026-10-23 · `T1:L355`] -> Probe -> Compose` [A]) tích hợp vào hệ thống quản trị `/agent` [B], Agent mắc 5 lỗi logic tinh vi bị che khuất bởi bộ unit test chỉ thêm file mới trong RAM [X]: (1) **Hợp nhất 2 cây cấu hình (`t_inv` và `t_dom`) bằng `merged.update(t_inv); merged.update(t_dom)` (`harness2_engine.py:642-644, 675-677`):** Vì cả `t_inv` lẫn `t_dom` đều là bản sao đầy đủ của `base_tree` cộng thêm phần sửa riêng của mỗi nhánh, việc gọi `merged.update(t_dom)` sau `merged.update(t_inv)` khiến toàn bộ các file có sẵn trong `base_tree` mà `t_inv` vừa sửa bị giá trị gốc chưa sửa trong `t_dom` ghi đè mất sạch 100%, đồng thời các file bị xóa ở `t_inv`/`t_dom` lại bị khôi phục từ `base_tree`; (2) **Thiếu `task_id` và nội dung `base_tree` trong khóa băm `_run_cached_session` (`harness2_engine.py:559-563`):** Gọi `call_fingerprint_sha256(label=label, mission=f"propose:{dim}:{label}", system=self.surface.name)` mà không băm kèm `task_id` và nội dung `base_tree`/`context_dir`, dẫn tới 2 task khác nhau dùng chung `cache_dir` sinh ra cùng mã băm cho `inv1`/`dom1` (Cross-Task Cache Poisoning); (3) **Lệch khóa `rollout_outputs["base"]` và dùng điểm số `score` trực tiếp trong hàm tuyển chọn ẩn danh (`harness2_engine.py:614, 653, 700-706`):** Lưu rollout cơ sở dưới khóa `"base1..basek"` / `"b1..bk"` nhưng khi chọn ứng viên lại tra cứu `rollout_outputs.get("base")` (luôn trả về `None` / `0.0`), khiến ứng viên `"base"` (`KEEP_BASE`) bị loại oan và làm vô hiệu hóa ý nghĩa chống thiên kiến vị trí của `id_map`; (4) **Đếm độ dài `always_on` bằng số ký tự `len(body)` thay vì số byte UTF-8 `len(body.encode("utf-8"))` (`harness2_engine.py:116`) và tự sinh mảng `s3_docs = [f"{pfx}_ok.md" ...]` truyền vào `can_commit_adopted_harness_to_domain_library` (`harness2_engine.py:711-716`):** Văn bản tiếng Việt dài `18,091` ký tự nhưng thực tế chiếm `20,483` bytes UTF-8, và việc tự tạo mảng `s3_docs` giả làm điều kiện `has_all_s3` luôn đúng một cách hình thức (`L044`); (5) **Bỏ sót việc tách file `tooling/orchestrator/proposer_axes.py` và gọi đồng bộ `build_lessons_index.py` đã ghi trong `08_tasks` mà không cập nhật `04_decisions`** -> Hậu quả [Y]: Code pass unit test và 3-Mutant Gate (do test chỉ kiểm tra trường hợp thêm file mới trên 1 task đơn lẻ) nhưng khi chạy thực tế với nhiều task hoặc khi nhánh `INV` sửa file có sẵn sẽ bị mất thay đổi và nhiễm bẩn cache. **Đúng:** (1) Khi hợp nhất 2 nhánh cấu hình trực giao `t_inv` và `t_dom` từ `base_tree`, BẮT BUỘC thực hiện **3-Way Diff Merge (`three_way_merge_trees(base, t_inv, t_dom)`)**: chỉ áp dụng các key có `t_branch[k] != base[k]` và xóa các key bị xóa ở `t_branch`; (2) Khóa băm SHA-256 của session cache BẮT BUỘC phải băm toàn bộ `(task_id, label, dim, system, base_tree_digest, context_dir)`; (3) Gắn kết quả rollout cơ sở vào khóa `"base"` (hoặc truyền hàm `selector_fn(anonymous_pool)` không phụ thuộc điểm số `score` ẩn); (4) Đo hạn ngạch `always_on_max` theo byte UTF-8 (`len(body.encode("utf-8"))`), tách YAML frontmatter theo dấu `---`, và kiểm tra `s3_docs_present` từ thư mục `workspace_dir` thật trên đĩa; (5) Giữ nghiêm kỷ luật `S5 -> S3 Re-Plan` (Propose-Only): ghi toàn bộ phát hiện và giải pháp vào `audit_report_*.md` + `09_tasks_solution_*.md`, chạy `/laya_router` và chờ User `APPROVE` trước khi sửa code.:** (1) Ghi bài học vào `lessons.md` và chạy `build_lessons_index.py`; (2) Chạy `/laya_router` (`DG3`) kiểm định toàn bộ 7 phát hiện phản biện (`AF-01` $\rightarrow$ `AF-07`); (3) Lưu ...
- **`L027` [2026-10-27 · `T2:L187`] Stuff…:** Khi mở rộng một hệ thống điều phối đa tác nhân (`Multi-Agent Orchestrator` [A]) để bổ sung định tuyến model động (`Dynamic Model Routing`), chạy song song theo tầng `Wave DAG` bằng...
- **`L011` [2026-10-27 · `T1:L74`] Bypas…:** Khi User [A] đối thoại hoặc chỉ đạo thay đổi kiến trúc trong khung **Co-Architect Studio (`/brain-delegate` · `Slot A` · `gemini-4-argon (antigravity)`)** trên `ai-orchestrator` [B...