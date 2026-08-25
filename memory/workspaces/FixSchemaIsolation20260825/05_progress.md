# 05_progress.md — Audit Log (Append-Only)

- [2026-08-25T14:35:00+07:00] [Agent:DeepSeek-R1] Initialized workspace FixSchemaIsolation20260825. Completed full-codebase scan for bare-table usages and schema collision vulnerabilities.
- [2026-08-25T14:50:00+07:00] [Agent:DeepSeek-R1] Full E2E fix completed:
  1. Updated cdc-cms-service: Added ShadowSchema, ShadowTable, SourceDatabase, SourceTable to ReconCheckCommand, ReconHealCommand, ExecuteHealCommand and normalized table prefix (scope.ShadowSchema.table) in TriggerCheck, TriggerHeal, TriggerExecuteHeal.
  2. Updated centralized-data-service: Normalized targetTable in CheckHealHandler, eliminated guessing fallbacks in resolveTargetTableConfig & helpers.go, switched Tier B lookups to GetByTargetTableAndSchema.
  3. Built and verified both cdc-cms-service (cmd/server) and centralized-data-service (cmd/worker) PASS (Exit code 0).

- [2026-08-25T15:40:00+07:00] [Agent:Claude] Rà soát toàn diện hoàn tất — phát hiện 10 vị trí trên 9 files cần fix.
- [2026-08-25T15:41:00+07:00] [Agent:Claude] Triển khai song song 3 executor subagents — hoàn thành toàn bộ 10 fix:
  - C1: recon_base_handler.go — Xóa 2 nhánh fallback pureTable
  - C2: helpers.go — return nil khi có schema mà GetByTargetTableAndSchema thất bại
  - H1: reconciliation_handler_commands.go — Forward schema fields cho wildcard "*"
  - H2: recon_sysops_handler.go HandleDebeziumSignal — Thêm ShadowSchema + normalize
  - H3: recon_sysops_handler.go HandleRetryFailed — Thêm ShadowSchema + normalize
  - H4: recon_execute_heal_handler.go — Thêm ShadowSchema/ShadowTable + normalize
  - H5: schedule_handler.go — Normalize targetTable khi có ShadowSchema
  - M1: transmuter.go — Log warning khi thiếu schema
  - M2: mapping_rule_v2_repo.go — Bỏ OR pureTable
  - M3: batch_transform_handler.go — Bỏ fallback sourceTable = pureTable
- [2026-08-25T15:41:00+07:00] [Agent:Claude] Build verification PASS: cdc-cms-service (Exit 0), centralized-data-service (Exit 0).

- [2026-08-25T15:51:00+07:00] [Agent:Gemini] Hoàn tất chuẩn hóa triệt để Tier 2 Master Schema (Shadow -> Master Isolation):
  1. Fix ResolveShadowTable (recon_stream_bucket_engine.go): Trả về qualified shadow_schema.shadow_table và parse schema cho fallback query.
  2. Fix HandleReconHeal (recon_check_heal_handler.go): Phân định độc lập normalization theo segment (SegmentShadowMaster dùng MasterSchema, SegmentSourceShadow dùng ShadowSchema).
  3. Fix resolveMasterFQN (recon_check_heal_handler.go) & resolveMasterBindingRef (recon_base_handler.go): Ưu tiên match chính xác FQN (MasterRel/ShadowRel) trước, ngăn chặn match nhầm binding đầu tiên khi trùng tên master_table.
  4. Fix CMS API Handlers (TriggerCheck, TriggerCheckAll, TriggerHeal, TriggerExecuteHeal): Hỗ trợ MasterSchema normalization khi ShadowSchema rỗng.
  5. Build Verification PASS: cdc-cms-service (Exit 0), centralized-data-service (Exit 0).

- [2026-08-25T15:58:00+07:00] [Agent:Gemini] Điều tra và khắc phục sự cố "Job Recon không hiển thị trong tab Progress":
  - Root Cause: Frontend query GET /api/reconciliation/jobs/active với target_table="hyperverge_audit_logs" (bảng trần). Nhưng DB cdc_system.recon_jobs lưu "shadow_gpay_ekyc.hyperverge_audit_logs" (qualified). Hàm GetActiveReconJobs ở backend dùng exact match WHERE target_table = ?, dẫn đến không tìm thấy job.
  - Fix:
    1. Update GetActiveReconJobs (recon_read_repo_gorm.go): Hỗ trợ match cả exact FQN lẫn fuzzy prefix (target_table = ? OR target_table LIKE '%.?').
    2. Update GetActiveJobs Handler (reconciliation_handler_reports.go): Nhận thêm query param shadow_schema và normalize trước khi query.
    3. Update Frontend (useReconStatus.ts & ReconPipelineGrid.tsx): Truyền historySchema vào hook useActiveReconJobs.
  - Build PASS: cdc-cms-service (Exit 0), cdc-cms-web (Exit 0).
