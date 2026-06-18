# Context: Tách cột Source / Shadow thành 3 cột Source, Shadow, Master trong tab Tổng quan (Overview) của trang Data Integrity

## Mô tả Task
Tách cột hiển thị gộp "Source / Shadow" hiện tại trong bảng Tổng quan (tab "overview") của trang Data Integrity (`DataIntegrity.tsx`) thành 3 cột riêng biệt:
- **Cột Source**: Hiển thị tên bảng/database nguồn kèm theo nhãn Connector (lấy từ `source_connection_code`).
- **Cột Shadow**: Hiển thị schema và table dạng `schema.table` của Shadow database, kèm theo tag `Ambiguous` nếu `scope_ambiguous = true`.
- **Cột Master**: Hiển thị schema và table dạng `schema.table` của Master database (nếu có mapping, ngược lại hiển thị `—`).

## Phân tích Governance & Root Cause
- **Governance Audit**: Phiên làm việc bắt đầu và Workspace được khởi tạo ngay lập tức trước khi thay đổi bất kỳ code nào.
- **Root Cause Analysis (nếu có vi phạm)**: Không phát hiện hành vi vi phạm quy trình Governance trong phiên khởi động.

## Tài liệu & Component liên quan
- Trang FE: [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
- Định nghĩa hook và interfaces: [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Grid Pipeline tham khảo: [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
