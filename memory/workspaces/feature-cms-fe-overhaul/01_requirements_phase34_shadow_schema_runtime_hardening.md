# Requirements — Phase 34 Shadow Schema Runtime Hardening

## Mục tiêu

- Bóc tiếp các helper/runtime path còn assume `public` trong `centralized-data-service`.
- Chỉ sửa những nhánh đang ảnh hưởng trực tiếp tới `cms-fe operator-flow` đã direct-V2 hóa:
  - `batch-transform`
  - `scan-raw-data`
  - `periodic-scan`
  - `drop-gin-index`
  - `scan-fields`
  - `schema-validator`

## Ràng buộc

- Không quét refactor toàn bộ codebase cùng lúc.
- Không làm gãy `auto-flow` Debezium hiện tại.
- Ưu tiên dùng metadata V2 để resolve `shadow_schema` theo `target_table`.
- Không tạo “V2 giả”: command/operator path phải chạy đúng schema vật lý, không chỉ đổi API surface.

## Definition of Done

- Worker helpers trên dùng đúng `shadow_schema` khi target nằm ngoài `public`.
- Có lookup V2 theo `target_table -> ResolvedSourceRoute`.
- `go test ./internal/service ./internal/handler ./internal/server` pass.
