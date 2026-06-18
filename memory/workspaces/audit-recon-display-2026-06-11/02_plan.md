# Plan: Fix recon display (trọn gói) — khóa pipeline (shadow_schema,shadow_table) + run_id

## Khóa thiết kế
- target_table bare KHÔNG unique → thêm (shadow_schema, shadow_table) làm khóa pipeline (có ở CẢ segment A `entry.ShadowSchema`/`entry.TargetTable` + B `ref.ShadowSchema`/`ref.ShadowTable`).
- run_id (uuid, đã có lib) gom row cùng 1 CheckAll = 1 phiên.
- Read-side query theo (shadow_schema, shadow_table) → unambiguous + gộp A+B; gom phiên theo run_id.

## Layers (bottom-up, verify từng tầng)
1. Migration 085 (cdc-cms-service/migrations/schema/recon_dlq): ADD shadow_schema/shadow_table/run_id + index. [DONE]
2. Worker model reconciliation_report.go: +ShadowSchema/ShadowTable/RunID. [DONE]
3. Worker recon_core.go: gen run_id ở CheckAll + CheckAllSegmentB; set shadow_schema/table/run_id mọi write site (A: entry; B: ref). Build PASS.
4. API cdc-cms-service: model +3 field; recon_read_repo GetTableHistory + ResolveTargetTableByScope query theo (shadow_schema,shadow_table); gom phiên run_id. Build PASS.
5. FE cdc-cms-web: truyền shadow_schema+shadow_table; hiển thị phiên đúng.
6. Verify: build 2 service + test + (runtime: recon chạy → query đúng pipeline).

## Rủi ro
- recon_core nhiều write site (~7) → set keys cẩn thận, build verify.
- Backward-compat: row cũ shadow_schema NULL → read-side fallback target_table (giữ tương thích).
- Migration idempotent (ADD COLUMN IF NOT EXISTS).
