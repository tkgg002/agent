# Walkthrough: CDC Worker Queue Monitoring

Tôi đã hoàn thành việc triển khai giải pháp giám sát tiến trình Queue và Buffer cho CDC Worker. Giờ đây bạn có thể theo dõi trạng thái hệ thống thực tế thông qua giao diện CMS.

## Thay đổi chính

### 1. Backend Instrumentation (Go)
- **Consumer Pool**: Đã thêm các bộ đếm `atomic` để theo dõi số lượng tin nhắn đã xử lý (`processed`), thất bại (`failed`) và số lượng worker đang hoạt động (`active_workers`).
- **Batch Buffer**: Đã thêm cơ chế theo dõi thời gian `flush` cuối cùng và kích thước buffer hiện tại.

### 2. API Monitoring
- Đã mở endpoint nội bộ: `GET http://localhost:8082/api/v1/internal/stats`.
- Endpoint này cung cấp cái nhìn tổng thể về hiệu suất của Worker mà không làm ảnh hưởng đến luồng xử lý chính.

### 3. Frontend Dashboard (React + Ant Design)
- **Trang mới**: `Queue Monitor` đã được thêm vào Sidebar.
- **Tính năng**:
  - Tự động tải lại dữ liệu (Polling) mỗi 2 giây.
  - Hiển thị biểu đồ trạng thái Worker Pool (Công suất đang sử dụng).
  - Hiển thị trạng thái Batch Buffer (Độ trễ ghi vào Data Warehouse).
  - Cảnh báo trạng thái Offline nếu Worker bị sập.

## Hướng dẫn xác minh

1. **Khởi động Worker**: Đảm bảo Worker đang chạy ở cổng `:8082`.
2. **Truy cập CMS**: Mở trình duyệt vào trang CMS và chọn mục **Queue Monitor** ở menu bên trái.
3. **Kiểm tra chỉ số**:
   - Bạn sẽ thấy các con số `Processed Events` bắt đầu tăng lên khi có dữ liệu CDC chảy qua.
   - Kiểm tra `Active Progress` để xem có bao nhiêu worker đang bận.
   - Kiểm tra `Batch Buffer` để biết dữ liệu có đang bị kẹt trong bộ nhớ hay đã được đẩy vào Postgres thành công.

> [!TIP]
> Nếu bạn thấy `Failed Events` tăng cao, hãy kiểm tra logs của Worker để tìm hiểu nguyên nhân (thường là do lỗi mapping hoặc kết nối DB).

> [!NOTE]
> Các chỉ số hiện tại được lưu in-memory, có nghĩa là chúng sẽ reset về 0 khi bạn restart Worker.
