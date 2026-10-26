---
tier: T2
suffix: real_repo_diff_queue_and_telegram_task_menu
stage: S2_S25
---
# 🔬 S2 Investigation & S2.5 Pre-Plan Adversarial Critique (`DG0`): Real Repo Diff (`centralized-export-service`), Hàng Đợi Task (Theo Repo) & Telegram `/task` Click-to-Run

## 1. S2 Evidence-Based Root Cause Analysis (`VERIFIED` vs `HYPOTHESIS`)

### 1.1. [VERIFIED] Tại sao `TASK-13` báo `DONE` nhưng trong `/Users/trainguyen/Documents/work/centralized-export-service` không có `git diff`?
- **Bằng chứng 1 (`internal/webui/telegram_bridge.go:1037-1109`):**
  - Hàm `synthesizeIsolatedRemediationDiffIfApproved` chỉ tạo chuỗi comment giả lập (`// Isolated Remediation Hunk for TASK-13...`) ghi vào `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/TASK-13/diff.patch` và không hề sửa file thật trong `/Users/trainguyen/Documents/work/centralized-export-service` (`git status -s` trả về rỗng).
- **Bằng chứng 2 (`internal/webui/server.go:1497-1503` & `1682-1688`):**
  - Trong `S4 Implement` và `S5 QA`, khi `filepath.Clean(repoPath) != filepath.Clean(s.Root)` và thư mục `.orchestrator/worktrees/TASK-13/centralized-export-service/.git` chưa được tạo bằng `git worktree add`, biến `execDir` bị gán bằng `""`. Do đó `S4/S5` bỏ qua hoàn toàn việc kiểm tra `git status --short` và `git diff --stat` trên `/Users/trainguyen/Documents/work/centralized-export-service`.
- **Bằng chứng 3 (`internal/webui/task_inspector.go:338-357` & `3124-3182`):**
  - Khi quét `/Users/trainguyen/Documents/work/centralized-export-service` cho câu lệnh `"con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret."`, hàm `extractKeywords` giữ lại từ `"con"` (khớp `consent-policy-export.params.ts`) và `"export"` (khớp `disbursement-trans-his-export.params.ts`), khiến `TASK-13-M01` và `TASK-13-M02` trỏ sai file thay vì các file thực tế của `trans_his` và PostgreSQL:
    1. `dbHandler/postgres.ts:1-60` (hiện đang bị comment `//` toàn bộ class `PostgresDb`).
    2. `models/trans-his/trans-his.model.ts:1-94` (hiện đang dùng `mongoose` `Connection` & `Schema<TransHisEntity>`).
    3. `defined/app-setting.ts:10-14, 67` (hiện đang khai báo `alias: "trans-his", type: "mongo"`).
    4. `svc-env.ts:40-41, 121-122` (chỉ có `TRANS_HIS_MONGO_URI`, thiếu cấu hình biến môi trường PostgreSQL `TRANS_HIS_POSTGRES_*`).
    5. `start.ts:6-7, 29-30, 233-268` (chỉ khởi tạo `MongoDb` và `MariaDb`, chưa nối `PostgresDb` khi `config.type === "postgres"`).

### 1.2. [VERIFIED] Tại sao bảng `Hàng Đợi Task (Theo Repo)` trên WebUI (`:8090`) chưa quản lý được theo Repo và khó biết task nào đang chạy?
- **Bằng chứng (`internal/webui/static/index.html:130-139` & `internal/webui/static/app.js:719-814`):**
  - Mặc dù tiêu đề cột là `Hàng Đợi Task (Theo Repo)`, hàm `renderSidebarTaskQueue()` chỉ render một danh sách phẳng (`allNodes.map(...)`) mà không gom nhóm theo `TargetRepo`.
  - Thiếu thanh tổng quan trạng thái hàng đợi (`⚡ Đang chạy`, `⏸ Chờ duyệt`, `⏳ Hàng đợi`, `✓ Đã xong`), thiếu bộ lọc nhanh (`Tất cả`, `Đang chạy / Chờ`, `Chờ duyệt`, `Đã xong`), thiếu vị trí thứ tự hàng đợi (`#1, #2...`), và thiếu chỉ báo `git diff` thực tế trên repo đích (`📝 N files · +X/-Y`).

### 1.3. [VERIFIED] Tại sao gõ `/task` trên Telegram chưa liệt kê danh sách task kèm nút bấm chạy ngay?
- **Bằng chứng (`internal/webui/telegram_bridge.go:560-578`):**
  - Hàm `processIncomingTelegramCommand` hiện chỉ kiểm tra `isApprove`; mọi lệnh khác như `/task`, `/tasks`, `/list`, hoặc callback `RUN_TASK <TaskKey>` đều bị từ chối với lỗi `telegram message does not contain APPROVE command`.

---

## 2. S2.5 Pre-Plan Adversarial Critique (`DG0` trên Local `:8200` — 0 LLM Token)
- **Claims File:** `06_laya_claims_dg0_real_repo_diff_queue_and_telegram_task_menu.json`
- **Summary File:** `06_laya_summary_dg0_20261023T165954.json`
- **Verdict:** `PASS` (`UNVALIDATED` trên Laya calibration — căn cứ trực tiếp bằng chứng code vật lý tại `models/trans-his/trans-his.model.ts:8-23`, `internal/webui/static/app.js:719-731`, `internal/webui/telegram_bridge.go:560-578`).
