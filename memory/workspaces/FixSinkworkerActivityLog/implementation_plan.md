# Kế hoạch Bổ sung Ghi nhận Activity Log 'sink-upsert' cho CDC Pipeline

Tài liệu này mô tả phương án bổ sung ghi nhận log hoạt động (`sink-upsert`) vào bảng `cdc_system.cdc_activity_log` khi CDC worker thực hiện ghi dữ liệu (upsert) từ Kafka vào Shadow DB qua `BatchBuffer`.

## User Review Required

> [!IMPORTANT]
> **Thay đổi cốt lõi:**
> 1. **Ghi log gom nhóm (Batched Logging):** Do CDC worker xử lý gom tin nhắn thành các lô (batch) thông qua `BatchBuffer`, việc ghi activity log sẽ được tích hợp trực tiếp vào hàm `batchUpsert`. 
> 2. **Hiệu năng & Tránh spam:** Mỗi lần flush batch (vd: sau mỗi 2 giây hoặc khi đạt 500 records), hệ thống chỉ ghi **1 dòng log duy nhất** cho toàn bộ lô cho từng bảng shadow, thay vì ghi log cho từng message riêng lẻ gây quá tải ("lằng nhằng").
> 3. **Phân loại Operation:** Hoạt động này được đặt tên operation là `sink-upsert` để khớp với filter trên giao diện CMS UI.

## Proposed Changes

### centralized-data-service

#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
- Import thêm `"centralized-data-service/internal/model/system"`.
- Trong hàm `batchUpsert`, khởi tạo `governance.ActivityLogger` bằng `bb.db` và `bb.logger`.
- Bắt đầu ghi log:
  ```go
  var logEntry *system.ActivityLog
  var act *governance.ActivityLogger
  if bb.db != nil {
      act = governance.NewActivityLogger(bb.db, bb.logger)
      targetFQN := schemaName + "." + tableName
      logEntry = act.Start("sink-upsert", targetFQN, "kafka-consumer")
  }
  ```
- Sử dụng block `defer` để cập nhật trạng thái `Complete` hoặc `Fail` dựa trên kết quả ghi DB:
  ```go
  defer func() {
      if act != nil && logEntry != nil {
          if err != nil {
              act.Fail(logEntry, err.Error())
          } else {
              details := map[string]any{
                  "batch_size": len(records),
                  "written":    written,
              }
              act.Complete(logEntry, int64(written), details)
          }
      }
  }()
  ```

---

## Verification Plan

### Automated Tests
1. Thực hiện biên dịch kiểm tra cdc-worker:
   ```bash
   go build ./cmd/worker/...
   ```
2. Chạy thử các test liên quan tới handler shadow:
   ```bash
   go test -v ./internal/handler/shadow/...
   ```

### Manual Verification
1. Chạy cdc-worker lên.
2. Thực hiện upsert/insert dữ liệu nghiệp vụ ở phía source database để sinh event CDC mới.
3. Truy cập `http://localhost:5173/activity-log`, kiểm tra xem đã xuất hiện dòng log có operation là `sink-upsert` của bảng tương ứng chưa.
