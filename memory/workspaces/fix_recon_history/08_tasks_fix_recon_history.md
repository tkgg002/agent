# Danh sách Task Chi Tiết: Sửa lỗi 500 Endpoint Lịch sử Đối soát

- [ ] **Phase 1: Phân tích & Tái hiện lỗi**
  - [ ] Viết integration test `recon_read_repo_gorm_real_test.go` kết nối DB thật.
  - [ ] Chạy test để lấy log lỗi chi tiết của SQL query.
  - [ ] Xác định nguyên nhân gốc rễ (Root Cause).
- [ ] **Phase 2: Thiết kế & Triển khai sửa lỗi**
  - [ ] Lập kế hoạch chi tiết tại `12_implementation_plan_fix_recon_history.md`.
  - [ ] Sửa đổi mã nguồn trong `recon_read_repo_gorm.go`.
- [ ] **Phase 3: Xác minh & Hoàn thành**
  - [ ] Chạy lại integration test để xác nhận query thành công (Green).
  - [ ] Kiểm tra xem có vi phạm các lessons nào không.
  - [ ] Cập nhật kết quả vào `05_progress_fix_recon_history.md` và `14_walkthrough_*.md`.
