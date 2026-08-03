# Yêu cầu chi tiết - Re-create Debezium Connector on Edit Connection

## 1. Yêu cầu Backend Go (`cdc-cms-service`)
- [REQ-1] Trong `UpdateSystemConnectorConfigHandler.Handle`: khi gọi `h.writer.GetConfig(ctx, cmd.Name)` gặp lỗi `ErrKafkaConnectNotFound` (HTTP 404):
  - Bắt lỗi an toàn, không trả về 502/404 fail cho client.
  - Tự động fallback coi config hiện tại là rỗng (`current = map[string]string{}`).
- [REQ-2] Giải mã credential từ DB (`source_fingerprints` table):
  - Nếu connector không tồn tại trên Debezium và `cmd.Config` chứa giá trị `__KEEP__` ở trường mật khẩu (ví dụ `database.password` hoặc `mongodb.connection.string`), Backend sẽ query `source_fingerprints` từ `sourceRepo` theo `cmd.Name`.
  - Lấy `options_json` từ DB để giải mã password/username cũ và điền vào `merged` config trước khi gửi sang Debezium.
- [REQ-3] Tự động Re-create Connector trên Debezium:
  - Gọi `h.writer.UpdateConfig` (hoặc `h.writer.Create` nếu cần) để đẩy full payload config lên Kafka Connect REST API `PUT /connectors/:name/config`.
  - Cập nhật lại `source_fingerprints` record trong DB qua `h.sourceRepo.Upsert`.
  - Trả về status HTTP 200/201 kèm thông báo thành công.

## 2. Yêu cầu Frontend (`cdc-cms-web`)
- [REQ-4] Giao diện Edit Connection:
  - Khi người dùng bấm Edit trên một Connection thuộc danh sách Orphan (connector không có trên Debezium), Modal Edit hiển thị đầy đủ thông tin cũ đã lưu từ DB.
  - Cho phép giữ nguyên mật khẩu cũ (`__KEEP__`) hoặc nhập mật khẩu mới.
  - Khi nhấn **Save**, gửi request `PATCH /api/v1/system/connectors/:name/config` kèm full payload config.
- [REQ-5] Trải nghiệm người dùng (UX):
  - Đảm bảo hiển thị notification rõ ràng: "Connector chưa có trên Debezium, đã tự động khởi tạo lại thành công!".
