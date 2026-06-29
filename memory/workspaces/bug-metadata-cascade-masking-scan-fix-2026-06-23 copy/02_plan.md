# Execution Plan

## Phase 1: Preparation & Investigation
- [x] Đọc lessons.md và active_plans.md.
- [x] Tạo cấu trúc thư mục workspace và các tài liệu governance.
- [ ] Dùng browser subagent để verify lỗi polling không stop trên FE và tìm hiểu cơ chế hoạt động thực tế.

## Phase 2: Implementation (gián tiếp qua Muscle CLI scripts)
- [ ] Viết Python script patch để sửa `source_repo_gorm.go` trong `cdc-cms-service` (bỏ cascade active).
- [ ] Viết Python script patch để sửa query lateral join trong `snapshot_progress_read_repo_gorm.go`.
- [ ] Viết Python script patch để sửa nạp chéo rules trong `metadata_registry_service.go` của `centralized-data-service`.
- [ ] Viết Python script patch để sửa lệch parameters trong `master_binding_repo.go` của `centralized-data-service`.
- [ ] Tối ưu hóa API dispatch-status hoặc frontend hook để dừng polling khi có kết quả.

## Phase 3: Verification & Security Check
- [ ] Rebuild cả hai services.
- [ ] Chạy manual test qua browser hoặc curl.
- [ ] Run security check.
