# Plan Phase 1 Implementation

## English

1. Inspect current repository/model patterns so the new V2 code matches the existing style.
2. Add V2 migrations with safe `IF NOT EXISTS` guards and backward-compatible legacy backfill.
3. Add V2 models with explicit `TableName()` bindings to `cdc_system.*`.
4. Add repository scaffolding for read/create/update use cases needed by future runtime refactors.
5. Run `gofmt` and targeted `go test` / compile validation.

## Tiếng Việt

1. Đọc pattern hiện tại của model/repository để scaffold mới không bị lệch style.
2. Thêm migration V2 với guard an toàn `IF NOT EXISTS` và backfill khởi tạo từ dữ liệu legacy.
3. Thêm model V2, bind rõ `TableName()` tới `cdc_system.*`.
4. Thêm repository scaffold phục vụ phase refactor runtime sau này.
5. Chạy `gofmt` và verify build/test ở mức phù hợp.
