# Solution — Phase 5 Master Finalization

## Delivered

Phase này chốt hai khoảng trống còn lại của master path V2:

1. `TransmuterModule` không còn giả định destination đã tồn tại sẵn.
   Nó sẽ gọi `MasterDDLGenerator.EnsureMaster()` trước khi chạy batch, nhờ đó schema/table/index được dựng idempotent ngay trên master DB đúng theo `master_binding`.

2. Luồng `transmute-shadow` không còn chỉ dựa vào `shadow_table`.
   Payload và lookup đã hiểu thêm `shadow_schema` + `shadow_connection_key`, giảm nguy cơ match nhầm khi cùng một tên bảng xuất hiện ở nhiều namespace khác nhau.

## Remaining Gaps

- `transmute_schedule` vẫn là legacy storage.
- Recon/DLQ/command legacy vẫn còn các điểm phụ thuộc `TableRegistry`.
- SinkWorker vẫn hardcode write shadow vào `cdc_internal`, mới chỉ được gắn identity rõ hơn ở trigger payload.
