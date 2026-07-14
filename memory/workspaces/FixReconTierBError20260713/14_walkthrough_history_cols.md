# Walkthrough: Thêm cột Lệch và Thời gian xử lý vào FE

## Thay đổi đã thực hiện

### 1. `cdc-cms-web`
- **[ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)**:
  - Thêm cột `Lệch` với width `130` sử dụng logic `fmtDrift(r.diff != null ? -r.diff : null)`.
  - Thêm cột `Thời gian xử lý` với width `110` hiển thị định dạng ms/giây dựa trên `r.duration_ms`.
  - Cấu hình `scroll={{ x: 920, y: 260 }}` để đảm bảo tính responsive của bảng.
- **[ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)**:
  - Loại bỏ biến `EMPTY_ARRAY` chưa sử dụng để sửa lỗi biên dịch TypeScript.

## Kết quả kiểm thử & xác minh

- Chạy kiểm tra TypeScript (`npx tsc --noEmit`) thành công.
- Biên dịch sản phẩm tĩnh (`npm run build`) thành công 100%.
