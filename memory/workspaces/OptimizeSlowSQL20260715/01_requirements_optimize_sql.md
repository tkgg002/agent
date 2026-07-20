# Yêu cầu: Tối ưu hóa các câu lệnh SQL chậm (Slow SQL Tuning)

Hệ thống ghi nhận một số câu lệnh SQL chậm (chạy mất hơn 200ms, thậm chí lên tới 1.2s) trong `recon_read_repo_gorm.go`.

## Các câu lệnh SQL cần tối ưu:
1. SQL chậm tại `recon_read_repo_gorm.go:601` (`ListFailedLogs`):
   - Query đếm số lượng log lỗi (`SELECT COUNT(*) FROM (...) AS failed_logs`).
   - Hiện trạng: Chạy mất khoảng 240ms - 250ms.
   - Nguyên nhân sơ bộ: Thực hiện subquery bọc toàn bộ logic query chính bao gồm 2 `LEFT JOIN LATERAL` và các phép JOIN bảng khác mà phép đếm `COUNT(*)` không thực sự cần thiết.

2. SQL chậm tại `recon_read_repo_gorm.go:237` (`ListLatest`):
   - Query lấy các báo cáo đối soát mới nhất (`listLatestPrimary`).
   - Hiện trạng: Chạy mất khoảng 1.2s - 1.3s.
   - Nguyên nhân sơ bộ: Thực hiện `DISTINCT ON` trên kết quả `UNION ALL` của hai bảng lịch sử lớn (`cdc_reconciliation_report` và `cdc_recon_smoke_result`), cùng với các phép `LEFT JOIN LATERAL` và JOIN khác.

## Mục tiêu (DoD):
- Tối ưu hóa hai câu truy vấn trên để giảm thời gian thực thi xuống dưới ngưỡng Slow SQL (200ms).
- Đảm bảo tính chính xác và tương thích ngược của kết quả trả về (không thay đổi cấu trúc dữ liệu trả về cho client).
- Kiểm thử và xác thực hiệu năng của các truy vấn mới.
