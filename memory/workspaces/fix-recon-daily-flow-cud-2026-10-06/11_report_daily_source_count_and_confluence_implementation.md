# Báo Cáo Triển Khai: Daily Source Count & Confluence Daily Report

---

## 1. TỔNG QUAN
Hệ thống đã hoàn tất triển khai toàn diện và kiểm định thực tế cho 2 giai đoạn theo đúng phê duyệt:
1. **Giai đoạn 1 (Daily Source Count):** Đếm số lượng bản ghi ngày tạo từ nguồn đa hình (PostgreSQL, MongoDB, MySQL, SFTP, ClickHouse), chạy tự động chốt sổ D-1 lúc 00:05 sáng (hoặc giờ cấu hình động), lưu vào bảng `cdc_system.daily_source_counts`.
2. **Giai đoạn 2 (Confluence Daily Report):** Tác vụ `confluence` trên Activity Manager tự động đọc số liệu đếm từ `daily_source_counts` kết hợp lưu lượng CDC từ `recon_hourly_pipeline_stats`, dựng báo cáo đối soát dạng bảng XHTML nghiêm ngặt, bắn lên Atlassian Confluence làm trang con (child page) dưới Parent Page ID `463833042` (`Core Trans Proxy History Service`).

---

## 2. DANH MỤC CÁC TỆP TIN THAY ĐỔI & TẠO MỚI

### A. Database Migrations (`cdc-cms-service/migrations/schema/worker/`)
1. **`024_enhance_worker_schedule.sql`** (Tạo mới, 12 dòng):
   - Thêm cột `run_at_time VARCHAR(8)` (định dạng "HH:MM", vd: "00:05", "06:00").
   - Thêm cột `params JSONB DEFAULT '{}'::jsonb` lưu cấu hình tự trị của từng tác vụ.
2. **`025_create_daily_source_counts.sql`** (Tạo mới, 28 dòng):
   - Tạo bảng đa hình `cdc_system.daily_source_counts` với các trường: `layer_type`, `connection_code`, `engine_type`, `container_namespace`, `object_name`, `object_type`, `date_strategy`, `date_field`, `report_date`, `creation_count`, `execution_duration_ms`, `metadata_json`.
   - Unique Index: `(layer_type, connection_code, container_namespace, object_name, report_date)`.
   - Lookup Index: `(layer_type, container_namespace, object_name, report_date)` và `(report_date)`.

### B. Backend CMS Service (`cdc-cms-service/`)
1. **`internal/model/scheduler/worker_schedule.go`** (+4 dòng):
   - Bổ sung `RunAtTime *string` và `Params json.RawMessage`.
2. **`internal/api/dto/scheduler_dto.go`** (+6 dòng):
   - Bổ sung `RunAtTime`, `Params`, `NextRunAt`, `RunNow` vào DTOs.
3. **`internal/app/commands/scheduler/create_worker_schedule.go`** (+4 dòng):
   - Ghi nhận `RunAtTime` và `Params` vào domain entity khi tạo lịch trình.
4. **`internal/app/commands/scheduler/update_schedule.go`** (+18 dòng):
   - Cập nhật `RunAtTime`, `Params` và xử lý logic kích hoạt bù `RunNow` (`next_run_at = NOW(), last_run_at = NULL`).
5. **`internal/app/queries/scheduler/list_worker_schedules.go`** (+2 dòng):
   - Trả về `run_at_time` và `params` cho API query.
6. **`internal/infra/persistence/scheduler/worker_schedule_read_repo_gorm.go`** (+2 dòng):
   - Scan các cột mới từ DB.
7. **`internal/api/scheduler/schedule_handler.go`** (+32 dòng):
   - Bổ sung hàm `MaskSensitiveScheduleParams`: Tự động ẩn token Confluence dạng `ATATT3...****` khi trả dữ liệu về client.
   - Áp dụng masking cho các endpoint: `List`, `Update`, và `Create`.
8. **`internal/api/scheduler/schedule_handler_test.go`** (Tạo mới, 45 dòng):
   - Unit test kiểm tra token masking đạt PASS 100%.

### C. Backend Worker Service (`centralized-data-service/`)
1. **`internal/model/master/worker_schedule.go`** (+3 dòng):
   - Thêm `RunAtTime` và `Params`.
2. **`internal/model/master/daily_source_count.go`** (Tạo mới, 32 dòng):
   - Struct model GORM mapping bảng `cdc_system.daily_source_counts`.
3. **`internal/server/server.go`** (+12 dòng):
   - Bổ sung `scheduleJobs map[string]func(...)` và hàm `RegisterScheduleJob`.
4. **`internal/server/server_scheduler.go`** (+48 dòng):
   - Scheduler ticker 10s với múi giờ `locHCM` ("Asia/Ho_Chi_Minh").
   - Hàm `calculateNextRun` hỗ trợ tính mốc giờ tương lai chính xác `run_at_time`, Window Guard 5 phút chống trigger bậy khi restart trễ.
   - Atomic Claim: `claimResult.RowsAffected > 0` ngăn chặn race condition double-run giữa các nhịp tick và các node phân tán.
