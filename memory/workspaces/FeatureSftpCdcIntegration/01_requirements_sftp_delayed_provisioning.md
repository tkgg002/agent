# Yêu cầu: Trì hoãn khởi tạo SFTP Connector theo luồng Active Binding chuẩn

## Bối cảnh & Hiện trạng
Hệ thống quản lý liên kết dữ liệu hoạt động theo luồng:
1. **Tạo Sync Table (Tạo Shadow Binding):** Hệ thống tạo bảng Shadow trong DB với các cột mặc định.
2. **Scan Fields:** Quét dữ liệu mẫu để sinh đề xuất Mapping Rules.
3. **Duyệt Mapping Rules:** Đồng bộ cấu trúc, thêm các cột nghiệp vụ vào bảng Shadow đã tồn tại.
4. **Active Binding:** Bấm kích hoạt để bắt đầu đồng bộ dữ liệu.

Tuy nhiên, đối với SFTP, connector đang bị tạo và khởi chạy quá sớm trên Kafka Connect (ngay từ bước khai báo Connection). Điều này khiến dữ liệu snapshot bị đẩy lên Kafka và bị Worker tiêu thụ/skip sớm khi bảng Shadow chưa được đồng bộ cấu trúc cột nghiệp vụ và liên kết chưa Active, gây mất dữ liệu snapshot ban đầu.

## Yêu cầu sửa đổi luồng nghiệp vụ Backend (cdc-cms-service)
1. **Trì hoãn tạo Connector:** 
   - Khi người dùng cấu hình SFTP connection, hệ thống chỉ lưu cấu hình vào bảng `ConnectionRegistry`. Tuyệt đối không tạo connector trên Kafka Connect ở bước này.
2. **Khởi chạy Connector tại bước Active Binding:**
   - Khi người dùng nhấn nút **"Active Binding"** (sau khi đã tạo Sync Table và duyệt Mapping Rules), hệ thống mới thực hiện gửi cấu hình và khởi chạy SFTP Connector trên Kafka Connect.
