# Yêu cầu — Migration + Frontend cho Luồng Chữa Lành Tương Tác

## Bối cảnh
Backend (Gateway + Worker) đã implement xong (13 files, BUILD PASS). Cần bổ sung:
1. DB Migration: 6 cột thống kê mới vào `cdc_reconciliation_report`
2. Frontend: hooks + modal 3 checkboxes + danh sách report chưa heal

## Yêu cầu chi tiết

### RF-1: Migration SQL
- Thêm 6 cột: `healed_mismatched_count`, `healed_mismatched_duration_ms`, `healed_missing_dest_count`, `healed_missing_dest_duration_ms`, `pruned_missing_src_count`, `pruned_missing_src_duration_ms`
- Tất cả `INT DEFAULT 0`, dùng `IF NOT EXISTS`
- File: `migrations/schema/recon_dlq/088_recon_interactive_heal_stats.sql` (tiếp theo 087)

### RF-2: Frontend hooks
- `useUnhealedReports(table, shadowSchema?)`: GET `/api/reconciliation/report/:table/unhealed`
- `useExecuteHealMutation()`: POST `/api/reconciliation/execute-heal`

### RF-3: Frontend UI
- Hiển thị danh sách report chưa heal (gom theo segment A/B)
- 3 checkboxes: Mismatched / Missing Dest / Prune Src
- Nút "Thực hiện" dispatch execute-heal
