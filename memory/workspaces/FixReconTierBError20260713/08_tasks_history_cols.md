# Checklist thực hiện thêm cột Nhật ký đối soát (30 phiên gần nhất)

- [x] Nghiên cứu cấu trúc code hiển thị bảng lịch sử tại `cdc-cms-web/src/components/ReconPipelineGrid.tsx`.
- [x] Thiết kế logic hiển thị và định dạng cho cột Chi tiết hợp nhất:
  - Định dạng hiển thị: `Thời gian xử lý : Chi tiết (lệch)`
- [x] Chỉnh sửa `ReconPipelineGrid.tsx` để gộp các cột và định dạng lại cột `Chi tiết`.
- [x] Tạo file phân tích kỹ thuật `13_analysis_history_cols.md`.
- [x] Tạo file tiến độ `05_progress_history_cols.md`.
- [x] Kiểm tra giao diện và đảm bảo build thành công (npm run build hoặc kiểm tra tĩnh).
- [x] Chạy linter quy trình `verify_governance.py`.
