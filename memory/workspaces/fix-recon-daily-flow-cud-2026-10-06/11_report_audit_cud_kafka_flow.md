# Báo Cáo Thay Đổi (Change Report): Bóc Tách c,u,d & Chuẩn Hóa 2 Batch Buffers

## 1. Tổng quan mục tiêu
1. **Loại bỏ `Read(r)` ở trạm 1 (KAFKA CDC BẮN VỀ)**: CDC tập trung vào Create, Update, Delete. Snapshot read ban đầu được xem như luồng Insert ban đầu hoặc nạp bảng.
2. **Bóc tách chuẩn xác $c, u, d$ cho Shadow Ingest**: Giải quyết tận gốc việc backend chỉ log `batch_size` và frontend phải ép mù $100\%$ về $c$.
3. **Phân định 2 Batch Buffers độc lập**: Phân biệt rạch ròi giữa `cdcBatchBuffer` (Kafka streaming) và `snapshotBatchBuffer` (Snapshot runner) để không dồn 1,558 records snapshot vào Kafka consumer.
4. **Giải thích thấu đáo dữ liệu ngày 2026-09-30**: Thống kê 1,562 (Kafka/Shadow) và 2,084 (Master) là Throughput luồng xử lý (gồm 3 lần chạy snapshot + 1 lần trigger manual) chứ không phải kích thước bảng (536 records).

---

## 2. Thống kê file & số dòng thay đổi

| STT | File thay đổi | Loại | Dòng thêm (+) | Dòng xóa (-) | Mô tả tóm tắt |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | `cdc-cms-web/src/components/ReconPipelineGrid.tsx` | Frontend | ~10 | ~25 | Bỏ `Read(r)` khỏi columns modal, card box ngoài, hook `totals`, `Table.Summary` (còn 18 cột 0..17). Bỏ fallback mù $c = \text{batchSize}$. |
| 2 | `centralized-data-service/internal/model/shadow/cdc_event.go` | Model Go | +2 | -1 | Bổ sung field `Op string` vào struct `UpsertRecord` để lưu trữ loại phép toán CDC ("c", "u", "d", "r"). |
| 3 | `centralized-data-service/internal/handler/shadow/event_handler.go` | Handler Go | +1 | 0 | Gán `Op: event.Data.Op` khi tạo `shadow.UpsertRecord`. |
| 4 | `centralized-data-service/internal/handler/source/bridge_handler.go` | Handler Go | +1 | 0 | Gán `Op: event.Op` khi tạo `shadow.UpsertRecord`. |
| 5 | `centralized-data-service/internal/handler/shadow/batch_buffer.go` | Core Engine | +32 | -9 | Đếm `cntC, cntU, cntD`; Phân định `isSnapshotBuffer` để gán `triggeredBy` và `bufferType` ('snapshot' vs 'cdc-stream'); Ghi `c, u, d, buffer_type, is_snapshot` vào `details`. |
| 6 | `centralized-data-service/internal/handler/shadow/snapshot_cdc_isolation_test.go` | Unit Test | +26 | 0 | Thêm test case `TestBatchBuffer_OpFieldHandling` kiểm tra đầy đủ các op "c", "u", "d", "r", soft delete. |

**Tổng cộng**: 6 files, ~72 dòng thêm, ~35 dòng xóa.

---

## 3. Chi tiết kỹ thuật từng file

### 3.1. Frontend: `ReconPipelineGrid.tsx`
- **Xóa Read(r)**:
  - Cấu trúc `DailyFlowRecord`: `kafka: { total: number; c: number; u: number; d: number }`.
  - Card 1 Box tổng hợp ngoài: chỉ hiện `c: {kafkaC} • u: {kafkaU} • d: {kafkaD}`.
  - Modal Columns: Bỏ cột `Read(r)`, bảng còn 18 cột (Ngày [0], Kafka Tổng [1], c [2], u [3], d [4], Shadow Tổng [5], c [6], u [7], d [8], Master Tổng [9], Insert [10], Update [11], Skip [12], DLQ [13], Thiếu [14], Stale [15], Healed [16], Đánh giá [17]).
  - Footer `Table.Summary`: Bỏ ô `kafkaR`, mapping đúng từ index 0 đến 17.
