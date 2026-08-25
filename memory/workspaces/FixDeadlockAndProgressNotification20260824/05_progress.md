# 05_progress.md — Audit Log (Append Only)

## [2026-08-24T11:31:00+07:00] [Agent:Brain] KHỞI TẠO WORKSPACE
- User báo lỗi deadlock `SQLSTATE 40P01` tại `master.bulk_upsert: master_payment_service.payments` trong cdc-worker
- Yêu cầu thêm: (2) thông báo đang chạy từ pipeline, (3) fix bug không hiện progress khi sync now & transform, (4) thêm traceID trên trang Transmute Schedules
- Root cause deadlock đã xác định: `concurrencyLimit=10` trong `getOrCreateDebouncer()` tạo 10 goroutine cùng upsert 1 table → deadlock chéo
- Đang lập implementation plan và workspace documents
- **Status: IN PROGRESS — Plan phase**

## [2026-08-24T14:45:00+07:00] [Agent:Muscle] HOÀN THIỆN ĐẦY ĐỦ CÁC CỘT /schedules, DEADLOCK 40P01 VÀ PROGRESS TRACKING
- **Migration 023:** `023_add_last_trace_id_to_transmute_schedule.sql` — thêm cột `last_trace_id VARCHAR(64)` vào `cdc_system.transmute_schedule`.
- **Backend Model & Worker:**
  - `centralized-data-service/internal/model/master/transmute_schedule.go`: Thêm field `LastTraceID`.
  - `centralized-data-service/internal/handler/master/transmute_handler.go`: Giảm `concurrencyLimit` từ 10 xuống 1 để ngăn chặn deadlock 40P01 trên cùng 1 master table; cập nhật `publishCompleted` truyền `traceID` vào event `cdc.evt.transmute.completed`.
  - `centralized-data-service/internal/service/master/job_monitor.go`: Nhận `TraceID` từ payload event và cập nhật vào `last_trace_id` trong DB.
- **Backend CMS (Read Query):**
  - `cdc-cms-service/internal/app/queries/scheduler/list_transmute_schedules.go`: Thêm `LastTraceID` và `LastSeenID` vào `TransmuteScheduleRow`.
  - `cdc-cms-service/internal/infra/persistence/scheduler/transmute_schedule_read_repo_gorm.go`: Bổ sung SQL JOIN với `cdc_system.sync_runtime_state` để lấy `last_seen_id` (`last_gpay_id`) và `last_trace_id`.
## [2026-08-24T15:26:00+07:00] [Agent:Muscle] FIX SCHEMA ISOLATION CHO SCHEDULE TRONG MASTER REGISTRY
## [2026-08-24T15:32:00+07:00] [Agent:Muscle] FIX TOÀN BỘ CỘT /schedules HIỂN THỊ GẠCH NGANG (—)
- **Nguyên nhân:**
  1. `Last Seen ID` bị gạch ngang: Trong `transmute_schedule_read_repo_gorm.go`, SQL query JOIN `sync_runtime_state` với điều kiện `srs.runtime_scope = 'transmute'`. Tuy nhiên trong bảng `sync_runtime_state`, `runtime_scope` cho master binding được lưu là `'master'` (check constraint: `'source'|'shadow'|'master'`). Vì vậy câu JOIN luôn trả về NULL cho `last_seen_id`.
  2. `Trace ID` và `Last run` chưa hiện ngay khi bấm: Trong `run_now.go`, `UpdateStatusRunning` chỉ cập nhật `last_status='running'` mà không gán `last_trace_id` ngay lập tức. Nếu OTel span context chưa active, `traceID` bị rỗng.
- **Fix:**
  - `transmute_schedule_read_repo_gorm.go`: Sửa `srs.runtime_scope = 'master'` để JOIN lấy đúng `last_seen_id` từ `sync_runtime_state`.
  - `run_now.go` & `transmute_schedule_repository_gorm.go`: Đảm bảo `traceID` luôn được sinh 32-hex UUID và cập nhật ngay vào `last_trace_id` trong `UpdateStatusRunning`, đồng thời truyền qua NATS `TransmuteRunCommand` (`cdc.cmd.transmute`) sang worker.
  - `transmute_handler.go`: Tiếp nhận `req.TraceID` và bắn vào `cdc.evt.transmute.completed` để `JobMonitor` cập nhật sau khi hoàn tất.
- **Build Verification:**
  - `centralized-data-service`: `go build` -> OK (Exit 0)
