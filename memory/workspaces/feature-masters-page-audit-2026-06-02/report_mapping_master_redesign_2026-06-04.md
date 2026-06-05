# report_mapping_master_redesign_2026-06-04.md

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-04
> Execute #1 (redesign mapping_rule_master) + #2 (gỡ Flatten khỏi master). Báo cáo dựa trên verify THỰC (build/test/migration/JOIN query/transmute live).

## #1 — Redesign mapping_rule_master (link mapping_v2_id, JOIN, KHÔNG copy)
- Bảng mới: `id, master_binding_id, mapping_v2_id(FK), target_column, is_active, status, notes, audit`. **BỎ**: source_field, source_path, data_type, source_data_type, source_format, transform_fn, is_nullable, default_value, is_sensitive, mask_strategy.
- Field nghiệp vụ lấy qua **JOIN mapping_rule_v2** (read-only). Clone filter theo **shadow_binding_id** (đúng nhất) + blacklist system cols, link mapping_v2_id (không copy).
- **"source_format là gì"**: cách extract (raw/jsonpath/expression) — thuộc tính rule nghiệp vụ ở mapping_rule_v2 → bỏ khỏi master, lấy qua JOIN.

## #2 — Gỡ "Scan Array (Flatten)" khỏi Master
- BE: xoá route flatten + method Flatten + helpers (discoverJsonPaths/extractPaths/normalizeTargetColumn/sanitizeIdentifier) + **bỏ shadowDB khỏi MasterMappingRuleHandler** → master KHÔNG còn kết nối shadow data plane.
- FE: xoá nút Scan Array (Flatten) + modal + handlers + cột Sensitive/Mask + data_type-edit + Add/Edit modal.

## Những file đã thay đổi
| File | Repo | Thay đổi | ~LoC |
|------|------|----------|------|
| `migrations/schema/cdc_system_model/075_redesign_mapping_rule_master.sql` | cms | NEW: DROP+CREATE schema mới | +40 |
| `internal/service/transmuter.go` | worker | loadRules JOIN mapping_rule_v2 | +6/-4 |
| `internal/service/master_ddl_generator.go` | worker | DDL query JOIN mapping_rule_v2 | +5/-4 |
| `internal/domain/mapping/master_rule.go` | cms | struct: +mapping_v2_id, joined read-only, bỏ sensitive/mask | +14/-14 |
| `internal/infra/persistence/master_mapping_rule_repo_gorm.go` | cms | rewrite: List/Get JOIN, Save theo mapping_v2_id | ~+120/-150 (gọn hơn) |
| `internal/api/master_mapping_rule_handler.go` | cms | rewrite: bỏ Flatten+shadowDB, Save theo mapping_v2_id | ~-200 (386→180 dòng) |
| `internal/app/commands/create_master.go` | cms | clone → INSERT...SELECT link mapping_v2_id filter shadow_binding_id | +22/-70 |
| `internal/router/router.go` | cms | gỡ route flatten | -1 |
| `internal/server/server.go` | cms | bỏ shadowDB khỏi master handler ctor | +1/-1 |
| `src/pages/MasterMappingFieldsPage.tsx` | web | rewrite lean (read-only join, bỏ flatten/sensitive/mask/add) | ~-350 (640→290 dòng) |

## Verify (THỰC TẾ)
- Build: worker `go build`=0 + test service/handler PASS; CMS `go build`=0; FE `tsc -b`=0 + `npm build` ✓.
- Restart worker + CMS → **migration 075 applied** (CMS log "migration applied 075"; schema confirm có `mapping_v2_id`, không còn source_field/data_type/sensitive/mask).
- LIVE clone: `INSERT...SELECT` filter shadow_binding_id=66 → **9 rule** link mapping_v2_id (14 v2 business − 5 system blacklist).
- LIVE worker JOIN: query trả `target_column | source_field | data_type | source_format` lấy **từ mapping_rule_v2** (đúng, không copy).
- LIVE transmute: trigger `cdc.cmd.transmute` sssss → activity_log `scanned=453` (worker JOIN chạy thật trong process) → status `degraded` vì sssss=flatten thiếu explode_path (đúng — P0-1 guard báo rõ, KHÔNG phải lỗi redesign).
- flatten route: gỡ khỏi code + build (POST trả 401 do auth gate trước; route không còn đăng ký).

## Lưu ý
- Master `sssss` vẫn degraded do **config flatten** (không có explode_path); muốn ra data thì tạo master `copy_1_to_1` (clone sẽ link 9 business rule qua mapping_v2_id → transmute ghi master). KHÔNG tự đổi config sssss.
- Đã populate 9 rule cho sssss (mô phỏng clone, created_by='verify') để verify JOIN — là dữ liệu đúng (clone production tạo y vậy).
- Toàn bộ thay đổi (gồm các phiên trước) vẫn **uncommitted** (chưa commit theo §8, chờ User).
