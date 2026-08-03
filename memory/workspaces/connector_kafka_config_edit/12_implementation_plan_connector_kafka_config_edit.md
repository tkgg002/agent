# Kế hoạch Triển khai AI - Bổ sung input Kafka Config khi Edit Connector

## Mục tiêu
Tích hợp trường cấu hình "Kafka Config" động thay thế cho giá trị hardcode `max.partition.fetch.bytes` hiện tại trong giao diện và API chỉnh sửa Connector.

## Các bước thực hiện dự kiến
1. **Tìm kiếm (Research):**
   - Tìm kiếm `max.partition.fetch.bytes` hoặc `max.partition.fetch.bytes` trong cả hai workspace `/Users/trainguyen/Documents/work/agent` và `/Users/trainguyen/Documents/work/data-hub`.
   - Xác định file UI Frontend chứa modal/form edit connector.
   - Xác định file API Backend xử lý lưu/cập nhật connector.
2. **Thiết kế giải pháp (Solution Design):**
   - Cấu trúc trường input mới "Kafka Config" (có thể dạng text hoặc JSON object tùy theo thiết kế hiện tại của project).
   - Truyền trường này qua payload API lên Backend.
   - Cập nhật connector config ở Backend để áp dụng các cấu hình Kafka động này.
3. **Phê duyệt:**
   - Trình bày giải pháp trong `09_tasks_solution_connector_kafka_config_edit.md` và xin ý kiến User.
4. **Thực thi (Muscle):**
   - Giao việc cho Muscle/Sub-agent chỉnh sửa.
5. **Kiểm thử & Báo cáo:**
   - Verify code biên dịch và chạy đúng.
