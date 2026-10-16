# 12 Implementation Plan: Tối ưu SLOW SQL GetTableHistory (4639ms)

## 1. Kế hoạch chi tiết

### Task 1 & 2 & 3: Sửa `recon_read_repo_gorm.go`
- **File**: `internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Hàm**: `GetTableHistory`
- **Thay đổi WHERE clause**:
  ```go
  where := "shadow_table = ? OR master_table = ?"
  args := []interface{}{table, table}
  if shadowSchema != "" {
      where = "shadow_schema = ? AND shadow_table = ?"
      args = []interface{}{shadowSchema, table}
      if masterTable != "" {
          where += " AND (segment <> 'shadow_master' OR master_table = ?)"
          args = append(args, masterTable)
      }
  }
  ```
- **Thay đổi countQuery khi `excludeSmoke = true`**:
  ```go
  countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cdc_system.cdc_reconciliation_report WHERE (%s) AND check_type NOT IN ('smoke', 'segment_b_smoke')", where)
  ```
- **Thay đổi countQuery khi `excludeSmoke = false`**:
  ```go
  countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (
      (SELECT 1 FROM cdc_system.cdc_reconciliation_report WHERE (%s) AND check_type IN ('smoke', 'segment_b_smoke'))
      UNION ALL
      (SELECT 1 FROM cdc_system.cdc_recon_smoke_result WHERE (%s))
  ) AS unioned`, where, where)
  ```
  `Scan(&total)` dùng `unionArgs`.

### Task 4 & 5: Áp dụng Migration SQL 102
- Kiểm tra file `migrations/schema/recon_dlq/102_fix_slow_table_history.sql`:
  - `idx_smoke_schema_table_checked ON cdc_system.cdc_recon_smoke_result (shadow_schema, shadow_table, checked_at DESC)`
  - `idx_recon_master_table ON cdc_system.cdc_reconciliation_report (master_table, checked_at DESC) WHERE master_table IS NOT NULL`
- Áp dụng vào PostgreSQL container `gpay-postgres-cdc` database `cdc_dw`.

### Task 6: Verification
- `git diff`
- `go build`
