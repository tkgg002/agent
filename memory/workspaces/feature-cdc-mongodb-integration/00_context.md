# Requirements: MongoDB Pre-flight Check & Integration

## 1. MongoDB Requirements Check (Pre-flight)
Hệ thống phải kiểm tra các điều kiện sau trước khi cho phép kết nối Debezium:
- [ ] **Replica Set**: MongoDB phải chạy ở chế độ Replica Set (bắt buộc cho CDC/Change Streams).
- [ ] **Auth/Permissions**: User cung cấp phải có quyền `readAnyDatabase`, `clusterMonitor` (hoặc cụ thể hơn tùy config).
- [ ] **Network Connectivity**: Worker có thể reach được MongoDB host/port.
- [ ] **Oplog access**: Có thể đọc được collection `local.oplog.rs`.

## 2. Integration Flow
- [ ] CMS-FE: Thêm Form đăng ký MongoDB (Host, Port, User, Password, DB, ReplicaSet Name).
- [ ] CMS-API: Endpoint `/api/v1/sources/check` để trigger pre-flight check.
- [ ] CMS-API: Sau khi check pass, gọi `admin-api` để register source.
- [ ] CMS-FE: Duyệt các collection → Materialize Shadow.
# Context: MongoDB CDC Integration Flow

## Goal
Tích hợp MongoDB làm Source DB cho hệ thống CDC, cho phép người dùng đăng ký qua CMS-FE, thực hiện kiểm tra cấu hình DB (Pre-flight check) trước khi kết nối Debezium.

## Components Involved
- **CMS-FE**: Giao diện đăng ký source, hiển thị kết quả check.
- **CMS-API**: Xử lý logic đăng ký, gọi Worker thực hiện check.
- **CDC-Worker**: Thực hiện truy vấn thực tế vào MongoDB để kiểm tra (ReplicaSet, Oplog, Auth).
- **Debezium**: Connector cho MongoDB.
