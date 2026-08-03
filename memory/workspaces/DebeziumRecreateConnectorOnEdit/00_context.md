# Bối cảnh & Phạm vi - Tự động tạo lại Debezium Connector khi Edit Connection

## 1. Bối cảnh
- Khi hạ tầng Debezium bị reset hoặc đổi cluster, toàn bộ connectors trên Debezium REST API bị mất.
- Tuy nhiên, dữ liệu cấu hình Connections (Source Fingerprints) vẫn còn được lưu trữ an toàn trong Postgres Database (`cdc_system.source_fingerprints`).
- Hiện tại, nếu connector chưa tồn tại trên Debezium, khi người dùng nhấn **Edit** và **Save** một Connection trên UI, Backend gọi `GET /connectors/:name/config` để merge config cũ -> bị lỗi 404 `connector_update_failed`. Người dùng buộc phải xóa connection trong DB rồi tạo mới lại từ đầu.

## 2. Mục tiêu
- Cho phép người dùng nhấn **Edit** và **Save** bất kỳ Connection nào.
- Nếu Backend phát hiện Connector chưa tồn tại trên Debezium (404), Backend sẽ tự động khôi phục credential từ DB (nếu giữ nguyên password `__KEEP__`) và **tự động Re-create Connector** trên Debezium.
- UI hiển thị phản hồi mượt mà, thông báo connector đã được tạo lại thành công.
