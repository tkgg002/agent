---
tier: T2
mode: standard
target_project: ai-orchestrator
suffix: transparency_ui_declutter_2026_11_04
created_at: 2026-11-04T17:51:30+07:00
---

# 08. Tasks Breakdown — Minh Bạch Thực Thi Model, Repo Memory Define & Tinh Gọn UI/UX (`ai-orchestrator`)

## Task 1 (`T-INVESTIGATE`): [DONE] Khảo sát thực trạng LLM Call Logging, Repo Memory & toàn bộ Label/Badge trên WebUI (`S2`)
- **Context:**
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css`
- **Logs/Test:** Kiểm tra toàn bộ chỗ ghi nhận lời gọi LLM hiện tại, chỗ quét/lưu thông tin repo, chỗ liên kết chéo task cũ (`sibling`/`cache`), và toàn bộ label text đang render trên `index.html` + `app.js`.
- **Definition of Done (DoD):** Liệt kê đầy đủ `file:line` của từng điểm gây thiếu minh bạch, từng điểm quét lòng vòng/liên kết chéo task, và lập bảng kiểm toán từng label text trên UI.

## Task 2 (`T-CRITIQUE-SOLUTION`): [DONE] Chạy Pre-Plan Critique (`DG0`), Thiết kế Giải pháp 1-1 & Code Demo (`S2.5` & `S3`)
- **Context:** Workspace `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-ai-orchestrator-transparency-ui-declutter-2026-11-04/09_tasks_solution_transparency_ui_declutter_2026_11_04.md`.
- **Logs/Test:** Chạy `laya_gate.py run --stage dg0` và `laya_gate.py run --stage dg1` đạt `PASS`.
- **Definition of Done (DoD):** Có bảng kiểm toán từng label text (Giữ / Xóa / Thu gọn), thiết kế chi tiết Feature 1 (Per-Task LLM Transparency Ledger + Repo Memory Define Bootstrap, cắt bỏ liên kết chéo task) & Feature 2 (UI/UX Decluttering), kèm Code Demo nguyên văn từ HEAD và dừng chờ `APPROVE`.

## Task 3 (`T-EXECUTE-S4`): [DONE] Triển khai Backend (`repo_memory_store.go`, `task_llm_trace.go`, đấu nối LLM Call Sites) & Frontend (`index.html`, `app.js`, `app.css`) + Unit/Integration Tests (`S4`)
- **Context:**
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/repo_memory_store.go`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_trace.go`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go:191-298`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/subtask_codegen_muscle.go:250-272`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/subtask_edit_service.go:311-333`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2794-2843`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css`
- **Logs/Test:** Chạy `go vet ./...` (`EV-20261105T103930-38b0`), `go test ./...` (`EV-20261105T104010-261e`), và `go run ./cmd/orchestrator daemon --once` (`EV-20261105T104649-8958`) qua `evidence_run.py`.
- **Definition of Done (DoD):** Mọi phiên gọi LLM của từng Task lưu vào `.orchestrator/tasks/<taskKey>/llm_call_ledger.json`, nạp/khởi tạo `repo_definition.json` cho `targetRepo` không liên kết chéo task, và giao diện WebUI tinh gọn hiển thị trực tiếp Tab `🔎 Minh Bạch Model & Prompt` cùng `🧠 Repo Memory` badge.

## Task 4 (`T-AUDIT-S5`): [DONE] Kiểm định Phản biện Cuối (`S5` — `git diff`, `evidence_run.py`, `DG3`, `audit_report_*`, `verify_governance.py`)
- **Context:** Workspace `/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-ai-orchestrator-transparency-ui-declutter-2026-11-04/audit_report_transparency_ui_declutter_2026_11_04.md`.
- **Logs/Test:** Chạy `git diff`, `evidence_run.py` (`unit`, `integration`, `runtime`), `laya_gate.py run --stage dg3` (`06_laya_summary_dg3_20261105T104535.json`), và `verify_governance.py`.
- **Definition of Done (DoD):** Đạt `PASS` tất cả cổng G1–G8, `DG3` và `verify_governance.py`.

