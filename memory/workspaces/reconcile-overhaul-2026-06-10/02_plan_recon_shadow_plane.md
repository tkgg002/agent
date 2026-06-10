# 02_plan_recon_shadow_plane.md — Reconcile Overhaul (Phase 1: hồi sinh CHECK path)

> Workspace: `reconcile-overhaul-2026-06-10` | 2026-06-10 | Muscle:Claude-Opus-4.8

## Tổng quan hiện trạng (verified bằng DB + trigger thật)
- Kiến trúc recon 3-tier (count_windowed / hash_window / bucket_hash) + heal + DLQ retry: code đầy đủ, lock 4 lớp, metric đủ.
- **NHƯNG recon CHƯA TỪNG chạy được trên kiến trúc hybrid Path B**:
  - `recon_runs` = 0 rows; `cdc_reconciliation_report` = 0 rows.
  - Trigger thật `cdc.cmd.recon-check {"tier":"1","table":"*"}` → activity_log `success` **tables_checked=0** (false-positive).
- `cdc_worker_schedule.reconcile` is_enabled=false (tắt từ 06-02).

## Root cause (3 lớp, code V1 public@5433 vs data V2 shadow_*@5436)
1. `synthesizeLegacyTableRegistry` KHÔNG mang `binding.ShadowSchema` → entry chỉ có TargetTable trần.
2. `CheckAll` gate `schemaAdapter.GetSchema(t)` = `GetSchemaInSchema("public", t)` trên **db 5433** → nil mọi bảng → skip hết → 0 checked + "success" im lặng.
3. `ReconDestAgent` nhận `db` 5433 + mọi query `FROM quoteIdent(table)` không schema-qualify → kể cả bỏ gate cũng đếm sai chỗ.

## Giải pháp (duy nhất, minimal, backward-compat V1)
1. `model/table_registry.go`: +field `ShadowSchema string` (`gorm:"-"` — synthetic, không đụng bảng V1) + method `QualifiedTarget()` (`schema.table`; schema rỗng → table trần).
2. `metadata_registry_service.go`: synthesize set `ShadowSchema: binding.ShadowSchema`.
3. `recon_dest_agent.go`: +`quoteRelation()` (tách `schema.table` quote từng phần; không dấu chấm → như `quoteIdent` cũ) — thay tại các vị trí quote TABLE (không đụng pkColumn).
4. `recon_core.go`: gate dùng `GetSchemaInSchema(shadowSchemaOf(entry), ...)`; call sites dest truyền `entry.QualifiedTarget()`; CheckAll đếm `skipped_no_schema` + Warn khi checked=0.
5. `worker_server.go`: ReconCore + DestAgent nhận **shadowDB** (fallback = db khi không config → V1 không đổi hành vi); adapter riêng `NewSchemaAdapter(shadowDB)` cho recon (KHÔNG đụng adapter của batchBuffer/ingest — giữ cô lập source→shadow).
6. `recon_handler.go`: CheckAll trả 0 → activity status `warning` (visibility, lesson L-CDC silent-skip).
7. FE 1 dòng: `DataIntegrity.tsx` invalidate key orphan `recon-status` → `recon-report`.

## Out-of-scope (phase sau, ghi nhận)
- Healer/Backfill/TsDetector đang disabled khi không có default Mongo client (cần per-source refactor); `extractSourceTsFromDoc` hardcode `updated_at` (G10); Tier3→Tier2 re-lock (G1); cap missing IDs (G3/G5); FE job_id poll/X-Action-Reason/TableHistory (GAP2/3/4).

## Verify plan
build+vet+test worker → restart worker local → re-trigger recon-check ALL → assert `recon_runs` có rows + report có data + activity không còn success-0.
