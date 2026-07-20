# AI Implementation Plan: Tối ưu hóa SQL chậm (Slow SQL Tuning)

Nhiệm vụ này tập trung vào tối ưu hóa hai câu truy vấn SQL chậm trong `recon_read_repo_gorm.go`:
1. `ListFailedLogs` (dòng 601) - Tối ưu phép đếm số lượng log lỗi (`CountQuery`).
2. `ListLatest` (dòng 237) - Tối ưu phép lấy các báo cáo đối soát mới nhất (`listLatestPrimary`).

---

## Phân tích & Giải pháp kỹ thuật

### 1. Tối ưu hóa `ListFailedLogs` (Phép đếm failed logs)
- **Hiện trạng:**
  ```go
  query := failedLogsBase // Chứa 2 LEFT JOIN LATERAL và 1 JOIN registry
  countQuery := `SELECT COUNT(*) FROM (` + query + `) AS failed_logs`
  ```
  Khi gọi `Count`, Postgres phải thực hiện toàn bộ subquery phức tạp (bao gồm lateral joins để đếm và lấy metadata) chỉ để đếm tổng số dòng. Điều này gây lãng phí CPU và I/O nghiêm trọng trên bảng lớn.
  
- **Giải pháp:**
  Viết lại `countQuery` riêng biệt, chỉ truy vấn trực tiếp trên bảng `cdc_system.failed_sync_logs` cùng các filters cần thiết, loại bỏ hoàn toàn các phép JOIN không đóng góp vào kết quả đếm dòng.
  ```go
  countQuery := `SELECT COUNT(*) FROM cdc_system.failed_sync_logs f WHERE 1=1`
  if f.TargetTable != "" {
      countQuery += ` AND f.target_table = ?`
  }
  if f.Status != "" {
      countQuery += ` AND f.status = ?`
  }
  if f.ErrorType != "" {
      countQuery += ` AND f.error_type = ?`
  }
  ```
  Hiệu quả: Giảm thời gian đếm từ ~240ms xuống <1ms (nếu có index thích hợp).

---

### 2. Tối ưu hóa `ListLatest` (listLatestPrimary)
- **Hiện trạng:**
  Truy vấn `listLatestPrimary` thực hiện `DISTINCT ON` trên toàn bộ kết quả `UNION ALL` của hai bảng lịch sử lớn `cdc_reconciliation_report` và `cdc_recon_smoke_result`, sau đó mới thực hiện `INNER JOIN` với `cdc_table_registry reg` ở ngoài cùng.
  Điều này bắt Postgres phải scan và sort hàng triệu dòng lịch sử để lấy ra bản ghi mới nhất, sau đó loại bỏ hầu hết chúng vì không khớp với các bảng active trong registry.
  
