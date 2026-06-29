# Context: bug-snapshot-v2-postgresql-support-2026-06-23

## Vấn đề
- **Hiện tại**: Lệnh snapshot v2 (`snapshot.v2`) trong service `centralized-data-service` ném lỗi:
  `snapshot.v2 currently supports engine=mongodb only (got "postgresql")`
- **Lý do**: File [snapshot_runner_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go#L129-L131) đang hardcode chặn các engine khác ngoài `mongodb` và chỉ implement logic đọc dữ liệu từ MongoDB driver.

## Yêu cầu
- Hỗ trợ engine `postgresql` cho lệnh `snapshot.v2`.
- Đọc dữ liệu trực tiếp từ database nguồn PostgreSQL bằng driver `pgx`.
- Lấy thông tin Primary Key của table nguồn từ `SourceObjectRegistry.PrimaryKeyField`. Sử dụng cột này để sắp xếp (`ORDER BY`) và lọc phục vụ việc resume (`WHERE primary_key > last_seen`).
- Chuyển đổi dữ liệu row từ PostgreSQL sang dạng Map, xử lý các kiểu dữ liệu đặc biệt (ví dụ: `JSON`, `JSONB`, `UUID`, `NUMERIC`) để khi serialise sang JSON không bị lỗi định dạng (ví dụ: base64 encode cho JSON/UUID).
- Đưa dữ liệu đã convert vào event bridge pipeline qua [EventHandler.HandleRaw](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L103) tương tự như MongoDB.
- Capture cluster time của Postgres từ database (hoặc epoch time nếu error) để điền vào CDC event payload nhằm bảo đảm Last-Write-Wins (LWW) guard hoạt động đúng.
