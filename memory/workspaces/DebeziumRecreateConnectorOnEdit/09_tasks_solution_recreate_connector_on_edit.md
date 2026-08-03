# Hồ sơ Giải pháp Kỹ thuật - Tự động Re-create Debezium Connector khi Save Connection

## 1. Phân tích nguyên nhân gốc rễ (Root Cause)
Khi Debezium bị reset hoặc đổi cluster:
1. Toàn bộ connector trên Debezium REST API bị mất.
2. Dữ liệu Connections vẫn còn trong bảng Postgres `cdc_system.source_fingerprints`.
3. Khi người dùng bấm **Edit** connection và nhấn **Save**, Backend (`UpdateSystemConnectorConfigHandler.Handle`) thực hiện:
   ```go
   current, err := h.writer.GetConfig(ctx, cmd.Name)
   if err != nil {
       return nil, err
   }
   ```
   Do connector không còn trên Debezium, Debezium REST API `/connectors/:name/config` ném lỗi HTTP 404 (`ErrKafkaConnectNotFound`), dẫn đến API trả về 502 `connector_update_failed`.

## 2. Giải pháp chi tiết

### 2.1 Backend (`cdc-cms-service`)

#### A. Cập nhật `internal/app/commands/source/debezium_connector.go`
Trong `UpdateSystemConnectorConfigHandler.Handle`:
1. Bắt lỗi `errors.Is(err, infrahttp.ErrKafkaConnectNotFound)`:
   ```go
   isNotFound := false
   current, err := h.writer.GetConfig(ctx, cmd.Name)
   if err != nil {
       if errors.Is(err, infrahttp.ErrKafkaConnectNotFound) {
           isNotFound = true
           current = make(map[string]string)
       } else {
           return nil, err
       }
   }
   ```

2. Khôi phục Credential từ DB nếu `isNotFound == true` và config gửi từ UI chứa `__KEEP__`:
   - Nếu `cmd.Config["database.password"] == "__KEEP__"` (hoặc `mongodb.connection.string`), ta kiểm tra `cmd.Fingerprint` hoặc gọi `sourceRepo` để lấy `options_json` / credentials cũ đã lưu trong DB.
   - Thay thế `__KEEP__` bằng password giải mã được từ DB.

3. Re-create Connector trên Debezium:
   - Kafka Connect REST API `PUT /connectors/:name/config` có cơ chế tự động tạo mới connector nếu connector chưa tồn tại và payload có đủ `connector.class`.
   - Gọi `h.writer.UpdateConfig(ctx, cmd.Name, merged)`.
   - Cập nhật `h.sourceRepo.Upsert(ctx, cmd.Fingerprint)` để đồng bộ trạng thái `source_fingerprints` trong DB về `"created"`.

### 2.2 Frontend (`cdc-cms-web`)

#### A. Cập nhật `src/pages/SourceConnectors.tsx`
- Trong Modal Edit Connection:
  - Khi edit một connection chưa có connector trên Debezium (`orphanFingerprint`), Form tự động điền đầy đủ cấu hình từ `SourceFingerprint` (như `connector_class`, `dbKind`, `topicPrefix`, `host`, `port`, `database`, `collectionNames`, `tableIncludeList`, `schemaIncludeList`).
  - Khi nhấn **Save**, `buildConnectorConfig` xây dựng đầy đủ payload config và gửi request `PATCH /api/v1/system/connectors/:name/config`.
  - Hiển thị thông báo thành công rõ ràng cho người dùng.
