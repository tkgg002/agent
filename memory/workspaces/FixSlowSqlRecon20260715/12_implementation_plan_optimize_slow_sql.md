# Kế hoạch triển khai chi tiết - Tối ưu hóa SQL Đối soát

## 1. Phân tích Nguyên nhân & Đề xuất Giải pháp

### 1.1. Tối ưu hóa `listLatestPrimary`
- **Nguyên nhân gốc rễ:**
  - Truy vấn sử dụng `cdc_reconciliation_report r` làm gốc và thực hiện `LEFT JOIN LATERAL` với `cdc_recon_smoke_result s` để lấy các chỉ số smoke check.
  - Phép `LEFT JOIN LATERAL` này được thực hiện cho **tất cả các dòng** của `cdc_reconciliation_report` trong subquery trước khi `DISTINCT ON` loại bỏ các bản ghi trùng lặp.
  - Vì bảng `cdc_reconciliation_report` lưu lịch sử chạy đối soát, số lượng bản ghi có thể lên đến hàng trăm nghìn dòng. Việc chạy lateral subquery hàng trăm nghìn lần khiến thời gian chạy tăng vọt (> 1.2s).
- **Giải pháp:**
  - Áp dụng phương pháp **Distinct-before-Join**: lọc ra danh sách các dòng mới nhất của mỗi bảng/segment từ cả hai nguồn (`cdc_reconciliation_report` và `cdc_recon_smoke_result`) trước.
  - Sau đó, thực hiện `UNION ALL` hai tập dữ liệu đã distinct (tối đa chỉ khoảng 200 dòng).
  - Tiếp tục `DISTINCT ON` lần cuối trên tập 200 dòng này để tìm ra dòng mới nhất thực sự.
  - Cuối cùng, thực hiện phép `LEFT JOIN LATERAL` với `cdc_recon_smoke_result` (chỉ khi dòng chiến thắng là từ `cdc_reconciliation_report`) và các bảng registry/lag khác trên tập kết quả cuối cùng (tối đa ~100 dòng).
  - Số lượng lateral subquery giảm từ **~100,000 lần** xuống còn **tối đa ~100 lần**.

### 1.2. Tối ưu hóa `ListFailedLogs` Count Query
- **Nguyên nhân gốc rễ:**
  - Hàm `ListFailedLogs` thực hiện đếm số dòng qua `SELECT COUNT(*) FROM (` + query + `) AS failed_logs`.
  - Biến `query` được định nghĩa là `failedLogsBase`, chứa 2 phép `LEFT JOIN LATERAL` đắt đỏ:
    - Quét `shadow_binding` tìm `source_object_id`, `shadow_schema`, `shadow_table` sắp xếp theo `updated_at DESC, id DESC LIMIT 1`.
    - Quét `shadow_binding` chạy `COUNT(*)` để xác định `scope_ambiguous`.
  - Các phép JOIN này hoàn toàn vô dụng đối với truy vấn `COUNT(*)` vì chúng không lọc bỏ hay nhân bản số lượng dòng của `failed_sync_logs` (vì đều là `LEFT JOIN LATERAL ... ON TRUE`).
- **Giải pháp:**
  - Tách biệt câu truy vấn đếm dòng. Thay vì dùng `failedLogsBase` làm gốc cho `COUNT(*)`, ta xây dựng một câu SQL `COUNT` tối giản trực tiếp trên bảng `cdc_system.failed_sync_logs`:
    ```sql
    SELECT COUNT(*) FROM cdc_system.failed_sync_logs f
    WHERE 1=1
    -- áp dụng các điều kiện lọc (f.target_table, f.status, f.error_type)
    ```
  - Loại bỏ hoàn toàn các JOIN, giảm độ phức tạp truy vấn từ $O(N \log M)$ xuống $O(N)$ (hoặc $O(\log N)$ nếu dùng index).

---

## 2. Chi tiết các file thay đổi

### 2.1. [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)

