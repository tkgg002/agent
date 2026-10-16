# DANH SÁCH NHIỆM VỤ: KHẮC PHỤC TRIỆT ĐỂ LỖI RUN NOW CHỈ CHẠY SCHEDULE-UPDATE

- [ ] **Task 1: Cập nhật `SourceCountRunner` trong `centralized-data-service`**
  - Inject `ActivityLogger` và `ConnectionManager` vào `SourceCountRunner`.
  - Thay thế hardcoded `r.dbMaster` bằng `r.connMgr.GetMasterDB(ctx, params.ConnectionCode)`.
  - Thay thế hardcoded `r.dbShadow` bằng `r.connMgr.GetShadowDB(ctx, params.ConnectionCode)`.
  - Ghi nhận `activityLogger.Quick` khi `source_count` thành công hoặc gặp lỗi.
  - Phân loại chính xác `triggered_by`: `"manual"` (Run Now) hoặc `"scheduler"`.

- [ ] **Task 2: Cập nhật `ConfluenceReportRunner` trong `centralized-data-service`**
  - Inject `ActivityLogger` vào `ConfluenceReportRunner`.
  - Ghi nhận `activityLogger.Quick` khi Confluence report thành công hoặc lỗi.

- [ ] **Task 3: Cập nhật `SetupSchedulerJobs` & `server_setup.go`**
  - Truyền `connectionManager` và `activityLogger` vào các runner trong `server_jobs.go`.
  - Giảm `ticker` trong `server_scheduler.go` xuống `2 * time.Second`.

- [ ] **Task 4: Tinh chỉnh Frontend Polling trong `cdc-cms-web`**
  - Bổ sung polling nhiều nhịp (1s, 2.5s, 4.5s) trong `handleRunNow`.

- [ ] **Task 5: Compile, Restart Services & Verification G1–G8**
  - Build binary mới cho worker và CMS.
  - Test Run Now trên schedule id=7 (`export_jobs_2` trên `master_2`).
  - Kiểm tra database: bảng `cdc_system.daily_source_counts`, `cdc_system.cdc_activity_log`, `cdc_system.cdc_worker_schedule`.
