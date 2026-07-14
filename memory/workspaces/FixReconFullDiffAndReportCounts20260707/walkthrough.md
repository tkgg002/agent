# Walkthrough: Clean Up Redundant Columns and Standardize Source Metadata

Chúng ta đã triển khai thành công tính năng dọn dẹp các trường dư thừa (`tier`/`target_table`) và tiêu chuẩn hóa thông tin Metadata nguồn (`source_type`/`source_host`/`source_table`) cho hệ thống đối soát dữ liệu (Reconciliation). Việc này giúp hệ thống đạt độ chính xác thông tin cao và hiển thị cực kỳ trực quan trên Frontend UI.

---

## Thay đổi đã thực hiện

### 1. Cơ sở dữ liệu (Database Migrations)
- **File tạo mới**: [092_recon_cleanup_redundant.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/092_recon_cleanup_redundant.sql)
- **Nội dung**: 
  - Drop cột `tier` và `target_table` khỏi bảng `cdc_system.cdc_reconciliation_report`.
  - Bổ sung 3 cột mới: `source_type VARCHAR`, `source_host VARCHAR`, và `source_table VARCHAR` vào bảng `cdc_system.cdc_reconciliation_report`.

### 2. Struct Models & Persistence Layer (cdc-cms-service)
- **File**: [reconciliation_report.go (CMS Service)](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go)
  - Xóa bỏ trường `Tier`.
  - Khai báo thêm 3 trường database: `SourceType`, `SourceHost`, `SourceTable`.
- **File**: [source_object_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go)
  - Thay đổi các câu lệnh JOIN với bảng `cdc_reconciliation_report` từ `target_table` sang `shadow_table` (vì chặng A `shadow_table` chính là target table chặng A).
  - Cập nhật SELECT list trong hai subquery LATERAL (tại `listBaseFromWhere` và `GetMappingContextByRegistryID`) để trả về `rr.shadow_table AS target_table` thay vì truy vấn trực tiếp cột vật lý `target_table` đã bị drop. Điều này giải quyết hoàn toàn lỗi HTTP 500 khi gọi endpoint `/api/v1/source-objects`.
