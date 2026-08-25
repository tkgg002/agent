# 01 Requirements: CDC Worker Multi-Pod Queue Group Support

## 1. Bối cảnh & Mục tiêu
- Khi triển khai môi trường Production với nhiều hơn 1 Pod `cdc-worker` (Kubernetes Deployment replicas = 2..5), các NATS Command từ CMS/Admin và Recon Job Worker đang sử dụng cơ chế Broadcast (`nats.Conn.Subscribe`) khiến tất cả các Pod cùng nhận lệnh và thực thi song song, gây ra duplicate execution, xung đột tài nguyên và tranh chấp DDL.
- Mục tiêu: Chuyển đổi các NATS Command và Recon Job Worker sang NATS Queue Group (`nats.Conn.QueueSubscribe`), đảm bảo mỗi message chỉ được phân phối cho ĐÚNG 1 POD rảnh rỗi xử lý.
- Đảm bảo 2 subject broadcast đặc thù (`schema.config.reload`, `cdc.cmd.kafka.refresh-topics`) tiếp tục duy trì Broadcast `Subscribe` để tất cả các Pod đều nhận được và reload in-memory cache/state.
- Bổ sung `FOR UPDATE SKIP LOCKED` cho DLQ poller để ngăn chặn 2 Pod quét trùng tập log lỗi.

## 2. Phạm vi thay đổi (Scope)
- `internal/server/server_setup.go`: Đổi các NATS Command sang `QueueSubscribe` với group `"cdc-worker-group"`.
- `internal/service/recon/recon_job_worker.go`: Thêm `QueueSubscribe` vào interface `NATSConsumer` và đổi subscription của `cdc.event.recon.job_created` sang `QueueSubscribe(..., "recon-job-workers", ...)`.
- `internal/handler/dlq/dlq_state_machine.go`: Bổ sung `FOR UPDATE SKIP LOCKED` vào câu query quét logs retry.
- Kiểm tra toàn bộ test suite để đảm bảo không gãy build và pass 100%.
