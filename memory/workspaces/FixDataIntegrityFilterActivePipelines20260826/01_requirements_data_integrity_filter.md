# Yêu cầu Chi tiết: Sửa lỗi Filter Active Pipelines tại /data-integrity

## Context
Trang `http://localhost:5173/data-integrity` hiện tại hiển thị toàn bộ các Pipelines cũ (bao gồm cả các pipelines đã bị tắt/xóa/inactive), mặc dù trên hệ thống thực tế chỉ có đúng 1 Pipeline đang hoạt động.

## Specs
1. Điều tra API / Query / UI logic load danh sách Pipelines ở trang `/data-integrity`.
2. Đảm bảo UI / API lọc đúng các Pipelines đang hoạt động (active / enabled) hoặc chỉ hiển thị các pipelines hợp lệ theo trạng thái thực tế.
3. Không làm gãy các tính năng khác trên Data Integrity Dashboard.
