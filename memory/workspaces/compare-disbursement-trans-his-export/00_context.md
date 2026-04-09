# Workspace: Compare Disbursement Trans His Export

## 00_context.md - Tổng quan bối cảnh

### Mục tiêu
So sánh sự khác biệt về logic giữa implementation cũ (Go) và implementation mới (TypeScript - Centralized Export Service) cho tính năng `DisbursementTransHisExport`.

### Cơ sở dữ liệu & Code
- **Code gốc (Go)**: `@/Users/trainguyen/Documents/work-feat/export/stepplaning/ExportDisbursementTransHis/`
- **Centralized Service (TS)**: `@/Users/trainguyen/Documents/work/centralized-export-service/logics/export/disbursement/`

### Task ID
`compare-disbursement-trans-his-export`

### Điểm phân biệt quan trọng
- Đây là logic **Lịch sử giao dịch (Trans History)**, khác hoàn toàn với logic **Yêu cầu chi (Ticket Export)**.
- Task này được tách riêng để tránh nhầm lẫn và đảm bảo tính "Elegance" của context.
