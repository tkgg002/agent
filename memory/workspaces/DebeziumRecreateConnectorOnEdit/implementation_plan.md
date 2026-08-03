# Kế Hoạch Kỹ Thuật: Tự Động Re-create Debezium Connector Khi Save Connection

Anh đã nêu rất đúng nhu cầu thực tế: Khi đổi instance Debezium, các Connector trên Debezium REST API bị mất nhưng dữ liệu Connection vẫn được lưu an toàn trong Postgres DB (`cdc_system.source_fingerprints`). Giải pháp tối ưu nhất là khi click **Edit** và **Save** một Connection trên UI, hệ thống sẽ **tự động khởi tạo lại (Re-create) Connector** tương ứng trên Debezium mà không bắt anh phải xóa rồi tạo lại thủ công.

---

## 1. Nguyên Nhân Gốc Rễ Đang Xảy Ra (Root Cause)

1. Khi click **Edit** & **Save** một Connection, Frontend gửi `PATCH /api/v1/system/connectors/:name/config`.
2. Tại Backend Go (`UpdateSystemConnectorConfigHandler.Handle`), hệ thống gọi `h.writer.GetConfig(ctx, cmd.Name)` để lấy cấu hình hiện tại từ Debezium REST API nhằm merge với cấu hình mới gửi lên.
3. Vì Debezium đã đổi/reset, REST API trả về lỗi **404 Not Found** (`ErrKafkaConnectNotFound`).
4. Backend văng lỗi 502 `connector_update_failed`, làm giao diện báo lỗi và không thể Save.

---

## 2. Phương Án Giải Quyết Tối Ưu (Single Best Approach)

### A. Backend Go (`cdc-cms-service`)
1. **Xử lý Graceful Fallback 404 trong `UpdateSystemConnectorConfigHandler`**:
   - Khi `h.writer.GetConfig` trả về `ErrKafkaConnectNotFound`, Backend không ném lỗi mà coi `current = map[string]string{}` (config cũ rỗng).
2. **Tự động Khôi phục Mật khẩu cũ từ DB**:
   - Nếu trong form Save có trường mật khẩu mang giá trị giữ nguyên `__KEEP__` (`database.password` hoặc `mongodb.connection.string`), Backend sẽ query `source_fingerprints` từ DB theo tên Connection để lấy `options_json` (chứa username/password cũ) và điền lại vào config.
3. **Tự động Re-create Connector trên Debezium**:
   - Gọi Kafka Connect REST API `PUT /connectors/:name/config` với full payload config mới. Kafka Connect REST API có cơ chế native: **Nếu connector chưa có, nó sẽ tự động CREATE MỚI connector**.
   - Cập nhật lại trạng thái record trong DB `source_fingerprints` sang `created`.

### B. Frontend (`cdc-cms-web`)
1. **Đảm bảo Modal Edit điền đủ thông tin từ DB**:
   - Khi bấm Edit một Connection thuộc danh sách chưa có trên Debezium (`orphanFingerprint`), Modal Edit tự động nạp đầy đủ thông tin từ DB (`SourceFingerprint`).
2. **Gửi Full Payload Config**:
   - Khi bấm **Save**, `buildConnectorConfig` xây dựng đầy đủ payload config (bao gồm `connector.class`, converter, signal topic...) và gửi request `PATCH /api/v1/system/connectors/:name/config`.
3. **Thông báo Trực quan**:
   - Hiển thị thông báo thành công: *"Đã cập nhật & tự động khởi tạo lại Connector thành công!"*.

---

## 3. Các File Sẽ Chỉnh Sửa

### Backend (`cdc-cms-service`)
- [debezium_connector.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go): Cập nhật `UpdateSystemConnectorConfigHandler.Handle` bắt lỗi `ErrKafkaConnectNotFound`, khôi phục credential cũ từ DB và tự động Re-create connector trên Debezium.

### Frontend (`cdc-cms-web`)
- [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx): Rà soát logic `openEdit` cho Orphan Connections và hiển thị thông báo thành công sau khi Save.

---

## 4. Kế Hoạch Kiểm Thử (Verification Plan)

### Automated Checks
- Chạy biên dịch Backend: `go build ./...` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`.
- Chạy kiểm tra Frontend: `npx tsc --noEmit` và `npm run build` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`.

### Manual Verification
- Kiểm tra Save một Connection khi Connector không tồn tại trên Debezium REST API -> Xác nhận API trả về 200 OK và Debezium Connector được khởi tạo lại thành công.
