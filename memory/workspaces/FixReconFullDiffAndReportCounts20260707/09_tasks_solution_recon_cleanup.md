# Technical Solution: Clean Up Redundant Columns and Standardize Source Metadata

Chi tiết mã nguồn cần chỉnh sửa để dọn dẹp các trường `tier`/`target_table` và bổ sung `source_type`/`source_host`/`source_table` cho `cdc_reconciliation_report`.

---

## 1. Component: cdc-cms-service

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go`
- Loại bỏ trường `Tier`.
- Đổi tag GORM cho `TargetTable` thành `gorm:"-"` (hoặc `gorm:"column:target_table"` để tương thích ngược khi scan).
- Thêm 3 trường `SourceType`, `SourceHost`, `SourceTable`.

```go
type ReconciliationReport struct {
	ID                  uint64          `gorm:"primaryKey" json:"id"`
	TargetTable         string          `gorm:"column:target_table" json:"target_table"` // Keep for dynamic scan alias
	SourceDB            string          `gorm:"column:source_db" json:"source_db"`
	SourceType          string          `gorm:"column:source_type" json:"source_type,omitempty"`
	SourceHost          string          `gorm:"column:source_host" json:"source_host,omitempty"`
	SourceTable         string          `gorm:"column:source_table" json:"source_table,omitempty"`
	SourceCount         int             `gorm:"column:source_count" json:"source_count"`
...
```

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go`
- Cập nhật các câu JOIN đến `cdc_reconciliation_report rr`:
  - Dòng 84:
    `WHERE rr.shadow_table = COALESCE(sb.shadow_table, tr.target_table)`
  - Dòng 268:
    `WHERE rr.shadow_table = tr.target_table`
  - Dòng 404:
    `WHERE rr.shadow_table = sb.shadow_table`

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- Trong `GetTableHistory` (dòng 246):
  - Xóa `tier` khỏi select lists.
  - Chọn trực tiếp các cột `source_type`, `source_host`, `source_table`, `recon_start_time`, `recon_end_time` từ bảng `cdc_reconciliation_report` và `cdc_recon_smoke_result`.
- Trong `listLatestPrimary`:
  - Chọn trực tiếp `r.source_type`, `r.source_host`, `r.source_table` từ bảng thay vì các giá trị `NULL::text`.

---

## 2. Component: centralized-data-service

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go`
- Loại bỏ `Tier`.
- Thiết lập `TargetTable` tag GORM thành `gorm:"-"`.
- Thêm 3 trường `SourceType`, `SourceHost`, `SourceTable`.

```go
type ReconciliationReport struct {
	ID          uint64 `gorm:"primaryKey" json:"id"`
	TargetTable string `gorm:"-" json:"target_table"`
	SourceDB    string `gorm:"column:source_db" json:"source_db"`
	SourceType  string `gorm:"column:source_type" json:"source_type"`
	SourceHost  string `gorm:"column:source_host" json:"source_host"`
	SourceTable string `gorm:"column:source_table" json:"source_table"`
...
```

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go`
- Cập nhật `stampA` để gán thông tin nguồn chi tiết từ `entry`:
  ```go
  report.SourceType = entry.SourceType
  report.SourceHost = extractHost(entry.SourceURL)
  report.SourceTable = entry.SourceTable
  ```
- Cập nhật `stampB` để gán thông tin shadow plane nguồn:
  ```go
  report.SourceType = "postgresql"
  report.SourceHost = "shadow_plane"
  report.SourceTable = ref.ShadowTable
  ```

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go`
- Xóa hàm `RunSmokeCheck`.

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go`
- Xóa hàm `RunSmokeCheckB`.

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go`
- Trong `validateAndEnrichContext`, loại bỏ trường hợp `TypeReconSmoke`.
- Trong `executeGenericCheck`, loại bỏ logic gọi `RunSmokeCheck`/`RunSmokeCheckB`.

---

## 3. Component: cdc-cms-web

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts`
- Xóa `tier?: number;` khỏi interface `ReconReport`.
- Thêm `source_host?: string | null;` và `source_table?: string | null;` vào `ReconReport`.

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx`
- Cập nhật `levelLabel` để map theo `check_type` thay vì `tier`.
- Cập nhật `resolvePipelineNames` để định dạng tên nguồn thâu tóm đầy đủ thông tin:
  ```typescript
  const getSourceDisplayName = (r: { source_type?: string | null; source_host?: string | null; source_db?: string | null; source_table?: string | null; target_table?: string | null }) => {
    if (!r) return '—';
    const typeStr = r.source_type ? `[${r.source_type}] ` : '';
    const hostStr = r.source_host ? `${r.source_host} / ` : '';
    const dbStr = r.source_db ?? '';
    const tableStr = r.source_table || r.target_table || '';
    if (!hostStr && !dbStr && !tableStr) return '—';
    return `${typeStr}${hostStr}${dbStr} . ${tableStr}`;
  };
  ```
  Sử dụng helper này để gán cho `sourceName`.

---

## 4. Component: cdc-cms-service (System Health & Legacy Query Fix)

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go`
- Cập nhật câu lệnh Raw SQL tại dòng 34 thay thế `target_table` vật lý bằng biểu thức động `CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END`:
```go
	err := db.Raw(
		`SELECT DISTINCT ON (CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END)
			id,
			run_id,
			segment,
			shadow_schema,
			shadow_table,
			CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END AS target_table,
			source_db,
			source_count,
			dest_count,
			diff,
			status,
			error_message,
			duration_ms,
			checked_at,
			total_source_count,
			total_dest_count,
			check_type
		FROM cdc_reconciliation_report
		ORDER BY CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END, checked_at DESC`,
	).Scan(&reports).Error
```

### File: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- Trong hằng số `listLatestLegacy` (dòng 186), cập nhật SQL tương tự để loại bỏ truy vấn trực tiếp cột `target_table`:
```go
		SELECT DISTINCT ON (CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END)
			id,
			run_id,
			segment,
			shadow_schema,
			shadow_table,
			CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END AS target_table,
			source_db,
			source_count,
			dest_count,
			diff,
			status,
			error_message,
			duration_ms,
			checked_at,
			total_source_count,
			total_dest_count,
			check_type
		FROM cdc_reconciliation_report
		ORDER BY CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END, checked_at DESC
```

---

## 5. Component: centralized-data-service (Reconciliation Report Repository Fix)

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go`
- Cập nhật các câu truy vấn GORM dùng `target_table = ?` để chuyển đổi sang logic check `shadow_table` hoặc `master_table` tương thích với cơ sở dữ liệu mới.
- Ánh xạ tham số `tier` thành tập hợp `check_type` (vì cột `tier` đã bị drop khỏi DB).
- Cụ thể:
  - `GetLatestByTable`:
    ```go
    q := r.db.WithContext(ctx)
    if segment == "shadow_master" {
    	q = q.Where("master_table = ?", targetTable)
    } else if segment == "source_shadow" {
    	q = q.Where("shadow_table = ?", targetTable)
    } else {
    	q = q.Where("shadow_table = ? OR master_table = ?", targetTable, targetTable)
    }
    if segment != "" {
    	q = q.Where("segment = ?", segment)
    }
    ```
  - `GetLatestMissingReport` & `GetLatestMissingReportWithSegment`:
    ```go
    checkTypes := getCheckTypesForTier(tier)
    // Query shadow_table OR master_table thay vì target_table
    // Query check_type IN checkTypes thay vì tier = ?
    ```
  - `GetUnhealedReports`:
    ```go
    Where("(shadow_table = ? OR master_table = ?) AND healed_at IS NULL AND (missing_count > 0 OR stale_count > 0 OR orphan_count > 0)", targetTable, targetTable)
    ```
  - Định nghĩa helper `getCheckTypesForTier(tier int) []string` ở cuối file:
    ```go
    func getCheckTypesForTier(tier int) []string {
    	switch tier {
    	case 1:
    		return []string{"smoke", "count_windowed"}
    	case 2:
    		return []string{"hash_window"}
    	case 3:
    		return []string{"bucket_hash", "deep_check"}
    	default:
    		return []string{}
    	}
    }
    ```


