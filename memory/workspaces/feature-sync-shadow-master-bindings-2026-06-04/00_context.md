# Context: Syncing Shadow And Master Bindings

## Goal
Đồng bộ hóa cấu hình và trạng thái giữa Shadow và Master database layers trong Data-Hub. Sửa lỗi SQL reference, đảm bảo an toàn cột hệ thống, hoàn thiện giao diện (In Shadow, In Master, Source Data Type) và các action (Create Mapping, Approve Modal).

## Scope
- **Backend (Go)**:
  - Sửa lỗi SQL 42703 (`master_name` vs `master_table`) trong `MasterColumns` (`master_mapping_rule_handler.go`).
  - Sửa lỗi SQL 42P01 trong `HandleScanArrayFields` (`command_handler.go`) bằng cách trỏ đúng vào `shadowDB` thay vì `db`.
  - Tinh chỉnh system columns blacklist để đảm bảo không lọt các cột hệ thống khi tạo/approve master rules.
- **Frontend (React/TypeScript)**:
  - Hiển thị cột "In Master", "In Shadow" và trạng thái Shadow.
  - Ngăn chặn check Approve vào Master table (s2) nếu shadow rules chưa được Approve / chưa "In Shadow".
  - Hiển thị "Source Data Type".
  - Thêm nút "Create Mapping" workflow thủ công để operator chủ động khởi tạo rules.
  - Hoàn thiện "Pending" review modal để Approve/Reject hàng loạt hoặc đơn lẻ các rules vào Master registry.
