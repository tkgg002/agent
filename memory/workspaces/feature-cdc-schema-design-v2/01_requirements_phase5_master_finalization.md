# Requirements — Phase 5 Master Finalization

## Scope

- Hoàn tất các điểm còn thiếu trên master path sau Phase 4.
- Giảm ambiguity ở luồng `cdc.cmd.transmute-shadow` bằng identity đầy đủ hơn.
- Đảm bảo transmuter có thể tự chuẩn bị destination master trước khi upsert.
- Ghi runtime state cho master path để phục vụ vận hành sau khi wipe/bootstrap.

## Required Outcomes

1. `TransmuterModule` phải auto-ensure master destination trước khi ghi.
2. `MasterDDLGenerator` phải cập nhật `cdc_system.sync_runtime_state` khi DDL success/fail.
3. `TransmuterModule` phải cập nhật runtime state success/fail/skipped cho `master_binding`.
4. `transmute-shadow` payload phải hỗ trợ thêm `shadow_schema` và `shadow_connection_key`.
5. `TransmuteHandler` phải ưu tiên lookup theo identity-aware route trước khi fallback theo `shadow_table`.
6. Các package bị ảnh hưởng phải pass:
   - `./internal/service`
   - `./internal/handler`
   - `./internal/server`
   - `./internal/sinkworker`

## Non-Goals

- Chưa purge toàn bộ `TableRegistry`/recon legacy trong phase này.
- Chưa thay scheduler storage khỏi `cdc_internal.transmute_schedule`.
- Chưa refactor toàn bộ sinkworker shadow storage sang multi-shadow runtime.
