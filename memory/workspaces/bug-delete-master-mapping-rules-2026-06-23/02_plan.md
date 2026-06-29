# Plan: Cấu hình quy tắc xóa cho mapping_rule_master (delete master mapping rules)

## Proposed Steps

### Phase 1: Research & Setup
- [x] Tạo workspace `bug-delete-master-mapping-rules-2026-06-23` và đăng ký trạng thái Active trong `active_plans.md`.
- [/] Đọc file `internal/app/commands/master/delete_master_rule.go` và `internal/domain/governance/master_rule.go` để nắm được logic hiện tại.

### Phase 2: Implementation
- [ ] Định nghĩa lỗi `ErrCannotDeleteApprovedInMaster` trong `delete_master_rule.go` (hoặc nơi định nghĩa lỗi domain tương tự).
- [ ] Chỉnh sửa logic kiểm tra trong hàm `Handle` của `DeleteMasterRuleHandler` tại `delete_master_rule.go`:
  - `isShadowSync := existing.CreatedBy != nil && *existing.CreatedBy == "shadow-sync"`
  - `isScanFlatten := existing.SourcePath != nil && *existing.SourcePath != ""`
  - Cho phép xóa nếu `!isShadowSync || isScanFlatten`. Nếu `isShadowSync && !isScanFlatten`, trả về `ErrCannotDeleteSync`.
  - Không cho phép xóa nếu `existing.Status == governance.StatusApproved && existing.InMaster` (trả về `ErrCannotDeleteApprovedInMaster`).

### Phase 3: Compile & Verification
- [ ] Chạy lệnh build dự án Go để kiểm tra cú pháp và đảm bảo code biên dịch thành công.
- [ ] Chạy unit test trong service `cdc-cms-service` để đảm bảo không phá vỡ logic kiểm thử cũ và có thể bổ sung unit test cho logic mới nếu cần.
