# Implementation Plan - Re-create Debezium Connector on Edit Connection

## 1. Tổng quan
Kế hoạch xử lý bài toán khi Debezium bị reset / đổi cluster làm mất connectors nhưng Connections vẫn còn trong DB.
Khi người dùng bấm **Edit** và **Save** Connection trên UI:
- Backend không bị crash 404 khi check config cũ trên Debezium.
- Tự động khôi phục mật khẩu cũ từ DB (nếu giữ nguyên `__KEEP__`).
- Tự động gọi Kafka Connect REST API `PUT /connectors/:name/config` để Re-create Connector trên Debezium cluster mới.

## 2. Các bước triển khai
1. **Sửa Backend (`cdc-cms-service`)**:
   - `internal/app/commands/source/debezium_connector.go`: Bắt lỗi 404 từ `h.writer.GetConfig`, tự động khôi phục password từ DB và gọi `UpdateConfig` / `Create` để tái tạo connector.
2. **Kiểm tra Frontend (`cdc-cms-web`)**:
   - `src/pages/SourceConnectors.tsx`: Đảm bảo Modal Edit điền đủ seed data và gửi full config payload khi Save.
3. **Verify & Test**:
   - Chạy `go build ./...` ở Backend.
   - Chạy `npm run build` ở Frontend.
