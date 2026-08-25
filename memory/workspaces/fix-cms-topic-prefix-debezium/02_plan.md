# 02 - Plan: Lộ trình chuẩn hóa Topic Prefix

## 1. Mục tiêu
Chuẩn hóa logic sinh topic prefix tại frontend `cdc-cms-web` để đảm bảo topic Kafka sinh ra bởi Debezium có đúng cấu trúc 4-segment `{prefix}.{database}.{collection}`, triệt tiêu lỗi duplicate segment, đồng thời mở quyền tùy biến cho người dùng khi có va chạm tên.

## 2. Kế hoạch thực hiện (Phases)
- **Phase 1: Phân tích & Định vị mã nguồn**
  - Xác định các vị trí gán chuỗi `.${connector_name}` trong `SourceConnectors.tsx` (dòng 394, 435, 484-490, 1596).
- **Phase 2: Thiết kế giải pháp kỹ thuật (Task Solution & Code Demo)**
  - Tách bạch hành vi giữa SFTP (`kafka-connect-fs`) và Debezium Connectors.
  - Bật quyền sửa cho input `topicPrefix` kèm tooltip hướng dẫn.
- **Phase 3: Thực thi & Kiểm thử**
  - Áp dụng các thay đổi vào `src/pages/SourceConnectors.tsx`.
  - Chạy `npx tsc --noEmit` xác thực toàn vẹn type.
- **Phase 4: Báo cáo & Audit**
  - Cập nhật tài liệu workspace, ghi nhận bài học vào `lessons.md`.
