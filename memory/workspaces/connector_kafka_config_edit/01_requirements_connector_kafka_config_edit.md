# Yêu cầu: Bổ sung input "Kafka Config" khi Edit Connector

Hiện tại, khi thực hiện edit connector, cấu hình `max.partition.fetch.bytes` đang bị gán tĩnh (hardcode) là 2MB. 
Yêu cầu:
1. Thay đổi phần chỉnh sửa (edit) connector.
2. Thêm một ô nhập liệu (input) có nhãn "Kafka Config" (nằm phía trên trường "Reason" trong form edit).
3. Khi người dùng lưu (save), cập nhật giá trị từ ô nhập liệu này vào cấu hình connector.

## Phạm vi tác động
- Giao diện Frontend (FE) nơi chỉnh sửa connector (form edit connector).
- Logic API Backend (BE) nhận request cập nhật connector và ghi nhận cấu hình Kafka Config vào DB / Connectors.
