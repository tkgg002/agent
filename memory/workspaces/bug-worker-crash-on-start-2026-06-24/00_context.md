# Context: bug-worker-crash-on-start-2026-06-24

## Vấn đề
- **Hiện tại**: Khi khởi động service `centralized-data-service` (worker), service bị crash/shutdown graceful ngay lập tức sau 5 giây.
- **Log crash**:
  ```text
  discovered kafka topics component=kafka_consumer op=discover_topics ...
  kafka consumer started component=kafka_consumer ...
  kafka consumer flush ticker configured interval=5s ...
  consumer pool stopped
  NATS disconnected
  CDC Worker stopped component=worker_server op=shutdown phase=complete ...
  worker server stopped, flushing remaining traces and logs
  OpenTelemetry shutdown
  ```
- **Lý do khả nghi**: Có lỗi khởi tạo trong phần wiring mới, hoặc một goroutine/service con bị crash/exit sớm kích hoạt trigger shutdown hệ thống.

## Yêu cầu
- Tìm gốc rễ nguyên nhân gây crash/shutdown khi start worker.
- Khắc phục lỗi bảo đảm worker khởi động và duy trì hoạt động bình thường.
