# Thiết kế kỹ thuật chi tiết: Concurrency & Batching Optimization

Tài liệu này ghi nhận chi tiết thiết kế mã nguồn, cấu trúc dữ liệu và giải pháp khắc phục Deadlock/Race Condition.

## 1. Cơ chế `DebounceBuffer`
*   Xem thiết kế chi tiết tại [12_implementation_plan_sink_transmute.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/FixSinkTransmuteConcurrency20260708/12_implementation_plan_sink_transmute.md).
*   Sử dụng một bản đồ bảo vệ bởi `sync.Mutex` để đảm bảo an toàn truy cập map đa luồng từ nhiều goroutine.
*   Cơ chế `flushCh` (channel của string) đóng vai trò điều phối luồng ghi, cô lập việc gom lô của từng bảng độc lập.

## 2. Phòng tránh Race Condition
*   Không chia sẻ mảng `NatsMsgTask` giữa luồng thêm tin nhắn (`Add`) và luồng ghi (`flushTable`). Khi flushTable bốc mẻ dữ liệu ra, lập tức xóa key đó khỏi map `db.tasks` dưới sự bảo vệ của mutex.
*   Đảm bảo `nats.Msg.Ack()` và `nats.Msg.Term()` được gọi bất đồng bộ an toàn từ worker pool.
