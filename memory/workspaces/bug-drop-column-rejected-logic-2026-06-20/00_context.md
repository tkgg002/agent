# Context: Sửa lỗi logic Drop Column đối với Rule bị Rejected

## 1. Mô tả lỗi
- **Hiện tượng**: Khi thực hiện drop một cột đã bị `rejected` (được quyền drop để xóa cột khỏi table và không được sync khi transmute), hệ thống trả về lỗi: `{"error":"cột đang được rule approved+active khác dùng — không drop"}`.
- **Nguyên nhân gốc rễ**:
  1. Trong `internal/app/commands/master/drop_column.go`, hàm `DropColumnHandler.Handle` thực hiện kiểm tra xung đột cột bằng cách gọi:
     ```go
     conflict, err := h.repo.CheckColumnConflict(ctx, rule.MasterBindingID, rule.TargetColumn, 0)
     ```
     Việc truyền `excludeID = 0` (thay vì `rule.ID`) làm cho truy vấn SQL đếm chính rule hiện tại (vốn đang có `target_column` trùng với cột cần drop), khiến `conflict` luôn trả về `true`.
  2. Hàm `CheckColumnConflict` trong `master_mapping_rule_repo_gorm.go` đếm tất cả các rule có cùng tên cột mà không lọc theo trạng thái hoạt động:
     ```sql
     SELECT count(*) FROM cdc_system.mapping_rule_master WHERE master_binding_id = ? AND id <> ? AND target_column = ?
     ```
     Điều này khiến một cột bị coi là xung đột ngay cả khi nó chỉ được sử dụng bởi các rule khác cũng đã bị `rejected` hoặc `inactive` (is_active = false).
  3. Logic đúng: Một cột chỉ bị coi là xung đột (không cho phép drop) khi và chỉ khi có ít nhất một rule **khác** (`id <> rule.ID`) có trạng thái là `approved` và đồng thời đang hoạt động (`is_active = true`) sử dụng cột đó.

## 2. Phạm vi ảnh hưởng
- Dự án: `cdc-cms-service`
- Các tệp tin liên quan:
  - `internal/app/commands/master/drop_column.go`
  - `internal/domain/master/repository.go`
  - `internal/infra/persistence/master/master_mapping_rule_repo_gorm.go`
  - `internal/app/commands/master/drop_column_test.go` (cần tạo mới để kiểm thử)

## 3. Quy trình Governance áp dụng
- Thực hiện phân tích Root Cause lỗi vi phạm governance (nếu có).
- Cập nhật tiến độ vào `05_progress.md` trước khi sửa code.
- Chạy unit test biên dịch thành công.
- Lưu tài liệu Stage 5: `03_implementation_drop_column_fix.md` và `report_drop_column_logic_fix.md` sau khi hoàn thành.
