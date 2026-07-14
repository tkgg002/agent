# Danh sách Task: Thêm Chức năng Xoá Phiên Đối Soát (cdc_reconciliation_report)

## Task List
- [x] Task 1: Sửa đổi `internal/api/recon/reconciliation_handler.go` và constructor để tiêm `db *gorm.DB`.
- [x] Task 2: Tạo file handler mới `internal/api/recon/reconciliation_handler_delete_report.go` chứa logic xoá report.
- [x] Task 3: Sửa đổi `internal/server/server.go` để truyền `db` khi khởi tạo `ReconciliationHandler`.
- [x] Task 4: Đăng ký route `DELETE` trong `internal/router/router.go`.
- [x] Task 5: Thêm mutation `useDeleteReportMutation` trong `cdc-cms-web/src/hooks/useReconStatus.ts`.
- [x] Task 6: Tích hợp nút Xoá và logic xác nhận vào bảng "Phiên chưa xử lý" trong `cdc-cms-web/src/components/ExecuteHealModal.tsx`.
- [x] Task 7: Biên dịch backend, check frontend và chạy thử nghiệm thực tế.
- [x] Task 8: Cập nhật helper `auditHeaders` để encode `X-Action-Reason` (chống lỗi setRequestHeader do tiếng Việt).
- [x] Task 9: Tích hợp chọn từng phiên chữa lành (row selection) vào bảng và gửi danh sách report ID được chọn chữa lành.
- [x] Task 10: Sửa đổi nút Xóa để bỏ qua validation lý do thủ công khi xóa.
- [x] Task 11: Giải quyết lỗi infinite loop (Maximum update depth exceeded) ở component `ExecuteHealModal.tsx` do dependency array không ổn định.
- [x] Task 12: Sửa lỗi không hiển thị phiên đối soát đã xử lý (healed reports) hỗ trợ trạng thái partially_healed ở UI và thêm onSuccess invalidate queries cho useExecuteHealMutation.
