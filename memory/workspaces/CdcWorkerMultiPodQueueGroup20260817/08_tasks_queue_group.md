# 08 Tasks: CDC Worker Multi-Pod Queue Group Support

- [x] Task 1: Cập nhật `internal/server/server_setup.go` chuyển toàn bộ các NATS command sang `QueueSubscribe` với `cdc-worker-group`, giữ nguyên `schema.config.reload` và `cdc.cmd.kafka.refresh-topics` ở dạng `Subscribe`.
- [x] Task 2: Cập nhật `internal/service/recon/recon_job_worker.go` thêm `QueueSubscribe` vào interface `NATSConsumer` và chuyển subscription `cdc.event.recon.job_created` sang `QueueSubscribe(..., "recon-job-workers", ...)`.
- [x] Task 3: Cập nhật `internal/handler/dlq/dlq_state_machine.go` thêm optimistic lock cho câu retry logs.
- [x] Task 4: Chạy test suite `go test` và build binary `go build ./cmd/worker` để verify PASS 100%.
- [x] Task 5: Cập nhật báo cáo `11_report_queue_group.md` và `05_progress_queue_group.md`.
