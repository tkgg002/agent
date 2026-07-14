# Phân tích kỹ thuật - Khắc phục Hiển thị Dữ liệu ID Diff

## 1. Nguyên nhân
Hàm `GetTableHistory` sử dụng câu lệnh `UNION ALL` để gộp bản ghi từ `cdc_reconciliation_report` và `cdc_recon_smoke_result`. Trong danh sách chiếu (SELECT list) của UNION:
- Chưa liệt kê các trường chứa mảng ID diff: `missing_ids`, `stale_ids`, `field_diffs`, và các chỉ số thống kê heal.
- Vì không có trong danh sách SELECT của subquery, GORM không thể scan các giá trị này vào struct models, dẫn tới các trường này luôn bị `null` hoặc `0` ở kết quả API.

## 2. Giải pháp sửa đổi
Bổ sung đầy đủ các cột tương ứng vào SELECT của 2 bảng trong UNION. Với bảng `cdc_recon_smoke_result` (không lưu các chi tiết này), ta gán các giá trị giả lập kiểu dữ liệu phù hợp (ví dụ: `NULL::jsonb` cho các trường JSONB và `0::integer` cho các trường INTEGER) để đảm bảo câu lệnh UNION chạy chuẩn xác trên PostgreSQL.
