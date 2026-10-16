# BÁO CÁO NGHIỆM THU: KHẮC PHỤC LỖI RUN NOW CHỈ CHẠY SCHEDULE-UPDATE & RELATION DOES NOT EXIST (SQLSTATE 42P01)

## 1. Tóm tắt Bối cảnh & Hiện tượng (Incident Summary)
- **Triệu chứng 1:** Người dùng bấm "Chạy ngay" trên giao diện `/activity-manager`, nhưng trên `/activity-log` chỉ xuất hiện tác vụ `schedule-update`, không thấy tác vụ nghiệp vụ `source_count`.
- **Triệu chứng 2:** Tooltip lỗi trên cột `last_error` báo:
  `execute count sql: ERROR: relation "master_centrallized_export_service.export_jobs_2" does not exist (SQLSTATE 42P01)`.

---

## 2. Nguyên nhân Gốc rễ (Root Cause Analysis - RCA)

| STT | Triệu chứng | Nguyên nhân kỹ thuật gốc rễ | Vị trí mã nguồn |
| :--- | :--- | :--- | :--- |
| **1** | Bấm "Chạy ngay" chỉ thấy log `schedule-update` | `SourceCountRunner` và `ConfluenceReportRunner` hoàn toàn không được inject `ActivityLogger`. Chạy xong chỉ cập nhật `daily_source_counts` và `last_error`, không ghi vào `cdc_activity_log`. Trong khi CMS khi nhận `run_now=true` thì tự ghi 1 record `schedule-update`. | `server_jobs.go:191`, `source_count_runner.go:138`, `update_schedule.go:145` |
| **2** | Lỗi `relation does not exist` trên bảng `export_jobs_2` | Nhánh `case "master"` của `SourceCountRunner` bị hardcode dùng `r.dbMaster` (kết nối tới `default_master` / db `goopay_dest` trên port 5434). Trong khi bảng `export_jobs_2` thực tế nằm ở kết nối `master_2` (`goopay_master_2` trên port 5437). Worker truy vấn trên DB sai nên PostgreSQL báo lỗi 42P01. | `source_count_runner.go:110` |
| **3** | Phản hồi chậm khi bấm "Chạy ngay" | `time.NewTicker(10 * time.Second)` trong scheduler poller khiến worker có thể trễ tới 10 giây mới phát hiện cờ `run_now`. | `server_scheduler.go:59` |
| **4** | `triggered_by` bị ghi nhận thành `scheduler` | Worker scheduler cập nhật `next_run_at = nextRun` (mốc tương lai) trước khi bàn giao cho goroutine runner, khiến điều kiện `!sched.NextRunAt.After(now)` trong runner bị sai. | `server_scheduler.go:120` |

---

## 3. Chi tiết các File và Dòng Code đã Thay đổi

### File 1: `centralized-data-service/internal/activity/taxonomy.go`
- **Thay đổi:** Bổ sung hằng số `TriggeredByManual TriggeredBy = "manual"` vào taxonomy enum, danh sách `KnownTriggeredBy()` và hàm kiểm tra `IsKnownTriggeredBy()`.
- **Số dòng thay đổi:** +5 dòng.

### File 2: `centralized-data-service/internal/service/sourcecount/source_count_runner.go`
- **Thay đổi:**
  + Inject `connMgr *servicesource.ConnectionManager` và `activityLogger *governance.ActivityLogger` vào struct và constructor.
  + Đọc `triggered_by` từ `ctx.Value("triggered_by")`.
  + Tự động định tuyến kết nối động: `case "master"` gọi `connMgr.GetMasterDB(queryCtx, params.ConnectionCode)` thay vì hardcode `r.dbMaster`.
  + Gọi `activityLogger.Quick("source_count", targetName, triggeredBy, status, rows, meta, errMsg)` khi gặp lỗi và khi thành công.
- **Số dòng thay đổi:** ~45 dòng.

