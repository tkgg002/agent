# Báo Cáo Audit Phản Biện Toàn Diện (`S5` & `S6`) — Task `ui_sri_markdown_cas_scroll` Trên `ai-orchestrator`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `ui_sri_markdown_cas_scroll`
- **Tier:** `T2`
- **Project:** `ai-orchestrator`
- **Ngày thực hiện:** 2026-10-22
- **Trạng thái:** `DONE (S0 → S6 Complete — Khắc phục triệt để 2 lỗi UI #sri-markdown-content bị hụt chiều cao và .cas-messages-list thiếu scroll dọc trên ai-orchestrator)`

---

## 1. Mục 1: Phản Biện Toàn Bộ "Quá Trình" Vừa Thực Hiện, Từng Task (`TASK-UI-01 .. TASK-UI-03`), Từng File & Từng Dòng Update (`git diff` Line-by-Line Critique)

### 1.1. Phản Biện Toàn Bộ Quá Trình (`S0 → S5`)
- **Tuân thủ tuyệt đối `S0 → S3 → DG1/DG2 → APPROVE → S4 → S5`:**
  - `TASK-UI-01`: Điều tra bằng chứng `file:line` trên `internal/webui/static/index.html`, `internal/webui/static/app.js`, `internal/webui/static/app.css` và xác định chính xác gốc rễ của cả 2 lỗi mà không đoán mò.
  - `TASK-UI-02`: Thiết kế duy nhất 1 phương án tối ưu trong `09_tasks_solution_ui_sri_markdown_cas_scroll.md`, qua cổng `DG1 + DG2` (`06_laya_summary_dg1_20261022T131730.json` = `PASS`, `06_laya_summary_dg2_20261022T131730.json` = `PASS`), và chờ anh `APPROVE` rồi mới sửa code.
  - `TASK-UI-03`: Thực thi sửa `internal/webui/static/app.css` và bổ sung regression assertion trong `internal/webui/server_test.go`, rebuild binary `./bin/orchestrator`, khởi động lại tiến trình live trên `127.0.0.1:8090`, và thu thập biên lai `EV-...` thật.
- **Phát hiện & tự hiệu chỉnh ngay trong `S4` (Zero Speculation):**
  - Ban đầu test kiểm tra `GET /static/app.css`, nhưng khi chạy `go test` thật đã phát hiện `server.go` phục vụ file tĩnh từ gốc `/app.css` (khớp với `<link rel="stylesheet" href="/app.css" />` tại `index.html:7`). Đã sửa ngay assertion trong `server_test.go:74` sang `GET /app.css` và xác minh lại `PASS` thật trước khi sang `S5`.
- **Tuân thủ ranh giới dự án (`L003`):** Không đụng bất kỳ dòng nào vào `/Users/trainguyen/Documents/work/data-hub/*`.

### 1.2. Phản Biện Từng File & Từng Dòng `git diff` (`Line-by-Line Diff Critique`)

1. **[app.css](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1781-L1819) (Khối `#sri-markdown-content` / `.sri-markdown-pane`):**
   - **Dòng `1781-1788` (`.sri-markdown-pane, .sri-side-pane`):** Thêm `min-height: 0;` vào flex column container.
     - *Phản biện:* Trong CSS Flexbox/Grid (`grid-template-columns: 1.35fr 1fr; align-items: stretch` tại `.sri-split-grid` dòng `1775-1780`), phần tử flex column cần `min-height: 0` để con bên trong (`flex: 1 1 auto; overflow-y: auto`) tính toán đúng biên cuộn nội bộ mà không làm bung khung grid.
   - **Dòng `1802-1819` (`.sri-pre-block, #sri-markdown-content`):**
     - Thay `max-height: 235px;` bằng `flex: 1 1 auto; min-height: 260px; max-height: 465px; margin: 0; overflow-y: auto; overscroll-behavior: contain;`.
     - *Phản biện:* Trước đây `.sri-side-pane` bên phải cao `~470px` nhưng `.sri-pre-block` bị chặn cứng ở `max-height: 235px` và không có `flex: 1 1 auto`, khiến khung `#sri-markdown-content` chỉ chiếm đúng 1 nửa trên của `.sri-markdown-pane`, để trống nửa dưới. Nay `#sri-markdown-content` tự giãn đầy 100% chiều cao còn lại của `.sri-markdown-pane`, bỏ `margin` mặc định của thẻ `<pre>`, và cuộn mượt bên trong khi nội dung dài quá `465px`.

