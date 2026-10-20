# Giải Pháp Kỹ Thuật Duy Nhất Tối Ưu (Single Best Approach): Chuẩn Hóa Metrics Theo Ngày & Theo Trạm Trên `:8082/metrics`

- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **Ngày lập:** `2026-10-16`
- **Tier:** `T2`
- **Trạng thái:** **PROPOSE-ONLY** — Chờ User phê duyệt (`APPROVE`) trước khi thực thi (`GEMINI.md` Rule #1, Rule #13).

---

## 1. Root Cause (`VERIFIED` kèm `file:line`)

1. **Source chưa khai báo rõ cột ngày tạo bản ghi (`created_at_field`) trên `cdc_source_object_registry` (`VERIFIED`):**
   - Trong [`source_object_registry.go:25-39`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L25-L39), model `SourceObjectRegistry` chỉ khai báo `TimestampField` (`timestamp_field`, thường dùng cho incremental/watermark `updatedAt`), chưa có trường tường minh xác định **cột ngày tạo bản ghi gốc** (`created_at_field`) để đối soát số bản ghi tạo mới theo ngày (`report_date`).
   - Hệ quả: Trong [`ActivityManager.tsx:439-446`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L439-L446), khi cấu hình job `source_count`, UI đang dùng `regex` đoán tên cột hoặc tự động điền cứng `date_field = '_created_at'` cho cả bảng Master (gây lỗi `SQLSTATE 42703: column "_created_at" does not exist` trên `master_data.*` vì bảng Master không có cột `_created_at`).
2. **`countSQL` trên PostgreSQL chưa lọc bản ghi xóa mềm `_deleted = false` (`VERIFIED`):**
   - Trong [`source_count_runner.go:288-293`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L288-L293), câu lệnh `SELECT COUNT(*) FROM %s WHERE %s >= $1 AND %s < $2` đếm cả các dòng đã bị xóa mềm (`_deleted = true`) trên bảng Shadow/Master có soft-delete.
3. **Kết quả đếm theo ngày (`cdc_system.daily_source_counts`) chưa được đẩy ra `:8082/metrics` (`VERIFIED`):**
   - Trong [`source_count_runner.go:359-400`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L359-L400), `saveResult` chỉ `INSERT ... ON CONFLICT DO UPDATE` vào bảng `cdc_system.daily_source_counts` mà không phát Prometheus Gauge ra `:8082/metrics`.
4. **Tồn tại các metric đếm tổng toàn bảng vô nghĩa trên hệ thống lớn (`VERIFIED`):**
   - Các phép đếm tổng toàn bảng (`source_total`, `full_source_count`, `window_source_count`, `shadow_total`, `master_total`) quét toàn bảng dữ liệu lớn gây tốn tài nguyên và không phản ánh đúng câu hỏi vận hành theo ngày/theo luồng sự kiện.

---

## 2. Phương Án Kỹ Thuật Duy Nhất Tối Ưu (Single Best Technical Solution — Rule #9 & Rule #12)

Tuyệt đối **KHÔNG** làm các tác vụ đếm tổng toàn bảng và **KHÔNG** dùng regex/fallback đoán mò cột ngày. Thực hiện đồng bộ 3 hạng mục kiến trúc chuẩn:

1. **Hạng mục 1 — Khai báo tường minh Metadata cột ngày tạo (`created_at_field`) từ Source $\rightarrow$ Shadow $\rightarrow$ Master:**
   - Bổ sung trường `CreatedAtField *string` (`column:created_at_field`) vào [`source_object_registry.go:25-39`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L25-L39).
   - Khi đăng ký bảng Source, bắt buộc chỉ định rõ bảng có cột ngày tạo hay không (`created_at_field`, ví dụ `createdAt` hoặc `created_at`).
   - Ánh xạ (mapping) tường minh cột ngày tạo này từ **Source (`created_at_field`) $\rightarrow$ Shadow (`shadow_created_at_col` hoặc `_created_at`) $\rightarrow$ Master (`master_created_at_col` qua Transmute Rule `approved`)**.
   - Bảng nào **chưa khai báo `created_at_field`** thì chỉ giám sát bằng **Event Counters (`insert` / `update` / `delete`)**, tuyệt đối không chạy job đếm theo ngày tạo và không đoán mò cột ngày.
   - Trên [`ActivityManager.tsx:439-446`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L439-L446), lấy trực tiếp cột ngày từ cấu hình mapping tường minh của từng trạm; nếu chưa khai báo thì yêu cầu người dùng chọn rõ, bỏ hoàn toàn regex đoán tên cột và bỏ gán cứng `_created_at` cho bảng Master.

2. **Hạng mục 2 — Lọc `_deleted = false` cho bản ghi Active tạo mới trong ngày tại `countSQL`:**
   - Trong [`source_count_runner.go:288-297`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go#L288-L297), với các bảng Shadow (luôn có cột `_deleted`) hoặc bảng Master có cấu hình soft-delete, bổ sung điều kiện `AND "_deleted" = false` vào truy vấn đếm theo khoảng thời gian `[startOfDay, endOfDay)` để số đếm phản ánh chính xác số bản ghi Active tạo mới trong ngày `report_date`.

3. **Hạng mục 3 — Phát Prometheus Gauge `cdc_daily_table_count` theo `report_date` ra `:8082/metrics` và Hydrate khi khởi động:**
   - Khai báo `DailyTableCount` (`cdc_daily_table_count`) và `DailyTableCountUpdatedAt` (`cdc_daily_table_count_updated_timestamp_seconds`) trong [`prometheus.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go) với nhãn `{report_date, target_layer, source_name, target_table, clean_table, date_field}`.
   - Cập nhật metric ngay trong `saveResult` của [`source_count_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go) và gọi `SyncRecentToMetrics(ctx, 7)` khi khởi động `cdc-worker` để nạp lại số đếm của 7 ngày gần nhất từ `cdc_system.daily_source_counts` lên RAM `:8082/metrics`.

---

## 3. Blast Radius & Rollback Plan

### 3.1. Blast Radius (Phạm vi ảnh hưởng)
| Component / File | Thay đổi | Mức độ ảnh hưởng |
| :--- | :--- | :--- |
| [`centralized-data-service/internal/model/source/source_object_registry.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go) | Thêm cột nullable `CreatedAtField *string` (`created_at_field`) | Tương thích ngược 100% với các bản ghi hiện có (`NULL` = chưa cấu hình ngày tạo). |
| [`centralized-data-service/pkgs/metrics/prometheus.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go) | Thêm 2 GaugeVec mới (`cdc_daily_table_count`, `cdc_daily_table_count_updated_timestamp_seconds`) | Chỉ thêm metric mới trên `:8082/metrics`, không phá vỡ scrape hay alert hiện hành. |
| [`centralized-data-service/internal/service/sourcecount/source_count_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go) | Emit metric sau `saveResult`, thêm `SyncRecentToMetrics`, lọc `_deleted = false` cho bảng Shadow | Chỉ tác động luồng `source_count` job và khởi động worker `:8082`. |
| [`cdc-cms-web/src/pages/ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx) | Bỏ gán cứng `_created_at` trên tầng Master và bỏ regex đoán cột | Chỉ ảnh hưởng form cấu hình `source_count` trên CMS UI. |

### 3.2. Rollback Plan (Kế hoạch hoàn tác)
- Trước khi thực thi ở bước S4, tạo local git restore-point (`git commit` local trên nhánh làm việc).
- Nếu phát sinh lỗi runtime sau khi sửa:
  1. Revert commit thay đổi trên `centralized-data-service` và `cdc-cms-web`, khởi động lại `cdc-worker` (`:8082`).
  2. Cột `created_at_field` là nullable nên không gây ảnh hưởng đến các truy vấn cũ của `cdc_source_object_registry`.

---

## 4. Code Demo Chi Tiết Từng File

### 4.1. File 1: [`centralized-data-service/internal/model/source/source_object_registry.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L25-L39)

Bổ sung trường `CreatedAtField` tường minh:

```go
	ReplicationMode   string     `gorm:"column:replication_mode;type:varchar(32);not null;default:CDC" json:"replication_mode"`
	TimestampField    *string    `gorm:"column:timestamp_field;type:varchar(128)" json:"timestamp_field,omitempty"`
	CreatedAtField    *string    `gorm:"column:created_at_field;type:varchar(128)" json:"created_at_field,omitempty"`
	BatchScheduleCron *string    `gorm:"column:batch_schedule_cron;type:varchar(64)" json:"batch_schedule_cron,omitempty"`
```

### 4.2. File 2: [`centralized-data-service/pkgs/metrics/prometheus.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go)

Bổ sung 2 Gauge metrics cho Daily Table Count theo ngày:

```go
	// DailyTableCount exposes the daily record count persisted in cdc_system.daily_source_counts.
	// Hydrated from DB on startup and updated immediately after each source_count job run.
	DailyTableCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cdc_daily_table_count",
		Help: "Daily record creation count per table and layer (source/shadow/master) by report_date",
	}, []string{"report_date", "target_layer", "source_name", "target_table", "clean_table", "date_field"})

	// DailyTableCountUpdatedAt exposes the Unix timestamp of the last successful daily count update.
	DailyTableCountUpdatedAt = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cdc_daily_table_count_updated_timestamp_seconds",
		Help: "Unix timestamp when cdc_daily_table_count was last updated for a table and layer",
	}, []string{"report_date", "target_layer", "source_name", "target_table", "clean_table"})
```

### 4.3. File 3: [`centralized-data-service/internal/service/sourcecount/source_count_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/sourcecount/source_count_runner.go)

1. **Lọc `_deleted = false` trên tầng Shadow trong `countSQL` (`source_count_runner.go:288-297`) và không đoán mò cột ngày:**

```go
	tableExpr := fmt.Sprintf(`"%s"."%s"`, target.ContainerNamespace, target.ObjectName)
	dateExpr := fmt.Sprintf(`"%s"`, target.DateField)
	whereClause := fmt.Sprintf(`%s >= $1 AND %s < $2`, dateExpr, dateExpr)
	if normalizeTargetLayer(target.TargetLayer) == TargetLayerShadow {
		whereClause += ` AND "_deleted" = false`
	}
	query := fmt.Sprintf(
		`SELECT COUNT(*) FROM %s WHERE %s`,
		tableExpr,
		whereClause,
	)
```

2. **Emit metric ngay sau khi `saveResult` lưu `daily_source_counts` và nạp lại 7 ngày gần nhất khi khởi động (`SyncRecentToMetrics`):**

```go
func emitDailyCountMetric(reportDateStr, targetLayer, sourceName, targetTable, dateField string, count int64, updatedAt time.Time) {
	layer := normalizeTargetLayer(targetLayer)
	cleanTable := extractCleanTableName(targetTable)
	metrics.DailyTableCount.WithLabelValues(
		reportDateStr,
		layer,
		sourceName,
		targetTable,
		cleanTable,
		dateField,
	).Set(float64(count))
	metrics.DailyTableCountUpdatedAt.WithLabelValues(
		reportDateStr,
		layer,
		sourceName,
		targetTable,
		cleanTable,
	).Set(float64(updatedAt.Unix()))
}

// SyncRecentToMetrics hydrates Prometheus gauges from cdc_system.daily_source_counts on worker startup.
func (r *SourceCountRunner) SyncRecentToMetrics(ctx context.Context, lookbackDays int) error {
	if r == nil || r.sysDB == nil {
		return nil
	}
	if lookbackDays <= 0 {
		lookbackDays = 7
	}
	var rows []DailySourceCountRow
	err := r.sysDB.WithContext(ctx).Raw(`
		SELECT id, report_date, source_name, target_layer, target_table, date_field, creation_count, updated_at
		FROM cdc_system.daily_source_counts
		WHERE report_date >= CURRENT_DATE - ($1 * INTERVAL '1 day')
		ORDER BY report_date ASC, id ASC
	`, lookbackDays).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("sync recent daily_source_counts to metrics: %w", err)
	}
	for _, row := range rows {
		reportDateStr := row.ReportDate.In(vnLoc).Format("2006-01-02")
		emitDailyCountMetric(reportDateStr, row.TargetLayer, row.SourceName, row.TargetTable, row.DateField, row.CreationCount, row.UpdatedAt)
	}
	return nil
}
```

### 4.4. File 4: [`cdc-cms-web/src/pages/ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L439-L446)

Bỏ hoàn toàn regex đoán cột và bỏ gán cứng `_created_at` trên tầng Master:

```tsx
      if (selectedLayer === 'source') {
        const configuredCol = selectedSourceObj?.created_at_field || '';
        form.setFieldsValue({ date_field: configuredCol });
      } else if (selectedLayer === 'shadow') {
        form.setFieldsValue({ date_field: '_created_at' });
      } else {
        const approvedCreatedAtRule = transmuteRules.find(
          r => r.approval_status === 'approved' && r.target_column === 'created_at'
        );
        form.setFieldsValue({ date_field: approvedCreatedAtRule ? 'created_at' : '' });
      }
```

---

## 5. Bảng Kết Quả Decision Gate `DG1` & `DG2` (`/laya_router`)

### 5.1. Cổng `DG1` (Solution Claims — `06_laya_summary_dg1_20261016T161657.json`)
| Claim ID | Nội dung Claim | Laya Decision | Resolution (`REF:`) | Gate Status |
| :--- | :--- | :--- | :--- | :--- |
| `S-RC` | `SourceObjectRegistry` tại `source_object_registry.go:25-39` chỉ có `TimestampField`, chưa có trường riêng cho cột ngày tạo (`created_at_field`). | `FLAG` -> `RESOLVED` | `REF: .../source_object_registry.go:25-39` xác nhận model chỉ có `TimestampField`. | **PASS** |
| `S-FIX` | `countSQL` tại `source_count_runner.go:288-293` đếm `SELECT COUNT(*) FROM %s WHERE %s >= $1 AND %s < $2` mà chưa có điều kiện `_deleted`. | `ACCEPT` | Ground truth được chấp nhận trực tiếp (`r1=0.696, r2=0.669, r3=0.903`). | **PASS** |
| `S-IMPACT` | `saveResult` tại `source_count_runner.go:359-400` chỉ ghi vào `cdc_system.daily_source_counts` mà không gọi `metrics` package. | `ACCEPT` | Ground truth được chấp nhận trực tiếp (`r1=0.531, r2=0.587, r3=0.818`). | **PASS** |
| `S-PATTERN` | `ActivityManager.tsx:444-446` gán `form.setFieldsValue({ date_field: '_created_at' })` ở nhánh `else` của `selectedLayer`. | `FLAG` -> `RESOLVED` | `REF: .../ActivityManager.tsx:444-446` xác nhận nhánh `else` gán cứng `_created_at`. | **PASS** |

### 5.2. Cổng `DG2` (Code Demo Claims — `06_laya_summary_dg2_20261016T163541.json`)
| Claim ID | Nội dung Claim | Laya Decision | Resolution (`REF:`) | Gate Status |
| :--- | :--- | :--- | :--- | :--- |
| `D-BEHAV` | Trong `countSQL`, hàm kiểm tra `validIdent.MatchString` trên `ContainerNamespace`, `ObjectName`, và `DateField` trước khi tạo câu SQL. | `FLAG` -> `RESOLVED` | `REF: .../source_count_runner.go:278-286` xác nhận kiểm tra `validIdent.MatchString` trên cả 3 trường. | **PASS** |
| `D-EDGE` | Trong `countSQL`, khi `db == nil` tại dòng `274`, hàm trả về ngay `0, nil` kèm lỗi `target SQL database connection is not available`. | `ACCEPT` | Ground truth được chấp nhận trực tiếp (`r1=0.649, r2=0.581, r3=0.887`). | **PASS** |
| `D-API` | Trong `countSQL`, câu truy vấn được thực thi qua `db.WithContext(ctx).Raw(query, start, end).Scan(&count).Error` tại dòng `297`. | `FLAG` -> `RESOLVED` | `REF: .../source_count_runner.go:297` xác nhận lệnh gọi GORM `.Raw(query, start, end).Scan(&count).Error`. | **PASS** |

