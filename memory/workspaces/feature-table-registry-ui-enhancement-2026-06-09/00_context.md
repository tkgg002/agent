# Context: Table Registry UI Enhancement

## Bối cảnh
User yêu cầu cải tiến giao diện (UI) của `TableRegistry.tsx` tại `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx`.
Cụ thể, chỉnh sửa cột "Source Actions" nơi render Switch và component `AsyncRowActions`.

## Yêu cầu chi tiết
1. Thêm style `padding-bottom: 5px` cho thẻ bao quanh Switch:
   ```tsx
   <div onClick={e => e.stopPropagation()}>
   ```
2. Thay đổi UI khi `AsyncRowActions` đang chạy hoặc hoàn tất:
   - Khi có hoạt động của `AsyncRowActions` (hoặc tương tự), không hiển thị class `ant-space css-dev-only-do-not-override-ch9ese ant-space-horizontal ant-space-align-center ant-space-gap-row-small ant-space-gap-col-small css-var-root` nữa (hoặc ẩn component wrapper `Space` mặc định bên trong `AsyncRowActions`).
   - Thay vào đó, cập nhật màu sắc và icon trực tiếp cho nút "Quét field" (Scan fields).
   - Khi hoàn tất quét field thành công: nút hiển thị dấu check (check icon) và border color màu xanh lá.
   - Khi đang chạy (scanning): hiển thị màu xanh dương (blue) và icon loading như hiện tại.

## Phân tích vi phạm Governance (Root Cause Analysis)
- Không có vi phạm Governance do đây là task mới bắt đầu trực tiếp từ yêu cầu của User trong session này, workspace được khởi tạo ngay lập tức tại Gate #0.