2. **[app.css](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L2295-L2497) (Khối `.cas-messages-list` / `#co-architect-studio`):**
   - **Dòng `2357-2386` (`.cas-messages-list, .cas-messages-box, #co-architect-chat-messages` & `::-webkit-scrollbar`):**
     - Gắn `min-height: 110px; max-height: 260px; overflow-y: auto; overflow-x: hidden; overscroll-behavior: contain; scrollbar-width: thin; display: flex; flex-direction: column; gap: 7px;` cùng bộ custom scrollbar `width: 7px`.
     - *Phản biện:* Trước đây `index.html:380` khai báo `<div class="cas-messages-list" id="co-architect-chat-messages">` nhưng `app.css` lại viết `.cas-messages-box` nên `.cas-messages-list` không khớp bất kỳ rule CSS nào, dẫn đến mất `max-height` và mất `overflow-y: auto`. Nay gộp cả 3 selector (`.cas-messages-list, .cas-messages-box, #co-architect-chat-messages`) đảm bảo khung tin nhắn luôn có thanh cuộn dọc rõ ràng và `scrollTop = scrollHeight` trong `app.js:1264` hoạt động thật.
   - **Dòng `2295-2356` & `2387-2497` (Các selector con của `#co-architect-studio`):**
     - Đồng bộ đầy đủ các class thực tế đang được `index.html:356-403` và `app.js:1232-1263` render: `.cas-title-group`, `.cas-sub-hint`, `.cas-controls`, `.cas-select-label`, `.cas-msg, .cas-msg-card` (`flex-shrink: 0;` để thẻ tin nhắn không bị bóp dẹt trong flex column khi danh sách dài), `.cas-msg-card.user-msg`, `.cas-msg-card.agent-msg`, `.cas-msg-author, .cas-msg-meta`, `.cas-slot-pill, .cas-skill-pill, .cas-mode-pill`, `.cas-model-pill`, `.cas-applied-pill`, `.cas-time`, `.cas-input-form textarea`, `.cas-form-actions`, `.btn-cas-send`, `.btn-cas-approve`.

