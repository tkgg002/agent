# 00_context — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Trigger
User report 2026-05-28:
> "_gpay_source_id đã có source_id, _gpay_deleted đã có _deleted. audit và bỏ toàn bộ các logic liên quan 2 field này. nó đang là rác kỹ thuật."

## Background
- Workspace `audit-shadow-create-bugs-2026-05-27` đã apply Bug #2 fix hôm qua (17:30 ICT), thêm `_gpay_source_id TEXT UNIQUE` + `_gpay_deleted BOOLEAN DEFAULT FALSE` vào `command_handler.go` CREATE TABLE và `cdcColumns[]` ALTER list. User bây giờ chỉ ra rằng 2 cột này thừa vì shadow tables vốn đã có `source_id` + `_deleted` (do `shadow_automator.go` tạo ra ở path khác).

## Symptom (rác kỹ thuật)
| Bảng | Cột anchor | Cột tombstone |
|---|---|---|
| FE shadow tạo bởi `cdc-cms-service/shadow_automator.go` | `source_id VARCHAR(200) UNIQUE` | `_deleted BOOLEAN` |
| FE shadow tạo bởi `centralized-data-service/HandleCreateDefaultColumns` (sau fix Bug #2 hôm qua) | `_gpay_source_id TEXT UNIQUE` + `<pkField> PK` | `_gpay_deleted` + `_deleted` (cả 2!) |
| Master tables (sinkworker + master_ddl_generator + transmuter) | `_gpay_source_id TEXT NOT NULL` + `_gpay_id BIGINT PK` | `_gpay_deleted` |

Cùng concept (external source anchor + soft-delete) nhưng 2 naming convention song song → cleanup confusion + drift risk.

## Scope tự nhiên
Grep `_gpay_source_id|_gpay_deleted` thu được **104 references** trải qua 3 service:
- `centralized-data-service/internal/sinkworker/` (5 files): upsert.go, schema_manager.go, sinkworker.go, envelope.go, sinkworker_test.go.
- `centralized-data-service/internal/service/` (3 files): transmuter.go, master_ddl_generator.go, schema_adapter.go.
- `centralized-data-service/internal/handler/` (2 files): command_handler.go (vừa thêm hôm qua), event_handler.go (tombstone INSERT).
- `centralized-data-service/test/` (2 files): schema_adapter_ordering_test.go, schema_adapter_test.go.
- `cdc-cms-service/internal/api/mapping_preview_handler.go`.
- `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go`.
- `cdc-cms-web/src/pages/MasterRegistry.tsx`.

## Critical observation: 2 path architecture
**Path A — FE Shadow (cdc-cms-service)**:
- `shadow_automator.go` tạo shadow với `source_id VARCHAR(200) NOT NULL UNIQUE` + `_deleted BOOLEAN DEFAULT FALSE`.
- `mapping_preview_handler.go` đọc `_gpay_source_id`/`_gpay_id` từ shadow để preview — **đây có thể là drift**: shadow_automator tạo `source_id` nhưng preview đọc `_gpay_source_id`. Cần xác minh runtime.

**Path B — FE Shadow (centralized-data-service `HandleCreateDefaultColumns`)**:
- Vừa thêm hôm qua: `_gpay_source_id TEXT UNIQUE` + `_gpay_deleted BOOLEAN DEFAULT FALSE`.
- Đây là path tôi MUSCLE-applied yesterday cho Bug #2.

**Path C — Master tables (sinkworker + master_ddl_generator + transmuter)**:
- `sinkworker/schema_manager.go createShadowTable` (kafka-consumer path) tạo shadow độc lập với `_gpay_id BIGINT PRIMARY KEY` + `_gpay_source_id TEXT NOT NULL` + `_gpay_deleted` + partial UNIQUE INDEX `(_gpay_source_id) WHERE NOT _gpay_deleted`.
- `master_ddl_generator.go` tạo master tables tương tự cấu trúc.
- `sinkworker/upsert.go buildUpsertSQL` dùng `ON CONFLICT (_gpay_source_id) WHERE NOT _gpay_deleted` — partial unique index.
- `transmuter.go shadowBatchRow` đọc `_gpay_source_id`/`_gpay_deleted` từ shadow → ghi vào master.
- Master tables KHÔNG có `source_id`/`_deleted` (chỉ có `_gpay_*` variants).

## Risk preview
| Action | Risk | Reversibility |
|---|---|---|
| Rollback `_gpay_*` ở `command_handler.go` (Bug #2 portion) | Low | Easy (re-apply nếu cần) |
| Bỏ `_gpay_*` ở `event_handler.go:236` tombstone INSERT | Low-Med | Easy nếu chưa deploy |
| Bỏ `_gpay_*` ở `sinkworker/upsert.go ON CONFLICT` | **High** | Production master tables có thể đã có data; partial UNIQUE INDEX phải drop & recreate; data migration |
| Bỏ `_gpay_*` ở `master_ddl_generator.go` | **High** | Tương tự |
| Bỏ `_gpay_*` ở `transmuter.go` | **High** | Cross-cutting; transmuter là CDC pipeline core |
| Cập nhật test (schema_adapter_test.go, sinkworker_test.go) | Low | Easy |
| Cập nhật FE `MasterRegistry.tsx` placeholder | Low | Easy |

## User constraint
- Đọc lesson trước → đã đọc `lessons.md` đặc biệt lesson `2026-05-20 "Verify ở destination"` và lesson `2026-05-20 "Bump dependency version trước khi reproduce bug = anti-pattern"`.
- Core /agent + GEMINI.md → đọc.
- Chỉ làm đúng yêu cầu.
- Không cheat DB hay đổi config.
- Plan rõ ràng + code demo chi tiết.
- Report dựa trên kết quả tính toán thực tế, file thay đổi + LOC delta.
- Verify build/test trước khi báo done.
- Luôn có `report_*.md`.

## Cross-reference lesson
- `lessons.md` 2026-05-20 line 3433-3450 "Bump dependency version trước khi reproduce bug = anti-pattern" → cảnh báo KHÔNG over-correct theo feedback. User chỉ ra "rác kỹ thuật" nhưng scope "toàn bộ" có nhiều layer khác nhau; cần kiểm chứng impact mỗi layer trước khi cut.
- `lessons.md` 2026-05-20 line 3415-3429 "Verify ở destination" → mỗi option phải có verify plan ở destination thực sự (PG shadow + master).
