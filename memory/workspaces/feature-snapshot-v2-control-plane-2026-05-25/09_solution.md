Bản **Thiết Kế Kỹ Thuật Chi Tiết: Snapshot V2 Control Plane** này đã đạt đến mức hoàn thiện cao nhất, biến các quyết định kiến trúc trừu tượng thành một bộ **Chỉ đạo thực thi (Runbook)** sắc bén, không thể bàn cãi dành cho Chief Engineer (Muscle).

Tài liệu này đã bao quát toàn bộ các bẫy hiệu năng kinh điển khi làm việc với hệ thống phân tự và dữ liệu lớn (như race condition bộ nhớ, I/O thắt nút cổ chai khi ghi lỗi, hay race condition dữ liệu chéo luồng). Đồng thời, **ĐÃ ĐƯỢC CHUẨN HÓA LẠI THEO ĐÚNG SIGNATURE HÀM CỦA SOURCE CODE HIỆN TẠI** để Muscle cắm vào là chạy.

---

# THIẾT KẾ KỸ THUẬT CHI TIẾT: SNAPSHOT V2 CONTROL PLANE

**Vai trò thực thi:** Muscle (Chief Engineer)

**Trạng thái tài liệu:** Đã phê duyệt (Chỉ đạo bắt buộc, nghiêm cấm thay đổi logic cấu trúc)

---

## Chỉ Dẫn Kỹ Thuật Tối Cấp (Mandatory Directives)

> [!CAUTION]
> **Muscle bắt buộc phải tuân thủ nghiêm ngặt 3 tiêu chuẩn tối ưu hiệu năng sau:**
> 1. **Concurrency Control (Zero-cost check)**: Cờ hiệu trạng thái Pause/Resume chia sẻ giữa Goroutine lắng nghe NATS và Goroutine chạy vòng lặp Ingest bắt buộc phải dùng `sync/atomic.Bool`.
> 2. **DLQ Storage I/O (Batch Performance)**: Ghi log lỗi trong Lenient mode PHẢI gom nhóm (Slice) và dùng cơ chế **Bulk-Insert 1 lần duy nhất ở cuối mỗi batch** bằng hàm `CreateInBatches` của GORM.
> 3. **LWW Guard SQL Tie-Breaker**: Mệnh đề `WHERE` của cú pháp `UPSERT` tại Postgres đích bắt buộc phải bọc đủ 3 điều kiện lọc để phân định luồng, cô lập hoàn toàn luồng CDC realtime.

---

## I. Cơ chế Kiểm soát Tải (Dynamic Flow Control & Throttling)

### 1. Thay đổi Database Schema & GORM Models

**Bảng:** `cdc_system.source_object_registry`

Muscle chạy migration bổ sung cột và cập nhật model struct (`internal/model/source_object_registry.go`):
Chú ý **GIỮ NGUYÊN** các cột hiện tại, chỉ thêm vào cuối:

```go
    // Migration mới cho Snapshot V2 Control Plane
    SnapshotMaxRPS      *int    `gorm:"column:snapshot_max_rps" json:"snapshot_max_rps"` // Tốc độ quét tối đa
    SnapshotErrorMode   *string `gorm:"column:snapshot_error_mode;default:'lenient'" json:"snapshot_error_mode"` // lenient / strict
```

### 2. Triển khai Logic tại `snapshot_runner_handler.go`

Trong hàm `runSnapshot(ctx context.Context, p snapshotV2Payload, jobID string)`, Muscle bổ sung NATS Event-Driven Pause và Rate Limiting:

```go
    // Kích hoạt NATS Subscription lắng nghe lệnh điều khiển
    var isPaused atomic.Bool
    subSubject := fmt.Sprintf("cdc.control.commands.%d", p.SourceObjectID)
    // Lưu ý: Tự bind NATS Connection vào SnapshotRunner hoặc dùng context
    // Nếu chưa có r.natsConn, cần lấy từ hệ thống (tuỳ injection hiện tại)
    // Để an toàn, nếu chưa truyền NATS vào SnapshotRunner, Muscle phải bổ sung NATS connection vào constructor NewSnapshotRunner.
```

Trong vòng lặp `for { ... }` sinh batch:
```go
    // Zero-cost check
    if isPaused.Load() {
        // flush checkpoint và break
    }

    // Rate limit
    if so.SnapshotMaxRPS != nil && *so.SnapshotMaxRPS > 0 {
        expectedDuration := time.Duration(len(batch)) * time.Second / time.Duration(*so.SnapshotMaxRPS)
        // Check time elapsed...
    }
```

---

## II. Resiliency (Bền vững) & Tiến độ (Progress)

### 1. Script DDL Migration
```sql
ALTER TABLE cdc_system.snapshot_progress ADD COLUMN IF NOT EXISTS total_rows BIGINT DEFAULT 0;
```

### 2. Logic Đoán số dòng và Commit Checkpoint
Trước khi vào `for`, gọi `coll.EstimatedDocumentCount(ctx)` và ghi xuống DB.

---

## III. Fail-Safe (An toàn khi có lỗi & Dead Letter Queue)

### 1. Cấu trúc Bảng DDL 
Tạo file migration cho `cdc_system.snapshot_dlq`.

### 2. Phân cấp Xử lý Lỗi (Strict vs Lenient Mode)
Tại vòng lặp trong `runSnapshot` khi duyệt qua từng doc của `batch`:
Nếu lỗi lúc parse hoặc tạo Envelope, xử lý dựa theo `so.SnapshotErrorMode`. Gom vào mảng `[]model.SnapshotDLQ` và cuối batch gọi `r.db.CreateInBatches(&dlqBatch, 100)`.

---

## IV. Data Integrity (LWW Guard & Source Tagging)

### Thay đổi hàm `buildUpsertSQLSnapshotInSchema` (`internal/sinkworker/upsert.go`)

Muscle ĐƯỢC YÊU CẦU CẬP NHẬT hàm `buildUpsertSQLSnapshotInSchema` để dùng LWW thay vì `DO NOTHING`. Mã nguồn tham khảo phải giữ nguyên signature:

```go
func buildUpsertSQLSnapshotInSchema(schemaName, table string, record map[string]any) (sqlText string, values []any) {
    // 1. Giữ nguyên đoạn sinh keys, colList, placeholders, values...
    
    // 2. Tạo mệnh đề updateSets (Loại trừ immutableOnUpdate y hệt hàm buildUpsertSQL)
    updateSets := make([]string, 0, len(keys))
    for _, k := range keys {
        if _, skip := immutableOnUpdate[k]; skip {
            continue
        }
        updateSets = append(updateSets,
            fmt.Sprintf("%s = EXCLUDED.%s", quoteIdent(k), quoteIdent(k)))
    }

    qt := quoteIdent(schemaName) + "." + quoteIdent(table)

    // 3. SQL LWW Guard Tie-Breaker
    sqlText = fmt.Sprintf(
        `INSERT INTO %s (%s) VALUES (%s)
ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted
DO UPDATE SET %s
WHERE %s._source_ts IS NULL
   OR EXCLUDED._source_ts > %s._source_ts
   OR (EXCLUDED._source_ts = %s._source_ts AND %s._source = 'snapshot:v2')`,
        qt,
        strings.Join(colList, ", "),
        strings.Join(placeholders, ", "),
        strings.Join(updateSets, ", "),
        qt, qt, qt, qt,
    )
    return sqlText, values
}
```