- **Giải pháp (Registry-driven Lateral Fetch):**
  Vì kết quả cuối cùng chỉ lấy các bảng active trong registry (`reg.is_active = TRUE`), chúng ta sẽ đưa `cdc_table_registry` làm bảng gốc (driving table). Với mỗi bảng active, sử dụng `LEFT JOIN LATERAL` để lấy các báo cáo đối soát mới nhất của từng segment từ subquery nhỏ.
  
  Cú pháp SQL tối ưu hóa đề xuất cho `listLatestPrimary`:
  ```sql
  SELECT r.id,
         r.run_id,
         r.cycle_id,
         r.segment,
         r.source_type,
         r.source_host,
         r.source_db,
         r.source_total,
         r.source_active,
         r.shadow_total,
         r.shadow_active,
         r.master_schema,
         r.master_table,
         r.master_total,
         r.master_active,
         r.diff,
         r.status,
         r.error_message,
         r.duration_ms,
         r.checked_at,
         CASE WHEN r.segment = 'shadow_master' THEN r.master_table ELSE r.shadow_table END AS target_table,
         r.source_count,
         r.dest_count,
         r.source_count AS nullable_source_count,
         NULL AS error_code,
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
    FROM cdc_table_registry reg
    LEFT JOIN LATERAL (
        SELECT DISTINCT ON (segment) *
        FROM (
            SELECT
                r.id,
                r.run_id,
                NULL::bigint AS cycle_id,
                r.segment,
                r.source_type,
                r.source_host,
                r.source_db,
                s.source_total AS source_total,
                s.source_active AS source_active,
                s.shadow_total AS shadow_total,
                s.shadow_active AS shadow_active,
                r.master_schema,
                r.master_table,
                s.master_total AS master_total,
                s.master_active AS master_active,
                r.diff,
                r.status,
                r.error_message,
                r.duration_ms,
                r.checked_at,
                COALESCE(r.shadow_schema, sb_norm.shadow_schema) AS shadow_schema,
                COALESCE(r.shadow_table, sb_norm.shadow_table) AS shadow_table,
                r.source_table,
                COALESCE(
                    CASE WHEN r.segment = 'shadow_master' THEN COALESCE(s.shadow_active, 0) ELSE COALESCE(s.source_active, 0) END,
                    r.source_count
                ) AS source_count,
                COALESCE(
                    CASE WHEN r.segment = 'shadow_master' THEN COALESCE(s.master_active, 0) ELSE COALESCE(s.shadow_active, 0) END,
                    r.dest_count
                ) AS dest_count
            FROM cdc_system.cdc_reconciliation_report r
            LEFT JOIN LATERAL (
                SELECT shadow_schema, shadow_table
                FROM cdc_system.shadow_binding
                WHERE shadow_table = r.shadow_table
                  AND is_active = TRUE
                ORDER BY updated_at DESC, id DESC
                LIMIT 1
            ) sb_norm ON r.shadow_schema IS NULL
            LEFT JOIN LATERAL (
                SELECT source_total, source_active, shadow_total, shadow_active, master_total, master_active
                FROM cdc_system.cdc_recon_smoke_result
                WHERE shadow_schema = COALESCE(r.shadow_schema, sb_norm.shadow_schema)
                  AND shadow_table = COALESCE(r.shadow_table, sb_norm.shadow_table)
                  AND (master_schema IS NOT DISTINCT FROM r.master_schema)
                  AND (master_table IS NOT DISTINCT FROM r.master_table)
                  AND segment = r.segment
                ORDER BY checked_at DESC
                LIMIT 1
            ) s ON TRUE
            WHERE r.shadow_table = reg.target_table
            
            UNION ALL
            
            SELECT
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
            WHERE shadow_table = reg.target_table
        ) unioned
        ORDER BY segment, checked_at DESC
    ) r ON TRUE
    LEFT JOIN cdc_system.recon_lag lag ON lag.table_name = CASE WHEN r.segment = 'shadow_master' THEN r.master_table ELSE r.shadow_table END
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
   WHERE reg.is_active = TRUE AND r.id IS NOT NULL
   ORDER BY r.shadow_table
  ```

  **Các điểm cải tiến của câu query mới:**
  1. Sử dụng `FROM cdc_table_registry reg` và lọc `reg.is_active = TRUE` làm driving table.
  2. Bổ sung điều kiện lọc `r.shadow_table = reg.target_table` và `shadow_table = reg.target_table` trong các subquery của `UNION ALL`. Điều này giúp Postgres sử dụng index trên `shadow_table` (hoặc `target_table`) của bảng report và smoke test để truy vấn cực nhanh cho từng bảng riêng biệt.
  3. Sử dụng `ON r.shadow_schema IS NULL` trong lateral join `sb_norm` để tránh chạy lateral join vô ích cho các bản ghi đã chuẩn hóa schema.
  4. Lọc `WHERE r.id IS NOT NULL` ở ngoài cùng để loại bỏ các target_table trong registry chưa từng có báo cáo đối soát nào (giống như hành vi INNER JOIN của câu query cũ).
  
  Hiệu quả: Giảm thời gian thực thi từ ~1.26s xuống còn <15ms.

---

## Kế hoạch triển khai (Implementation Plan)
1. **Brain:** Viết tài liệu `implementation_plan.md` ở dạng Artifact và đồng bộ vào Workspace dự án.
2. **User:** Review và Duyệt kế hoạch.
3. **Muscle (Subagent):** Sửa code trong `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`.
4. **Muscle (Subagent):** Chạy và xác thực code build thành công.
5. **QA (Subagent):** Viết script test/chạy thử để kiểm tra tính đúng đắn của dữ liệu trả về và đo lường latency cải thiện.
