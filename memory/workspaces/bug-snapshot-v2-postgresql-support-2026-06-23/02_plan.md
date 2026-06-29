# Implementation Plan: bug-snapshot-v2-postgresql-support-2026-06-23

## Kế hoạch triển khai

### Phase 1: Research & Setup (Hoàn thành)
- [x] Định vị lỗi tại [snapshot_runner_handler.go:129](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go#L129).
- [x] Tìm hiểu cấu trúc và cách kết nối PostgreSQL bằng driver `pgx` trong codebase.
- [x] Xác định các kiểu dữ liệu đặc biệt của Postgres cần convert (`json`, `jsonb`, `uuid`, `numeric`).

### Phase 2: Implementation
- [ ] Mở rộng check engine type trong `snapshot_runner_handler.go` để chấp nhận cả `postgresql`.
- [ ] Bổ sung helper `capturePGClusterTime` trong `snapshot_runner_utils.go` để lấy transactional timestamp từ PostgreSQL bằng query `SELECT (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::bigint`.
- [ ] Implement logic loop cursor đọc table PostgreSQL:
  - Connect database Postgres nguồn bằng DSN resolve từ connection registry.
  - Query lấy approximate count của table để update `total_rows`.
  - Phân trang bằng `ORDER BY {primary_key}` và `WHERE {primary_key} > {last_seen}`.
  - Quét từng row và scan data thành map. Xử lý map các type PostgreSQL sang JSON representation an toàn (đặc biệt là UUID, JSON, JSONB, Numeric).
  - Marshal row data và wrap vào Debezium-compatible envelope qua `buildSnapshotEnvelope`.
  - Gọi `eventHandler.HandleRaw(...)` để đưa data vào pipeline đồng bộ.

### Phase 3: Verification
- [ ] Viết unit tests giả lập PostgreSQL snapshot runner trong `snapshot_runner_test.go` hoặc test suite tương ứng để bảo đảm SQL syntax và data casting hoạt động đúng.
- [ ] Chạy `go build` và `go test` trên service `centralized-data-service`.
