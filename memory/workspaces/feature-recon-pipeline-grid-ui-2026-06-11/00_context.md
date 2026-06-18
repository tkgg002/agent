# Context: Recon Pipeline Grid UI Enhancement

## Bối cảnh
User yêu cầu cải tiến giao diện cột "Pipeline" trong bảng lineage tại file `ReconPipelineGrid.tsx` (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`).

## Yêu cầu chi tiết
1. Thiết kế lại cột render Pipeline theo dạng 2 hàng:
   - **Hàng trên**: Hiển thị tên bảng (table name) của `source` → `shadow` → `master`.
   - **Hàng dưới**: Hiển thị schema/database tương ứng của `source` → `shadow` → `master`.
2. Định dạng riêng cho `source` ở hàng dưới: hiển thị thêm icon cơ sở dữ liệu (`DatabaseOutlined`) và tag của connector (`sourceConnector`) nếu có.

## Phân tích vi phạm Governance (Root Cause Analysis)
- Không có vi phạm Governance do đây là task mới và workspace được khởi tạo ngay lập tức khi bắt đầu.