## Task 5 (`T-SELF-HEAL-S5-S3-S4-S5`): [DONE] Vòng Lặp Phản Tỉnh & Tự Khắc Phục (`AF-1..AF-5` — Tách Module `NOW <= HEAD`, Chặn Láo `REAL_LLM_OK` Khi Lỗi 401, Định Tuyến Nhánh `feat/auto-recon-export`, Test HTTP Thật & Rebuild Live `:8090`)
- **Context:**
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/repo_memory_store.go:1-336`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_trace.go:1-349`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/repo_memory_trace_test.go:1-183` (Mới — tách khỏi `task_llm_bridge_test.go`)
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/task_transparency.js:1-156` (Mới — tách khỏi `app.js`)
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/task_transparency.css:1-23` (Mới — tách khỏi `app.css`)
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go`, `server.go`, `task_llm_bridge.go`, `subtask_codegen_muscle.go`, `subtask_edit_service.go`, `static/index.html`, `static/app.js`, `static/app.css`
- **Logs/Test:** Đo `wc -l` xác nhận toàn bộ các file lớn đều đạt `NOW <= HEAD`; chạy `go vet ./...` (`EV-20261105T103930-38b0`), `TestRepoMemoryBootstrap_BranchAwareResolution_Honest401Guard_AndPerTaskHTTPIsolation` (`EV-20261105T103947-4543`), `TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch` (`EV-20261105T103844-2cfd`), `go test ./...` (`EV-20261105T104010-261e`), rebuild `./bin/orchestrator`, restart tiến trình `:8090` và kiểm tra trực tiếp `/task_transparency.js` (`EV-20261105T104712-acfa`) cùng `/api/dashboard` (`EV-20261105T104723-112c`) trên `:8090` qua `evidence_run.py --kind runtime`, chạy `laya_gate.py run --stage dg1` (`06_laya_summary_dg1_20261105T104507.json`) + `laya_gate.py run --stage dg3` (`06_laya_summary_dg3_20261105T104535.json`).
- **Definition of Done (DoD):** Toàn bộ 5 lỗ hổng `AF-1..AF-5` được đóng sạch 100%, `NOW <= HEAD` trên mọi file cũ, live server `:8090` phục vụ code & bundle mới, và `audit_report_transparency_ui_declutter_2026_11_04.md` ghi nhận trung thực từng dòng diff + nơi cần phối hợp với User để trace trình duyệt thực tế.

## Task 6 (`T-RESTORE-WORKFLOW-WAVE-UI`): [DONE] Tái cấu trúc đồng bộ Kiến trúc Thông tin 3 Phân vùng (`3-Zone Clean IDE Layout`) & Khôi phục Sơ đồ Workflow `S1→S6` + Dải Wave `0..N` Sub-Tasks (`#autopilot-flow`)
- **Context:**
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:145-152,289-468`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:746-930,1307-1553,1958-2006`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/task_transparency.js:1-335`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/task_transparency.css:1-294`
  - `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/repo_memory_trace_test.go:180-215`
