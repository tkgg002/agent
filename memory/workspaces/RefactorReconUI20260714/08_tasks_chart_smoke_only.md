# Danh sách Task: Lọc Biểu đồ chỉ vẽ Smoke Check

## Phase 1: Phân tích & Lên kế hoạch
- [x] Đọc GEMINI.md và lessons.md.
- [x] Tạo các tài liệu yêu cầu, tiến độ, danh sách task.
- [x] Soạn thảo kế hoạch triển khai chi tiết 12_implementation_plan_chart_smoke_only.md.

## Phase 2: Triển khai
- [x] Cập nhật file `ReconPipelineGrid.tsx`:
  - [x] Lọc `chartData` chỉ cho các row có `check_type === 'smoke'` hoặc `check_type === 'segment_b_smoke'`.
  - [x] Lọc dữ liệu đầu vào cho `yDomain` tương ứng.
- [x] Biên dịch dự án `cdc-cms-web` để đảm bảo build thành công.

## Phase 3: Hoàn thành & Nghiệm thu
- [x] Chạy linter verify_governance.py.
- [x] Cập nhật nhật ký tiến độ và hoàn thành walkthrough.md.
