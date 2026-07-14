# Báo cáo Thay đổi (Change Report) - ReconDeleteReport

Tài liệu tổng hợp các file thay đổi, số lượng dòng code và mô tả tổng quan về các thay đổi thực hiện cho tính năng xoá phiên đối soát.

## 1. Danh sách các file đã thay đổi

| File Path | Trạng thái | Số dòng thay đổi (Ước lượng) | Mô tả thay đổi |
| --- | --- | --- | --- |
| `internal/api/recon/reconciliation_handler.go` | Modified | ~30 | Import `"gorm.io/gorm"`, tiêm `db *gorm.DB` vào struct và constructor của `ReconciliationHandler`. |
| `internal/api/recon/reconciliation_handler_delete_report.go` | Created | ~30 | Tạo mới handler `DeleteReport` để xử lý logic xoá phiên đối soát bằng ID từ Database qua GORM. |
| `internal/server/server.go` | Modified | ~2 | Truyền biến `db` vào constructor của `ReconciliationHandler` ở bước khởi tạo handler. |
| `internal/router/router.go` | Modified | ~4 | Đăng ký route `DELETE /api/reconciliation/report/:id` và `DELETE /api/v1/reconciliation/report/:id` thông qua `destructiveChain`. |
| `cdc-cms-web/src/hooks/useReconStatus.ts` | Modified | ~35 | Định nghĩa và sau đó cập nhật mutation hook `useDeleteReportMutation` gửi request xoá kèm audit headers tự động gán lý do mặc định 'Xóa phiên đối soát'. Bổ sung onSuccess invalidation cho `useExecuteHealMutation` để tự động tải lại các query status, history và report khi heal thành công. |
| `cdc-cms-web/src/components/ExecuteHealModal.tsx` | Modified | ~70 | Tích hợp cột "Thao tác" chứa nút Xoá và Popconfirm xác nhận vào bảng "Phiên chưa xử lý". Loại bỏ logic lý do thủ công khi xóa. Sửa lỗi infinite loop và cập nhật healedReports filter bao gồm cả status partially_healed và record đã heal > 0 để hiển thị đầy đủ các phiên đối soát đã xử lý. |

## 2. Kết quả kiểm tra
* **Backend Build**: Biên dịch thành công với lệnh `go build ./cmd/server/...` tại thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`.
* **Frontend TSC**: Kiểm tra kiểu dữ liệu thành công với lệnh `npx tsc --noEmit` tại thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` sau khi cập nhật logic mới.

* **Quy trình Linter**: Chạy `python3 tooling/verify_governance.py` thành công tốt đẹp (`GOVERNANCE AUDIT PASSED`).