## [2026-08-24T15:43:00+07:00] [Agent:Brain] MID-SESSION FIX — KỶ LUẬT BỘ BA ĐỊNH DANH (DB, SCHEMA, TABLE)
- **User feedback:** Phản ánh việc liên tục xảy ra lỗi do bỏ quên kiểm tra bộ ba định danh `(db/connection, schema, table)` trên 2 tầng `Source → Shadow` và `Shadow → Master`, vội vàng sửa nhanh mà không đối chiếu DDL DB thực tế.
- **Thực thi quy tắc Mid-Session Fix (Rule #5):**
  1. Dừng ngay toàn bộ hành động sửa mò.
  2. Ghi nhận bài học kinh nghiệm nghiêm túc vào `agent/memory/global/lessons.md`: `[2026-08-24] Bỏ quên bộ ba định danh Metadata (DB/Connection, Schema, Table) trên 2 tầng Source→Shadow và Shadow→Master gây lỗi gãy cách ly dữ liệu và gãy query`.
## [2026-08-24T16:01:00+07:00] [Agent:Muscle] FIX LỖI SQLSTATE 42703 (COLUMN _id DOES NOT EXIST) VÀ LƯU TRẠNG THÁI PROGRESS KHI F5
- **Nguyên nhân gốc rễ:**
  1. `ERROR: column "_id" of relation "bank_requests" does not exist (SQLSTATE 42703)`: Trong `transmuter.go` (line 958), `conflictTarget` bị ghi đè thành `pkCol` (`_id`) từ `transform_spec` mà không kiểm tra xem bảng master đích có cột `_id` hay unique constraint trên `_id` không. Mọi bảng master trong hệ thống đều có PK là `_gpay_id` (`"_gpay_id" BIGINT PRIMARY KEY`). Khi câu `INSERT ... ON CONFLICT ("_id")` chạy, PostgreSQL văng lỗi 42703 làm job crash.
  2. Bấm "Sync ngay" không lưu progress sau khi F5: Do job bị crash bởi lỗi 42703 ở trên khiến `transmute_jobs` lưu `status = 'FAILED'`. Đồng thời trước đó câu `LEFT JOIN LATERAL transmute_jobs` trong `master_read_repo_gorm.go` chỉ tìm theo short name `WHERE tj.master_table = mb.master_table`, bị miss các job lưu dưới dạng FQN.
- **Biện pháp xử lý:**
  - `centralized-data-service/internal/service/master/transmuter.go`: Giữ `conflictTarget = "_gpay_id"` cố định theo đúng DDL Primary Key của bảng master đích, loại bỏ việc ép `conflictTarget = "_id"`.
  - `centralized-data-service/internal/service/master/job_monitor.go`: Dùng `COALESCE(NULLIF(?, ''), last_trace_id)` để bảo toàn `last_trace_id` trong DB khi event không có trace ID mới.
  - `cdc-cms-service/internal/infra/persistence/master/master_read_repo_gorm.go`: LATERAL JOIN `transmute_jobs` đối soát cả 2 dạng: schema-qualified FQN (`master_bidv_connector_service.bank_requests`) và short table name (`bank_requests`).
## [2026-08-24T17:10:00+07:00] [Agent:Brain] ĐIỀU CHỈNH KẾ HOẠCH: TUYỆT ĐỐI KHÔNG CHẠM CONFIG HẠ TẦNG CỦA USER
- **Nhận định & Kỷ luật:**
  - File cấu hình hạ tầng (`config-local.yml`, `.env`...) thuộc quyền quản trị môi trường testing của User. Agent tuyệt đối không được tự ý sửa hay đề xuất ghi đè.
  - Tập trung 100% vào việc sửa mã nguồn Golang phòng thủ:
    1. `dlq_helper.go`: Xử lý an toàn nhị phân / UTF-8 cho `msg.Key` (Avro) để triệt tiêu lỗi PostgreSQL `SQLSTATE 22021 (invalid byte sequence for encoding "UTF8": 0x9f)`.
    2. `avro_helper.go`: Thêm guard thông báo lỗi cấu hình tường minh nếu `SchemaRegistryURL` chưa được cung cấp.
    3. `topic_helper.go`: Phân định chuẩn xác luồng Debezium CDC (`ParseDebeziumTopic`) và luồng SFTP (`ExtractSFTPTable`).
    4. `metadata_registry_utils.go`: Chuẩn hóa `buildRouteLookupKeys` với các biến thể `-` ↔ `_`.
- **Trạng thái Artifact:** Đã cập nhật [implementation_plan.md](file:///Users/trainguyen/.gemini/antigravity/brain/87f6c137-a2ad-4433-b6fb-32f40936b0d4/implementation_plan.md) chờ phê duyệt.

## [2026-08-24T17:15:00+07:00] [Agent:Muscle] CẬP NHẬT KIỂM TRA SFTP Ở VỊ TRÍ THỨ 2 (PARTS[1])
- **File thay đổi:** [topic_helper.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/topic_helper.go)
- **Nội dung thực hiện:**
  - `parts := strings.Split(topic, ".")`
  - Nếu `len(parts) >= 2 && strings.Contains(parts[1], "sftp")`: `tableName = ExtractSFTPTable(topic)`
  - Ngược lại nếu `len(parts) >= 4`: `tableName = parts[len(parts)-1]`
- **Kết quả xác minh:** Lệnh `go build ./internal/... ./cmd/...` chạy thành công (Exit code 0).

