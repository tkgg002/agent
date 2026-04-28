# Plan — Phase 5 Master Finalization

## Execution Plan

1. Mở rộng `MasterDDLGenerator` để ghi `sync_runtime_state` ở scope `master`.
2. Tiêm `MasterDDLGenerator` vào `TransmuterModule` dưới dạng ensurer để auto-prepare namespace/table.
3. Ghi runtime state success/failure/skipped từ `TransmuterModule`.
4. Mở rộng payload `cdc.cmd.transmute-shadow` với `shadow_schema` và `shadow_connection_key`.
5. Cập nhật `TransmuteHandler` để match binding theo identity-aware metadata trước.
6. Verify bằng `gofmt` + targeted `go test`.

## Design Notes

- Vẫn giữ fallback theo `shadow_table` để không làm gãy caller cũ.
- SinkWorker legacy sẽ gửi mặc định `shadow_schema=cdc_internal` và `shadow_connection_key=default`.
- Runtime state chỉ ghi ở `master` scope vì đây là chỗ cần observability nhất cho vòng wipe/bootstrap sắp tới.
