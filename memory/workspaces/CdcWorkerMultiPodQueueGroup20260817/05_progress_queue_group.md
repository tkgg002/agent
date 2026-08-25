# 05 Progress: CDC Worker Multi-Pod Queue Group Support

- [2026-08-17 15:18:00] [Brain:Gemini-3.7-Flash] Khởi tạo workspace và tài liệu requirements / plan.
- [2026-08-17 15:18:05] [Brain:Gemini-3.7-Flash] Tiến hành thực thi chuyển đổi NATS Command sang Queue Group trong `server_setup.go`, `recon_job_worker.go`, `dlq_state_machine.go`.
- [2026-08-17 15:18:32] [Muscle:Gemini-3.7-Flash] Cập nhật `internal/server/server_setup.go`: Thêm `workerQueueGroup = "cdc-worker-group"` và chuyển 30+ NATS Commands sang `QueueSubscribe`.
- [2026-08-17 15:18:56] [Muscle:Gemini-3.7-Flash] Cập nhật `internal/service/recon/recon_job_worker.go`: Thêm `QueueSubscribe` vào interface `NATSConsumer` và đổi `cdc.event.recon.job_created` sang `QueueSubscribe(..., "recon-job-workers", ...)`.
- [2026-08-17 15:19:01] [Muscle:Gemini-3.7-Flash] Cập nhật `internal/service/recon/recon_job_worker_test.go`: Bổ sung `QueueSubscribe` mock.
- [2026-08-17 15:19:10] [Muscle:Gemini-3.7-Flash] Cập nhật `internal/handler/dlq/dlq_state_machine.go`: Thêm optimistic concurrency check (`status IN ('pending','failed')` và kiểm tra `RowsAffected == 0`).
- [2026-08-17 16:22:08] [Muscle:Gemini-3.7-Flash] Chạy `go test ./internal/service/recon/... ./internal/handler/...` -> PASS 100%.
- [2026-08-17 16:39:15] [Muscle:Gemini-3.7-Flash] Chạy `go build ./cmd/worker` -> BUILD OK, hoàn thành nghiệm thu.
