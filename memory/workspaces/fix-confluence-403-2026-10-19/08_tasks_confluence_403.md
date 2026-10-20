# 08. Tasks Checklist — `fix-confluence-403-2026-10-19` (`confluence_403`)

## Task 1: [S2 Investigate] Truy vết mã nguồn, cấu hình DB và tái hiện lỗi HTTP 403 từ Confluence API
- **[Mô tả & Phạm vi]:**
  - Tìm và đọc mã nguồn xử lý Confluence report & search content (`check existing confluence page failed: search content returned status 403`) trong `data-hub/centralized-data-service`.
  - Truy vấn read-only DB (`worker_schedules`, `cdc_activity_log`) và kiểm tra biến môi trường / config của Confluence client.
  - Thực hiện lệnh kiểm tra kết nối thực tế (reproduce) để xem chi tiết response body / headers của mã lỗi `403`.
- **[Context & Logs]:**
  - Log lỗi: `06:00:02 19/10/2026 confluence cdc_transaction_history error - - scheduler - check existing confluence page failed: search content returned status 403`.
- **[Definition of Done]:**
  - Xác định chính xác `file:line` sinh ra lỗi, cấu hình job đang chạy lúc 06:00, và nguyên nhân gốc rễ (`VERIFIED`) khiến Confluence API trả về `403`.

## Task 2: [S3 Solution & DG1/DG2] Lập hồ sơ giải pháp `09_tasks_solution_confluence_403.md` và qua cổng `/laya_router` (DONE)
- **[Mô tả & Phạm vi]:**
  - Trình bày 1 phương án giải quyết triệt để kèm Code Demo trong `09_tasks_solution_confluence_403.md`.
  - Chạy `laya_gate.py run --stage dg1` và `--stage dg2`.
- **[Context & Logs]:**
  - Kết quả điều tra từ Task 1 (`06_laya_summary_dg1_20261019T133833.json`, `06_laya_summary_dg2_20261019T134022.json`).
- **[Definition of Done]:**
  - `DG1` và `DG2` đạt `PASS`, cập nhật `05_progress_confluence_403.md` và trình User chờ `APPROVE`.

## Task 3: [S4 Execute & S5 Audit — Chờ User APPROVE] Triển khai Fallback `findChildPageByTitle`, Retry 403 đầy đủ & Scheduled Retry trong ngày
- **[Mô tả & Phạm vi]:**
  - `centralized-data-service/pkgs/confluence/client.go`: Thực hiện đủ 4 lần retry cho `403` trong `doWithRetry`, bổ sung `findChildPageByTitle` fallback qua `ParentPageID` trong `CreateOrUpdateChildPage`.
  - `centralized-data-service/internal/service/confluence/confluence_runner.go` & `centralized-data-service/internal/server/server_scheduler.go`: Tự động lên lịch retry sau 15 phút (trong cửa sổ 2 giờ sau `run_at_time`) khi Daily Job (`confluence`) thất bại.
- **[Context & Logs]:**
  - `pkgs/confluence/client.go:82-151, 218-264`, `internal/service/confluence/confluence_runner.go:706-713`, `internal/server/server_scheduler.go:229-275`.
- **[Definition of Done]:**
  - Unit/Integration test PASS (`go test -v ./pkgs/confluence/... ./internal/service/confluence/... ./internal/server/...`), qua cổng S5 Adversarial Audit (`DG3`) và xuất `audit_report_confluence_403.md`.

