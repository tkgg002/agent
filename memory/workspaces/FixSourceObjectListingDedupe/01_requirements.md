# 01_requirements — FixSourceObjectListingDedupe

## Functional
- FR-1: `GET /api/v1/source-objects` MUST trả 1 row cho mỗi `(source_object_registry.id, shadow_binding.id)` cặp.
- FR-2: Field `registry_id` MUST trỏ đúng `cdc_table_registry.id` tương ứng với cùng `source_connection_id` của source_object (hoặc legacy NULL nếu không có exact match).
- FR-3: `total` count MUST khớp số row thực tế trả về (sau filter + dedupe).
- FR-4: Không thay đổi wire shape (JSON keys) của response.

## Non-functional
- NFR-1: Backward compat — legacy `cdc_table_registry` rows có `source_connection_id IS NULL` (chưa backfill match được) vẫn fallback được, không break listing.
- NFR-2: Idempotent — chạy migration 054/055/056 lại không phá fix.
- NFR-3: Determinism — nhiều legacy NULL rows trùng (db, table, target) phải pick deterministic 1 row (theo `source_connection_id NOT NULL preferred, then id ASC`).
- NFR-4: Performance — không tăng latency listing > 20%. JOIN LATERAL `LIMIT 1` đã có index `idx_ctr_source_connection (source_connection_id, source_db, source_table)`.

## Acceptance Criteria
- AC-1: Với data hiện tại trong DB (id 1, 36, 18, 5), listing trả về 4 rows + `total=4`.
- AC-2: id=1 hiện duy nhất 1 row với `registry_id` = registry thuộc connection_id của so id=1.
- AC-3: id=36 hiện duy nhất 1 row với `registry_id` thuộc connection_id=42.
- AC-4: `go build ./...` PASS, `go vet ./...` PASS, `go test ./internal/...` PASS (no regression).
