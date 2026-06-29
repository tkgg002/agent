# Context: bug-metadata-cascade-masking-scan-fix-2026-06-23

## Vấn đề 1: Bỏ cascade is_active cho shadow_binding
- **Hiện tại**: Khi bật/tắt `is_active` cho source_object_registry (qua API `PATCH /api/v1/source-objects/:id`), logic trong `UpdateMetadata` tự động cascade update `is_active` cho tất cả các shadow_bindings thuộc source_object đó.
- **Yêu cầu**: Bỏ cascade này. Toggled active của source_object không tự động cập nhật active của shadow_binding.

## Vấn đề 2: Cột Shadow hiển thị sai ở snapshot-monitor
- **Hiện tại**: Query trong `ListSnapshotProgress` sử dụng `LEFT JOIN LATERAL` lấy shadow binding active mới nhất dựa trên `source_object_id`. Điều này sai vì khi chạy snapshot ta đã có binding_id cụ thể của progress (đã được lưu trong `shadow_binding_id` của bảng `snapshot_progress`).
- **Yêu cầu**: Sửa query để join chính xác qua `shadow_binding_id` (với fallback về binding mới nhất của source_object nếu `shadow_binding_id` null).

## Vấn đề 3 & 4: Masking Strategy không chạy khi snapshot & upstream
- **Hiện tại**: `MetadataRegistryService.ReloadAll` nạp `mappingCache` bằng cách lấy tất cả `v2Rules` của source object đó mà KHÔNG kiểm tra xem rule đó có khớp với `ShadowBindingID` hay không.
- **Hệ quả**: `mappingCache[bindingID]` của một binding chứa cả rules của binding/clone khác. Khi map data, cột nhạy cảm có thể bị map qua rule của clone khác (có `IsSensitiveField = false` hoặc không cấu hình masking), dẫn đến masking strategy không hoạt động.
- **Yêu cầu**: Sửa reload registry để filter chính xác `v2.ShadowBindingID == bindingID`.

## Vấn đề 5: Scan fields chạy trên table rỗng và báo lỗi
- **Hiện tại**: Command `scan-fields` được trigger. Trong `ScanFieldsDebezium`, nếu shadow table rỗng, nó sẽ ném lỗi: `shadow table %s is empty`.
- **Yêu cầu**: Cải thiện hoặc xử lý trường hợp shadow table rỗng để tránh báo lỗi không cần thiết, hoặc không tự động trigger `scan-fields` từ FE khi chưa có data. Đồng thời, kiểm tra tại sao FE đang poll không stop dù api chạy thành công.

## Vấn đề 6: Transmute shadow -> master không chạy sau upstream
- **Hiện tại**: Sau khi thực hiện upstream (sync data thành công vào shadow table), `SinkWorker` bắn event `cdc.cmd.transmute-shadow` qua NATS, nhưng transmuter không chạy.
- **Nguyên nhân**: Lệch số lượng placeholder và parameters trong query `ListMasterTablesByShadowIdentity` (`master_binding_repo.go`). Query chỉ có 7 placeholders `?` nhưng lại truyền vào 8 arguments (dư một biến `shadowConnectionKey` ở cuối), khiến query lỗi và trả về không có master tables nào để transmute.
- **Yêu cầu**: Xóa argument dư trong query bind.