#### Thay đổi 1: Cập nhật `listLatestPrimary`
```go
const listLatestPrimary = `
	SELECT r.id,
	       r.run_id,
	       r.cycle_id,
	       r.segment,
	       r.source_type,
	       r.source_host,
	       r.source_db,
	       COALESCE(r.source_total, s.source_total) AS source_total,
	       COALESCE(r.source_active, s.source_active) AS source_active,
	       COALESCE(r.shadow_total, s.shadow_total) AS shadow_total,
	       COALESCE(r.shadow_active, s.shadow_active) AS shadow_active,
	       r.master_schema,
	       r.master_table,
	       COALESCE(r.master_total, s.master_total) AS master_total,
	       COALESCE(r.master_active, s.master_active) AS master_active,
	       r.diff,
	       r.status,
	       r.error_message,
	       r.duration_ms,
	       r.checked_at,
	       -- Tương thích ngược với ReconciliationReport
	       CASE WHEN r.segment = 'shadow_master' THEN r.master_table ELSE r.shadow_table END AS target_table,
	       r.source_count,
	       r.dest_count,
	       r.source_count AS nullable_source_count,
	       NULL AS error_code,
	       -- Enrichment metadata
	       reg.sync_engine, reg.timestamp_field,
	       reg.timestamp_field_source, reg.timestamp_field_confidence,
	       reg.full_source_count, reg.full_dest_count, reg.full_count_at,
	       lag.ingest_lag_ms, lag.transmute_lag_ms, lag.worker_backlog,
	       sb.source_object_id,
	       COALESCE(r.source_table, so.source_object_name) AS source_table,
	       COALESCE(r.shadow_schema, sb.shadow_schema) AS shadow_schema,
	       COALESCE(r.shadow_table, sb.shadow_table) AS shadow_table,
	       cr.connection_code AS source_connection_code,
	       r.master_schema,
	       COALESCE(scope_counts.binding_count, 0) > 1 AS scope_ambiguous
	  FROM (
		SELECT DISTINCT ON (shadow_schema, shadow_table, master_schema, master_table, segment) *
		  FROM (
			-- 1. Lọc Distinct trên cdc_reconciliation_report trước
			SELECT DISTINCT ON (COALESCE(r.shadow_schema, sb_norm.shadow_schema), r.shadow_table, r.master_schema, r.master_table, r.segment)
				r.id,
				r.run_id,
				NULL::bigint AS cycle_id,
				r.segment,
				r.source_type,
				r.source_host,
				r.source_db,
				NULL::bigint AS source_total,
				NULL::bigint AS source_active,
				NULL::bigint AS shadow_total,
				NULL::bigint AS shadow_active,
				r.master_schema,
				r.master_table,
				NULL::bigint AS master_total,
				NULL::bigint AS master_active,
				r.diff,
				r.status,
				r.error_message,
				r.duration_ms,
				r.checked_at,
				COALESCE(r.shadow_schema, sb_norm.shadow_schema) AS shadow_schema,
				r.shadow_table,
				r.source_table,
				r.source_count,
				r.dest_count
			FROM cdc_system.cdc_reconciliation_report r
			-- Chỉ normalize shadow_schema khi giá trị bị NULL
			LEFT JOIN LATERAL (
				SELECT shadow_schema
				FROM cdc_system.shadow_binding
				WHERE shadow_table = r.shadow_table
				  AND is_active = TRUE
				ORDER BY updated_at DESC, id DESC
				LIMIT 1
			) sb_norm ON r.shadow_schema IS NULL
			ORDER BY COALESCE(r.shadow_schema, sb_norm.shadow_schema), r.shadow_table, r.master_schema, r.master_table, r.segment, r.checked_at DESC

			UNION ALL

			-- 2. Lọc Distinct trên cdc_recon_smoke_result trước
			SELECT DISTINCT ON (shadow_schema, shadow_table, master_schema, master_table, segment)
				id,
				run_id,
				cycle_id,
				segment,
				source_type,
				source_host,
				source_db,
				source_total,
				source_active,
				shadow_total,
				shadow_active,
				master_schema,
				master_table,
				master_total,
				master_active,
				diff,
				status,
				error_message,
				duration_ms,
				checked_at,
				shadow_schema,
				shadow_table,
				source_table,
				CASE WHEN segment = 'shadow_master' THEN COALESCE(shadow_active, 0) ELSE COALESCE(source_active, 0) END AS source_count,
				CASE WHEN segment = 'shadow_master' THEN COALESCE(master_active, 0) ELSE COALESCE(shadow_active, 0) END AS dest_count
			FROM cdc_system.cdc_recon_smoke_result
			ORDER BY shadow_schema, shadow_table, master_schema, master_table, segment, checked_at DESC
		  ) unioned
		 ORDER BY shadow_schema, shadow_table, master_schema, master_table, segment, (source_active IS NOT NULL) DESC, checked_at DESC
	  ) r
	  -- 3. Chỉ thực hiện enrichment lateral join trên các bản ghi final kết quả (ở đây s chỉ join nếu dòng thắng là từ report)
	  LEFT JOIN LATERAL (
		SELECT source_total, source_active, shadow_total, shadow_active, master_total, master_active
		FROM cdc_system.cdc_recon_smoke_result
		WHERE shadow_schema = r.shadow_schema
		  AND shadow_table = r.shadow_table
		  AND (master_schema IS NOT DISTINCT FROM r.master_schema)
		  AND (master_table IS NOT DISTINCT FROM r.master_table)
		  AND segment = r.segment
		ORDER BY checked_at DESC
		LIMIT 1
	  ) s ON r.source_active IS NULL
	  LEFT JOIN cdc_system.recon_lag lag ON lag.table_name = CASE WHEN r.segment = 'shadow_master' THEN r.master_table ELSE r.shadow_table END
	  INNER JOIN cdc_table_registry reg ON reg.target_table = r.shadow_table AND reg.is_active = TRUE
	  LEFT JOIN LATERAL (
		SELECT source_object_id, shadow_schema, shadow_table
		FROM cdc_system.shadow_binding
		WHERE shadow_table = r.shadow_table
		  AND is_active = TRUE
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	  ) sb ON TRUE
	  LEFT JOIN LATERAL (
		SELECT COUNT(*)::int AS binding_count
		FROM cdc_system.shadow_binding
		WHERE shadow_table = r.shadow_table
		  AND is_active = TRUE
	  ) scope_counts ON TRUE
	  LEFT JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id
	  LEFT JOIN cdc_system.connection_registry cr ON cr.id = so.source_connection_id
	 ORDER BY r.shadow_table
