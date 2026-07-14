# Yêu cầu: Sửa lỗi 500 Endpoint Lịch sử Đối soát (schedule_histories)

## 1. Mô tả bài toán
Endpoint `GET /api/reconciliation/report/schedule_histories?page=1&page_size=30&shadow_schema=shadow_testss&master_table=schedule_histories` trả về lỗi 500 Internal Server Error.

## 2. Mục tiêu
- Tìm ra nguyên nhân gốc rễ gây lỗi 500 khi chạy SQL query trong `GetTableHistory` của `recon_read_repo_gorm.go`.
- Sửa lỗi đảm bảo query hoạt động ổn định và trả về kết quả đúng cấu trúc `ReconciliationReport`.
- Viết integration test kết nối DB thật để tái hiện lỗi (Red) và xác minh sau khi sửa thành công (Green).
- Đảm bảo không phá vỡ bất kỳ logic hoặc cấu trúc hiện tại nào của hệ thống.