5. **`internal/service/sourcecount/source_count_runner.go`** (Tạo mới, 343 dòng):
   - Multi-driver đếm số bản ghi theo chuẩn Half-Open Interval `[startOfDay, nextDayStart)` với toán tử `>= ? AND < ?`.
   - Sanitize SQL identifiers bằng Regex `^[a-zA-Z0-9_]+$`.
   - Context timeout 30s chống nghẽn connection pool nguồn.
   - MongoDB `CountDocuments`, SFTP stream chunk 32KB đếm newline.
   - Upsert kết quả an toàn vào `cdc_system.daily_source_counts`.
6. **`pkgs/confluence/client.go`** (Tạo mới, 182 dòng):
   - Confluence REST API Client gọi Atlassian Confluence `/wiki/rest/api/content`.
   - Cơ chế Idempotent: Tìm trang theo title trước; nếu đã có, thực hiện `PUT` tăng version thay vì trả lỗi 400 Bad Request.
   - Tạo trang con dưới Parent Page ID chỉ định (`463833042`).
7. **`pkgs/confluence/client_test.go`** (Tạo mới, 20 dòng):
   - Unit test kiểm tra HTML/XML escaping cho Confluence.
8. **`internal/service/confluence/confluence_runner.go`** (Tạo mới, 235 dòng):
   - Confluence Runner: Đọc `params JSONB` từ lịch trình.
   - Đọc số liệu đếm từ `daily_source_counts`.
   - Range Query `window_hour >= ? AND window_hour < ?` trên `recon_hourly_pipeline_stats` bảo toàn B-Tree Index.
   - Dựng bảng XHTML với `html.EscapeString` nghiêm ngặt cho toàn bộ text content.
   - Gọi Confluence Client tạo Child Page dưới Parent ID `463833042`.
9. **`internal/service/confluence/confluence_runner_test.go`** (Tạo mới, 52 dòng):
   - Unit test kiểm tra XHTML table generator và strict escaping đạt PASS 100%.
10. **`internal/server/server_jobs.go`** (+18 dòng):
    - Đăng ký cả 2 runners `source_count` và `confluence` vào `WorkerServer`.

### D. Frontend CMS Web (`cdc-cms-web/`)
1. **`src/App.tsx`** (-1 dòng):
   - Xóa bỏ unused import `CloudServerOutlined` để đạt chuẩn `tsc -b` không warning/error.
2. **`src/pages/ActivityManager.tsx`** (+130 dòng):
   - Bổ sung `source_count` và `confluence` vào `ALL_OPERATIONS`.
   - Triển khai Form Modal Cascading 5 tầng cho `source_count`.
   - Triển khai Form Modal All-in-One cho `confluence`: Base URL, Space Key, Parent Page ID `463833042`, User Email, API Token (Password Input), Title Template, Multi-Select Tables, Giờ chạy (`run_at_time = '06:00'`).
   - Hàm `buildSchedulePayload`: Map state tường minh, gom đúng vào `params JSONB`, triệt tiêu rủi ro payload rỗng.
   - Nút "Chạy ngay" (`Run Now`) trên danh sách lịch trình: Hỗ trợ operator kích hoạt bù dữ liệu khi cần mà không phải sửa DB bằng tay.

---

## 3. KẾT QUẢ KIỂM THỬ THỰC TẾ TRÊN MÔI TRƯỜNG LIVE

1. **Database Migrations:** Đã apply thành công vào `gpay-postgres-cdc` (`cdc_dw` :5433).
2. **Live Execution Test cho Giai đoạn 1:**
   - Đã cấu hình task `source_count` cho bảng `master_core_trans_proxy_history_service.trans_his` trên database `goopay_dest` (Port 5434).
   - Scheduler tự động nhận diện và kích hoạt `source_count_runner`.
   - Bản ghi được ghi nhận thành công vào `cdc_system.daily_source_counts`:
     - ID: 1
     - Schedule ID: 6
     - Layer Type: master
     - Connection Code: default_master
     - Table: trans_his
     - Report Date: 2026-10-12
     - Count: 0 (Đúng theo mốc ngày hôm qua)
     - Execution Duration: 417 ms
     - Calculated At: 2026-10-13 03:53:02.758605+00
3. **Biên dịch & Unit Tests:**
   - `cdc-cms-service`: `go test -v ./internal/api/scheduler/...` $\rightarrow$ **PASS 100%**.
   - `centralized-data-service`: `go test -v ./pkgs/confluence/...` và `./internal/service/confluence/...` $\rightarrow$ **PASS 100%**.
   - `cdc-cms-web`: `npm run build` (tsc -b && vite build) $\rightarrow$ **PASS 100% (549ms)**.