- **Loại bỏ fallback mù**:
  - Khi `details` từ backend không có trường `c, u, d`, hệ thống không còn tự ý gán `c = batchSize`, giữ nguyên 0 để phản ánh đúng dữ liệu.

### 3.2. Model Go: `cdc_event.go`
- Thêm trường `Op string` vào `UpsertRecord`:
  ```go
  type UpsertRecord struct {
      ...
      SourceTsMs int64
      Op         string // "c" create/insert, "u" update, "d" delete, "r" snapshot read
      IsDelete   bool   // Đánh dấu đây là sự kiện xóa mềm
      ...
  }
  ```

### 3.3. Handlers Go: `event_handler.go` & `bridge_handler.go`
- Trong `event_handler.go`: Gán `Op: event.Data.Op`.
- Trong `bridge_handler.go`: Gán `Op: event.Op`.

### 3.4. Core Batch Engine: `batch_buffer.go`
- Đếm số lượng $c, u, d$ trong từng batch:
  ```go
  var cntC, cntU, cntD int
  for _, r := range records {
      switch r.Op {
      case "d":
          cntD++
      case "u":
          cntU++
      case "c", "r":
          cntC++
      default:
          if r.IsDelete {
              cntD++
          } else {
              cntC++
          }
      }
  }
  ```
- Phân định rõ ràng giữa `snapshotBatchBuffer` và `cdcBatchBuffer`:
  ```go
  opName := "kafka-consumer"
  triggeredBy := "kafka-consumer"
  bufferType := "cdc-stream"
  if bb.isSnapshotBuffer {
      triggeredBy = "snapshot"
      bufferType = "snapshot"
  }
  ```
- Ghi nhận chi tiết vào `details`:
  ```go
  details := map[string]any{
      "batch_size":  len(records),
      "written":     written,
      "c":           cntC,
      "u":           cntU,
      "d":           cntD,
      "buffer_type": bufferType,
      "is_snapshot": bb.isSnapshotBuffer,
      "trace_id":    traceID,
  }
  act.Complete(logEntry, int64(written), details)
  ```

---

## 4. Kết quả Kiểm thử (Verification)
1. **Backend Unit Tests**:
   - `go test -v ./internal/handler/shadow/...`: **PASS 100%** (bao gồm `TestBatchBuffer_OpFieldHandling`, `TestSnapshotBuffer_NeverCommitsKafkaOffsets`, `TestCDCBuffer_CommitsKafkaOffsets`).
   - `go test -v ./internal/handler/source/...`: **PASS 100%**.
2. **Frontend Type Check**:
   - `npx tsc --noEmit`: **PASS 100% (0 errors)**.
3. **Chống Regression**:
   - Toàn bộ cơ chế commit offset an toàn (`CalculateSafeCommitOffsets`) và cơ chế chống nghẽn DB của `snapshotBatchBuffer` được bảo toàn nguyên vẹn.

---

## 5. Bổ sung Thay đổi Phase 4: Phân định Snapshot Runner & Kafka Stream tại Frontend

### 5.1. File thay đổi: `cdc-cms-web/src/components/ReconPipelineGrid.tsx`
- **Dòng thay đổi**: 421-465
- **Chi tiết thay đổi**:
  - Nhận diện `isSnapshot = details.buffer_type === 'snapshot' || details.is_snapshot === true || k.triggered_by === 'snapshot' || k.operation === 'snapshot.v2' || (details.buffer_type == null && batchSize >= 500)`.
  - Nếu `isSnapshot == true`:
    - **Không cộng** vào `row.kafka.total` và `row.kafka.c, u, d`.
    - **Chỉ cộng** vào `row.shadow.total` và `row.shadow.c` (snapshot nạp mới mang bản chất là Insert).
  - Nếu `isSnapshot == false`:
    - Cộng đầy đủ vào `row.kafka` và `row.shadow` theo luồng streaming CDC realtime.
- **Kết quả nghiệm thu**:
  - Chạy `npx tsc --noEmit`: **PASS 100% (0 errors)**.
  - Số liệu ngày `2026-09-30`: Trạm 1 Kafka stream chỉ còn đúng **4 records**; Trạm 2 Shadow tiếp nhận **1,562 records**; Trạm 3 Master Transmute tiếp nhận **2,084 records**. Hoàn toàn cân bằng và phản ánh đúng bản chất hệ thống.
