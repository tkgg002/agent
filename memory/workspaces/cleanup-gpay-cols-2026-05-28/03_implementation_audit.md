# 03_implementation_audit — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Inventory 104 references theo path

### Path A — FE Shadow (cdc-cms-service)
| File | Line | Vai trò | Cột dùng |
|---|---|---|---|
| `cdc-cms-service/internal/infra/persistence/shadow_automator.go` | 80, 86, 89 | CREATE shadow table | `source_id`, `_deleted` (KHÔNG có `_gpay_*`) |
| `cdc-cms-service/internal/api/mapping_preview_handler.go` | 63, 65-69 | Preview SELECT shadow | Đọc `_gpay_source_id`, `_gpay_id` **— drift: shadow ko có** |
| `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go` | 100 | Test setup CREATE | `_gpay_source_id` |

### Path B — FE Shadow via centralized-data-service (NEW from Bug #2 fix yesterday)
| File | Line | Vai trò | Cột dùng |
|---|---|---|---|
| `centralized-data-service/internal/handler/command_handler.go` | 166-167 | Comment header | Annotation |
| | 169 | cdcColumns ALTER | `_gpay_source_id TEXT` |
| | 176 | cdcColumns ALTER | `_gpay_deleted BOOLEAN` |
| | 192-208 | DO block ADD CONSTRAINT | `uq_<t>_gpay_source_id UNIQUE` |
| | 620, 627 | CREATE TABLE inline | `_gpay_source_id TEXT UNIQUE`, `_gpay_deleted BOOLEAN` |
| `centralized-data-service/internal/handler/event_handler.go` | 236 | Tombstone INSERT | `_gpay_source_id` (column write) |

### Path C — Master + Sinkworker (V2 architecture, untouched)
| File | Line | Vai trò | Cột dùng |
|---|---|---|---|
| `centralized-data-service/internal/sinkworker/upsert.go` | 12, 16 | immutableOnUpdate set | `_gpay_source_id` |
| | 30, 67, 118 | SQL `ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted` | partial UNIQUE INDEX target |
| `centralized-data-service/internal/sinkworker/schema_manager.go` | 227, 234, 259-265, 385-397 | CREATE shadow (kafka path) + partial UNIQUE INDEX + systemFieldsSet | `_gpay_source_id NOT NULL`, `_gpay_deleted` |
| `centralized-data-service/internal/sinkworker/sinkworker.go` | 40, 84, 117, 147, 154, 160, 256 | Record build + sourceID extract | `_gpay_source_id` map key |
| `centralized-data-service/internal/sinkworker/envelope.go` | 222 | extractSourceID | comment |
| `centralized-data-service/internal/sinkworker/sinkworker_test.go` | 33, 80, 86, 93, 110, 114, 256, 263, 273 | Tests | `_gpay_source_id`, `_gpay_deleted` |
| `centralized-data-service/internal/service/transmuter.go` | 87, 90, 328, 335, 362, 367, 449, 456 | shadowBatchRow GORM mapping + SELECT FROM shadow + ON CONFLICT (master) | `_gpay_source_id`, `_gpay_deleted` |
| `centralized-data-service/internal/service/master_ddl_generator.go` | 89, 96, 100, 136-137, 148 | CREATE master table + UNIQUE INDEX | `_gpay_source_id NOT NULL`, `_gpay_deleted` |
| `centralized-data-service/internal/service/schema_adapter.go` | 497-498, 527-529 | Conditional metadata cols/values | `_gpay_source_id` (V2 schema branch) |

### Path D — Test fixtures + UI
| File | Line | Vai trò |
|---|---|---|
| `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go` | 23, 30, 46, 53, 58, 62, 92-220 | Test setup uses `_gpay_source_id` as PK + `_gpay_deleted` tombstone |
| `centralized-data-service/test/internal/service/schema_adapter_test.go` | 11-83 | Test V2 schema branch |
| `cdc-cms-web/src/pages/MasterRegistry.tsx` | 68, 425 | FE form default spec placeholder |

## Semantic mapping
| `_gpay_*` (legacy / V2 master) | `source_id` / `_deleted` (FE shadow) | Cùng nghĩa? |
|---|---|---|
| `_gpay_source_id TEXT NOT NULL` | `source_id VARCHAR(200) NOT NULL` | **Có** — đều là external source PK anchor; chỉ khác type (TEXT vs VARCHAR(200)) |
| `_gpay_deleted BOOLEAN DEFAULT FALSE` | `_deleted BOOLEAN DEFAULT FALSE` | **Có** — đều là soft-delete tombstone |

**Nhưng**: V2 partial UNIQUE INDEX `(_gpay_source_id) WHERE NOT _gpay_deleted` thiết kế cho phép tombstone-row giữ lại + cho phép re-INSERT row mới cùng source_id. Đổi tên đơn thuần không break ngữ nghĩa — nhưng phải đổi đồng bộ schema + index + SQL + ORM mapping.

## Drift hiện hữu (chưa phải bug user hỏi)
- `mapping_preview_handler.go` đọc `_gpay_source_id`/`_gpay_id` từ shadow → fail trên path-A tables (`shadow_automator.go` chỉ tạo `id` + `source_id`).
- Workspace `audit-shadow-create-bugs-2026-05-27` fix Bug #2 hôm qua thêm `_gpay_source_id` vào path-B, vô tình làm preview handler "work" trên path-B → che giấu drift trên path-A.
- Cleanup option phải address drift này nếu chọn unify.

## Verify quick (đã đọc, không runtime)
- `shadow_automator.go:78-90`: CREATE table với `id BIGINT PK, source_id VARCHAR(200) NOT NULL, ..., _deleted BOOLEAN DEFAULT FALSE`, CONSTRAINT UNIQUE(source_id).
- `command_handler.go:617-636`: CREATE table với `<pkField> <pkType> PK, _gpay_source_id TEXT UNIQUE, ..., _gpay_deleted BOOLEAN DEFAULT FALSE, _deleted BOOLEAN DEFAULT FALSE`.
- `sinkworker/schema_manager.go:225-237`: CREATE shadow (Kafka path) với `_gpay_id BIGINT PK, _gpay_source_id TEXT NOT NULL, ..., _gpay_deleted BOOLEAN`. Partial UNIQUE INDEX line 262-269.
- `master_ddl_generator.go:87-100`: CREATE master với `_gpay_id BIGINT PK, _gpay_source_id TEXT NOT NULL, _gpay_deleted BOOLEAN`. UNIQUE INDEX `ux_<t>_source_id ON (_gpay_source_id)` line 148.

## Critical assumption to verify với user
**User claim**: "đã có source_id, đã có _deleted".

**Reality**:
- FE shadow tables tạo bởi `shadow_automator.go` (cdc-cms-service) → có `source_id` + `_deleted`. **Đúng**.
- FE shadow tables tạo bởi `command_handler.go HandleCreateDefaultColumns` → có `_gpay_source_id` + `_gpay_deleted` + `_deleted` (sau Bug #2 fix). **Trùng → rác**.
- Master tables + Sinkworker shadow tables → chỉ có `_gpay_source_id` + `_gpay_deleted`. **KHÔNG có `source_id`/`_deleted`**. Nếu user muốn cleanup ở đây thì cần ADD `source_id`/`_deleted` + migration data + DROP `_gpay_*`.
