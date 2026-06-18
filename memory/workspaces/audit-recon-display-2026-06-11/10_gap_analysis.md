# Audit: Recon display sai — "Nhật ký đối soát (30 phiên)" + "Biến động số lượng theo phiên"
Ngày 2026-06-11 | Agent: claude-opus-4-8 (Muscle) | Trigger: user báo recon chạy 30p/lần nhưng UI hiển thị sai, có pipeline "ko hiện gì".

## Bằng chứng (DB cdc_dw:5433 cdc_system + code cdc-cms-service/centralized-data-service)
1. Mọi master_binding active+approved ĐỀU có recon record (NOT EXISTS query = rỗng) → "ko hiện gì" KHÔNG do thiếu data.
2. shadow_table TRÙNG TÊN: `export_jobs` ở 2 binding active (shadow_aaaa_centrallized_export_service, shadow_dev000).
3. Read: ResolveTargetTableByScope (recon_read_repo_gorm.go:205-255) trả `sb.shadow_table` + LIMIT 2 → tên trùng → ErrAmbiguousScope → view rỗng.
4. GetTableHistory (recon_read_repo_gorm.go:137-155) lọc `WHERE target_table = <bare>` → gộp nhiều pipeline cùng tên. Recon `export_jobs`: source_db "centralized-export-service"(3) + "centrallized-export-service"(46) trộn chung → biến động sai.
5. Segment A (source_shadow) ghi target_table = tên SHADOW; segment B (shadow_master) ghi tên MASTER (export_jobs vs export_jobs_mt) → 1 pipeline tách 2 tên; resolver → shadow_table → chỉ thấy segment A, segment B vô hình.
6. source_db không nhất quán: A=tên source ("wallet-service"); B="schema.table" shadow ("shadow_goopay_lc_ws_wallet_service.wallet_capsets").
7. cdc_reconciliation_report KHÔNG có cột shadow_schema/binding_id/segment-key (chỉ source_db mơ hồ) — model reconciliation_report.go.
8. Pipeline source unreachable (export_jobs_2/4/test, events) → segment A toàn SRC_TIMEOUT/UNKNOWN, count=0 → biến động phẳng-0 (dev không tới prod Mongo 10.200.18x.x).
9. checked_at bị stagger (Task 9 spread 5') + không có run_id/session_id → gom "phiên" cross-table không chuẩn.

## Root cause
CHÍNH: recon report key bằng `target_table` (bare name) → (a) trùng tên across-schema → resolve ambiguous → rỗng; (b) trộn nhiều pipeline 1 tên → biến động sai; (c) segment A/B khác tên → mất nửa lịch sử.
PHỤ: không session_id + checked_at stagger → "phiên" sai; pipeline lỗi count=0 → biểu đồ phẳng.
Bản chất: data-model có sẵn (write bare target_table thiếu khóa định danh), LỘ RA sau khi Task 9 làm recon chạy lại. KHÔNG phải data sai.

## Fix direction (multi-tầng — chờ user chốt scope/thứ tự)
1. Migration: ALTER cdc_reconciliation_report ADD shadow_schema + shadow_binding_id (+ master_binding_id), + run_id.
2. Worker recon_core.go: set shadow_schema+binding_id+run_id khi ghi report cả segment A/B.
3. Read recon_read_repo + ResolveTargetTableByScope: query theo (shadow_schema,target_table) hoặc binding_id → hết ambiguous + gộp đủ A/B; gom phiên theo run_id.
4. FE: truyền khóa unambiguous.

## Global Pattern
[Report/log key bằng tên-nghiệp-vụ bare (table name)] mà tên đó KHÔNG unique (trùng across-schema, hoặc đổi nghĩa theo segment/stage) → read-side ambiguous/trộn/mất dữ liệu. Đúng: key bằng ID ổn định (binding_id) + run_id cho mỗi batch; tên bare chỉ để hiển thị. Áp dụng mọi audit/recon/log đa-stage.
