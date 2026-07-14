# Checklist thực hiện thêm cột Nhật ký đối soát (30 phiên gần nhất)

- [ ] Nghiên cứu cấu trúc code hiển thị bảng lịch sử tại `cdc-cms-web/src/components/ReconPipelineGrid.tsx`.
- [ ] Thiết kế logic hiển thị và định dạng cho 2 cột mới:
  - Cột "Lệch" sử dụng hàm `fmtDrift(r.diff)`.
  - Cột "Thời gian xử lý" sử dụng helper format duration theo giây/ms.
- [ ] Chỉnh sửa `ReconPipelineGrid.tsx` để thêm 2 cột mới vào bảng `Table<ReconReport>`.
- [ ] Tạo file phân tích kỹ thuật `13_analysis_history_cols.md`.
- [ ] Tạo file tiến độ `05_progress_history_cols.md`.
- [ ] Kiểm tra giao diện và đảm bảo build thành công (npm run build hoặc kiểm tra tĩnh).
- [ ] Chạy linter quy trình `verify_governance.py`.
