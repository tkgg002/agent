# Hồ sơ Giải pháp: Sửa lỗi hiển thị modal Chữa lành đối soát

Hồ sơ này mô tả chi tiết thay đổi mã nguồn cần thiết để sửa lỗi không hiển thị danh sách chữa lành do lệch FQN.

## Thay đổi trong `recon_read_repo_gorm.go`

Đường dẫn tệp: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`

### 1. Hàm `ListUnhealedReports` (Bắt đầu từ dòng 549)

#### Trước khi sửa:
```go
func (r *reconReadRepoGorm) ListUnhealedReports(ctx context.Context, table, shadowSchema string) ([]reconmodel.ReconciliationReport, error) {
	var reports []reconmodel.ReconciliationReport
	q := r.db.WithContext(ctx).
		Table("cdc_system.cdc_reconciliation_report").
		Where("(shadow_table = ? OR master_table = ?)", table, table).
		Where("healed_at IS NULL").
		Where("(missing_count > 0 OR stale_count > 0 OR orphan_count > 0)")
	if shadowSchema != "" {
		q = q.Where("shadow_schema = ?", shadowSchema)
	}
	err := q.Order("checked_at DESC").Find(&reports).Error
	return reports, err
}
```

#### Sau khi sửa:
```go
func (r *reconReadRepoGorm) ListUnhealedReports(ctx context.Context, table, shadowSchema string) ([]reconmodel.ReconciliationReport, error) {
	if strings.Contains(table, ".") {
		parts := strings.Split(table, ".")
		if len(parts) > 1 {
			if shadowSchema == "" {
				shadowSchema = parts[0]
			}
			table = parts[len(parts)-1]
		}
	}

	var reports []reconmodel.ReconciliationReport
	q := r.db.WithContext(ctx).
		Table("cdc_system.cdc_reconciliation_report").
		Where("(shadow_table = ? OR master_table = ?)", table, table).
		Where("healed_at IS NULL").
		Where("(missing_count > 0 OR stale_count > 0 OR orphan_count > 0)")
	if shadowSchema != "" {
		q = q.Where("shadow_schema = ?", shadowSchema)
	}
	err := q.Order("checked_at DESC").Find(&reports).Error
	return reports, err
}
```

---

### 2. Hàm `GetTableHistory` (Bắt đầu từ dòng 220)

#### Trước khi sửa:
```go
func (r *reconReadRepoGorm) GetTableHistory(ctx context.Context, table, shadowSchema, masterTable string, page, pageSize int) ([]reconmodel.ReconciliationReport, int64, error) {
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

#### Sau khi sửa:
```go
func (r *reconReadRepoGorm) GetTableHistory(ctx context.Context, table, shadowSchema, masterTable string, page, pageSize int) ([]reconmodel.ReconciliationReport, int64, error) {
	if strings.Contains(table, ".") {
		parts := strings.Split(table, ".")
		if len(parts) > 1 {
			if shadowSchema == "" {
				shadowSchema = parts[0]
			}
			table = parts[len(parts)-1]
		}
	}
	if strings.Contains(masterTable, ".") {
		parts := strings.Split(masterTable, ".")
		masterTable = parts[len(parts)-1]
	}

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
