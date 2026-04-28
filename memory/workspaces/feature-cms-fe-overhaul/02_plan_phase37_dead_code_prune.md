# Phase 37 — Plan

## Audit caller (đã thực hiện trong session gốc)

1. **TransformService**:
   - Grep `TransformService` / `BatchTransform` callers ở `centralized-data-service/internal/...` và `cdc-cms-service/internal/...`.
   - Kết quả: 0 caller runtime. Code chỉ tồn tại như helper-without-user.
   - → Prune.

2. **`registry_repo` legacy raw SQL**:
   - Methods `ScanRawKeys`, `PerformBackfill`, `GetDBColumns` (public-centric).
   - Phase 36 đã thêm wrappers `*InSchema()`.
   - Audit: methods cũ không còn caller.
   - → Prune phần cũ, giữ wrappers schema-aware.

3. **EventBridge**:
   - Có test file → vẫn được CI verify.
   - Hiện không trong runtime chính (poller không wired vào worker hot path).
   - Nhưng có giá trị nếu poller fallback được khôi phục.
   - → Giữ, đánh dấu compatibility reserve.

## Steps

1. Xóa file `centralized-data-service/internal/service/transform_service.go`.
2. Xóa methods legacy trong `cdc-cms-service/internal/repository/registry_repo.go`.
3. Thêm comment header "Compatibility Reserve" cho `event_bridge.go`.
4. Run `go test` ở cả 2 service để verify không gãy.
5. Tạo file vật lý workspace Phase 37.

## Verification matrix

- `go test ./...` trong `cdc-cms-service`
- `go test ./...` trong `centralized-data-service`
- `gofmt` pass
- grep `TransformService`: 0 hits sau prune
- grep methods cũ trong `registry_repo`: 0 hits sau prune
