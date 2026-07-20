# Danh sách Task - Tối ưu hóa SQL Đối soát

- `[ ]` **Phase 1: Phân tích & Lập Kế hoạch**
  - `[x]` Phân tích cấu trúc SQL hiện tại và tìm nguyên nhân gây chậm trễ.
  - `[x]` Đề xuất phương án tối ưu hóa tối giản, giảm thiểu tác động.
  - `[ ]` Tạo tài liệu Kế hoạch triển khai chi tiết (`12_implementation_plan_optimize_slow_sql.md`) và `implementation_plan.md` ở thư mục Artifacts của Gemini, đợi User Approve.
- `[ ]` **Phase 2: Triển khai Kỹ thuật (Muscle)**
  - `[ ]` Cập nhật câu truy vấn `listLatestPrimary` trong `recon_read_repo_gorm.go` áp dụng phương pháp Distinct-before-Join.
  - `[ ]` Cập nhật hàm `ListFailedLogs` trong `recon_read_repo_gorm.go` tách biệt câu truy vấn đếm (`Count`) để loại bỏ toàn bộ các phép JOIN dư thừa.
- `[ ]` **Phase 3: Kiểm thử & Xác nhận**
  - `[ ]` Viết test script SQL hoặc script Go chạy thử để so sánh performance (trước vs sau).
  - `[ ]` Chạy toàn bộ test suite của package `recon` đảm bảo không có regression.
  - `[ ]` Lưu kết quả kiểm thử vào `06_test_cases_optimize_slow_sql.md`.
  - `[ ]` Cập nhật tài liệu Walkthrough (`14_walkthrough_optimize_slow_sql.md`).
