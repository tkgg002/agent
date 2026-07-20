# Phân tích kỹ thuật

## 1. Dữ liệu Backend
- API `/api/reconciliation/report/:table/unhealed` được định nghĩa trong route dual-stack của backend:
  `dual("GET", shared, "/reconciliation/report/:table/unhealed", h.Recon.GetUnhealedReports)`
- Hàm `ListUnhealedReports` của repository backend sử dụng GORM raw query hoặc `.Find(&reports)` để lấy dữ liệu từ bảng `cdc_reconciliation_report`.
- Bảng này lưu trữ `source_count` và `dest_count` kiểu `bigint`. Struct Golang `ReconciliationReport` ánh xạ các trường này sang JSON với tag `json:"source_count"` và `json:"dest_count"`.
- Vì thế, API backend đã trả về đầy đủ hai trường này, Frontend chỉ việc map và hiển thị lên giao diện.

## 2. Thay đổi phía Frontend
- `useReconStatus.ts`: Cần bổ sung kiểu dữ liệu để TypeScript không báo lỗi khi truy cập các thuộc tính này trên đối tượng thuộc kiểu `UnhealedReport`.
- `ExecuteHealModal.tsx`:
  - `reportColumns` là mảng chứa cấu hình cột cho bảng "Phiên chưa xử lý". Ta cần chèn thêm 2 cột "Nguồn" (source_count) và "Đích" (dest_count).
  - Sử dụng hàm `.toLocaleString()` để hiển thị số dễ đọc hơn (ví dụ: `1,234` thay vì `1234`).
  - Hỗ trợ cuộn ngang cho table bằng thuộc tính `scroll={{ x: 'max-content', y: 200 }}` để đảm bảo tính mỹ thuật khi thêm cột.
