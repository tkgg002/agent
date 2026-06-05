# 00_context — FixSourceObjectListingDedupe

**Created**: 2026-05-21
**Owner**: Muscle (Claude Code CLI)
**Service**: `cdc-cms-service`
**Trigger**: User báo listing `GET /api/v1/source-objects` trả 6 rows trong khi chỉ có 4 source_objects thật sự (id 1, 36, 18, 5). 2 row id=1 và 2 row id=36 bị duplicate. User: "ko biết check id để bảo vào cho đúng mà làm vậy".

## Phạm vi
- File chính: `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go`
- Function: `ListEnriched` (line 79) — COUNT + SELECT cùng JOIN
- Function liên quan: `GetMappingContextByRegistryID` (line 164) — có cùng pattern bug nhưng cap bằng `LIMIT 1`

## Lịch sử relevant migrations
- `001_init_schema.sql`: tạo `cdc_table_registry` với UNIQUE `(source_db, source_table)`
- `053_relax_table_registry_unique.sql`: relax UNIQUE → `(source_db, source_table, target_table)`
- `054_v1_add_source_connection_id.sql`: ADD COLUMN `source_connection_id BIGINT FK connection_registry`
- `055_backfill_v1_source_connection_id.sql`: first-wins backfill cho row cũ
- `056_relax_v1_unique_with_connection.sql`: UNIQUE thành `(source_connection_id, source_db, source_table, target_table)` — connection-aware identity

→ Schema đã đúng. Bug nằm ở JOIN logic (chưa migrate theo schema).

## Khái niệm liên quan
- `source_object_registry` (V2): source-of-truth identity hiện tại, có `source_connection_id` đầy đủ.
- `cdc_table_registry` (V1 legacy bridge): vẫn dùng cho `sync_interval`, `priority`, `timestamp_field`, `notes`, `is_table_created` fallback.
- `shadow_binding` (V2): bridge sang tầng shadow (schema + table physical name).
- Mỗi source_object ↔ 1 connection ↔ 1 (db, table). cdc_table_registry rows giờ phải scope theo connection.

## Lessons related (đã đọc)
- `L-debezium-schema-evolution-compat` — schema drift cần preempt
- Lesson "Per-entity manual config = O(N)" — fix systematic, không patch per-row
- Lesson "Drift loại 2 — Model thêm field, quên migration" — schema và code drift
