# Plan: Fix Missing Debezium PG Plugin 2026-06-23

## Kế hoạch hành động

1. **Tìm hiểu cách thức vận hành Kafka Connect ở server `10.200.186.203`** [ĐÃ XONG]:
   - Xác định: Kafka Connect chạy dưới dạng Pod trong Kubernetes namespace `data-hub` (Service `cdc-kafka-connect-connect`).
   - Cổng kết nối REST API là `8083`. Máy local có thể gọi trực tiếp sang `10.200.186.203:8083`.

2. **Xác minh quyền truy cập và cài đặt plugin** [ĐÃ XONG]:
   - Kiểm tra SSH: Cổng 22 bị từ chối kết nối (`Connection refused`).
   - Kiểm tra `kubectl` local: File kubeconfig trống rỗng, không có context hợp lệ kết nối sang cluster.
   - Các file deploy Kafka Connect nằm ở một repository hạ tầng khác (không thuộc các active workspace hiện tại).

3. **Cung cấp giải pháp cho User & Đợi phê duyệt** [ĐÃ XONG]:
   - Viết tài liệu `implementation_plan.md` mô tả chi tiết lỗi và hướng dẫn bạn tự thực thi lệnh cài đặt plugin Debezium Postgres.
   - User phản hồi: Đã cài được plugin thành công nhưng gặp lỗi `permission denied to start WAL sender` trên PostgreSQL.

4. **Xử lý lỗi phân quyền WAL sender (PostgreSQL)** [ĐANG THỰC HIỆN]:
   - Xác định nguyên nhân: User `cdc_user` thiếu thuộc tính `REPLICATION` trên database nguồn PostgreSQL `10.200.186.203:5432`.
   - Cập nhật `implementation_plan.md` với các lệnh SQL cần thiết (`ALTER ROLE cdc_user WITH REPLICATION;`) để user chạy trên database bằng tài khoản superuser.
   - Đợi user chạy lệnh SQL cấp quyền và restart connector.

5. **Xác minh sau khi cài đặt**:
   - Gọi API GET `http://10.200.186.203:8083/connectors/pg_dev/status` từ máy local để kiểm tra xem connector đã ở trạng thái `RUNNING` chưa.
