# Yêu cầu: Khắc phục logic tự sửa đổi index (self-healing index) trong Transmuter & Bổ sung đề xuất trên UI

## 1. Vấn đề thiết kế (Architectural Issue)
- **Cơ chế hiện tại (Self-healing in runtime):** Khi transmuter chạy, nó tự động kiểm tra xem shadow table có index `idx_<table_name>_source_id` hay không. Nếu thiếu hoặc invalid, transmuter tự động chạy `CREATE INDEX CONCURRENTLY` ngầm dưới nền.
- **Lỗ hổng thiết kế (Deadlock & Starvation):**
  - Câu lệnh `CREATE INDEX CONCURRENTLY` bắt buộc phải đợi tất cả các transaction hiện tại (cả đọc và ghi) trên bảng đó hoàn thành mới có thể build thành công.
  - Trong lúc transmuter đang chạy liên tục (realtime CDC trigger liên tục), các transaction đọc/ghi diễn ra không ngừng.
  - Do đó, câu lệnh `CREATE INDEX CONCURRENTLY` do transmuter tự spawn dưới nền sẽ **luôn luôn bị block**, dẫn đến treo, timeout và chuyển sang trạng thái `INVALID` vĩnh viễn. Việc tự động sửa ở runtime trong transmuter là một "fix bẩn" (hacky fix) và là tác nhân gây lock-storm nghiêm trọng.
- **Giải pháp chuẩn:**
  - **Tách biệt vai trò (Separation of Concerns):** Loại bỏ việc thực thi DDL tạo index tự động ra khỏi hot-path của transmuter. Transmuter chỉ làm nhiệm vụ đồng bộ dữ liệu. Nếu phát hiện thiếu index, transmuter chỉ ghi log cảnh báo (`Warn`) để người vận hành biết.
  - **Đề xuất và tạo trước (Check & Recommend on CMS UI):** Cung cấp API hoặc bổ sung thông tin trong lệnh `introspect-indexes` để CMS UI hiển thị cảnh báo thiếu/invalid index `_source_id` và cho phép người vận hành chủ động bấm nút tạo index trước khi kích hoạt sync (hoặc tạm dừng sync để tạo index an toàn).

## 2. Các yêu cầu chi tiết (Definition of Done - DoD)
- **DoD 1:** Gỡ bỏ hoàn toàn logic tự động tạo index `CREATE INDEX CONCURRENTLY` và `DROP INDEX CONCURRENTLY` trong hàm `ensureShadowSourceIDIndex` của `transmuter.go`. Thay vào đó:
  - Chỉ thực hiện truy vấn kiểm tra index.
  - Nếu thiếu hoặc invalid, ghi log cảnh báo `Warn` (sử dụng cache để chỉ log 1 lần tránh spam log).
- **DoD 2:** Cập nhật `IndexManager` và `IndexHandler` để bổ sung tính năng gợi ý/đề xuất index (Index Recommendations):
  - Khi API `introspect-indexes` được gọi, ngoài danh sách index hiện tại, API phải trả về danh mục `recommendations`.
  - Nếu là shadow table và thiếu hoặc invalid index trên `_source_id`, tự động trả về đề xuất tạo index `idx_<table_name>_source_id`.
  - Nếu thiếu hoặc invalid index trên `_deleted` (dành cho CountDeletedRows), trả về đề xuất tạo partial index `idx_<table_name>_deleted_partial` hoặc `idx_<table_name>__deleted`.
- **DoD 3:** Viết unit test kiểm chứng logic đề xuất index trong `index_manager_test.go` hoạt động chính xác.
- **DoD 4:** Không tự ý chạy git commit.
