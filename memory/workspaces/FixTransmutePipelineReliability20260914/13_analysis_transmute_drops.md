# Phân Tích Hiện Trạng & Điểm Rơi Rớt Transmute (Root Cause Analysis)

## 1. Kiến Trúc Luồng Sự Kiện (Event Flow)
```
Kafka CDC Ingest -> SinkWorker -> NATS `cdc.cmd.transmute-shadow`
  -> HandleTransmuteShadow() -> Query master_binding
  -> NATS `cdc.cmd.transmute`
  -> HandleTransmute()
  -> TableDebouncer.Add() (Gom lô 500 records / 100ms - 1s)
  -> TableDebouncer.workerLoop() -> runDebouncedTransmute()
  -> TransmuterModule.Run()
```

## 2. Phân Tích Chi Tiết 6 Điểm Gây Mất Dữ Liệu / Treo Job

### Điểm 1: NATS Core QueueSubscribe (At-most-once)
- **Vấn đề**: `QueueSubscribe` dùng NATS Core thuần. Khi worker khởi động lại hoặc redeploy, các message đang bay trong NATS broker bị mất hoàn toàn nếu không có consumer sẵn sàng hoặc ACK timeout.
- **Hậu quả**: Toàn bộ CDC events phát sinh trong khoảng thời gian deployment bị mất, không được transmute sang master.

### Điểm 2: Timeout & Silent Drop Trong HandleTransmuteShadow
- **Vấn đề**: Hàm `HandleTransmuteShadow` đặt context timeout cố định 5s cho DB query `ListMasterTablesByShadowTable` hoặc `ListMasterTablesByShadowIdentity`. Khi DB tải cao, query này dễ bị timeout (> 5s).
- **Hậu quả**: Khi gặp lỗi hoặc không tìm thấy binding, code thực hiện `return` ngay mà không có retry, không metric cảnh báo, làm biến mất trigger transmute.

### Điểm 3: Bỏ Qua Lỗi Khi NATS Publish `cdc.cmd.transmute`
- **Vấn đề**: Lệnh `_ = h.natsConn.PublishMsg(outMsg)` tại dòng 150 của `transmute_handler.go` phớt lờ lỗi.
- **Hậu quả**: Nếu kết nối NATS bị ngắt tạm thời hoặc buffer đầy, message bị rớt mà không ai hay biết.

### Điểm 4: TableDebouncer Tắc Nghẽn & Backpressure Thiếu Visibility
- **Vấn đề**: `TableDebouncer` có `concurrencyLimit = 1` cho mỗi master table để chống deadlock. Khi một mẻ transmute lớn đang xử lý (có thể mất nhiều phút), các message mới dồn vào slice `tasks`. Khi `len(tasks) >= maxSize * 2`, debouncer sleep 10ms trong vòng lặp kín mà không cảnh báo ra bên ngoài.
- **Hậu quả**: Tích tụ độ trễ lớn, không thể theo dõi được tình trạng ứ đọng trên Dashboard/Prometheus.

### Điểm 5: TransmuteScheduler Kẹt Trạng Thái `running`
- **Vấn đề**: Scheduler chỉ giải phóng các job stuck `running` duy nhất 1 lần khi worker reboot (`cleanupStuckSchedules`). Nếu goroutine transmute bị OOM, crash hoặc context leak giữa phiên chạy mà worker không restart, schedule sẽ kẹt vĩnh viễn ở `running`.
- **Hậu quả**: Cron định kỳ bị tê liệt, không bao giờ được quét lại.

### Điểm 6: Full Table Scan Timeout Trong Recon & Hash Window
- **Vấn đề**:
  - `recon_smoke.go` gọi `agent.CountRows(traceCtx, relation, "_gpay_id")` -> thực thi `SELECT COUNT("_gpay_id") FROM shadow_...` gây full scan bảng hàng chục triệu dòng -> timeout 30s.
  - Bảng master/shadow thiếu index trên trường timestamp nghiệp vụ (`updatedAt`), khiến `HashWindow` quét toàn bộ bảng dẫn đến `context deadline exceeded`.
- **Hậu quả**: Recon liên tục báo lỗi đỏ, không thể tự động bù đắp dữ liệu rớt.
