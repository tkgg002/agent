# Requirements: GORM DB Tracing Context Propagation

## Mục tiêu
Khắc phục triệt để việc mất traces truy vấn database (SQL query spans) trong `centralized-data-service` bằng cách truyền context `WithContext(ctx)` cho tất cả các truy vấn GORM.

## Các yêu cầu chi tiết
1. **Rà soát các tệp chứa truy vấn database không có context**:
   - `internal/handler/shadow/batch_buffer.go`
   - `internal/handler/recon/recon_sysops_handler.go`
   - `internal/server/server_scheduler.go`
   - `internal/service/shadow/schema_adapter.go`
   - `internal/service/source/bridge_service.go`
   - `internal/service/governance/masking_service.go`
   - `internal/service/governance/schema_validator.go`
   - `internal/service/governance/partition_dropper.go`
   - `internal/service/governance/activity_logger.go`
   - `internal/service/recon/recon_engine_segment_b.go`

2. **Cập nhật code**:
   - Bổ sung tham số `ctx context.Context` vào các hàm/method nghiệp vụ liên quan.
   - Thay đổi các lệnh gọi query từ `db.Raw(...)` hay `db.Exec(...)` thành `db.WithContext(ctx).Raw(...)` hay `db.WithContext(ctx).Exec(...)` để OpenTelemetry GORM plugin có thể capture và liên kết với spans cha.
