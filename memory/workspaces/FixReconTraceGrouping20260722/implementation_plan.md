# Kết Quả Audit & Kế Hoạch Sửa Lỗi Luồng Recon Shadow-Master

## Nguyên Nhân Gốc Rễ (Root Cause Analysis)

Từ Jaeger trace log của `POST /api/reconciliation/check` trên bảng `payment_bills`:
1. **Thiếu trường `Segment` trong NATS Event**: `ReconJobCreatedEvent` trong `recon_job_worker.go` và `recon_check_handler.go` chỉ đóng gói `JobID`, `TargetTable`, `StartTime`, `EndTime` mà **bỏ quên `Segment`** (`source_shadow`, `shadow_master`, `both`).
2. **`ChunkStreamBucketEngine` chỉ đối soát Segment A (Source vs Shadow)**: `ChunkStreamBucketEngine` hiện tại chỉ giữ `sourceAgent` (Source Mongo) và `destAgent` (Shadow PG). Engine **chưa được wire `masterAgent`** (Master PG) và chưa có logic đối soát Segment B (`shadow_master`).
3. **Bỏ sót chênh lệch giữa Shadow DB và Master DB**: Khi người dùng kích hoạt đối soát `POST /api/reconciliation/check`, worker chỉ so sánh Source DB vs Shadow DB (khớp 100%), trả về `COMPLETED` với diff = 0, hoàn toàn bỏ qua việc kiểm tra chênh lệch giữa Shadow DB (`shadow_testpbs.payment_bills`) và Master DB (`master_payment_bill_service.payment_bills`).

---

## User Review Required

> [!IMPORTANT]
> 1. `ReconJobCreatedEvent` sẽ được bổ sung field `Segment` để truyền lựa chọn đối soát (`source_shadow`, `shadow_master`, hoặc `both`).
> 2. `ChunkStreamBucketEngine` sẽ được wire `masterAgent` và triển khai logic đối soát Segment B giữa Shadow DB và Master DB với cùng cấu trúc cây Trace Spans 4 tầng.

---

## Proposed Changes

### Centralized Data Service (`centralized-data-service`)

#### [MODIFY] [recon_job_worker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker.go)
- Bổ sung `Segment string json:"segment"` vào struct `ReconJobCreatedEvent`.
- Truyền `event.Segment` từ `HandleJobEvent` vào engine `ExecuteSegment`.

#### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
- Đóng gói `Segment: payload.Segment` vào `eventPayload` khi publish NATS message `ReconJobCreatedSubject`.

#### [MODIFY] [recon_stream_bucket_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go)
- Thêm `masterAgent *ReconDestAgent` và `db *gorm.DB` vào `ChunkStreamBucketEngine`.
- Bổ sung phương thức `WithMasterAgent(master *ReconDestAgent, db *gorm.DB) *ChunkStreamBucketEngine`.
- Thêm phương thức `ExecuteSegment(ctx, entry, segment, startTime, endTime)`:
  - Nếu `segment == "shadow_master"`: Tra cứu `MasterBindingRef` của `TargetTable`, chạy `checkDayChunkB` đối soát giữa `destAgent` (`shadow_schema.shadow_table`) và `masterAgent` (`master_schema.master_table`) trên các cột `_gpay_id` và `_source_ts`.
  - Nếu `segment == "both"`: Chạy cả Segment A và Segment B, tổng hợp toàn bộ chênh lệch.

#### [MODIFY] [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go)
- Wire `masterAgent` và `db` vào `chunkEngine` khi khởi tạo worker server.

---

## Verification Plan

### Automated Tests
- Run full unit tests:
  ```bash
  go test -v ./internal/...
  ```

### Manual Verification
- Gọi API `POST /api/reconciliation/check` với `segment: "shadow_master"` hoặc `"both"` cho `payment_bills` và xác nhận trên Jaeger UI xuất hiện các span `pg.hash_window: shadow_testpbs.payment_bills` vs `pg.hash_window: master_payment_bill_service.payment_bills`, phát hiện chính xác chênh lệch `shadow_master`.
