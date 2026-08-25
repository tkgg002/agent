# 11 Report: CDC Worker Multi-Pod Queue Group Support

## 1. Tổng quan thay đổi
Hệ thống đã được nâng cấp để hỗ trợ triển khai đa Pod (Multi-Pod / Multi-Replica) cho `cdc-worker` trên môi trường Production mà không gặp lỗi chạy trùng hay tranh chấp tài nguyên khi nhận các command từ CMS / Admin hoặc các event background.

## 2. Chi tiết các file thay đổi & số dòng code
| File thay đổi | Số dòng sửa / thêm | Mô tả thay đổi |
| :--- | :---: | :--- |
| `centralized-data-service/internal/server/server_setup.go` | +38 / -36 | Khai báo `workerQueueGroup = "cdc-worker-group"` và chuyển toàn bộ các NATS command (`cdc.cmd.*`) sang `QueueSubscribe`. Giữ nguyên Broadcast `Subscribe` cho `schema.config.reload` và `cdc.cmd.kafka.refresh-topics`. |
| `centralized-data-service/internal/service/recon/recon_job_worker.go` | +6 / -4 | Bổ sung `QueueSubscribe` vào interface `NATSConsumer` và chuyển subscription `cdc.event.recon.job_created` sang `QueueSubscribe(..., "recon-job-workers", ...)`. |
| `centralized-data-service/internal/service/recon/recon_job_worker_test.go` | +4 / -0 | Bổ sung method `QueueSubscribe` vào `mockNATSConsumer` trong unit test. |
| `centralized-data-service/internal/service/recon/recon_smoke_test.go` | +4 / -4 | Cập nhật tham số `traceID` trong unit test `RunTotalOnlyA` & `RunTotalOnlyB`. |
| `centralized-data-service/internal/handler/dlq/dlq_state_machine.go` | +4 / -1 | Bổ sung điều kiện `status IN ('pending','failed')` và kiểm tra `RowsAffected == 0` khi đánh dấu `retrying` để ngăn 2 pod replay trùng bản tin lỗi. |

## 3. Kết quả Kiểm thử (Verification)
- `go test ./internal/service/recon/... ./internal/handler/...`: **PASS 100%** (10/10 package test ok).
- `go build ./cmd/worker`: **BUILD THÀNH CÔNG** (exit code 0).
