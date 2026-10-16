# PHÂN TÍCH GỐC RỄ & THIẾT KẾ GIẢI PHÁP: RUN NOW CHỈ CHẠY SCHEDULE-UPDATE & KHÔNG GHI ACTIVITY LOG

## 1. Hiện tượng & Triệu chứng (Trigger)
- Người dùng phản hồi trực diện:
  > *"có thêm lỗi. khi click chạy ngay, nó ko chạy sau khi đã update, click chạy ngay nó chỉ chạy schedule-update"*
- Khi người dùng bấm nút "Chạy ngay" trên trang `/activity-manager` của CMS đối với tác vụ `source_count` hoặc `confluence`:
  - Trong trang Lịch sử tác vụ (`/activity-log`), chỉ xuất hiện 1 dòng log: `schedule-update | export_jobs_2 | success | manual`.
  - Hoàn toàn KHÔNG xuất hiện log của chính tác vụ vừa bấm (`operation: source_count` hoặc `confluence`).
  - Lần chạy kế tiếp (`next_run_at`) hoặc `last_run_at` trên UI chậm cập nhật.

---

## 2. Truy vết Nguyên nhân Gốc rễ (Root Cause Analysis - RCA)

### Gốc rễ 1: Thiếu hoàn toàn ActivityLogger trong `SourceCountRunner` và `ConfluenceReportRunner`
- Trong `centralized-data-service/internal/server/server_jobs.go`:
  - Tất cả các tác vụ nền khác (`bridge`, `transform`, `field-scan`, `partition-check`, `reconcile`) đều gọi `activityLogger.Quick(op, table, triggeredBy, status, rows, meta, errStr)`.
  - DUY NHẤT 2 runner mới là `sourceCountRunner` và `confluenceRunner` được khởi tạo mà **KHÔNG ĐƯỢC INJECT `activityLogger`**.
  - Trong `source_count_runner.go`: Sau khi thực thi xong (dù thành công hay lỗi), code chỉ upsert vào bảng `cdc_system.daily_source_counts` và cập nhật cột `last_error` của bảng `cdc_worker_schedule`. Hoàn toàn không ghi 1 dòng nào vào `cdc_system.cdc_activity_log`!
  - Trong `confluence_runner.go`: Tương tự, sau khi tạo trang xong, chỉ cập nhật `notes` và `last_error`, không ghi vào `cdc_activity_log`.
  - Trong khi đó, ở phía CMS (`cdc-cms-service/internal/app/commands/scheduler/update_schedule.go:145`): Khi nhận lệnh PATCH `/api/worker-schedule/:id` (với payload `run_now: true`), CMS ghi 1 dòng log vào `cdc_activity_log` với `Operation: "schedule-update"`.
  - **Hậu quả trực quan:** Người dùng vào `/activity-log` chỉ thấy dòng `schedule-update`, không hề thấy `source_count` $\rightarrow$ Kết luận tất yếu: *"nó ko chạy, chỉ chạy schedule-update"*.

### Gốc rễ 2: Hardcode Master Database Pool (`r.dbMaster`) khiến `source_count` fail ngầm trên connection `master_2`
- Trong `source_count_runner.go:110`:
  ```go
  case "master":
      countResult, meta, countErr = r.countSQL(queryCtx, r.dbMaster, params, startOfDay, nextDayStart)
  ```
- `r.dbMaster` trong `server_jobs.go` được truyền từ `destDB` (tức `RoleDestination` - kết nối default_master trên port 5434 / db `goopay_dest`).
- Trong khi đó, bảng cấu hình `export_jobs_2` thực tế nằm ở connection `master_2` (`goopay_master_2` trên port 5437)!
- Khi worker chạy, nó thực thi query `SELECT COUNT(1) FROM "master_centrallized_export_service"."export_jobs_2"` trên `r.dbMaster` $\rightarrow$ PostgreSQL bắn lỗi:
  `ERROR: relation "master_centrallized_export_service.export_jobs_2" does not exist (SQLSTATE 42P01)`.
- Worker ghi lỗi này vào `last_error` của `cdc_worker_schedule` rồi dừng (`return`), không có Activity Log báo lỗi cho người dùng.

### Gốc rễ 3: Ticker Poller 10 Giây & Khoảng cách Polling của Frontend
- Trong `server_scheduler.go:59`:
  `ticker := time.NewTicker(10 * time.Second)`
  Khi người dùng click "Chạy ngay", worker có thể mất tới 10 giây mới tick đến chu kỳ quét DB.
- Trong khi đó, `ActivityManager.tsx:516` chỉ `setTimeout(() => fetchData(), 2500)`. Sau 2.5s, frontend fetch lại nhưng worker vẫn chưa kịp claim và chạy task.

---

## 3. Kiến trúc Giải pháp Tối ưu (The Single Best Approach)

1. **Inject `ConnectionManager` vào `SourceCountRunner`:**
   - Sử dụng `connMgr.GetMasterDB(ctx, params.ConnectionCode)` cho Layer `master` để tự động kết nối đúng DB instance (`master_2`, `postgres-develop`, `default_master`).
   - Sử dụng `connMgr.GetShadowDB(ctx, params.ConnectionCode)` cho Layer `shadow`.
   - Dynamic resolve cho Layer `source` qua `connection_registry`.

2. **Inject `ActivityLogger` vào cả `SourceCountRunner` và `ConfluenceReportRunner`:**
   - Ghi Activity Log đầy đủ cả khi thành công lẫn thất bại (`status: "success" | "error"`).
   - Tự động nhận diện `triggeredBy`: Nếu chạy do người dùng bấm chạy ngay $\rightarrow$ `"manual"`, nếu chạy theo lịch $\rightarrow$ `"scheduler"`.

3. **Tối ưu Ticker Scheduler xuống 2 Giây:**
   - Giảm từ 10s xuống 2s để worker phản hồi gần như tức thì khi có request Run Now.
   - Thêm nhịp polling thông minh trên UI frontend (1s, 2.5s, 4.5s).
