# Yêu cầu: Thêm Chức năng Xoá Phiên Đối Soát (cdc_reconciliation_report)

## 1. Mô tả yêu cầu
Trên giao diện Chữa lành đối soát -> Phiên chưa xử lý (Unprocessed Sessions), hiện tại chưa có tính năng xoá các phiên đối soát (các bản ghi trong bảng `cdc_system.cdc_reconciliation_report`).
Yêu cầu bổ sung chức năng xoá phiên đối soát chưa xử lý để dọn dẹp các phiên rác hoặc các phiên không cần chữa lành nữa.

Yêu cầu chi tiết:
- **Backend API:** Thêm HTTP endpoint `DELETE /api/reconciliation/report/:id` (và `/api/v1/reconciliation/report/:id`) thuộc nhóm destructive route (cần quyền OpsAdmin và Audit log lý do thực hiện).
- **Frontend Mutation:** Thêm mutation hook `useDeleteReportMutation` trong `useReconStatus.ts` để gọi API xoá.
- **Frontend UI:** Thêm cột "Thao tác" vào bảng "Phiên chưa xử lý" trong `ExecuteHealModal.tsx` với nút Xoá dạng thùng rác nguy hiểm. Khi click, yêu cầu người dùng xác nhận qua Popconfirm. Yêu cầu người dùng điền lý do ở ô textarea bên dưới trước khi thực hiện để đáp ứng Audit Log.

## 2. Definition of Done (DoD)
- Code backend biên dịch thành công (`go build ./cmd/server/...`).
- Code frontend TypeScript check pass (`npx tsc --noEmit`).
- Khi click Xoá trên UI và đã điền lý do, phiên đối soát được xoá khỏi database và bảng trên UI tự động reload lại dữ liệu mới.
