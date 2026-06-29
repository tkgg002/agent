# Plan: Cannot Create Two Master Tables with the Same Name in Different Schemas

## Phase 1: Research & Diagnosis (Hoàn thành)
1. **Tìm kiếm các bảng lưu trữ master table**:
   - Xác định bảng `cdc_system.master_binding` có constraint UNIQUE trên bộ ba `(master_connection_id, master_schema, master_table)`.
   - Về phía database cho phép trùng tên bảng khác schema.
2. **Tìm kiếm logic validate trong mã nguồn**:
   - Phát hiện lỗi `ambiguous_master_name` xuất phát từ repository layer của `cdc-cms-service`: `master_repo_gorm.go`.
   - Các phương thức query theo `master_table = ?` mà không kèm `master_schema` (ApproveSchemaTx, RejectSchema, RevertSchemaTx, ResolveMasterBindingByName).
   - `masterNameRe` regex trong `master_registry_handler.go` không cho phép dấu chấm `.`.

## Phase 2: Design & Implementation Plan (Đang thực hiện - Cập nhật 6)
- [x] Tạo `implementation_plan.md` chi tiết (cho phép truyền `schema.table` và parse/validate/query tương ứng).
- [x] Cải tiến logic backend: Khi người dùng chỉ truyền tên bảng đơn thuần (như `export_jobs`), hệ thống sẽ tự động lọc theo `schema_status` (ví dụ `pending_review` cho Approve) để tự động nhận diện chính xác bản ghi cần tác động mà không gây lỗi `ambiguous_master_name` cho Frontend.
- [ ] Chờ User phê duyệt kế hoạch.

## Phase 3: Execution & Verification
- [ ] Thực hiện sửa đổi trong `cdc-cms-service`:
  - [ ] Sửa lại SQL query trong `master_repo_gorm.go` (`ApproveSchemaTx`, `RejectSchema`, `RevertSchemaTx`) để bổ sung điều kiện lọc `schema_status`.
- [ ] Chạy unit tests trong `cdc-cms-service` để verify:
  - [ ] `go test ./...`
- [ ] Cập nhật walkthrough và báo cáo User.
