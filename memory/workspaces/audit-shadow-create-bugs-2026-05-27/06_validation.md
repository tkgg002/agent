# 06_validation — Audit Shadow Create Bugs

## Build verify (baseline trước fix)
Lý do verify baseline: chứng minh state hiện tại compile sạch, để khi áp fix ở phase sau, build PASS = do fix đúng (không nhiễu noise).

| Service | Command | Result |
|---|---|---|
| `centralized-data-service` | `go build ./...` | **PASS** (no output, exit 0) |
| `cdc-cms-service` | `go build ./...` | **PASS** (no output, exit 0) |
| `cdc-cms-web` | `npx vite build` | **PASS** (built in 797ms, 9 chunks) |

## Static verification (code-reading evidence)

### Bug 1 evidence
- Repo file `mapping_rule_v2_repo.go` line 54-61 — JOIN by `so.source_object_name` only.
- Caller `command_handler.go` line 649 — pass `payload.SourceTable` (string).
- Alternative API có sẵn: `ListActiveBySourceObject(ctx, sourceObjectID)` line 37-44 — filter bằng `source_object_id`.

### Bug 2 evidence
- `command_handler.go:586-602` (CREATE TABLE) — list cột thiếu `_source_ts`, `_gpay_source_id`, `_gpay_deleted`.
- `command_handler.go:163-172` (`ensureCDCColumnsInSchema`) — cdcColumns slice thiếu cùng 3 cột.
- Cross-check:
  - `sinkworker/schema_manager.go:231` → `"_source_ts" BIGINT` (path runtime, đúng spec).
  - `sinkworker/upsert.go:69-122` → `EXCLUDED._source_ts > shadow._source_ts` (OCC guard tham chiếu cột, sẽ crash nếu shadow thiếu).
  - `service/master_ddl_generator.go:92` + `service/transmuter.go:89` → đều dùng `_source_ts`.
  - `recon/recon_handler.go:263` → recon hash dùng `_source_ts`.

→ Evidence cho thấy `_source_ts` là cột mandatory, được tham chiếu rộng rãi. Bug 2 là drift của 1 path DDL builder (FE-trigger) so với spec của các path runtime.

## Runtime verify (sau khi user approve + apply fix — chưa thực thi)
1. FE `/shadow` → tạo `sd_test_audit_001`, đợi NATS handler hoàn tất.
2. `psql -h localhost -p 5436 -U postgres -d shadow -c "\\d+ shadow_xxx.sd_test_audit_001"` → kỳ vọng có đủ:
   - `id` (hoặc PK) + `_gpay_source_id TEXT UNIQUE`
   - `_raw_data jsonb`, `_source varchar(20)`, `_source_ts bigint`
   - `_synced_at timestamp`, `_version bigint`, `_hash varchar(64)`
   - `_gpay_deleted boolean`, `_deleted boolean`
   - `_created_at timestamp`, `_updated_at timestamp`
   - Indexes: `idx_<table>_raw` (GIN), `idx_<table>_source_ts`
   - Constraint: `uq_<table>_gpay_source_id`
3. Cross-leak test: 
   - Tạo registry `A: src=export_jobs, target=sd_aaa`, approve mapping rules `field1,field2` cho A.
   - Tạo registry `B: src=export_jobs, target=sd_bbb`.
   - psql `\\d+ shadow_xxx.sd_bbb` → kỳ vọng CHỈ có 11 cột system + PK. KHÔNG có `field1`, `field2`.
4. OCC guard test (sinkworker ingest):
   - Insert event vào `sd_test_audit_001` với `_source_ts=1000`.
   - Insert event cùng PK với `_source_ts=500` (older).
   - Kỳ vọng: row không bị overwrite (older-wins skipped). Query `_source_ts` = 1000.

## Files đã thay đổi trong audit phase
| File | LOC delta |
|---|---|
| (source code) | **0 dòng** — audit-only, không sửa code |
| workspace docs | +9 files (see report) |
