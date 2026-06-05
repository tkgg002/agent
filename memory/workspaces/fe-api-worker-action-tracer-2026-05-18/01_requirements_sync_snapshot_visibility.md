# Requirements — Sync Fields + Snapshot Now Visibility

## Origin
User: "đã chạy lại đc quét field, làm tiếp vụ sync field to shadow và snapshot now"

Bối cảnh: sau khi MongoDB connection URL chuẩn → bước "quét field" (auto-discovery scan) trong `HandleCreateDefaultColumns` đã PASS. Còn 2 bước cuối chưa verify được:
1. **Sync Fields to Shadow** — `mappingV2Repo.GetActiveRulesBySourceTable` + ALTER TABLE ADD COLUMN.
2. **Snapshot Now** — `HandleDebeziumSignal` insert Mongo `debezium_signal` doc.

## Vấn đề observability hiện tại
- `HandleCreateDefaultColumns` line 497: `if err == nil { for _, rule ... }` → swallow err của repo. User thấy "success / 0 columns" mà không biết do query fail hay do rules thật sự rỗng.
- `processDiscoveryRows` line 365: `if err := Create(&rule); err == nil { added++ }` → swallow Create err. Unique constraint hoặc FK fail bị giấu.
- `HandleDebeziumSignal` line 326-353: 2 nhánh dispatch (signalClient vs mongo direct insert) nhưng KHÔNG log nhánh nào chạy → khó debug khi snapshot không xảy ra.

## Functional Requirements
- FR-1: Mỗi click `Sync Fields to Shadow` phải có log đủ để user grep theo `trace_id` thấy:
  - Số rules trả về từ `GetActiveRulesBySourceTable` (hoặc err).
  - Mỗi ALTER TABLE: column + data_type + result (added / skipped).
  - Summary cuối: `rules_total`, `columns_added`, `columns_skipped`.
- FR-2: Mỗi insert rule trong `processDiscoveryRows` fail phải log với `source_object_id`, `field`, `data_type`, `error`. Summary: `discovered_total`, `already_mapped`, `inserted`, `insert_errors`.
- FR-3: `HandleDebeziumSignal` phải log:
  - Nhánh dispatch (`signal_client` hay `mongo_direct_insert`) + lý do (signal nil / not configured).
  - Mongo client nil → hint chỉ rõ `worker_server gating reconCore=nil`.
  - Err dispatch kèm `database`, `collection`, `dispatch_path`.

## Out of Scope
- Sửa logic chính của 2 luồng (chỉ add log).
- Đổi FE — `handleSyncFields` và `handleSnapshot` đã truyền đủ trace + payload.
- Migration DB.

## Definition of Done
- [ ] Worker build/vet/test PASS.
- [ ] Khi user click Sync Fields, log có `loaded mapping_rule_v2 for ALTER` + `ALTER TABLE summary`.
- [ ] Khi user click Snapshot Now, log có `debezium signal: using <path>` + `debezium signal dispatched`.
- [ ] Workspace docs đầy đủ.
