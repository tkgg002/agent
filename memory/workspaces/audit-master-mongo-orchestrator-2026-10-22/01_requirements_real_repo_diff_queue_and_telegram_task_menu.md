---
tier: T2
suffix: real_repo_diff_queue_and_telegram_task_menu
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T16:52:30+07:00
---

# 01 Requirements — Thực thi Code Diff Thật trên Repo Đích (`centralized-export-service`), Quản lý "Hàng Đợi Task (Theo Repo)" & Lệnh `/task` Chọn Chạy Task trên Telegram

## 1. Yêu cầu Nguyên văn từ User
> "đã thấy báo done, nhưng trong repo export của tao, ko thấy code diff, thứ 2 là Hàng Đợi Task (Theo Repo) chưa đc quản lý, ko biết cái nào chạy hay ko. lên cho tao 1 event ở tele, nếu tao /task thì nó sẽ list ra list task hiện tại, tao cllick vào cái nào thì nó sẽ chạy"

## 2. Phân rã 3 Mục tiêu Cốt lõi (Scope & Acceptance Criteria)

### Mục tiêu 1: S4 Thực thi Code Diff Thật trên Repo Đích (`centralized-export-service` cho `TASK-13`) — Tuyệt đối không báo `DONE` bằng `diff.patch` giả khi repo đích trống trơn
- **Hiện trạng & Nguyên nhân:**
  - `TASK-13` (*"con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret."*) nhắm tới repo `/Users/trainguyen/Documents/work/centralized-export-service`.
  - Ở phiên trước, hàm `synthesizeIsolatedRemediationDiffIfApproved` (`internal/webui/telegram_bridge.go:1016-1109`) chỉ ghi vài dòng comment giả (`// Isolated Remediation Hunk...`) vào `.orchestrator/tasks/TASK-13/diff.patch` mà không hề sửa code thực tế trong `/Users/trainguyen/Documents/work/centralized-export-service`, dẫn đến `git diff` trong `/Users/trainguyen/Documents/work/centralized-export-service` bằng `0` dòng!
  - Hơn nữa, bộ phân rã Sub-Task (`inspectTargetRepoSymbols` / `DecomposeMultiRepoCodebaseSubTasks` tại `internal/webui/task_inspector.go`) khi quét `centralized-export-service` đã chọn nhầm `consent-policy-export.params.ts` thay vì đúng các file `trans_his` đang dùng `@InjectModel(TransHis.name, 'trans_his')` (`mongoose`) cần chuyển sang PostgreSQL (`TypeORM` / `pg` / `@InjectRepository` hoặc `DataSource`).
- **Yêu cầu chuẩn xác:**
  1. Khảo sát thực tế cấu trúc `/Users/trainguyen/Documents/work/centralized-export-service` (các file `trans-his` / `trans_his`, module Mongo vs Postgres hiện có trong repo) để xác định chính xác từng `file:line` của `trans_his`.
  2. Khi một Task được User `APPROVE` và chạy `S4 Implement` trên môi trường thật (`!s.isTestTaskOrTempRoot`), hệ thống phải thực thi thay đổi code thật trên repo đích (`TargetRepo` / `AffectedRepos`) tạo ra **`git diff` thực tế trong repo đích** (và đồng bộ bản chụp `git diff` thật đó vào `.orchestrator/tasks/<TaskKey>/diff.patch` để hiển thị trên UI), đồng thời `VerifySubTaskCodeOnDisk` kiểm chứng `git status --porcelain` / `git diff` thật trên repo đích.

### Mục tiêu 2: Quản lý "Hàng Đợi Task (Theo Repo)" trên Giao diện WebUI (`:8090`)
- **Hiện trạng:**
  - Khung **🗂️ Hàng Đợi Task (Theo Repo)** ở cột trái (`internal/webui/static/index.html` & `internal/webui/static/app.js`) hiện chỉ hiển thị danh sách tĩnh, chưa quản lý rõ trạng thái hàng đợi: không nhìn thấy ngay task nào **đang chạy (`⚡ ĐANG CHẠY`)**, task nào **đang chờ duyệt (`⏸ CHỜ APPROVE`)**, task nào **đang xếp hàng chờ chạy (`⏳ HÀNG ĐỢI #1, #2...`)**, hay task nào **đã xong (`✓ ĐÃ XONG`)**, cũng như không hiển thị **số file / dòng `git diff` thật trên repo** và thiếu nút bấm **▶ Chạy / ✓ Duyệt & Chạy / ⏹ Dừng** ngay trên từng dòng task trong Hàng Đợi.
