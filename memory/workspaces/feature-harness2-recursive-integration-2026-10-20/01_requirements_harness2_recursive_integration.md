# 01 — Requirements: Phân tích Chuyên sâu `google-research/harness2` & Thiết kế Tích hợp Đệ quy 2 Tầng vào Hệ sinh thái `/agent` (`harness2_recursive_integration`)

tier: T2
suffix: harness2_recursive_integration
workspace: feature-harness2-recursive-integration-2026-10-20
environment: local
status: PROPOSED (Waiting for User APPROVE)

---

## 1. Bối cảnh & Yêu cầu của User (Phiên 3 — Tiếp nối Phiên 1 `Open-Makers` & Phiên 2 `Queue-Driven Orchestrator`)

User cung cấp kho mã nguồn nghiên cứu mới nhất của Google Research: **[google-research/harness2](https://github.com/google-research/harness2)** (*Harness²: Recursive Agent Harnessing for an Open World*, arXiv:2610.14224) và yêu cầu:
> `https://github.com/google-research/harness2 => tiếp tục đọc cái này`

Mục tiêu của tác vụ:
1. Giải phẫu toàn diện bản chất vận hành vật lý (`Macro Runtime Topology` — tuân thủ Lesson `L001`) và kiến trúc mã nguồn thực tế (`file:line`) của `google-research/harness2`.
2. Đối sánh trực diện 8 thành phần `c1..c8` (`SurfaceSpec`) và cơ chế **Đệ quy 2 Tầng (`Task-Level Propose–Probe–Compose` + `Harnessing-Level Experience Reflector`)** của `Harness²` với **Bộ `/agent` hiện tại (`S0 → S6`)** cùng **Hệ thống `Modular Queue-Driven Agent Orchestrator` (Phiên 1 + Phiên 2)**.
3. Xác định chính xác các khoảng trống kiến trúc (`F1–F6`) và thiết kế **1 phương án hợp nhất tối ưu duy nhất** (ánh xạ 1:1 theo từng khoảng trống `F1..F6` — tuân thủ Lesson `L002`) để nâng cấp hệ thống thành **Kiến trúc 3 Tầng Hoàn chỉnh**:
   - **Tầng 1 (Inner Governance Kernel):** Bộ `/agent` (`GEMINI.md`, `skills/`, `hooks/agy/agy_guard.py` `H1–H7`, `laya_gate.py` `DG1–DG3`, `mutation_gate.py`, `verify_governance.py`).
   - **Tầng 2 (Middle Queue-Driven Factory):** Băng chuyền `Scout ($0) → Sorting → Planner (Opus Wave 0..N DAG) → Coder (Dynamic Opus/Sonnet + Git Worktree) → Reviewer (Autopilot)`.
   - **Tầng 3 (Outer Meta-Harness Optimizer — `Harness²`):** Động cơ tự tiến hóa đệ quy 2 cấp (`INV/DOM Proposer`, `SurfaceSpec Validator & Auto-Trigger`, `Disk ContextTree & Leak Tripwire`, `Blind Anonymized Selector`, `Deterministic Budget Identity R=4k/P=3k`, và `Per-Domain Harness Library`).

---

## 2. Danh sách Yêu cầu Chức năng Chi tiết (`R0` – `R6`)

| Mã Req | Mô tả Yêu cầu | Tiêu chí Nghiệm thu (Acceptance Criteria) |
| :--- | :--- | :--- |
| **R0** | **Bức tranh Tổng quan Vận hành Vật lý (`Macro Runtime & Integration Topology` — Lesson `L001`)** | Mở đầu báo cáo bằng sơ đồ và giải thích vật lý rõ ràng: `Harness` là gì, `Harness²` là tiến trình OS gì, khác biệt thế nào với phiên chat Antigravity IDE hiện tại và ghép vào Tầng 1 (`/agent`) + Tầng 2 (`Queue-Driven Orchestrator`) ở vị trí vật lý nào. |
| **R1** | **Đóng `F1` — Tích hợp Động cơ Đệ quy 2 Tầng (`Task-Level Propose–Probe–Compose` + `Harnessing-Level Domain Library`)** | Chuyển hóa cơ chế tự học thủ công sau lỗi (`GEMINI.md:47-55`) thành vòng lặp tự tiến hóa tự động khi Worker gặp task khó hoặc trượt QA: chạy `Propose (INV + DOM) → Probe → Compose` (`recursion.py:343-567`) và lưu trọn bộ cây `/agent` đã thắng kèm `EXPERIENCE.md` vào thư viện theo từng domain (`library.py:106-181`). |
| **R2** | **Đóng `F2` — Phân tách 2 Trục Đề xuất Trực giao (`INV` vs `DOM`) & Thứ bậc Tiết kiệm Ngân sách Context (`Skill → Script → Subagent → Hook → Always-On`)** | Tách biệt rõ `INV Proposer` (chỉ sửa quy trình, script `.py`, hook `agy_guard.py`) và `DOM Proposer` (chỉ sửa kiến thức chuyên ngành trong `skills/<name>/SKILL.md`), đồng thời áp dụng thứ bậc ưu tiên chống phình to `Always-On` (`prompts.py:129-146, 347-407`). |
| **R3** | **Đóng `F3` — Xây dựng Bộ Kiểm định Bề mặt Cấu hình `/agent` (`AgentSurfaceValidator & Auto-Trigger Compiler`)** | Bổ sung trình kiểm định cấu hình `/agent` theo chuẩn `SurfaceSpec` (`tree.py:40-62, 192-390`): kiểm tra trần ký tự `always_on_max` của `GEMINI.md`, kiểm tra frontmatter `name:`/`description:` của mọi `SKILL.md`, kiểm tra dòng Trigger trong `GEMINI.md` (`autotrigger`), và kiểm tra mọi script `tooling/*.py` được tham chiếu đều tồn tại & compile AST sạch (`_check_script_refs`, `_py_compiles`). |
| **R4** | **Đóng `F4` — Chuẩn hóa Cây Bằng chứng Vật lý (`ContextTree`) & Chốt chặn Tiền kiểm Phòng sạch (`Pre-Flight Leak & Grader Verdict Tripwire`)** | Đóng gói bằng chứng cho Sub-agent/Reviewer/Improver dưới dạng cây thư mục trên đĩa (`context/task.md`, `rollouts/`, `deliverables/`, `harness/`, `pool/`) kèm prompt con trỏ ngắn (`context.py:42-103, 521-546`) và quét chặn cứng các file nguồn cấm (`assert_no_leak_paths`) + nhãn phán quyết (`assert_no_grader_verdicts`) **trước** khi khởi chạy tiến trình Model (`context.py:399-519`), nâng cấp chốt hậu kiểm hiện tại ở `hooks/agy/agy_guard.py:282-301`. |
| **R5** | **Đóng `F5` — Tuyển chọn Ẩn danh Chống Thiên kiến (`Blind Anonymized Selection`) & Đo Độ Đa dạng (`Pairwise Line-Jaccard Diversity`)** | Khi so sánh nhiều bản sửa hoặc giữa bản gốc (`KEEP_BASE`) với bản tinh chỉnh, ẩn danh hóa nhãn ứng viên thành `A, B, C...` bằng SHA-1 có seed (`selection.py:71-86`), kết hợp đo độ hội tụ `diversity()` (`tree.py:407-428`) và thẩm định qua `/laya_router` (`DG3`). |
| **R6** | **Đóng `F6` — Hằng đẳng thức Ngân sách Tất định (`Ledger.assert_identity: R=4k, P=3k`) & Content-Addressed SHA-256 Cache + `flock` Resume** | Bổ sung kiểm toán số lượt chạy `R=4k, P=3k` chống ghi đè trùng nhãn (`ledger.py:89-113`), bộ nhớ đệm phiên chạy theo mã băm SHA-256 của toàn bộ cây bằng chứng (`proposer.py:507-534`), và khóa tệp nguyên tử `fcntl.flock` (`schedule.py:289-329`) vào tầng điều phối. |
