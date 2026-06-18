# REPORT — Fix recon display "trọn gói" (khóa pipeline shadow_schema+shadow_table + run_id)
Ngày 2026-06-11 | Agent claude-opus-4-8 | 3 service: centralized-data-service (worker), cdc-cms-service (API), cdc-cms-web (FE)

## Vấn đề (user)
recon chạy 30p/lần nhưng "Nhật ký đối soát (30 phiên)" + "Biến động số lượng theo phiên" hiển thị sai, có pipeline "ko hiện gì".

## Root cause (audit 10_gap_analysis)
cdc_reconciliation_report định danh pipeline bằng `target_table` (bare name) KHÔNG unique: (a) trùng tên across-schema (export_jobs ở 2 schema) → trộn/ambiguous; (b) segment A ghi tên shadow / B ghi tên master → 1 pipeline tách 2 tên, read theo 1 tên mất nửa lịch sử / DrillDown "ko hiện".

## Giải pháp: KHÓA PIPELINE (shadow_schema, shadow_table) có ở CẢ 2 segment + run_id gom phiên.

## Thay đổi theo tầng
### 1. Migration (cdc-cms-service/migrations/schema/recon_dlq/085_recon_pipeline_key.sql, NEW 24 dòng)
ADD shadow_schema/shadow_table/run_id + index (shadow_schema,shadow_table,checked_at), (run_id). Idempotent. ĐÃ apply dev.

### 2. Worker (centralized-data-service)
- model reconciliation_report.go +ShadowSchema/ShadowTable/RunID; table_registry.go +RunID (transient); MasterBindingRef +RunID.
- recon_core.go: helper stampA(entry)/stampB(ref) set khóa+run_id rồi Create; thay 6 site seg-A + 1 site seg-B; CheckAll/CheckAllSegmentB gen run_id (uuid) + set vào entry/ref.

### 3. API (cdc-cms-service)
- model +3 field. GetTableHistory(table, shadowSchema): shadowSchema!="" → WHERE shadow_schema=? AND shadow_table=? (gộp A+B, hết collision); rỗng → fallback target_table. Interface+Query+Handler+TableHistory đọc ?shadow_schema.
- listLatestPrimary: COALESCE(r.shadow_schema, sb.shadow_schema) AS shadow_schema (đặt cuối → thắng cột trùng r.*; seg B dùng stamped); DISTINCT ON (shadow_schema,target_table,segment) → pipeline trùng tên không gộp.

### 4. FE (cdc-cms-web)
- useReconStatus.useTableHistory(table, shadowSchema?) → ?shadow_schema. ReconPipelineGrid.DrillDown: historyTable=rowA/rowB.shadow_table (thay masterName bare) + historySchema → query khóa pipeline.

## Files & LOC
| Service | File | ghi chú |
|---|---|---|
| cms | 085_*.sql (NEW 24) + model(+23) + recon_read_repo(+61) + recon_reader(+8) + get_table_history(+9) + handler_reports(+7) + queries_test(+25) | |
| worker | reconciliation_report(+21) + table_registry(+32) + recon_core (stamp+run_id ~40 dòng của task này; phần lớn diff 949 là Task 9) | |
| FE | useReconStatus.ts(+52, tracked) + ReconPipelineGrid.tsx (DrillDown ~6 dòng, UNTRACKED) | |

## Verify (Rule 16-G3/G8)
- Migration 085 apply dev: OK (ALTER + 2 INDEX).
- cdc-cms-service: go build ./... PASS; go test TestGetTableHistory PASS.
- centralized-data-service: go build ./... PASS.
- cdc-cms-web: tsc -b PASS (0 error recon files).
- Query thật: COALESCE + DISTINCT ON (shadow_schema,target_table,segment) chạy OK; seg A + same-name seg B có khóa; seg-B-khác-tên (export_jobs_mt) NULL tới khi worker re-stamp.
- Fix lỗi build: backtick quanh export_jobs trong comment SQL đóng sớm Go raw-string → bỏ.

## CHƯA Done runtime (cần user)
Redeploy worker (stamp shadow_schema/shadow_table/run_id cho row recon MỚI) + cms + FE → recon re-run (~30min) → tôi query đúng plane verify: grid + DrillDown hiện ĐỦ A+B per pipeline, pipeline trùng tên (export_jobs) tách riêng, hết "ko hiện gì". Old rows seg-B-khác-tên NULL tới khi re-stamp (transitional, tự lành).

## Restore-point (Rule 18)
ReconPipelineGrid.tsx UNTRACKED (git không revert được về bản trước-edit) — chỉ đổi 3 dòng DrillDown, additive. Các file khác tracked → git checkout revert được.
