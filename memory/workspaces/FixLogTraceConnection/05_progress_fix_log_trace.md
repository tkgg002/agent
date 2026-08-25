# Progress: Fix Log Trace Connection

## Root Cause Analysis
- **Vấn đề**: Các file logs nghiệp vụ và logs xử lý sự kiện/lệnh trong `centralized-data-service` không được liên kết với traces trên giao diện SigNoz/Grafana.
- **Nguyên nhân gốc rễ**: 
  1. Trong Go, OpenTelemetry trích xuất trace/span context và lưu trữ trong `context.Context`. Để zap logger đính kèm `trace_id` và `span_id` vào log entry, cần phải gọi thông qua `observability.Ctx(ctx, logger)`.
  2. Tuy nhiên, rất nhiều file handler, strategy, helper và base handler đang trực tiếp gọi `logger.Info(...)`, `logger.Warn(...)`, `logger.Error(...)` mà không bọc qua `observability.Ctx(ctx, logger)` dù có sẵn `ctx context.Context` trong phạm vi hàm.
  3. Một số hàm/phương thức nhận NATS messages chưa trích xuất trace context từ headers hoặc chưa truyền context xuyên suốt vào các lớp nghiệp vụ bên dưới.

## Audit Log
- `[2026-08-05 11:00:00] [Antigravity:Gemini 3.5 Flash] Khởi tạo workspace FixLogTraceConnection. Phân tích nguyên nhân gốc rễ lỗi đứt gãy log-trace.`
- `[2026-08-05 11:05:00] [Antigravity:Gemini 3.5 Flash] Thiết kế giải pháp kỹ thuật, tích hợp cơ chế trích xuất context tự động từ NATS message header trong NatsPublish.`
- `[2026-08-05 11:33:00] [Antigravity:Gemini 3.5 Flash] Subagent muscle-executor hoàn thành chỉnh sửa base_handler.go, bridge_handler.go, và bridge_mongo.go. Build test unsandboxed PASS thành công. Giao việc Phase 2 cho subagent thực hiện sửa đổi các handler còn lại.`
- `[2026-08-05 15:05:00] [Antigravity:Gemini 3.5 Flash] Chạy unit tests toàn bộ handler phát hiện 2 file test (scan_handler_test.go, batch_transform_handler_test.go) bị fail do lệch mock query sql với thực tế của service. Cập nhật thiết kế kỹ thuật và uỷ quyền subagent muscle-executor sửa đổi.`
- `[2026-08-05 15:06:00] [Antigravity:Gemini 3.5 Flash] Chạy lại test suite phát hiện lỗi biên dịch ở recon_job_handler_test.go (thiếu GetActiveJobs mock) và lỗi lệch đối số ở scan_handler_test.go (hasAfter = true đổi path thành after.items). Cập nhật thiết kế kỹ thuật và uỷ quyền subagent muscle-executor sửa đổi lần 2.`
- `[2026-08-05 15:07:00] [Antigravity:Gemini 3.5 Flash] Chạy lại test suite sau khi subagent cập nhật mock test lần 2. Kết quả: TOÀN BỘ TESTS PASS 100%. Đóng task, viết walkthrough.md.`
