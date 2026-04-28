# Phase 38 — Task list

| # | Subject | Owner | Status |
|---:|---|---|---|
| T-38.1 | Apply `ALTER ROLE … SET search_path` + lưu migration 039 | Muscle | ✅ |
| T-38.2 | Replace `cdc_internal.*` → `cdc_system.*` (schema_proposal_handler, transmute_schedule_handler) | Muscle | ✅ |
| T-38.3 | Rewrite `transmute_schedule` List/RunNow Raw SQL JOIN master_binding | Muscle | ✅ |
| T-38.4 | Drop `tr.sync_status` + `tr.recon_drift` references trong source_objects_handler | Muscle | ✅ |
| T-38.5 | Flat scan + transpose cho WorkerScheduleResponse | Muscle | ✅ |
| T-38.6 | Build + restart CMS, login `admin/admin123`, curl 11 endpoints | Muscle | ✅ (11/11 200) |
| T-38.7 | Verify auto-flow: connector RUNNING, topics tồn tại, worker không panic | Muscle | ✅ infra-green |
| T-38.8 | Document data-linkage gap registry ↔ Debezium include list (debt cho phase sau) | Brain | ✅ ghi vào 01_requirements/06_validation |
| T-38.9 | 2 lesson mới vào `agent/memory/global/lessons.md` | Brain | ✅ append-only |

## Open follow-ups (debt → phase mới)

- `D-38.A` — Quyết policy cho `shadow_schema='cdc_internal'`: tạo schema thật
  hay đổi binding sang `cdc_system`/schema khác. Cần đụng 9 row +
  shadow_automator + mapping_preview_handler.
- `D-38.B` — Bổ sung migration thêm cột `sync_status`, `recon_drift` cho
  `cdc_table_registry` nếu muốn giữ semantic; hoặc xóa cột khỏi mọi
  fallback khác.
- `D-38.C` — Sync `cdc_system.source_object_registry` với
  Debezium `collection.include.list` để auto-flow không bị starve.
- `D-38.D` — Hoặc đổi Debezium config về collection `payments` nếu đó là
  intent gốc (cần xác nhận với business-owner).