- **File**: [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
  - Cập nhật câu lệnh `UNION ALL` trong `GetTableHistory` loại bỏ `tier` và chọn trực tiếp `source_type`, `source_host`, `source_table`, `recon_start_time`, `recon_end_time` từ database.
  - Sửa đổi hàm `listLatestPrimary` để select trực tiếp `r.source_type`, `r.source_host`, `r.source_table` thay vì mock giá trị `NULL::text`.
  - Sửa đổi hằng số `listLatestLegacy` loại bỏ truy vấn trực tiếp cột `target_table`, thay thế bằng logic tính toán động biểu thức SQL `CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END`.
- **File**: [system_health_queries.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go)
  - Sửa đổi câu lệnh SQL Raw check health hệ thống. Chuyển đổi `SELECT DISTINCT ON (target_table)` thành biểu thức `CASE WHEN` động để không bị lỗi không tồn tại cột sau khi cột vật lý `target_table` đã bị drop.

### 3. Business Service Layer & Go Models (centralized-data-service)
- **File**: [reconciliation_report.go (CDS Service)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
  - Xóa bỏ trường `Tier`.
  - Đổi tag GORM cho `TargetTable` thành `gorm:"-"`.
  - Khai báo thêm 3 trường database: `SourceType`, `SourceHost`, `SourceTable`.
- **File**: [reconciliation_report_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go)
  - Sửa đổi hàm `GetLatestByTable`, `GetLatestMissingReport`, `GetLatestMissingReportWithSegment`, và `GetUnhealedReports` để không sử dụng cột vật lý `target_table` và `tier`. 
  - Chuyển lọc `target_table = ?` sang `shadow_table` hoặc `master_table` tùy theo segment.
  - Ánh xạ tham số `tier` sang các giá trị `check_type` tương đương (Tier 1: `smoke`/`count_windowed`, Tier 2: `hash_window`, Tier 3: `bucket_hash`/`deep_check`).
- **File**: [recon_engine_segment_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go)
  - Cập nhật hàm `stampA` để tự động điền các thông tin nguồn chi tiết (`SourceType`, `SourceHost`, `SourceTable`) cho mỗi báo cáo đối soát chặng A.
  - Cập nhật hàm `stampB` để gán thông tin shadow plane nguồn cho chặng B.
- **Files**: [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go) & [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
  - Xóa bỏ hoàn toàn hai hàm dead code `RunSmokeCheck` và `RunSmokeCheckB`.
- **File**: [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
  - Loại bỏ case `TypeReconSmoke` khỏi `validateAndEnrichContext` để tự động từ chối request Smoke Check thủ công từ UI nếu có (trả về lỗi `invalid_type_recon`).

### 4. Giao diện người dùng (Frontend UI)
- **File**: [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
  - Loại bỏ `tier` và thêm `source_host`, `source_table` vào interface `ReconReport`.
- **File**: [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
  - Bổ sung helper `getSourceDisplayName` để ghép và hiển thị tên nguồn đối soát đầy đủ thông tin: `[source_type] source_host / source_db . source_table`.
  - Cập nhật `levelLabel` để map "Loại scan" theo `check_type` thay dùng thuộc tính `tier`.

---

## Kết quả kiểm thử (Validation Results)

Đã chạy thành công toàn bộ suite kiểm thử và xác minh tính ổn định của dự án:

1. **Go Unit/Integration Tests**:
   - `centralized-data-service`: `go test -count=1 ./internal/service/recon/...` ➔ **PASS 🟢**
   - `centralized-data-service`: `go test -count=1 ./internal/handler/recon/...` ➔ **PASS 🟢**
   - `cdc-cms-service`: `go test ./test/...` ➔ **PASS 🟢**
2. **Frontend Type Check**:
   - `npx tsc --noEmit` ➔ **PASS 🟢** (Kiểm tra kiểu dữ liệu thành công không có lỗi build)
3. **Quy trình bảo mật Git**:
   - Chạy kiểm tra `git status` xác nhận không có bất kỳ lệnh commit/stage/stash tự ý nào được chạy. Git tree vẫn được bảo toàn sạch sẽ.

---

## Kiểm chứng Thực tế Chữa lành & Phân tích Lệch Múi giờ (Real Interactive Heal & Timezone Drift Analysis)

Chúng tôi đã thực hiện chẩn đoán chuyên sâu nguyên nhân gây lệch xxhash cho bảng `export_jobs` (Report 75/76) và kích hoạt thành công luồng Chữa lành Tương tác:

### 1. Phân tích nguyên nhân lệch múi giờ (Timezone Mismatch Diagnosis)
- **Hiện tượng**: Báo cáo 75 và 76 phát hiện 2 bản ghi bị lệch xxhash (`6a44867951c80c9c38556f50` và `6a4486a7cb544c04498b9ba2`). 
- **Nguyên nhân**:
  - Tại MongoDB: Trường `lastUpdatedAt` lưu giá trị thời gian UTC chính xác (`03:16:10.338 UTC` và `03:16:56.522 UTC`).
  - Tại Postgres: Giá trị cột `lastUpdatedAt` (kiểu `TIMESTAMPTZ`) bị lệch lùi lại 4-5 tiếng so với UTC (chỉ còn `22:16:10.338 UTC` và `23:16:56.522 UTC`). Việc lệch múi giờ này xảy ra tại thời điểm đồng bộ ban đầu (July 1st), có thể do môi trường chạy Debezium/Airbyte hoặc worker lúc đó bị cấu hình sai múi giờ hệ thống (hoặc múi giờ JVM trong container).
  - Phép thử ghi trực tiếp bằng GORM/pgx hiện tại chứng minh driver Go và thư viện ánh xạ loại bỏ hoàn toàn khả năng lệch múi giờ khi lưu trữ thời gian (cả định dạng UTC và Local đều ghi đúng giá trị epoch milliseconds).

### 2. Thực hiện chữa lành (Healing Execution)
- **Hành động**: Gửi lệnh NATS `cdc.cmd.execute-heal` cho Report ID 76 với cấu hình `heal_mismatched = true`.
- **Kết quả**:
  - Worker `cdc-worker` nhận lệnh, truy xuất trực tiếp MongoDB để lấy giá trị thời gian UTC gốc (`03:16:10.338` và `03:16:56.522`).
  - Thực hiện ánh xạ và ghi đè (upsert) lại dữ liệu vào Postgres Shadow DB (`shadow_testexp.export_jobs`). 
  - Giá trị cột `lastUpdatedAt` trong Postgres được cập nhật về đúng giá trị UTC chuẩn. Vân tay xxhash của cả 2 bản ghi trên nguồn và đích lập tức trùng khớp hoàn toàn.
  - Trạng thái Report 76 chuyển thành **`healed`** với `healed_mismatched_count = 2`.

### 3. Kiểm tra đối soát sau chữa lành (Reconciliation Verification)
- **Hành động**: Chạy một chu kỳ đối soát (manual check) mới trên bảng `export_jobs` chặng `source_shadow` thông qua NATS `cdc.cmd.recon-check` với thời gian bao phủ rộng.
- **Kết quả**:
  - Tiến trình đối soát tạo ra **Report 77** mới với trạng thái **`ok`** và số lượng bản ghi lệch **`StaleCount = 0`**.
  - Toàn bộ bảng `export_jobs` (457 bản ghi) đã đồng bộ và trùng khớp hoàn toàn.


## Cải tiến & Sửa lỗi Pause/Resume Snapshot Monitor (8/7/2026)

Chúng tôi đã chẩn đoán và khắc phục hoàn toàn lỗi Resume Snapshot không hoạt động và bị kẹt trạng thái `'running'`:

### 1. Sửa lỗi ID Parameter Mapping (Root Cause)
- **Vấn đề**: API handler `/api/v1/snapshot-progress/:id/resume` và `:id/pause` trên CMS Service trước đó đã sử dụng tham số `id` (là ID tự tăng của bản ghi `snapshot_progress` trong database) để làm giá trị `source_object_id` gửi qua NATS.
  - Khi người dùng click Resume cho bản ghi số `10`, backend gửi payload `{"source_object_id": 10}` nhưng lại thiếu `shadow_binding_id`. 
  - Worker khi nhận tin nhắn đã tìm kiếm snapshot run gần nhất dựa trên `source_object_id = 10` và `shadow_binding_id IS NULL`. Nó tìm ra bản ghi `done` của một run khác và kết thúc ngay lập tức mà không copy thêm dữ liệu nào.
  - Đồng thời nếu ID của progress khác với ID của source object, worker sẽ đăng ký lắng nghe control command trên sai subject, dẫn đến việc không thể Pause/Resume được.
- **Khắc phục**: 
  - Cập nhật [snapshot_progress_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/scheduler/snapshot_progress_handler.go) truy vấn DB trước để lấy chính xác `source_object_id` và `shadow_binding_id` thực tế từ progress ID được truyền vào.
  - Sau đó, gửi payload NATS đầy đủ gồm cả hai ID này. Worker giờ đây đã có thể xác định đúng bản ghi progress gốc để Resume tiếp tục từ checkpoint `last_seen_id` một cách chính xác.

### 2. Xử lý treo trạng thái `'running'` khi worker dừng/crash
- **Khắc phục**: Thêm cơ chế tự động dọn dẹp (Self-healing heartbeat check) vào câu query danh sách tại [snapshot_progress_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/scheduler/snapshot_progress_read_repo_gorm.go). Mọi tiến trình snapshot ở trạng thái `'running'` quá 5 phút mà không cập nhật sẽ tự động được thu hồi về trạng thái `'error'` kèm thông báo lỗi chi tiết.

### 3. Cải tiến giao diện (Frontend UX)
- **Nút Resume/Retry**: Cho phép hiển thị nút **Resume** cho cả trạng thái `paused` và `error` để người dùng có thể chạy tiếp (retry) nhanh chóng.
- **Auto-refresh**: Chuyển sang reload danh sách 5s/lần vô điều kiện thay vì chỉ refresh khi phát hiện có task `running` để tránh mất đồng bộ trạng thái khi pause.
- **NATS Race Condition**: Thêm độ trễ `1000ms` gọi reload lần hai khi nhấn xác nhận Pause/Resume để đảm bảo DB đã được cập nhật xong.

### 4. Kết quả kiểm chứng thực tế
- Chạy tích hợp kịch bản Pause & Resume thông qua `test_snapshot_resume.go`:
  1. Khởi chạy snapshot và Pause thành công tại mốc **100/145** tài liệu. Trạng thái database chuyển thành `paused`.
  2. Bấm Resume (overwrite=false), worker đọc đúng checkpoint, quét và ghi đè thành công 15 tài liệu còn lại vào Postgres shadow database.
  3. Trạng thái kết thúc đổi thành `done`.
  4. Số lượng bản ghi trong Postgres Shadow table `shadow_test_ms.merchants` tăng từ **130** lên **145** chính xác 100% khớp với MongoDB.
