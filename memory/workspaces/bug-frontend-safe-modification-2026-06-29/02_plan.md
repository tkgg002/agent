# Plan: Sửa đổi an toàn cho mã nguồn Frontend

## Mục tiêu
Đảm bảo mã nguồn Frontend chạy an toàn, chống crash app khi dữ liệu API bị null/undefined, hiển thị đúng các giá trị Date và chỉ số Drift/Lag, và đảm bảo dự án build thành công.

## Kế hoạch triển khai

### Phase 1: Chuẩn bị & Cấu hình Governance (Brain)
1. Tạo workspace `bug-frontend-safe-modification-2026-06-29` (Đã hoàn thành).
2. Tạo file `00_context.md` và `02_plan.md` (Đang hoàn thành).
3. Đăng ký workspace hoạt động trong registry `active_plans.md` (Brain thực hiện).
4. Phân công cho Chief Engineer (Muscle - Subagent) thực thi sửa đổi code.

### Phase 2: Sửa đổi mã nguồn (Muscle)
#### 1. File: `ReconPipelineGrid.tsx`
- Sửa hàm `buildPipelines` hoặc các callsite sử dụng `p.shadowName.split('.')` sang optional chaining an toàn.
- Bọc logic `new Date(v)` hiển thị `checked_at` bằng check an toàn `v ? new Date(v).toLocaleString() : '—'`.

#### 2. File: `DataIntegrity.tsx`
- Sửa hàm `resolvePipelineNames` loại bỏ lặp `record.shadow_table`.
- Tại cột Drift %: Ép kiểu an toàn `Number(v)` trước khi gọi `toFixed(2)`.
- Bọc an toàn các chỉ số: lag (kiểm tra `Number(v)` isNaN), `percent_done` (đảm bảo fallback 0), `started_at` và `created_at` (kiểm tra ngày trước khi định dạng).

### Phase 3: Biên dịch & Kiểm thử (Muscle)
- Chạy `npm run build` ở thư mục `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web` để đảm bảo dự án frontend compile thành công không có lỗi TypeScript hay linter.

### Phase 4: Tổng kết & Báo cáo (Brain)
- Muscle báo cáo kết quả và log build thành công.
- Brain cập nhật `05_progress.md`, update registry `active_plans.md` sang `✅ Done`.
- Phản hồi kết quả cụ thể tới User.