`
```

#### Thay đổi 2: Tách biệt query Count trong `ListFailedLogs`
```diff
 func (r *reconReadRepoGorm) ListFailedLogs(ctx context.Context, f recon.FailedLogFilter, page, pageSize int) ([]recon.FailedLogRow, int64, error) {
-	query := failedLogsBase
-	args := make([]interface{}, 0, 4)
-	if f.TargetTable != "" {
-		query += ` AND f.target_table = ?`
-		args = append(args, f.TargetTable)
-	}
-	if f.Status != "" {
-		query += ` AND f.status = ?`
-		args = append(args, f.Status)
-	}
-	if f.ErrorType != "" {
-		query += ` AND f.error_type = ?`
-		args = append(args, f.ErrorType)
-	}
-
-	var total int64
-	countQuery := `SELECT COUNT(*) FROM (` + query + `) AS failed_logs`
-	if err := r.db.WithContext(ctx).Raw(countQuery, args...).Scan(&total).Error; err != nil {
-		return nil, 0, err
-	}
+	// 1. Xây dựng truy vấn đếm tối giản không chứa JOIN
+	countQuery := `SELECT COUNT(*) FROM cdc_system.failed_sync_logs f WHERE 1=1`
+	countArgs := make([]interface{}, 0, 3)
+	if f.TargetTable != "" {
+		countQuery += ` AND f.target_table = ?`
+		countArgs = append(countArgs, f.TargetTable)
+	}
+	if f.Status != "" {
+		countQuery += ` AND f.status = ?`
+		countArgs = append(countArgs, f.Status)
+	}
+	if f.ErrorType != "" {
+		countQuery += ` AND f.error_type = ?`
+		countArgs = append(countArgs, f.ErrorType)
+	}
+
+	var total int64
+	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&total).Error; err != nil {
+		return nil, 0, err
+	}
+
+	// 2. Xây dựng truy vấn lấy dữ liệu phân trang (giữ nguyên failedLogsBase cũ)
+	query := failedLogsBase
+	args := make([]interface{}, 0, 4)
+	if f.TargetTable != "" {
+		query += ` AND f.target_table = ?`
+		args = append(args, f.TargetTable)
+	}
+	if f.Status != "" {
+		query += ` AND f.status = ?`
+		args = append(args, f.Status)
+	}
+	if f.ErrorType != "" {
+		query += ` AND f.error_type = ?`
+		args = append(args, f.ErrorType)
+	}
 
 	pagedQuery := query + ` ORDER BY f.created_at DESC OFFSET ? LIMIT ?`
 	pagedArgs := append(args, (page-1)*pageSize, pageSize)
```

---

## 3. Kế hoạch Kiểm tra (Verification Plan)

### 3.1. Kiểm thử tự động (Automated Tests)
- Chạy unit/integration test suite của `recon` service để xác minh các truy vấn vẫn hoạt động chính xác và trả về dữ liệu đúng định dạng:
  ```bash
  go test -v ./internal/infra/persistence/recon/...
  ```

### 3.2. Kiểm thử hiệu năng thủ công (Manual Verification)
- Chúng ta sẽ tạo một file test script Go (`/Users/trainguyen/Documents/work/agent/memory/workspaces/FixSlowSqlRecon20260715/benchmark_test.go` hoặc tương đương) kết nối vào database thực tế của môi trường local.
- Thực thi đo đạc thời gian chạy (Execution Time) của query cũ và query mới với 100 lần lặp để so sánh:
  - Thời gian chạy trung bình của truy vấn `ListLatest` (Cũ vs Mới).
  - Thời gian chạy trung bình của truy vấn `CountFailedLogs` (Cũ vs Mới).
- Kết quả benchmark sẽ được ghi nhận chi tiết vào file `06_test_cases_optimize_slow_sql.md`.
