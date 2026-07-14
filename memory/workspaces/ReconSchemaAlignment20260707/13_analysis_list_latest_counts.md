# Phân tích kỹ thuật - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

## 1. Bản chất vấn đề
API `/api/reconciliation/report` (hàm `ListLatest`) chịu trách nhiệm trả về trạng thái tổng quan mới nhất của các luồng đối soát để hiển thị lên Dashboard.
Do `cdc_reconciliation_report` (kết quả chạy Full Search) và `cdc_recon_smoke_result` (kết quả chạy Smoke Test định kỳ) được gộp lại bằng `UNION ALL` rồi `DISTINCT ON` lấy dòng mới nhất, khi vừa chạy Full Search xong, bản ghi Full Search sẽ đại diện cho luồng đối soát đó.
Tuy nhiên, Full Search chạy trên một cửa sổ thời gian (window) hữu hạn (ví dụ: 30 ngày qua), dẫn đến cột `source_count` và `dest_count` chỉ ghi nhận số lượng thay đổi trong cửa sổ này (ví dụ: 8 bản ghi).
Dashboard hiển thị counts này làm active counts cho toàn bảng, gây ra hiện tượng không nhất quán số lượng với các segment khác (ví dụ: segment `shadow_master` vẫn hiển thị 457 bản ghi) dẫn đến thông tin hiển thị bị sai lệch nghiêm trọng.

## 2. Giải pháp kỹ thuật
Để đảm bảo số lượng hiển thị trên Dashboard luôn là tổng số lượng thực tế của bảng (thông tin này chỉ được cập nhật chính xác qua các lượt chạy Smoke Test quét toàn bộ bảng), ta sử dụng `LEFT JOIN LATERAL` ở nhánh `cdc_reconciliation_report` để tham chiếu đến bản ghi smoke test mới nhất của cùng một pipeline (`shadow_schema, shadow_table, segment`).
Sử dụng toán tử `IS NOT DISTINCT FROM` trên PostgreSQL để so sánh các cột nullable (`master_schema`, `master_table`) một cách an toàn.
Các hàm `COALESCE` được sử dụng để fallback về counts của chính báo cáo nếu hệ thống chưa có bản ghi smoke test nào.
