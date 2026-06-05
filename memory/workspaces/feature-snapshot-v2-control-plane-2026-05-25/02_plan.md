# Kế Hoạch Triển Khai Chi Tiết: Snapshot V2 Control Plane

Tài liệu này là bản đặc tả kỹ thuật (Technical Specification) đầy đủ dành cho Muscle (Chief Engineer) để thực thi. Mọi chỉ đạo tại đây là bắt buộc nhằm đáp ứng chuẩn Enterprise cho luồng dữ liệu cực lớn.

---

## 1. Cơ chế Kiểm soát Tải (Dynamic Flow Control & Throttling)

### Thay đổi Database & Model
**File tác động**: `centralized-data-service/internal/model/source_object_registry.go` (hoặc tương đương) và schema SQL.
1. **Schema `cdc_system.snapshot_progress`**: 
   - Không cần thêm field mới cho status nếu field status đang là string, chỉ cần thống nhất sử dụng value `paused` tại luồng update.
2. **Schema `cdc_system.source_object_registry`**: 
   - Thêm cột `max_rps` (kiểu INT, nullable).
   - Thêm cột `snapshot_error_mode` (kiểu VARCHAR, nullable, chứa `strict` hoặc `lenient`).
   - Cập nhật struct GORM tương ứng.

### Logic Xử lý (`snapshot_runner_handler.go`)
1. **Cơ chế Event-Driven Pause (Zero-cost check)**:
   - Worker KHÔNG ĐƯỢC poll DB mỗi batch.
   - Khi khởi tạo luồng chạy (`runSnapshot`), tạo thêm 1 NATS Subscriber nội bộ hoặc dùng context lắng nghe NATS Subject: `cdc.control.snapshot.pause`.
   - **Mệnh lệnh kỹ thuật**: Sử dụng `isPaused := &atomic.Bool{}` để lưu cờ trạng thái. Khi nhận event từ NATS trùng `source_object_id`, gọi `isPaused.Store(true)`.
   - Cuối mỗi batch của Cursor, đọc `if isPaused.Load() { ... }`. Nếu `true`, flush checkpoint hiện tại (`last_seen_id`) xuống DB với status = `paused` và `break` vòng lặp.
   
2. **Rate Limiting (MaxRPS)**:
   - Đọc cấu hình `MaxRPS` từ `SourceObject`. Tính toán thời gian cần delay giữa các batch bằng Token Bucket đơn giản: `thời_gian_đã_chạy_của_batch` so với `kỳ_vọng_dựa_trên_MaxRPS`. Nếu chạy quá nhanh, dùng `time.Sleep`.

---

## 2. Resiliency (Bền vững) & Tiến độ (Progress)

### Thay đổi Database
**File tác động**: Script Migration SQL mới.
- Thêm cột `total_rows BIGINT` vào `cdc_system.snapshot_progress`.

### Logic Xử lý (`snapshot_runner_handler.go`)
1. **Tính tổng tiến độ**:
   - Trước vòng lặp Cursor, gọi hàm `EstimatedDocumentCount(ctx)` của MongoDB. (Tuyệt đối **không** dùng `CountDocuments()` vì quét full collection gây treo DB).
   - Lưu kết quả vào biến `total_rows` và update xuống bảng `snapshot_progress`.
2. **Persistent Checkpoint**:
   - Chỉ cập nhật `last_seen_id` và `rows_processed` sau khi toàn bộ event trong batch đã được Event Handler ACK thành công (Đã có logic `checkpoint`, chỉ cần giữ nguyên vị trí gọi hiện hành).
3. **Phân biệt Full Re-snapshot vs Resume**:
   - Khi NATS trigger lệnh `snapshot.v2`, nếu command quy định chạy từ đầu (Full Re-snapshot), bỏ qua việc tìm kiếm `last_seen_id` cũ, và tạo row progress mới với `last_seen_id = NULL`.

---

## 3. Fail-Safe (An toàn khi có lỗi & DLQ)

### Thay đổi Database
**File tác động**: Script Migration SQL mới và Model GORM.
1. Tạo bảng `cdc_system.snapshot_dlq`:
   - `id BIGSERIAL PRIMARY KEY`
   - `progress_id BIGINT`
   - `source_object_id BIGINT`
   - `document_id VARCHAR(255)`
   - `payload JSONB`
   - `error_msg TEXT`
   - `created_at TIMESTAMP DEFAULT NOW()`

### Logic Xử lý (`snapshot_runner_handler.go`)
1. **Phân cấp Quản trị Rủi ro (Strict vs Lenient)**:
   - Đọc giá trị `snapshot_error_mode` (lenient / strict) của `SourceObject`.
   - Bọc logic parse BSON và `eventHandler.HandleRaw` trong khối bắt lỗi (của từng document).
2. **Strict Mode (Bảng Core)**:
   - Khi có 1 document lỗi, log lỗi, lập tức update `snapshot_progress` thành `error` và `return` (thoát vòng lặp, dừng tiến trình).
3. **Lenient Mode (Bảng Log/Audit)**:
   - Khi document lỗi, append document lỗi vào một `[]model.SnapshotDLQ`.
   - **Mệnh lệnh kỹ thuật (Bulk-Insert)**: Ở cuối batch, nếu slice DLQ có dữ liệu, thực hiện 1 câu lệnh GORM `db.CreateInBatches(dlqRecords, len(dlqRecords))` duy nhất. Cấm gọi `INSERT` từng dòng lẻ tẻ làm sập I/O throughput của batch (tránh phốt Docker Mac/K8s).

---

## 4. Data Integrity (LWW Guard Nâng cao)

### Logic Xử lý (`upsert.go` hoặc tương đương nơi sinh câu SQL cho Snapshot)
1. **Thay đổi Logic Ghi đè**:
   - Hàm `buildUpsertSQLSnapshot` hiện tại dùng `ON CONFLICT DO NOTHING`. Bắt buộc phải thay bằng `ON CONFLICT (...) DO UPDATE SET ...`.
2. **SQL Tie-Breaker Mệnh lệnh**:
   - Điều kiện cập nhật (WHERE clause của UPDATE) phải bao gồm 3 lớp bảo vệ:
     ```sql
     WHERE qt._source_ts IS NULL 
        OR EXCLUDED._source_ts > qt._source_ts 
        OR (EXCLUDED._source_ts = qt._source_ts AND qt._source = 'snapshot:v2')
     ```
   - **Phân tích kỹ thuật**:
     - `IS NULL`: Bảo vệ tập data cũ chưa có cột TS.
     - `EXCLUDED._source_ts > qt._source_ts`: Đảm bảo CDC luôn thắng Snapshot (vì CDC real-time sinh ra sau khi Snapshot đã chốt $clusterTime).
     - `= ... AND qt._source = 'snapshot:v2'`: Edge-case Re-snapshot 2 lần liên tiếp trùng khít TS, cho phép đè.
3. **Bảo vệ Cột Immutable**:
   - Các cột `_gpay_id`, `_gpay_source_id`, `_created_at` tuyệt đối không xuất hiện trong mệnh đề `SET`.
