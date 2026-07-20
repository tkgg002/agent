# Walkthrough: Thêm cột Lệch và Thời gian xử lý vào FE

## Thay đổi đã thực hiện

### 1. `cdc-cms-web`
- **[ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)**:
  - Loại bỏ các cột riêng lẻ `Lệch` và `Thời gian xử lý` để tránh cuộn ngang.
  - Gộp tất cả vào cột `Chi tiết` với định dạng `"Thời gian xử lý : Chi tiết (lệch)"`. Ví dụ: `85ms : 2,713,267 → 2,713,279 (-12)`.
  - Giữ nguyên cấu hình `scroll={{ y: 260 }}` gọn gàng.
- **[ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)**:
  - Loại bỏ biến `EMPTY_ARRAY` chưa sử dụng để sửa lỗi biên dịch TypeScript.

## Kết quả kiểm thử & xác minh

- Đã biên dịch sản phẩm tĩnh (`npm run build`) thành công 100% không có lỗi cảnh báo TypeScript.
