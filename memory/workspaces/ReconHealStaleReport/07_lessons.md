# Lessons Learned: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## 1. Cần kiểm tra cột thời gian trong SQL Mock
- **Bài học**: Khi logic code Go sử dụng `time.Since()` hoặc so sánh thời gian dựa trên các trường của struct được scan từ database, unit test sử dụng `sqlmock` buộc phải cung cấp cột thời gian đó với giá trị giả định hợp lệ.
- **Hậu quả nếu thiếu**: GORM gán giá trị zero-time (0001-01-01) cho trường. Phép so sánh `time.Since(zeroTime)` trả về khoảng thời gian lớn hơn 50 năm, kích hoạt nhánh logic chạy check động và gây lỗi panic do thiếu mock dependencies ở các hàm liên quan.

## 2. Quy trình Governance Workspace-First Rule
- **Bài học**: Luôn khởi tạo đầy đủ cấu trúc thư mục workspace và các tài liệu liên quan trước khi thực hiện chỉnh sửa source code hay chạy test. Việc thiếu các tệp tin mô tả yêu cầu, thiết kế và tiến độ vi phạm nghiêm trọng quy chuẩn quản trị dự án quy mô lớn.
