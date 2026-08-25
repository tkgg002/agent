# Requirements: Isolated Transform Job Status per Source Object / Connector

## Bối cảnh
Khi hệ thống có nhiều connectors khác nhau (ví dụ `testces` và `testces1`) cùng đồng bộ các bảng từ cùng Database nguồn (hoặc có các object trùng tên target table / shadow table như `payment_bill`), việc query `last_transform_status` ở Backend đang bị trôi/bleed dữ liệu status từ connector này sang connector khác.

## Nguyên nhân
1. Bảng `cdc_system.transform_jobs` chưa lưu `source_object_id`.
2. LATERAL JOIN `tj` trong `source_object_read_repo_gorm.go` chỉ filter theo `WHERE tj.target_table = COALESCE(sb.shadow_table, tr.target_table, so.source_object_name)`.
3. Do đó, khi 1 connector hoàn thành transform, object trùng tên của connector khác cũng nhận nhầm `last_transform_status = 'COMPLETED'`.

## Yêu cầu kỹ thuật
1. Tạo migration `088_add_source_object_id_to_transform_jobs.sql` bổ sung cột `source_object_id BIGINT` (nullable, index).
2. Cập nhật `transform_job_repo.go` (Struct `TransformJob`, hàm `Create`) để ghi nhận `source_object_id`.
3. Cập nhật `source_object_actions_handler.go` để truyền `sourceObjectID` khi gọi `transformJobRepo.Create`.
4. Cập nhật SQL LATERAL JOIN `tj` trong `source_object_read_repo_gorm.go` để lọc ưu tiên theo `tj.source_object_id = so.id`.
5. Đảm bảo 100% không làm breaking changes đối với các job cũ chưa có `source_object_id` (dùng `COALESCE/OR fallback`).
