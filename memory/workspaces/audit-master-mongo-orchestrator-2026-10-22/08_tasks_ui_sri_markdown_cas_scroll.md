# 08 Tasks: Khắc phục Layout `.sri-markdown-content` & Thanh cuộn `.cas-messages-list` trên `ai-orchestrator`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `ui_sri_markdown_cas_scroll`
- **Tier:** `T2`

---

## Danh mục Task

- [x] **TASK-UI-01 (S2 Investigate Root Cause `file:line` trong `index.html`, `app.css`, `app.js`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html:288-395`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css:1776-1858,2269-2443`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js:947-986`.
  - **Mô tả:** Đối chiếu cấu trúc DOM trong `index.html` và `app.js` với các selector CSS trong `app.css` để xác định chính xác nguyên nhân `.sri-markdown-content` chỉ cao bằng 1 nửa `.sri-markdown-pane` và `.cas-messages-list` mất thanh cuộn.
  - **Logs/Test:** Kiểm tra trực tiếp mã nguồn `index.html:289-380`, `app.css:1781-1810,2348-2358`, `app.js:949-985`.
  - **Definition of Done:** Chỉ rõ `file:line` đã `VERIFIED` cho cả 2 lỗi.

- [x] **TASK-UI-02 (S3 Solution & Code Demo + Kiểm định `/laya_router` DG1/DG2):**
  - **Context (`file:line`):** `09_tasks_solution_ui_sri_markdown_cas_scroll.md`, `06_laya_claims_dg1_ui_sri_markdown_cas_scroll.json`.
  - **Mô tả:** Thiết kế 1 phương án tối ưu nhất kèm Code Demo sửa `internal/webui/static/app.css`, chạy `laya_gate.py run --stage dg1` đạt `PASS`, và trình anh `APPROVE`.
  - **Logs/Test:** `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg1` đạt `PASS`.
  - **Definition of Done:** File `09_tasks_solution_ui_sri_markdown_cas_scroll.md` hoàn chỉnh, `DG1/DG2 PASS`, giữ nguyên kỷ luật Propose-Only chờ anh `APPROVE`.

- [x] **TASK-UI-03 (S4 Execute & S5/S6 Verify sau khi có `APPROVE`):**
  - **Context (`file:line`):** `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css:1781-1819,2295-2497`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go:73-85`.
  - **Mô tả:**
    1. Cập nhật `.sri-markdown-pane`, `.sri-pre-block` (`#sri-markdown-content`) trong `app.css:1781-1819`: thêm `flex: 1 1 auto; min-height: 260px; max-height: 465px; margin: 0;` cho `.sri-pre-block, #sri-markdown-content` để tự giãn lấp đầy 100% chiều cao của `.sri-markdown-pane` cân xứng với cột `.sri-side-pane` bên phải và có thanh cuộn dọc rõ ràng.
    2. Cập nhật `.cas-messages-list` (cùng `.cas-messages-box, #co-architect-chat-messages`) và toàn bộ các class con của `#co-architect-studio` (`.cas-title-group`, `.cas-sub-hint`, `.cas-controls`, `.cas-select-label`, `.cas-msg-card`, `.cas-msg-card.user-msg`, `.cas-msg-card.agent-msg`, `.cas-msg-author`, `.cas-msg-meta`, `.cas-input-form textarea`, `.cas-form-actions`, `.btn-cas-send`, `.btn-cas-approve`) trong `app.css:2295-2497`: đặt `max-height: 260px; min-height: 110px; overflow-y: auto; overscroll-behavior: contain;` kèm styling scrollbar cho `.cas-messages-list`.
    3. Bổ sung kiểm tra regression trong `internal/webui/server_test.go:73-85`, build lại `./bin/orchestrator`, khởi động lại daemon `:8090`, kiểm tra live qua `evidence_run.py`, chạy `DG3` và xuất `audit_report_ui_sri_markdown_cas_scroll.md`.
  - **Logs/Test:** `go vet ./...` (`EV-20261022T132404-f53c`), `go test -v -race ./internal/webui/...` (`EV-20261022T132404-fa5c`), `go run ./cmd/orchestrator daemon --once` (`EV-20261022T132431-836d`), kiểm tra HTTP `/app.css` trên `:8090` qua `evidence_run.py` (`EV-20261022T132644-2c7d`).
  - **Definition of Done:** `#sri-markdown-content` lấp đầy 100% khung `.sri-markdown-pane`, `.cas-messages-list` có thanh cuộn dọc hoạt động mượt mà trên `http://127.0.0.1:8090/`, pass đủ `DG3` và `verify_governance.py`.
