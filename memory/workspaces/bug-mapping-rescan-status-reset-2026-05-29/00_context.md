# 00_context — Bug: Mapping rescan should reset status to pending + show "in shadow"

## Page liên quan
- FE: `http://localhost:5173/shadow/:id/mappings` (MappingFieldsPage.tsx)
- BE control-plane: `cdc-cms-service` (Go + Fiber + GORM, PG `cdc_system.*`)
- BE worker: `centralized-data-service` (Go + NATS handler, owns shadow DDL)

## State observed (user report, 2026-05-29)
- Shadow table id=20 — đã DROP rồi recreate cùng tên. Lần scan mới không phát hiện field mới (đúng — schema không đổi). NHƯNG cột Status trong UI vẫn hiển thị "approved" từ lifecycle cũ — sai context của lần recreate.

## Source of truth — `cdc_system.mapping_rule_v2`
- Row tồn tại với `status='approved', is_active=true` từ lifecycle cũ.
- Scan KHÔNG đụng tới mapping_rule_v2 → status carry over.

## System Default Fields (auto-created on shadow table)
- Truth source: `centralized-data-service/internal/handler/command_handler.go:168-179` `ensureCDCColumnsInSchema`
  - 10 cột CDC: `source_id`, `_raw_data`, `_source`, `_source_ts`, `_synced_at`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at`
- Cộng 1 cột PK (Mongo `_id` rename → `id`, hoặc PK SQL nguyên gốc) → **tổng 11**
- FE hiện chỉ list 8 (thiếu `source_id`, `_source_ts`, PK)

## Endpoints liên quan
- `GET /api/introspection/scan-raw/:table` — wires worker `cdc.cmd.scan-raw-data` (publishes/awaits reply, không đụng mapping_rule_v2)
- `GET /api/mapping-rules` — `ListMappingRulesHandler` → `mapping_rule_v2 JOIN source_object_registry + shadow_binding`
- `PATCH /api/mapping-rules/:id` — `UpdateMappingRuleHandler` (in-process UPDATE)
- (mới) `GET /api/v1/source-objects/:id/shadow-columns` — list cột thực tế đang tồn tại trong shadow PG (information_schema)
