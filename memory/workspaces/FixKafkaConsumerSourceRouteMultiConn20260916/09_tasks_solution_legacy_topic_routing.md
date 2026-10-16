# 09_tasks_solution_legacy_topic_routing.md

## Hồ sơ Giải pháp Kỹ thuật Chi tiết

### 1. Kiến trúc Nguồn Chân Lý
- Thay vì hardcode cấu hình hoặc sửa DB thủ công, hệ thống khai thác trực tiếp bảng `cdc_system.sources`.
- Bảng `cdc_system.sources` là nơi CMS lưu trữ toàn bộ thông tin đăng ký connector:
  - `connector_name`: Định danh kết nối (tương ứng `connection_code` trong `cdc_system.connection_registry`).
  - `topic_prefix`: Tiền tố topic Kafka mà connector Debezium phát sinh.

### 2. Thuật toán Longest Prefix Match
- Khi một topic Kafka đến worker (ví dụ `cdc.goopaylocal.core-trans-proxy-history-service.trans-his`), hàm `ResolveConnectionByTopic` duyệt qua các tiền tố đã nạp:
  - So sánh `strings.HasPrefix(topic, pre + ".") || topic == pre`.
  - Chọn prefix có độ dài lớn nhất (`len(pre)` lớn nhất) để tránh việc prefix ngắn (`cdc.goopaylocal`) nuốt nhầm prefix dài (`cdc.goopaylocal.traitestmongodevct`).
- Độ phức tạp: $O(N)$ với $N$ là số lượng connector (thường < 50), chi phí CPU thực thi < 1 microsecond mỗi message.

### 3. Tương thích ngược (Backward Compatibility)
- Nếu topic không khớp bất kỳ connector prefix nào trong bảng `cdc_system.sources`, hệ thống tự động fallback sang cơ chế bóc tách segment:
  - 5 segments: lấy segment thứ 3 (`parts[2]`).
  - 4 segments: nếu segment thứ 2 không phải generic prefix thì lấy `parts[1]`.
  - Nếu vẫn rỗng: giữ nguyên rỗng để `ResolveSourceRoutes` trả về toàn bộ routes của `(db, table)` như thiết kế truyền thống (không làm gián đoạn các luồng đơn kết nối đang chạy).