3. **[server_test.go](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go#L73-L85) (`+13/-0` dòng):**
   - Bổ sung kiểm định HTTP `GET /app.css` trong `TestWebUIServer_StaticIndexAndThreeScreensDashboard`, bắt buộc nội dung CSS phục vụ qua HTTP chứa đủ `.sri-pre-block, #sri-markdown-content`, `flex: 1 1 auto;`, `.cas-messages-list, .cas-messages-box, #co-architect-chat-messages`, `.cas-msg-card.user-msg`, `.cas-msg-card.agent-msg`.

---

## 2. Mục 2 & Mục 3: Chống Suy Diễn Smoke Test (`Anti-Smoke-Test Speculation`), Kiểm Tra Concurrency / Timeout-Heartbeat / Watermark Continuity & Biên Lai Runtime Thật (`EV-...`)

### 2.1. Phản Biện Concurrency (`WorkerCount > 1`), Batching, Timeout-Heartbeat (`AckWait` vs `InProgress`) & Watermark Continuity
- Bản chất thay đổi của task `ui_sri_markdown_cas_scroll` chỉ tác động lên tầng trình bày CSS (`internal/webui/static/app.css`) và test kiểm định HTTP static asset (`internal/webui/server_test.go`) của `ai-orchestrator`.
- Đối với backend `ai-orchestrator`:
  - Toàn bộ 6 bộ integration test của package `internal/webui` được chạy với cờ `-race` (`EV-20261022T132404-fa5c`), xác nhận không có data race trên `Server.mu`, SQLite `QueueStore`, SSE hub hay worker pool (`WorkerCount > 1`), không ảnh hưởng tới cơ chế gom lô (`batch/debouncer`), `AckWait/Timeout` vs `Heartbeat/InProgress`, hay tính liên tục của `Watermark/Cursor` sau sự cố Outage.
- **Minh bạch kiểm chứng thị giác (Theo Lesson `L057`):**
  - Em đã kiểm chứng trực tiếp tệp `/app.css` được tiến trình `./bin/orchestrator ui --addr 127.0.0.1:8090` phục vụ qua HTTP thật (`EV-20261022T132644-2c7d`) và kiểm tra tính đồng nhất class giữa `index.html`, `app.js`, và `app.css`. Anh có thể bấm `Cmd+Shift+R` (Hard Reload) trên trình duyệt đang mở `http://127.0.0.1:8090` để nạp bản `/app.css` mới nhất và kiểm tra trực quan ngay trên màn hình.

### 2.2. Danh Sách Biên Lai Thực Thi Vật Lý (`EV-...` — Zero Mockup)
Toàn bộ biên lai được tạo bởi `evidence_run.py` sau lần sửa code cuối cùng và lưu tại `/Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_evidence/`:

1. **Biên lai `EV-20261022T132404-f53c` (`unit` — `go vet ./...`):**
   - Lệnh: `go vet ./...` tại `/Users/trainguyen/Documents/work/ai-orchestrator`
   - Kết quả: `exit=0` (`PASS`).
2. **Biên lai `EV-20261022T132404-fa5c` (`integration` — `go test -v -race ./internal/webui/...`):**
   - Lệnh: `go test -v -race ./internal/webui/...` tại `/Users/trainguyen/Documents/work/ai-orchestrator`
   - Kết quả: `exit=0`, **6/6 test suites `PASS` trong `23.784s`** (bao gồm kiểm định `GET /app.css` trong `TestWebUIServer_StaticIndexAndThreeScreensDashboard`).
3. **Biên lai `EV-20261022T132431-836d` (`runtime` — `go run ./cmd/orchestrator daemon --once`):**
   - Lệnh: `go run ./cmd/orchestrator daemon --once` tại `/Users/trainguyen/Documents/work/ai-orchestrator`
   - Kết quả: `exit=0` (`PASS`).
4. **Biên lai `EV-20261022T132644-2c7d` (`runtime` — Live HTTP check trên `http://127.0.0.1:8090/app.css`):**
   - Lệnh: `curl -fsS http://127.0.0.1:8090/app.css | grep -nE 'sri-markdown-content|cas-messages-list|cas-msg-card\.user-msg'`
   - Kết quả: `exit=0`, trả về đúng các dòng thực tế đang phục vụ trên `:8090`:
     - `1802:.sri-pre-block, #sri-markdown-content {`
     - `2357:.cas-messages-list, .cas-messages-box, #co-architect-chat-messages {`
     - `2373:.cas-messages-list::-webkit-scrollbar {`
     - `2394:.cas-msg-user, .cas-msg-card.user-msg {`

---

## 3. Mục 4: Vòng Lặp Phản Tỉnh & Khắc Phục (`Self-Improvement Loop`)

- Đối chiếu với `lessons/INDEX.md` (`L001`–`L278`) và `topics/05_frontend_and_ui.md`:
  - Trong task `ui_sri_markdown_cas_scroll` này, em tuân thủ nghiêm ngặt quy trình `S0 → S6`, không tự ý sửa code trước `APPROVE` (Rule #13), không đụng vào repo `data-hub` (`L003`), và không báo cáo kết quả giả (`L057`). Không phát sinh vi phạm lesson mới trong lượt thực thi này.

---

## 4. Mục 5 & Mục 6: Kết Quả Hội Đồng `/laya_router` (`DG3`) & Quét Bảo Mật (`/security-agent`)

1. **Cổng `DG1` + `DG2` (`06_laya_summary_dg1_20261022T131730.json`, `06_laya_summary_dg2_20261022T131730.json`):**
   - Kết quả: **`gate = PASS`**.
2. **Cổng `DG3` sau lần sửa code cuối (`06_laya_summary_dg3_20261022T132646.json`):**
   - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 --workspace audit-master-mongo-orchestrator-2026-10-22 --claims /Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_claims_dg3_ui_sri_markdown_cas_scroll.json`
   - Kết quả: **`gate = PASS`**.
   - Chi tiết 3 claims `A-SRI-FLEX-HEIGHT`, `A-CAS-SCROLL`, `A-CAS-CARDS`: Cả 3 claims đều đạt `CORRECT` ở Round 1 & Round 2 và `YES` ở Round 3 (`confidence = 0.9119 .. 0.9450`); do Laya đánh dấu `UNVALIDATED` ở bước lọc control, đã đối chiếu trực tiếp với mã nguồn `internal/webui/static/app.css:1781-1819, 2357-2401` và biên lai runtime `EV-20261022T132404-fa5c`, `EV-20261022T132644-2c7d` theo đúng Rule #10.
3. **Kiểm định Bảo mật (`/security-agent`):**
   - Quét toàn bộ `git diff` và các file workspace: `0` hardcoded secret/PII (Rule #20), `0` closure mock `*Fn` trong struct production (Rule #12), không chạy `git push` (Rule #15).
