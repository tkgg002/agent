# Kế hoạch Triển khai: Bổ sung input Kafka Config khi Edit Connector

Bổ sung trường "Kafka Config" động vào Form chỉnh sửa/thêm mới Connector trên giao diện Web Frontend, cho phép quản trị viên cập nhật linh hoạt cấu hình Kafka (ví dụ như tăng kích thước message `producer.override.max.request.size` từ 2MB lên 10MB) mà không bị ghi đè tĩnh trong code.

## Proposed Changes

### Frontend (Web Component)

#### [MODIFY] [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx)
- Bổ sung trường `kafkaConfig?: string;` vào interface `ConnectionFormValues`.
- Thêm tập hợp `NATIVE_CONFIG_KEYS` liệt kê toàn bộ các key cấu hình Debezium mặc định đang được quản lý tĩnh thông qua Form UI.
- Thêm hàm helper `extractKafkaConfig(cfg)` để trích xuất các cấu hình Kafka custom (các key không nằm trong `NATIVE_CONFIG_KEYS`) dưới dạng chuỗi JSON string thụt lề thụt dòng.
- Cập nhật hàm `buildConnectorConfig` để tự động giải mã `kafkaConfig` từ form và merge (ghi đè) vào cấu hình Debezium config trước khi POST/PATCH lên Backend.
- Cập nhật các hàm `openCreate`, `openEdit`, và `openEditFromConnector` để trích xuất và set giá trị cho trường `kafkaConfig` của form.
- Thêm trường `kafkaConfig` dưới dạng `Input.TextArea` hỗ trợ validator JSON hợp lệ nằm ngay trên trường `Reason` của form.

---

## Verification Plan

### Automated Tests
- Do đây là thay đổi về UI Frontend, không có unit test cụ thể cho component này. Tuy nhiên, ta sẽ chạy build dự án Frontend để xác nhận không lỗi cú pháp hoặc TypeScript compiler error.
- Chạy lệnh build của Vite trong `cdc-cms-web`:
  `npm run build` (hoặc build tĩnh bằng `tsc --noEmit`).

### Manual Verification
1. Mở Modal "Edit Config" của một Connector có sẵn.
2. Kiểm tra xem trường "Kafka Config" đã hiển thị trên trường "Reason" chưa.
3. Kiểm tra xem các thuộc tính config custom (ví dụ: `producer.override.max.request.size` và `producer.override.compression.type`) có được tự động trích xuất vào ô nhập liệu dưới dạng JSON hay không.
4. Thử cập nhật giá trị `producer.override.max.request.size` từ `2097152` thành `10485760` (10MB) và nhập Reason (tối thiểu 10 ký tự), sau đó bấm Update.
5. Kiểm tra request API PATCH bắn lên xem payload config gửi lên đã chứa giá trị mới hay chưa.
