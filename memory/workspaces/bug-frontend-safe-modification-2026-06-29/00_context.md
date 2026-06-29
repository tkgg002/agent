# Context: Sửa đổi an toàn cho mã nguồn Frontend

## Yêu cầu sửa đổi
1. Tệp `ReconPipelineGrid.tsx`:
   - Thay thế `p.shadowName.split('.')` bằng `p.shadowName?.split('.') || []` để phòng ngừa lỗi crash app khi `shadowName` bị null.
   - Đảm bảo an toàn các đoạn định dạng Date (kiểm tra `v ? new Date(v)... : '—'`) ở các dòng hiển thị `checked_at`.

2. Tệp `DataIntegrity.tsx`:
   - Sửa hàm `resolvePipelineNames`: `const shadowTable = record.shadow_table || record.target_table || null;` (bỏ đoạn lặp `shadow_table`).
   - Tại cột Drift %: Ép kiểu an toàn sang Number trước khi gọi `toFixed` (`numVal.toFixed(2)`).
   - Tại các cột hiển thị thời gian, lag, tiến độ backfill: Bổ sung logic kiểm tra null/undefined an toàn (`r.percent_done || 0`, `v ? new Date(v)... : '—'`) để không hiển thị "Invalid Date" hay "NaN".

3. Chạy build dự án frontend (`npm run build`) để xác định toàn bộ dự án frontend biên dịch thành công.

## Phân tích Governance
- Khởi tạo workspace `bug-frontend-safe-modification-2026-06-29` trước khi sửa code.
- Brain chịu trách nhiệm lập kế hoạch, kiểm soát tiến độ, giám sát.
- Muscle chịu trách nhiệm trực tiếp sửa code và chạy build kiểm thử.
- RCA: Hiện tại quy trình được tuân thủ nghiêm ngặt, không có vi phạm governance.
