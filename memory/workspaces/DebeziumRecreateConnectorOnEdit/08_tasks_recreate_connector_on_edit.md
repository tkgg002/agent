# Danh sách Task - Re-create Debezium Connector on Edit Connection

## Phase 1: Backend Implementation (`cdc-cms-service`)
- [ ] Task 1.1: Bổ sung phương thức `GetByName` hoặc truy vấn `options_json` trong `SourceFingerprintRepo` / `SystemConnectorRepo`.
- [ ] Task 1.2: Cập nhật `UpdateSystemConnectorConfigHandler.Handle` trong `internal/app/commands/source/debezium_connector.go`:
  - Bắt lỗi `errors.Is(err, infrahttp.ErrKafkaConnectNotFound)` khi `h.writer.GetConfig`.
  - Nếu connector 404, khôi phục credential cũ từ `source_fingerprints` nếu config chứa `__KEEP__`.
  - Gọi `h.writer.UpdateConfig` để Re-create connector trên Debezium REST API.
- [ ] Task 1.3: Chạy unit test / `go build ./...` verify Backend compiled clean.

## Phase 2: Frontend Verification & Polish (`cdc-cms-web`)
- [ ] Task 2.1: Rà soát `SourceConnectors.tsx`, kiểm tra `openEdit` cho cả Linked và Orphan Connections.
- [ ] Task 2.2: Verify Modal Edit gửi đầy đủ payload config và hiển thị thông báo thành công.
- [ ] Task 2.3: Verify FE build bằng `npm run build` / `npx tsc --noEmit`.
