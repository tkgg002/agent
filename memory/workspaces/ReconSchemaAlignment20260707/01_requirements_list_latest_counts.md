# Yêu cầu - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

## 1. Bối cảnh
Khi chạy đối soát sâu (Full Search / Full Diff), hệ thống ghi nhận kết quả vào bảng `cdc_reconciliation_report`.
Tuy nhiên, API `ListLatest` (lấy danh sách trạng thái mới nhất cho mỗi pipeline) sử dụng một câu lệnh `UNION ALL` gộp kết quả của `cdc_reconciliation_report` và `cdc_recon_smoke_result`, sau đó `DISTINCT ON` theo cặp schema/table/segment lấy dòng mới nhất.
Khi một bản ghi `cdc_reconciliation_report` (Full Search) mới hơn bản ghi smoke check, nó được hiển thị lên Dashboard.
Do Full Search chỉ quét một khoảng thời gian giới hạn (ví dụ: 30 ngày qua), nên cột `source_count` và `dest_count` trong báo cáo này chỉ phản ánh số lượng bản ghi của khoảng thời gian đó (ví dụ: 8 bản ghi), thay vì số lượng bản ghi thực tế của toàn bảng (ví dụ: 457 bản ghi).
Điều này làm cho số lượng hiển thị trên dashboard bị sai lệch (Source: 8, Shadow: 8, Master: 457), dẫn đến tính toán sai số chênh lệch (`transmute: +449 thừa`) và báo trạng thái "Lệch" sai thực tế.

## 2. Mục tiêu
- Điều chỉnh câu truy vấn `listLatestPrimary` trong hàm `ListLatest` của repository `recon_read_repo_gorm.go`.
- Đối với các dòng dữ liệu có nguồn gốc từ `cdc_reconciliation_report`, số lượng bản ghi (`source_total`, `source_active`, `shadow_total`, `shadow_active`, `master_total`, `master_active`, `source_count`, `dest_count`) phải được lấy từ bản ghi `cdc_recon_smoke_result` mới nhất của pipeline đó (sử dụng `LEFT JOIN LATERAL` tìm smoke result mới nhất).
- Tránh việc hiển thị số lượng của cửa sổ thời gian Full Search làm nhiễu thông tin trạng thái tổng thể.

## 3. Definition of Done (DoD)
- [ ] Hàm `ListLatest` lấy đúng các trường active/total counts từ `cdc_recon_smoke_result` khi dòng mới nhất là của `cdc_reconciliation_report`.
- [ ] Dự án compile thành công.
- [ ] Chạy unit test suites của queries thành công.
- [ ] Xác minh kết quả qua API `ListLatest` trả về đúng số lượng thực tế của smoke test.
