# Yêu cầu: Sửa lỗi hiển thị modal Chữa lành đối soát (Interactive Heal Visibility)

## 1. Vấn đề hiện tại
Khi người dùng mở modal "Chữa lành đối soát", danh sách các bản ghi chưa được chữa lành (`unhealed reports`) không hiển thị (mảng trả về rỗng).
- **Lý do**: API `/api/reconciliation/report/:table/unhealed` nhận tham số `:table` dưới dạng Fully Qualified Name (FQN), ví dụ: `shadow_testexp.export_jobs`.
- **Query thực tế**: API gọi hàm `ListUnhealedReports` trong `recon_read_repo_gorm.go`. Hàm này lọc trực tiếp:
  `Where("(shadow_table = ? OR master_table = ?)", table, table)`
  Trong khi đó, ở cơ sở dữ liệu `cdc_reconciliation_report`, cột `shadow_table` chỉ lưu tên bảng thô (`export_jobs`), còn schema được lưu riêng ở cột `shadow_schema` (`shadow_testexp`). Việc so khớp trực tiếp `"export_jobs" = "shadow_testexp.export_jobs"` dẫn đến không có dữ liệu trả về.
- **Vấn đề tương tự**: API lịch sử bảng `/api/reconciliation/report/:table` (gọi hàm `GetTableHistory`) cũng gặp lỗi tương tự khi so khớp `shadow_table = ?` với giá trị FQN.

## 2. Yêu cầu chi tiết
- **Yêu cầu 1**: Chuẩn hóa tham số `table` trong hàm `ListUnhealedReports` của repo GORM. Nếu `table` chứa dấu chấm (`.`), thực hiện split để lấy phần schema làm `shadowSchema` (nếu `shadowSchema` rỗng) và phần tên bảng thô làm `table`.
- **Yêu cầu 2**: Thực hiện chuẩn hóa tương tự cho hàm `GetTableHistory` của repo GORM đối với các tham số `table` và `masterTable`.
- **Yêu cầu 3**: Biên dịch và chạy thử hệ thống để xác nhận không lỗi cú pháp hoặc hồi quy.