- **Logs/Test:** Chạy `laya_gate.py run --stage dg1` đạt `PASS` (`06_laya_summary_dg1_20261105T133030.json`); sau khi User `APPROVE`, thực thi S4, biên dịch `./bin/orchestrator`, khởi động lại `:8090`, chạy `go vet ./...` (`EV-20261105T133558-c2c9`), `TestRepoMemoryBootstrap_BranchAwareResolution_Honest401Guard_AndPerTaskHTTPIsolation` (`EV-20261105T133542-751b`), `go test ./...` (`EV-20261105T133601-9c94`), và kiểm chứng trực tiếp cổng `:8090` qua `evidence_run.py` (`EV-20261105T134310-87ce`, `EV-20261105T134412-02c7`, `EV-20261105T134420-bb4d`), giữ nguyên trần dòng `NOW <= HEAD` (`app.js = 3357 <= 3409`, `index.html = 1013 <= 1027`).
- **Definition of Done (DoD):** (1) Khôi phục `#autopilot-flow` hiển thị 2 dải trực quan: **Sơ đồ Bước Quy Trình (`S1 → DG0 → S3 → Gate → S4 → S5`)** gọn 1 dòng và **Dải Wave Thực Thi Sub-Tasks (`Wave 0 → Wave 1 → ...`)** của riêng Task đang chọn; (2) Xóa bỏ triệt để tình trạng sửa chắp vá & chồng chéo 7 khối trên 1 màn hình: ẩn thanh `.sri-header` lặp lại `Repo Memory`/`APPROVE`, tách `🧩 Sub-Tasks` và `💬 Đối Thoại & Log` thành các Tab chế độ xem độc lập trên `#sri-tabs-bar` (khi xem `S1..S5` chỉ hiển thị 2 cột Báo cáo Markdown & Code Demo full-height sạch sẽ; khi bấm Tab `🔎 Minh Bạch Model` hiển thị danh sách phiên + Prompt bên trái và Raw Response bên phải; khi bấm Tab `🧩 Sub-Tasks` hoặc chip Wave hiển thị toàn khung Sub-Tasks & Fractal Inspector; khi bấm Tab `💬 Đối Thoại & Log` hiển thị Co-Architect Chat + Live Console); (3) Tinh gọn thẻ Task ở Sidebar trái và thẻ Sub-Task không lặp lại nhãn model/role dư thừa.

### Task 7: Trình Duyệt Bản Mẫu Trực Quan UI/UX, Hệ Màu Ngữ Nghĩa 5 Trạng Thái & Hệ Thống Progress Bar 3 Tầng Trước Khi Sửa Code (`[IN_PROGRESS — PROPOSE-ONLY]`)
- **Context (`file:line`):**
  - `internal/webui/static/app.css:777-810` (Hệ màu ngữ nghĩa gốc của `.ap-subcard`: `status-done` Emerald `#22c55e`, `status-waiting` Amber `#f59e0b`, `status-running` Cyan `#38bdf8` + `runningPulseGlow`, `status-pending` Slate dashed `#64748b`).
  - `internal/webui/static/app.css:2116-2172` & `internal/webui/static/app.js:1647-1683` (Thanh tiến độ `#live-telemetry-hud` đang bị ẩn mất khi `!isRunning` tại `app.js:1675`).
  - `internal/webui/static/task_transparency.js:258-322` & `internal/webui/static/task_transparency.css:11-193` (Thanh `.twf-stage-pill` hiện tại đang tô đồng loạt màu xám tối `rgba(30, 41, 59, 0.78)` làm mất phân biệt màu sắc giữa các trạng thái và thiếu thanh Progress Bar tổng thể + mini progress bar cho từng Wave).
- **Logs / Test:**
  - Cung cấp Interactive Visual UI/UX Prototype (`ui_ux_preview.html`) nhúng trực tiếp trên khung chat + bảng thiết kế màu sắc & thanh tiến độ 3 tầng.
  - Tuyệt đối giữ chế độ **Propose-Only** (không sửa bất kỳ dòng code nào trước khi User xem bản mẫu UI/UX và `APPROVE`).
- **Definition of Done (DoD):** (1) Trình bày rõ ràng bản mẫu trực quan UI/UX (có thể bấm chuyển đổi trạng thái `READY 0%`, `RUNNING S2 35%`, `NEEDS_YOU S3 55%`, `RUNNING Wave 1 75%`, `DONE 100%` và chuyển đổi các Tab) ngay trên chat; (2) Đặc tả chi tiết Bảng màu ngữ nghĩa 5 trạng thái (`DONE` Xanh Ngọc, `RUNNING` Xanh Lam Shimmer, `NEEDS_YOU` Vàng Hổ Phách, `ERROR/BLOCK` Đỏ Hồng, `PENDING` Xám Nét Đứt) và Hệ thống Progress Bar 3 tầng (Overall Task Progress Strip luôn hiển thị + 6-Stage Workflow Progress + Per-Wave Sub-Task Progress Bar); (3) Qua cổng `DG0` và `DG1/DG2` (`PASS`) và dừng chờ User `APPROVE`.
