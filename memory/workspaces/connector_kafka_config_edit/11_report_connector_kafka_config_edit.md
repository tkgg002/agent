# Báo cáo thay đổi chi tiết - Thêm input Kafka Config khi Edit Connector

## Danh sách tệp tin thay đổi
*   [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx)

## Thống kê số dòng code thay đổi (Git Diff Stats)
*   **Số lượng tệp tin:** 1 file
*   **Insertions (+):** 120 dòng
*   **Deletions (-):** 35 dòng
*   **Tổng số dòng thay đổi:** 155 dòng

## Tóm tắt nội dung thay đổi
1.  **Interface ConnectionFormValues:** Thêm thuộc tính tùy chọn `kafkaConfig` để lưu giá trị cấu hình Kafka custom từ Form.
2.  **Khởi tạo hằng số & Helper Function:**
    *   `NATIVE_CONFIG_KEYS`: Danh sách Set chứa các keys cấu hình mặc định được quản lý qua các ô nhập liệu tĩnh trên giao diện Form.
    *   `extractKafkaConfig`: Lọc các key không thuộc `NATIVE_CONFIG_KEYS` trong config của Connector hiện tại (như `producer.override.max.request.size`, `producer.override.compression.type`,...) để đưa ra ô nhập liệu custom Kafka Config dưới dạng JSON string.
3.  **Hàm buildConnectorConfig:** Thay đổi cách cấu trúc map config từ return compactConfig trực tiếp sang gán vào biến `config`. Cuối hàm, nếu có `values.kafkaConfig`, tiến hành `JSON.parse` và gộp đè các key-value này vào `config` trước khi return.
4.  **Hàm openCreate, openEdit, openEditFromConnector:** Bổ sung việc trích xuất và thiết lập giá trị trường `kafkaConfig` khi mở Modal chỉnh sửa hoặc thêm mới.
5.  **Form Item (JSX):** Thêm ô nhập liệu `Kafka Config` (TextArea) ngay trên trường `Reason`, hỗ trợ validator định dạng JSON hợp lệ để đảm bảo an toàn dữ liệu.
