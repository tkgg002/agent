# Plan: Syncing Shadow And Master Bindings

## Steps
1. **Sửa lỗi SQL Reference 42703**:
   - Thay thế `master_name` thành `master_table` trong truy vấn SQL và struct tương ứng tại `master_mapping_rule_handler.go:MasterColumns`.
2. **Sửa lỗi SQL Relation Missing 42P01**:
   - Cập nhật `HandleScanArrayFields` trong `command_handler.go` để chạy query scan trên `execDB` (trỏ đến `h.shadowDB` nếu cấu hình shadow db) thay vì dùng `h.db`.
3. **Thắt chặt System Columns Blacklist**:
   - Đảm bảo trong `SyncFromShadow` và `create_master.go` (cũng như bất kỳ chỗ approve nào), các cột hệ thống (`_gpay_id`, `_source_id`, `_raw_data`, `_source`, `_source_ts`, `_synced_at`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at`) đều bị loại trừ và không bao giờ được sync hoặc tạo DDL trên Master.
4. **Hoàn thiện UI Master Mapping**:
   - Sửa đổi `MappingFieldsPage.tsx` hoặc trang Master Mapping tương ứng để:
     - Hiển thị cột: In Shadow, In Master, Source Data Type.
     - Vô hiệu hóa checkbox Approve vào Master nếu Shadow Status chưa Approved hoặc chưa "In Shadow".
     - Thêm nút "Create Mapping" workflow thủ công.
     - Triển khai "Pending" review modal để Approve/Reject các rules.
