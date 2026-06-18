
## [2026-06-11][Agent:claude-opus-4-8] BẮT ĐẦU fix trọn gói recon display (khóa pipeline + run_id)
- T1 DONE: migration 085_recon_pipeline_key.sql (ADD shadow_schema/shadow_table/run_id + ix_recon_pipeline + ix_recon_run, idempotent ADD COLUMN IF NOT EXISTS). cdc-cms-service/migrations/schema/recon_dlq.
- T2 DONE: worker model reconciliation_report.go +ShadowSchema/ShadowTable/RunID.
- Khóa: (shadow_schema, shadow_table) unique pipeline ở cả A (entry) + B (ref) → gộp A+B, hết collision export_jobs. run_id=uuid gom phiên.
- TODO: T3 recon_core (run_id + set keys ~7 write site), T4-5 API, T6 FE, T7 verify.

## [2026-06-11][Agent:claude-opus-4-8] T3 DONE worker recon_core (run_id + khóa pipeline)
- TableRegistry +RunID (gorm:"-" transient); MasterBindingRef +RunID; ReconciliationReport +ShadowSchema/ShadowTable/RunID.
- Helper stampA(report,entry)/stampB(report,ref): set shadow_schema/shadow_table/run_id rồi Create. Thay 6 site segment-A (RunTier1/2/3 + orphan + errorReport) → stampA; 1 site segment-B (RunSegmentB) → stampB. Chỉ còn 2 rc.db.Create trong helper.
- CheckAll: runID=uuid.NewString() + entry.RunID=runID trước RunTier1. CheckAllSegmentB: runID + ref.RunID trước RunSegmentB.
- Chiến lược tránh đệ quy: đổi B trước → replace_all A → mới thêm helper.
- Verify: gofmt clean; go build ./... PASS; grep xác nhận stamp đủ.
- Backward-compat: row cũ shadow_schema/run_id NULL → read-side cần fallback.

## [2026-06-11][Agent:claude-opus-4-8] T4-5 DONE API (cdc-cms-service) query theo khóa pipeline
- Model reconciliation_report.go +ShadowSchema/ShadowTable/RunID.
- recon_read_repo GetTableHistory(table, shadowSchema, page, size): shadowSchema!="" → WHERE shadow_schema=? AND shadow_table=? (gộp A+B, hết collision); rỗng → fallback target_table (legacy/row cũ NULL).
- ReconReader interface + GetTableHistoryQuery +ShadowSchema; Handle truyền; TableHistory handler đọc c.Query("shadow_schema").
- Fix stub test (signature + histSchema field).
- Verify: gofmt; go build ./... PASS; go test TestGetTableHistory PASS.

## [2026-06-11][Agent:claude-opus-4-8] T6 FE + T7 verify DONE — trọn gói recon display
- T6 FE: useReconStatus.useTableHistory(table, shadowSchema?) → truyền ?shadow_schema. ReconPipelineGrid.DrillDown: historyTable = rowA/rowB.shadow_table (thay masterName bare), historySchema = rowA/rowB.shadow_schema → query theo KHÓA pipeline (gộp A+B, hết collision).
- API thêm: listLatestPrimary COALESCE(r.shadow_schema, sb.shadow_schema) AS shadow_schema (đặt cuối → GORM thắng cột trùng r.*), seg B dùng stamped; DISTINCT ON (shadow_schema,target_table,segment) → pipeline trùng tên không bị gộp.
- FIX build: backtick quanh export_jobs trong comment SQL đóng sớm Go raw-string → bỏ backtick.
- T7 verify: migration 085 applied dev (idempotent OK); cdc-cms-service go build PASS + go test TestGetTableHistory PASS; centralized-data-service go build PASS; cdc-cms-web tsc -b PASS; query COALESCE+DISTINCT chạy trên data thật OK.
- CHƯA verify runtime end-to-end: cần redeploy worker (stamp shadow_schema/run_id row mới) + cms + FE → recon re-run ~30min → grid/DrillDown hiện đủ A+B per pipeline + hết "ko hiện gì". Old rows seg-B-khác-tên (export_jobs_mt) NULL tới khi re-stamp.

## [2026-06-11][Agent:claude-opus-4-8] T7 RUNTIME VERIFIED — fix recon display chạy đúng trên data thật
- User đã redeploy + chạy recon. Query cdc_dw:5433 xác nhận:
- Stamp: 69/112 row (1h) có shadow_schema+run_id (43 cũ NULL). 7 phiên.
- Gộp A+B theo (shadow_schema,shadow_table): shadow_dev000.export_jobs A=4/B=12; export_jobs_testid1, export_jobs_4, wallet_capsets đều có A+B.
- export_jobs_mt (seg B trước NULL) → stamp shadow_dev000.export_jobs → master B nối đúng pipeline shadow.
- Mô phỏng GetTableHistory(shadow_dev000,export_jobs) → trả CẢ source_shadow + shadow_master = "30 phiên"/"biến động" hiện đủ A+B. ROOT FIX confirmed.
- run_id gom phiên đúng: CheckAll seg A 12 pipeline/run; CheckAllSegmentB 4 pipeline/run.
- Lưu ý môi trường (không phải bug): seg A export_jobs status=error src=0 = source Mongo prod unreachable từ dev (Task 9). Row vẫn stamp+nối đúng.
- DONE G1-G8: code build/test PASS + runtime data thật PASS. Còn UI confirm trực quan từ user.

## [2026-06-12][Agent:claude-opus-4-8] FIX over-group segment B: 1 shadow fan-out N master
- User phát hiện: pipeline shadow_dev000.export_jobs→aaa hiện 4 dòng Segment B/phiên. Verify: aaa/aaaa2/export_jobs_mt/export_jobs_mt_02 CÙNG đọc shadow_dev000.export_jobs → khóa (shadow_schema,shadow_table) gom CHUNG seg B của 4 master.
- Root: khóa pipeline đúng cho seg A (source→shadow, dùng chung), nhưng seg B là per-master (shadow→master) → cần scope thêm master_table.
- Fix (READ-side, KHÔNG đụng worker — data đã stamp đúng): GetTableHistory(table, shadowSchema, masterTable) → where seg B thêm `target_table=masterTable`; seg A không lọc. + interface/query/handler(c.Query master_table) + FE useTableHistory(...,masterTable) + DrillDown truyền rowB.target_table. + stub test.
- Verify: go build+test PASS; tsc PASS; query scoped (shadow_dev000,export_jobs,aaa) → 1 A + 1 B/phiên (hết 4 B). 
- Deploy: chỉ cần redeploy cms (API) + FE — worker data sẵn sàng (đã stamp). KHÔNG cần re-run recon.
- Global Pattern: khóa định danh phải khớp CARDINALITY từng quan hệ: source→shadow (N:1 nhiều master dùng chung shadow) khác shadow→master (1:1 per master). Khóa chung cho leg dùng-chung, khóa + discriminator cho leg per-entity. Đừng dùng 1 khóa cho mọi segment khi fan-out khác nhau.