### File 3: `centralized-data-service/internal/service/confluence/confluence_runner.go`
- **Thay đổi:**
  + Inject `activityLogger *governance.ActivityLogger` vào struct và constructor.
  + Đọc `triggered_by` từ context.
  + Ghi `activityLogger.Quick` khi lỗi và khi tạo trang Confluence thành công.
- **Số dòng thay đổi:** ~35 dòng.

### File 4: `centralized-data-service/internal/server/server_jobs.go`
- **Thay đổi:**
  + Cập nhật chữ ký hàm `SetupSchedulerJobs` nhận thêm `connMgr *servicesource.ConnectionManager`.
  + Truyền `connMgr` và `activityLogger` vào `NewSourceCountRunner` và `NewConfluenceReportRunner`.
- **Số dòng thay đổi:** ~10 dòng.

### File 5: `centralized-data-service/internal/server/server_setup.go`
- **Thay đổi:** Truyền `connectionManager` vào hàm gọi `SetupSchedulerJobs`.
- **Số dòng thay đổi:** 1 dòng.

### File 6: `centralized-data-service/internal/server/server_scheduler.go`
- **Thay đổi:**
  + Giảm chu kỳ poller: `time.NewTicker(2 * time.Second)` (thay vì 10s).
  + Truyền `context.WithValue(ctx, "triggered_by", triggerType)` vào runner.
- **Số dòng thay đổi:** ~15 dòng.

### File 7: `cdc-cms-web/src/pages/ActivityManager.tsx`
- **Thay đổi:** Bổ sung cơ chế polling đa nhịp (1000ms, 2500ms, 4500ms) trong hàm `handleRunNow`.
- **Số dòng thay đổi:** ~8 dòng.

---

## 4. Bằng chứng Thực nghiệm Runtime (Verification G1–G8)

### (1) Kiểm tra trạng thái Schedule (`cdc_system.cdc_worker_schedule`)
```sql
SELECT id, operation, target_table, last_run_at, next_run_at, run_count, last_error 
FROM cdc_system.cdc_worker_schedule WHERE id = 7;
```
- **Kết quả:**
  + `id`: 7
  + `operation`: `source_count`
  + `target_table`: `export_jobs_2`
  + `last_run_at`: `2026-10-13 09:25:53.965445+00` (16:25:53 GMT+7)
  + `next_run_at`: `2026-10-13 17:05:00+00` (**00:05 sáng mai GMT+7, không bị drift**)
  + `run_count`: **12** (đã tăng)
  + `last_error`: **NULL (Đã sạch lỗi hoàn toàn)**

### (2) Kiểm tra bảng đếm nguồn (`cdc_system.daily_source_counts`)
```sql
SELECT id, schedule_id, layer_type, connection_code, container_namespace, object_name, creation_count, calculated_at, execution_duration_ms 
FROM cdc_system.daily_source_counts WHERE object_name = 'export_jobs_2';
```
- **Kết quả:**
  + `id`: 4
  + `schedule_id`: 7
  + `layer_type`: `master`
  + `connection_code`: `master_2`
  + `container_namespace`: `master_centrallized_export_service`
  + `object_name`: `export_jobs_2`
  + `creation_count`: 0 (đúng với số record ngày hôm qua trong `goopay_master_2`)
  + `execution_duration_ms`: **52ms**

### (3) Kiểm tra Nhật ký Tác vụ (`cdc_system.cdc_activity_log`)
```sql
SELECT id, operation, target_table, status, triggered_by, rows_affected, started_at 
FROM cdc_system.cdc_activity_log WHERE operation = 'source_count' ORDER BY id DESC LIMIT 2;
```
- **Kết quả:**
  + `id`: **112892**
  + `operation`: `source_count`
  + `target_table`: `export_jobs_2`
  + `status`: **success**
  + `triggered_by`: **manual**
  + `rows_affected`: 0
  + `started_at`: `2026-10-13 09:25:54.055296+00` (16:25:54 GMT+7)