- **Yêu cầu chuẩn xác:**
  1. Nâng cấp **Hàng Đợi Task (Theo Repo)** (`server.go`, `index.html`, `app.js`, `app.css`) để quản lý hàng đợi thực sự:
     - Hiển thị thanh tóm tắt trạng thái Hàng Đợi ở đầu khung: `⚡ X Đang chạy · ⏸ Y Chờ duyệt · ⏳ Z Hàng đợi · ✓ W Đã xong`.
     - Sắp xếp và gắn nhãn thứ tự ưu tiên rõ ràng trong từng nhóm Repo (`⚡ ĐANG CHẠY` lên đầu $\rightarrow$ `⏸ CHỜ APPROVE` $\rightarrow$ `⏳ HÀNG ĐỢI #1, #2...` $\rightarrow$ `✓ ĐÃ XONG`), kèm bộ lọc nhanh (`Tất cả`, `Đang chạy / Chờ`, `Đã xong`).
     - Mỗi thẻ task trong Hàng Đợi hiển thị: Trạng thái chạy trực quan (có chấm xung nhịp `pulse` khi đang chạy), Tiến độ Sub-Tasks (`done/total`), Trạng thái `git diff` thật trên repo (`+X/-Y lines` hoặc `N files modified`), và nút hành động trực tiếp ngay trên thẻ (`▶ Chạy Plan`, `✓ Duyệt & Chạy`, `⏹ Dừng`) mà không cần chuyển màn hình.

### Mục tiêu 3: Sự kiện `/task` trên Telegram — Liệt kê danh sách Task hiện tại & Click vào Task nào là Chạy ngay Task đó
- **Hiện trạng:**
  - Trong `internal/webui/telegram_bridge.go` (`processIncomingTelegramCommand:560-578`), mọi tin nhắn đến nếu không chứa từ khóa `approve` / `duyệt` đều bị từ chối (`ignored — not an APPROVE command`).
- **Yêu cầu chuẩn xác:**
  1. Khi anh gõ `/task` (hoặc `/tasks`, `/list`, `task`) trên Telegram group/chat:
     - Bot lập tức phản hồi một bản tin **📋 DANH SÁCH HÀNG ĐỢI TASK HIỆN TẠI (THEO REPO)** liệt kê rõ từng Task (`TaskKey`, `Title`, `Repo`, `Stage` hiện tại: `⚡ RUNNING`, `⏸ NEEDS_YOU`, `⏳ READY`, `✓ DONE`, tiến độ Sub-Tasks).
     - Đính kèm **Inline Keyboard Buttons (`inline_keyboard`)** cho từng Task (ví dụ: `▶️ Chạy TASK-13 (centralized-export-service · NEEDS_YOU)`, `▶️ Chạy TASK-12 (...)`, ...).
  2. Khi anh **click vào bất kỳ nút Task nào** trên Telegram (`callback_data: "RUN_TASK <TaskKey>"`):
     - Nếu Task đang ở `RAW` / `PRIORITIZED` / `READY` / `PLANNING`: tự động kích hoạt chạy `Phase 1 (S0 -> DG0 -> S3 Plan)` và gửi bản thiết kế S3 + nút `✅ APPROVE` về Telegram.
     - Nếu Task đang ở `NEEDS_YOU` (đã có Plan S3) hoặc `DONE` (muốn chạy lại): tự động kích hoạt `approve_and_run` (`S4 Implement trên repo thật -> S5 Audit -> DG3 -> DONE`) và gửi thông báo kết quả + `git diff --stat` thật về Telegram!
