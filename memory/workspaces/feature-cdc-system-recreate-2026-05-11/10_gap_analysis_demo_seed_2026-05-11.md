# 10 — Gap Analysis: Demo Seed Leak + Table Usage Audit

> **Date**: 2026-05-11
> **Trigger**: User phát hiện `cdc_system.source_object_registry` chứa 10 row demo (goopay_wallet/payment/order/main + 2 mysql legacy) sau khi cold-boot service qua `make run`. Quote: "mớ demo này là gì. chạy product mà mày add tùm lum vậy à."
> **Rule mới user giao**: "khi tạo 1 table migration, tự check lại hệ thống xem 2 thằng api và cdc-worker có xài ko. rồi báo cáo lại tao."

## §1 Findings — DB state thực tế (docker exec verify)

| Table | Rows | Nguồn | Đánh giá |
|---|---:|---|---|
| `cdc_system.cdc_table_registry` | **10** | `001_init_schema.sql:228-241` hardcode INSERT 10 pilot rows | 🔴 DEMO LEAK |
| `cdc_system.source_object_registry` | **11** | `035_v2_backfill_legacy_registry.sql:99` fan-out từ cdc_table_registry (10) + `049_mariadb_seed_legacy_orders.sql` (1) | 🔴 DEMO LEAK (chain từ #1) |
| `cdc_system.shadow_binding` | **10** | `035` fan-out từ source_object_registry (10 cdc_table_registry rows, 049 chưa có shadow_binding vì 049 inactive) | 🔴 DEMO LEAK (chain) |
| `cdc_system.connection_registry` | **4** | `035` (3 legacy_*) + `049` (mariadb_legacy_default) | 🟡 Mixed: 3 generic config + 1 demo |
| `cdc_system.master_binding` | 0 | — | OK (cdc_internal.master_table_registry empty before 035 ran) |
| `cdc_system.mapping_rule_v2` | 0 | — | OK (cdc_mapping_rules empty before 035) |
| `cdc_system.cdc_mapping_rules` | 0 | — | OK |
| `cdc_system.cdc_worker_schedule` | **6** | 5 từ `007_worker_schedule.sql` (bridge, transform, field-scan, partition-check, airbyte-sync) + 1 row `reconcile/30m` (chưa truy được nguồn — không có grep match trong migrations/*.sql, không có grep match trong Go code) | 🟢 4 row OK (config) / 🟡 1 row reconcile cần investigate |
| `cdc_system.enum_types` | **3** | `020_mapping_rule_jsonpath.sql:93` (payment_state, api_type, currency_iso) | 🟡 Domain seed |

## §2 Migration audit — 5 chỗ INSERT data

| File | Loại seed | Rows | Risk | Khuyến nghị |
|---|---|---|---|---|
| `001_init_schema.sql:228-241` | Pilot demo (goopay_wallet/payment/order/main/legacy) | 10 | 🔴 HIGH — root cause | Xoá khỏi migration, move sang `scripts/seed_dev.sql` |
| `007_worker_schedule.sql:25-31` | Config defaults (bridge/transform/field-scan/partition-check/airbyte-sync) | 5 | 🟢 OK — runtime config | GIỮ |
| `020_mapping_rule_jsonpath.sql:93` | Enum domain values (payment_state/api_type/currency_iso) | 3 | 🟡 Medium — domain coupling | Có thể giữ (enum metadata), hoặc tách dev seed |
| `035_v2_backfill_legacy_registry.sql:6/35/64/99/177/217/256` | 3 connection rows + 4 SELECT-fan-out (registry/binding/mapping) | 3 fixed + dynamic | 🔴 HIGH — amplifier | Xoá 4 fan-out, giữ 3 connection nếu dùng được; hoặc xoá toàn bộ migration |
| `036_v2_transmute_schedule.sql:34` | Migrate schedule legacy → V2 | Dynamic | 🟡 Dependent on legacy | Sẽ no-op vì legacy table rỗng, có thể giữ |
| `049_mariadb_seed_legacy_orders.sql:12/48` | MariaDB pilot + legacy_orders draft | 2 | 🟡 Demo nhưng inactive=false, draft | Move sang `scripts/seed_dev.sql` |

## §3 Table usage audit — 43 tables × 2 service

Audit qua Agent Explore, grep `*.go` trên `cdc-cms-service/` + `centralized-data-service/`, match: bare-name TableName(), schema-qualified TableName(), raw SQL.

### §3.1 USED-BY-BOTH (18 tables)

`cdc_table_registry`, `cdc_mapping_rules`, `pending_fields`, `schema_changes_log`, `cdc_activity_log`, `cdc_worker_schedule`, `cdc_reconciliation_report`, `failed_sync_logs`, `recon_runs`, `schema_proposal`, `connection_registry`, `source_object_registry`, `shadow_binding`, `master_binding`, `mapping_rule_v2`, `sync_runtime_state`, `transmute_schedule`, `cdc_jobs`.

### §3.2 USED-BY-CMS-ONLY (4 tables)

`sources`, `cdc_wizard_sessions`, `admin_actions`, `cdc_alerts`.

### §3.3 USED-BY-WORKER-ONLY (2 tables)

- `worker_registry` — qua PL/pgSQL `claim_machine_id` + `heartbeat_machine_id` được worker gọi từ `cmd/sinkworker/main.go:70` + `:250`.
- `enum_types` — qua raw SQL `SELECT FROM cdc_system.enum_types` trong `type_resolver.go`.

### §3.4 UNUSED — candidate xoá (2 tables verified, 3rd claim sai)

| Table | Verified? | Note |
|---|---|---|
| `cdc_system.table_registry_legacy` | ✓ exists, 0 rows | Rename leftover từ migration 037. 0 Go reference. |
| `cdc_system.master_table_registry_legacy` | ✓ exists, 0 rows | Rename leftover từ 037/038. Go code `master_registry_handler_resolve.go:20` query `cdc_system.master_table_registry` (KHÔNG còn tồn tại) → handler dead code/bug (sẽ throw `relation does not exist` mỗi khi endpoint được hit). |
| `cdc_system.transmute_schedule_legacy` | ✗ **NOT EXISTS** | Audit agent claim sai. Migration 037/038 conditional rename, nhưng `cdc_internal.transmute_schedule` rỗng nên không thực sự rename. Không cần action. |

## §4 Disagreement với audit agent (corrections)

| Audit agent claim | Reality | Action |
|---|---|---|
| `transmute_schedule_legacy` exists & unused | KHÔNG tồn tại trong cdc_system | Bỏ khỏi candidate xoá |
| `cdc_worker_schedule` có 5 row (từ 007) | Có **6 row** trong DB | Row #6 `reconcile/30m` chưa truy được nguồn — investigate riêng |

## §5 Bug phụ phát hiện qua audit

**`master_registry_handler_resolve.go:20`** query `FROM cdc_system.master_table_registry` — table này KHÔNG tồn tại (đã rename thành `_legacy` bởi migration 037/038). Endpoint sẽ throw `relation "cdc_system.master_table_registry" does not exist` mỗi khi được hit từ FE. Hiện tại không có log error vì endpoint chưa được gọi trong probe loop.

Fix khả dĩ:
- (a) Xoá handler nếu UI không còn dùng.
- (b) Sửa query → `cdc_system.master_binding` (V2) hoặc `cdc_system.master_table_registry_legacy`.
- Cần xem FE/route có gọi endpoint này không trước khi quyết định.

## §6 Đề xuất fix — 3 nhóm, chờ user approve

### Fix #1 — Tách demo seed khỏi production migrations (PRIORITY 🔴)
1. `001_init_schema.sql`: xoá block INSERT 10 row line 228-241 + `SELECT create_all_pending_cdc_tables();` line 244.
2. `035_v2_backfill_legacy_registry.sql`: review xem 3 connection_registry rows (legacy_system_db, legacy_shadow_default, legacy_master_default) có còn dùng không, nếu không thì xoá luôn. Xoá 4 fan-out SELECT blocks (source_object/shadow/master/mapping).
3. `049_mariadb_seed_legacy_orders.sql`: xoá hoặc move.
4. Tạo `cdc-cms-service/scripts/seed_dev.sql` chứa toàn bộ data trên + `Makefile` target `make seed-dev`.

### Fix #2 — Wipe dead schema + dead code
1. Tạo `053_drop_legacy_rename_leftovers.sql`: `DROP TABLE IF EXISTS cdc_system.table_registry_legacy, cdc_system.master_table_registry_legacy CASCADE;`
2. Xử lý `master_registry_handler_resolve.go:20` — xoá handler hoặc fix query.

### Fix #3 — Process rule (đã append lessons.md hôm nay)
- Global Pattern Z — production migration seed leak (lessons.md).
- New rule: audit table usage on both consumer services BEFORE merging migration (lessons.md).

## §7 Skill đã dùng
Read, Bash (grep, docker exec psql, wc), Edit, Write, Agent Explore (very thorough — audit 43 tables × 2 codebase × functions × seed inserts).
