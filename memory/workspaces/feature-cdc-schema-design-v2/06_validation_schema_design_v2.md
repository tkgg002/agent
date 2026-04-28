# Validation

## Validation Method

1. Đối chiếu thiết kế với code hiện tại của `centralized-data-service`.
2. Xác nhận proposal xử lý được các pain point user nêu:
   - multi source
   - multi shadow
   - multi master
   - system riêng
   - preserve schema cha / namespace cha
3. Xác nhận proposal có đường rollout, không yêu cầu big bang rewrite.

## What Was Verified

1. Thiết kế xử lý đúng vấn đề `cdc_internal` bị dùng sai nghĩa:
   - V2 dùng `cdc_system` cho metadata.
2. Thiết kế xử lý đúng bài toán preserve parent namespace:
   - có `source_database`, `source_schema`, `source_namespace`
   - có `shadow_schema`, `master_schema`
3. Thiết kế xử lý đúng bài toán chọn đích riêng cho từng table:
   - `shadow_binding`
   - `master_binding`
4. Thiết kế xử lý đúng bài toán runtime hiện tại chỉ có một DB:
   - có đề xuất `connection_manager.go`
5. Thiết kế xử lý đúng chỗ current code đang hardcode:
   - `event_handler.go`
   - `registry_service.go`
   - `master_ddl_generator.go`
   - `transmuter.go`

## Residual Gaps

1. Chưa viết migration SQL thật trong repo.
2. Chưa code compatibility adapter.
3. Chưa chạy integration test vì đây là task thiết kế/tài liệu, chưa phải implementation runtime.
