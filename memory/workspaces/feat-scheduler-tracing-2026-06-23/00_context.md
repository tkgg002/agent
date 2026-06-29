# Context: Scheduler Tracing & Log Grouping

## Bối cảnh
Khi scheduler của `cdc-worker` chạy các tác vụ định kỳ (chẳng hạn như `reconcile`, `transform`, `field-scan`, `partition-check`...):
1. Scheduler thực hiện lấy khóa Redis (lock) và bắt đầu một chu kỳ chạy các scheduled operations.
2. Các logs ghi nhận từ lúc lấy lock cho đến khi chu kỳ reconcile hoàn tất (hoặc các operation khác hoàn tất) hiện tại KHÔNG được liên kết với một trace ID cụ thể nào (không có OpenTelemetry traces).
3. Người dùng muốn gom nhóm toàn bộ mớ logs này thành một nhóm duy nhất dưới một trace span bắt đầu từ khi log ra thông báo `"Acquired scheduler lock, running cycle"`.

## Mục tiêu
1. Bọc chu kỳ tick scheduler (hoặc chu kỳ chạy của các schedules) vào một OpenTelemetry trace span gốc (ví dụ: `cdc.worker.scheduler_cycle`).
2. Các chu kỳ nhỏ của từng operation (như `reconcile_cycle`, `transform_cycle`, `field_scan_cycle`, `partition_check_cycle`) sẽ kế thừa và trở thành các Child Spans của `cdc.worker.scheduler_cycle`.
3. Cập nhật các câu lệnh ghi log trong `worker_server.go` và các cycle handlers (đặc biệt là package `recon` phục vụ reconcile cycle) để sử dụng `observability.Ctx(ctx, logger)` thay vì `logger` trực tiếp, từ đó đính kèm `trace_id` và `span_id` chính xác vào logs, giúp SigNoz/Jaeger nhóm chúng lại một cách dễ dàng.
