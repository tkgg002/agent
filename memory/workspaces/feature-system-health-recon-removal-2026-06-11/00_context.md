# Context: System Health Reconciliation Removal

## Bối cảnh
User yêu cầu loại bỏ bảng "Đối soát dữ liệu" (Reconciliation) khỏi trang SystemHealth (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SystemHealth.tsx`).
Trang đối soát dữ liệu (Reconciliation) hiện đã có trang chuyên biệt là DataIntegrity (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx`), do đó việc hiển thị bảng này ở trang SystemHealth là dư thừa và làm chật giao diện.

## Yêu cầu chi tiết
1. Loại bỏ UI render của HealthSection "Đối soát dữ liệu" trong `SystemHealth.tsx`.
2. Dọn dẹp code thừa liên quan đến `ReconciliationBody` và `ReconRow` trong `SystemHealth.tsx`.

## Phân tích vi phạm Governance (Root Cause Analysis)
- Không có vi phạm Governance do đây là task mới và workspace được khởi tạo ngay lập tức khi bắt đầu.
